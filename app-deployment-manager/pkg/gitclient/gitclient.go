// SPDX-FileCopyrightText: (C) 2023 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package gitclient

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	_http "net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/open-edge-platform/orch-library/go/dazl"

	vaultAPI "github.com/hashicorp/vault/api"
	"github.com/open-edge-platform/app-orch-deployment/app-deployment-manager/pkg/utils"
	"github.com/open-edge-platform/app-orch-deployment/app-deployment-manager/pkg/vault"
	"github.com/open-edge-platform/orch-library/go/pkg/errors"

	"code.gitea.io/sdk/gitea"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/plumbing/transport"
	"github.com/go-git/go-git/v5/plumbing/transport/http"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/client"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/codecommit"
	"github.com/aws/aws-sdk-go/service/codecommit/codecommitiface"
)

// A Repository provides a set of operations on a remote Git repo
type Repository interface {
	ExistsOnRemote() (bool, error)
	Initialize(basedir string) error
	Clone(basedir string) error
	CommitFiles() error
	PushToRemote() error
	Delete() error

	// CodeCommit
	CreateCodeCommitClient() error
	CreateRepo() error
	DeleteRepo() error
	// End CodeCommit
}

type GitClient struct {
	Server       string
	User         string
	Password     string
	RepoName     string
	RemoteURL    string
	CABundle     []byte
	Proxy        string
	GitProvider  string
	Repo         *git.Repository
	RemoteConfig config.RemoteConfig
	CcClient     codecommitiface.CodeCommitAPI
}

var log = dazl.GetPackageLogger()

func getGitClientWithSecretService(repoName, server, gitProvider, gitProxy, gitCaCert string) (Repository, error) {
	vaultManager := vault.NewManager(utils.GetSecretServiceEndpoint(), utils.GetServiceAccount(), utils.GetSecretServiceMount())
	vaultClient, err := vaultManager.GetVaultClient(context.Background())
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := vaultManager.Logout(context.Background(), vaultClient); err != nil {
			log.Errorf("failed to logout from vault: %v", err)
		}
	}()
	user, err := vaultManager.GetSecretValueString(context.Background(), vaultClient, utils.GetSecretServiceGitServicePath(), utils.GetSecretServiceGitServiceKVKeyUsername())
	if err != nil {
		return nil, err
	}
	password, err := vaultManager.GetSecretValueString(context.Background(), vaultClient, utils.GetSecretServiceGitServicePath(), utils.GetSecretServiceGitServiceKVKeyPassword())
	if err != nil {
		return nil, err
	}

	remote := getRemoteURL(server, user, repoName, gitProvider)
	return &GitClient{
		Server:      server,
		User:        user,
		Password:    password,
		CABundle:    []byte(gitCaCert),
		RepoName:    repoName,
		RemoteURL:   remote,
		Proxy:       gitProxy,
		GitProvider: gitProvider,
		RemoteConfig: config.RemoteConfig{
			Name: "origin",
			URLs: []string{remote},
		},
	}, nil
}

func getGitClientWithoutSecretService(repoName, server, gitProvider, gitProxy, gitCaCert string) (Repository, error) {
	user, err := utils.GetGitUser()
	if err != nil {
		return nil, err
	}
	password, err := utils.GetGitPassword()
	if err != nil {
		return nil, err
	}

	remote := getRemoteURL(server, user, repoName, gitProvider)
	return &GitClient{
		Server:      server,
		User:        user,
		Password:    password,
		CABundle:    []byte(gitCaCert),
		RepoName:    repoName,
		RemoteURL:   remote,
		Proxy:       gitProxy,
		GitProvider: gitProvider,
		RemoteConfig: config.RemoteConfig{
			Name: "origin",
			URLs: []string{remote},
		},
	}, nil
}

type ClientCreator func(string) (Repository, error)

// Allow overrriding the NewGitClient function for testing error handling in the controller.
// The test framework can easily return errors to the controller's reconcile loop
// to test that the controller reports the error correctly and recovers when the error resolves.
var NewGitClient = func(repoName string) (Repository, error) {
	server, err := utils.GetGitServer()
	if err != nil {
		return nil, err
	}
	gitProvider, err := utils.GetGitProvider()
	if err != nil {
		return nil, errors.NewUnavailable("GIT_PROVIDER env var not set")
	}
	gitProxy := utils.GetGitProxy()
	gitCaCert := utils.GetGitCaCert()
	flag, err := utils.IsSecretServiceEnabled()
	if err != nil {
		return nil, err
	}
	if flag {
		return getGitClientWithSecretService(repoName, server, gitProvider, gitProxy, gitCaCert)
	}

	return getGitClientWithoutSecretService(repoName, server, gitProvider, gitProxy, gitCaCert)
}

// Return the Git remote's URL given a server, user, and UID.
func getRemoteURL(server string, user string, repoName string, gitProvider string) string {
	if gitProvider == "codecommit" {
		return fmt.Sprintf("%s/%s.git", server, repoName)
	}
	return fmt.Sprintf("%s/%s/%s.git", server, user, repoName)
}

func GetRemoteURL(repoName string) (string, error) {
	server, err := utils.GetGitServer()
	if err != nil {
		return "", err
	}
	gitProvider, err := utils.GetGitProvider()
	if err != nil {
		return "", errors.NewUnavailable("GIT_PROVIDER env var not set")
	}

	flag, err := utils.IsSecretServiceEnabled()
	if err != nil {
		return "", err
	}

	if flag {
		return getRemoteURLWithSecretService(repoName, server, gitProvider)
	}

	return getRemoteURLWithoutSecretService(repoName, server, gitProvider)
}

// GetRemoteURLWithCreds will build remote url of repo used by Fleet.
func GetRemoteURLWithCreds(deploymentID string) (string, error) {
	// Get initial remoteUrl
	remoteURL, err := GetRemoteURL(deploymentID)
	if err != nil {
		return "", err
	}

	gitProvider, err := utils.GetGitProvider()
	if err != nil {
		return "", err
	}

	remoteType, ok := os.LookupEnv("FLEET_GIT_REMOTE_TYPE")
	if !ok {
		return "", errors.NewUnavailable("FLEET_GIT_REMOTE_TYPE is not set")
	}

	var (
		user, passwd, awsSSHKeyID string
		vaultManager              vault.Manager
		vaultClient               *vaultAPI.Client
	)

	secretServiceEnabled, err := utils.IsSecretServiceEnabled()
	if err != nil {
		return "", err
	}

	if secretServiceEnabled {
		secretServiceEndpoint := utils.GetSecretServiceEndpoint()

		vaultManager = vault.NewManager(secretServiceEndpoint,
			utils.GetServiceAccount(),
			utils.GetSecretServiceMount())

		vaultClient, err = vaultManager.GetVaultClient(context.Background())
		if err != nil {
			return "", err
		}
		defer func() {
			if err := vaultManager.Logout(context.Background(), vaultClient); err != nil {
				log.Info(fmt.Sprintf("Error logging out from Vault: %v\n", err))
			}
		}()

		user, err = vaultManager.GetSecretValueString(context.Background(),
			vaultClient,
			utils.GetSecretServiceGitServicePath(),
			utils.GetSecretServiceGitServiceKVKeyUsername())
		if err != nil {
			return "", err
		}

		passwd, err = vaultManager.GetSecretValueString(context.Background(),
			vaultClient,
			utils.GetSecretServiceGitServicePath(),
			utils.GetSecretServiceGitServiceKVKeyPassword())
		if err != nil {
			return "", err
		}
	} else {
		user, err = utils.GetGitUser()
		if err != nil {
			return "", err
		}

		passwd, err = utils.GetGitPassword()
		if err != nil {
			return "", err
		}
	}

	switch remoteType {
	case "http":
		fallthrough
	case "https":
		// Only add creds to URL if gitProvider is codecommit
		if gitProvider != "codecommit" {
			return remoteURL, nil
		}

		// Encode any special characters so it can be safely placed inside the URL
		encodedPasswd := url.QueryEscape(passwd)

		// TODO: remove embedding basic auth creds in remote URL for CodeCommit
		// when gitjob v0.1.77 is released
		// TODO: handle https and http properly
		// Build URL
		replaceURL := "https://" + user + ":" + encodedPasswd + "@"
		updatedRemoteURL := strings.Replace(remoteURL, "https://", replaceURL, -1)
		return updatedRemoteURL, nil
	case "ssh":
		gitServerURL, err := url.Parse(remoteURL)
		if err != nil {
			return "", err
		}

		if gitProvider == "codecommit" {
			if secretServiceEnabled {
				awsSSHKeyID, err = vaultManager.GetSecretValueString(context.Background(),
					vaultClient,
					utils.GetSecretServiceAWSServicePath(),
					utils.GetSecretServiceAWSServiceKVKeySSHKey())
				if err != nil {
					return "", err
				}
			} else {
				awsSSHKeyID, err = utils.GetAwsSSHKey()
				if err != nil {
					return "", err
				}
			}

			return strings.Replace(remoteURL, "https://", "ssh://"+awsSSHKeyID+"@", -1), nil
		} else if gitProvider == "gitea" {
			return fmt.Sprintf("git@%s:%s/%s.git", gitServerURL.Hostname(), user, deploymentID), nil
		}
		return "", errors.NewUnavailable("unsupported git provider type")
	default:
		return remoteURL, nil
	}
}

func getRemoteURLWithSecretService(repoName, server, gitProvider string) (string, error) {
	secretServiceEndpoint := utils.GetSecretServiceEndpoint()
	vaultManager := vault.NewManager(secretServiceEndpoint, utils.GetServiceAccount(), utils.GetSecretServiceMount())
	vaultClient, err := vaultManager.GetVaultClient(context.Background())
	if err != nil {
		return "", err
	}
	defer func() {
		if err := vaultManager.Logout(context.Background(), vaultClient); err != nil {
			log.Info(fmt.Sprintf("Error logging out from Vault: %v\n", err))
		}
	}()
	user, err := vaultManager.GetSecretValueString(context.Background(), vaultClient, utils.GetSecretServiceGitServicePath(), utils.GetSecretServiceGitServiceKVKeyUsername())
	if err != nil {
		return "", err
	}
	return getRemoteURL(server, user, repoName, gitProvider), nil
}

func getRemoteURLWithoutSecretService(repoName, server, gitProvider string) (string, error) {
	user, err := utils.GetGitUser()
	if err != nil {
		return "", err
	}
	return getRemoteURL(server, user, repoName, gitProvider), nil
}

// Check if this Repository already exists on the remote server.
func (g *GitClient) ExistsOnRemote() (bool, error) {
	r := git.NewRemote(nil, &g.RemoteConfig)
	_, err := r.List(&git.ListOptions{
		Auth: &http.BasicAuth{
			Username: g.User,
			Password: g.Password,
		},
		CABundle: g.CABundle,
		ProxyOptions: transport.ProxyOptions{
			URL: g.Proxy,
		},
	})

	if err != nil {
		switch {
		case strings.Contains(err.Error(), "repository not found"):
			return false, nil
		case strings.Contains(err.Error(), "authentication required"):
			return false, nil
		default:
			return false, err
		}
	}

	return true, nil
}

// Initialze a local Git repository
func (g *GitClient) Initialize(basedir string) error {
	if g.Repo != nil {
		return errors.NewUnavailable("GitClient already initialized")
	}

	// If g.GitProvider is codecommit, create aws sdk client
	// and create repo with client since cannot with git client
	if g.GitProvider == "codecommit" {
		err := g.CreateCodeCommitClient()
		if err != nil {
			return err
		}

		err = g.CreateRepo()
		if err != nil {
			return err
		}
	}

	if err := os.MkdirAll(basedir, os.ModePerm); err != nil {
		return err
	}

	r, err := git.PlainInit(basedir, false)
	if err != nil {
		return err
	}

	_, err = r.CreateRemote(&g.RemoteConfig)
	if err != nil {
		return err
	}

	g.Repo = r
	return nil
}

// Clone a local Git repository
func (g *GitClient) Clone(basedir string) error {
	if g.Repo != nil {
		return errors.NewUnavailable("GitClient already initialized")
	}
	if err := os.MkdirAll(basedir, os.ModePerm); err != nil {
		return err
	}

	r, err := git.PlainClone(basedir, false, &git.CloneOptions{
		Auth: &http.BasicAuth{
			Username: g.User,
			Password: g.Password,
		},
		CABundle: g.CABundle,
		URL:      g.RemoteURL,
		ProxyOptions: transport.ProxyOptions{
			URL: g.Proxy,
		},
		Depth: 1,
	})

	if err != nil {
		return err
	}

	g.Repo = r
	return nil

}

// Commit any file that has changed to the local Git repository
func (g *GitClient) CommitFiles() error {
	if g.Repo == nil {
		return errors.NewUnavailable("git repo not yet initialized or cloned")
	}
	w, err := g.Repo.Worktree()
	if err != nil {
		return err
	}

	status, err := w.Status()
	if err != nil {
		return err
	}
	if len(status) == 0 {
		// Nothing has changed, just return
		return nil
	}

	err = w.AddGlob("*")
	if err != nil {
		return err
	}

	_, err = w.Commit("Generated Fleet configs", &git.CommitOptions{
		Author: &object.Signature{
			Name:  "App Deployment Manager",
			Email: "adm@app-orch.com",
			When:  time.Now(),
		},
	})
	if err != nil {
		return err
	}
	return nil
}

// Push the modified local Git repository to the remote server
func (g *GitClient) PushToRemote() error {
	if g.Repo == nil {
		return errors.NewUnavailable("git repo not yet initialized or cloned")
	}

	err := g.Repo.Push(&git.PushOptions{
		RemoteName: "origin",
		Auth: &http.BasicAuth{
			Username: g.User,
			Password: g.Password,
		},
		CABundle: g.CABundle,
		ProxyOptions: transport.ProxyOptions{
			URL: g.Proxy,
		},
	})

	if err == git.NoErrAlreadyUpToDate {
		return nil
	}

	return err
}

// getCodeCommitSecrets returns GIT_ACCESSKEY, GIT_SECRET_ACCESSKEY, and GIT_REGION
func (g *GitClient) getCodeCommitSecrets() (string, string, string, error) {
	flag, err := utils.IsSecretServiceEnabled()
	if err != nil {
		return "", "", "", err
	}

	if flag {
		return g.getCodeCommitSecretsWithSecretService()
	}

	return g.getCodeCommitSecretsWithoutSecretService()
}

func (g *GitClient) getCodeCommitSecretsWithSecretService() (string, string, string, error) {
	secretServiceEndpoint := utils.GetSecretServiceEndpoint()
	vaultManager := vault.NewManager(secretServiceEndpoint, utils.GetServiceAccount(), utils.GetSecretServiceMount())
	vaultClient, err := vaultManager.GetVaultClient(context.Background())
	if err != nil {
		return "", "", "", err
	}
	defer func() {
		if err := vaultManager.Logout(context.Background(), vaultClient); err != nil {
			log.Info(fmt.Sprintf("Error logging out from Vault: %v\n", err))
		}
	}()
	accessKeyID, err := vaultManager.GetSecretValueString(context.Background(), vaultClient, utils.GetSecretServiceAWSServicePath(), utils.GetSecretServiceAWSServiceKVKeyAccessKey())
	if err != nil {
		return "", "", "", err
	}
	secretAccessKey, err := vaultManager.GetSecretValueString(context.Background(), vaultClient, utils.GetSecretServiceAWSServicePath(), utils.GetSecretServiceAWSServiceKVKeySecretAccessKey())
	if err != nil {
		return "", "", "", err
	}
	gitRegion, err := vaultManager.GetSecretValueString(context.Background(), vaultClient, utils.GetSecretServiceAWSServicePath(), utils.GetSecretServiceAWSServiceKVKeyRegion())
	if err != nil {
		return "", "", "", err
	}
	return accessKeyID, secretAccessKey, gitRegion, nil
}

func (g *GitClient) getCodeCommitSecretsWithoutSecretService() (string, string, string, error) {
	accessKeyID, err := utils.GetGitAccessKey()
	if err != nil {
		return "", "", "", err
	}
	secretAccessKey, err := utils.GetGitSecretAccessKey()
	if err != nil {
		return "", "", "", err
	}
	gitRegion, err := utils.GetGitRegion()
	if err != nil {
		return "", "", "", err
	}
	return accessKeyID, secretAccessKey, gitRegion, nil
}

// CreateCodeCommitClient creates the AWS sdk client.
func (g *GitClient) CreateCodeCommitClient() error {
	accessKeyID, secretAccessKey, gitRegion, err := g.getCodeCommitSecrets()
	if err != nil {
		return err
	}

	var ccSession client.ConfigProvider
	creds := credentials.NewStaticCredentials(accessKeyID, secretAccessKey, "")
	awsconfig := aws.Config{
		Credentials: creds,
		Region:      aws.String(gitRegion),
	}
	if g.Proxy != "" {
		proxyURL, _ := url.Parse(g.Proxy)
		awsconfig.HTTPClient = &_http.Client{
			Transport: &_http.Transport{
				Proxy: _http.ProxyURL(proxyURL),
			},
		}
	}

	// Create new session with creds, region and proxy defined
	ccSession, err = session.NewSession(&awsconfig)

	if err != nil {
		return err
	}

	// Create codecommit service client with session config
	client := codecommit.New(ccSession)
	g.CcClient = client

	return nil
}

// CreateRepo creates a repository using the AWS sdk client. Only works with Codecommit.
func (g *GitClient) CreateRepo() error {
	createInput := &codecommit.CreateRepositoryInput{
		RepositoryName: &g.RepoName,
	}

	_, err := g.CcClient.CreateRepository(createInput)

	if err != nil {
		return err
	}

	return nil
}

// DeleteRepo delete a repository using the AWS sdk client. Only works with Codecommit.
func (g *GitClient) DeleteRepo() error {
	deleteInput := &codecommit.DeleteRepositoryInput{
		RepositoryName: &g.RepoName,
	}

	_, err := g.CcClient.DeleteRepository(deleteInput)
	if err != nil {
		return err
	}

	return nil
}

// DeleteGitea Deletes a Git repository using the Gitea client. Only works with Gitea.
func (g *GitClient) DeleteGitea() error {
	var giteaClient *gitea.Client
	var err error
	if len(g.CABundle) != 0 {
		// Use CA Cert if one is provided
		caCertPool := x509.NewCertPool()
		if !caCertPool.AppendCertsFromPEM(g.CABundle) {
			log.Errorf("unable to append ca cert: %v", string(g.CABundle))
			return errors.NewInvalid("invalid ca bundle")
		}

		// Create an HTTP client with the custom transport
		httpClient := &_http.Client{
			Transport: &_http.Transport{
				TLSClientConfig: &tls.Config{
					RootCAs:    caCertPool,
					MinVersion: tls.VersionTLS12,
				},
			},
		}

		// Use the custom httpClient for interacting with Gitea
		giteaClient, err = gitea.NewClient(g.Server, gitea.SetBasicAuth(g.User, g.Password), gitea.SetHTTPClient(httpClient))
		if err != nil {
			return err
		}
	} else {
		giteaClient, err = gitea.NewClient(g.Server, gitea.SetBasicAuth(g.User, g.Password))
		if err != nil {
			return err
		}
	}
	_, err = giteaClient.DeleteRepo(g.User, g.RepoName)
	if err != nil {
		return err
	}
	return nil
}

// Delete a Git repository using the AWS SDK client.  Only works with CodeCommit.
func (g *GitClient) DeleteCodeCommit() error {
	err := g.CreateCodeCommitClient()
	if err != nil {
		return err
	}

	err = g.DeleteRepo()
	if err != nil {
		return err
	}

	return nil
}

func (g *GitClient) Delete() error {
	if g.GitProvider == "codecommit" {
		err := g.DeleteCodeCommit()
		if err != nil {
			return err
		}
	} else {
		err := g.DeleteGitea()
		if err != nil {
			return err
		}
	}

	return nil

}
