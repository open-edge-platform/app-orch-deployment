// SPDX-FileCopyrightText: (C) 2024 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package utils

import (
	"context"
	stdErr "errors"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"

	clientv1beta1 "github.com/open-edge-platform/app-orch-deployment/app-deployment-manager/pkg/appdeploymentclient/v1beta1"
	"github.com/open-edge-platform/app-orch-deployment/app-deployment-manager/pkg/utils/k8serrors"
	"github.com/open-edge-platform/app-orch-deployment/app-deployment-manager/pkg/utils/ratelimiter"
	"github.com/open-edge-platform/orch-library/go/dazl"
	fleetv1alpha1 "github.com/rancher/fleet/pkg/apis/fleet.cattle.io/v1alpha1"
	"github.com/rancher/wrangler/v3/pkg/genericcondition"
	"google.golang.org/grpc/metadata"
	v1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/utils/clock"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/open-edge-platform/app-orch-deployment/app-deployment-manager/api/v1beta1"
	"github.com/open-edge-platform/app-orch-deployment/app-deployment-manager/internal/catalogclient"
	"github.com/open-edge-platform/orch-library/go/pkg/auth"
)

var (
	Clock         clock.Clock = clock.RealClock{}
	log                       = dazl.GetPackageLogger()
	utilsLog                  = dazl.GetPackageLogger().WithSkipCalls(1)
	gitCaCert     []byte
	gitCaCertLock sync.RWMutex
)

const (
	appRefFilter = ".metadata.deployment-package-ref"
)

// CreateClient creates the k8s client
func CreateClient(kubeconfig string) (*clientv1beta1.AppDeploymentClient, error) {
	var (
		config *rest.Config
		err    error
	)

	if kubeconfig == "" {
		log.Infof("using in-cluster configuration")
		config, err = rest.InClusterConfig()
	} else {
		log.Infof("using configuration from '%s'", kubeconfig)
		config, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
	}

	if err != nil {
		return nil, err
	}
	qps, burst, err := ratelimiter.GetRateLimiterParams()
	if err != nil {
		log.Warnw("Failed to get rate limiter parameters", dazl.Error(err))
		return nil, err
	}

	config.QPS = float32(qps)
	config.Burst = int(burst)

	crClient, err := clientv1beta1.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	return crClient, nil
}

// LogActivity logs activity of user
func LogActivity(ctx context.Context, verb string, thing string, args ...string) {
	var user []string
	md, ok := metadata.FromIncomingContext(ctx)
	if ok && len(md.Get("name")) > 0 {
		user = md.Get("name")
	} else {
		user = md.Get("client_id")
	}

	utilsLog.Infof("User '%s' %s %s %s", user, verb, thing, strings.Join(args, "/"))
}

func CreateRestConfig(kubeConfig string) (*rest.Config, error) {
	var config *rest.Config
	var err error
	if kubeConfig == "" {
		log.Info("KubeConfig is empty using in cluster kubeConfig")
		config, err = rest.InClusterConfig()
		if err != nil {
			log.Warnw("Failed to get in cluster kubeConfig", dazl.Error(err))
			return nil, err
		}

	} else {
		config, err = clientcmd.BuildConfigFromFlags("", kubeConfig)
		if err != nil {
			log.Warnw("Failed to build k8s config from flags", dazl.Error(err))
			return nil, err
		}
	}
	qps, burst, err := ratelimiter.GetRateLimiterParams()
	if err != nil {
		log.Warnw("Failed to get rate limiter parameters", dazl.Error(err))
		return nil, err
	}
	config.QPS = float32(qps)
	config.Burst = int(burst)

	return config, nil
}

func WriteFile(basedir string, filename string, data []byte) error {
	if err := os.MkdirAll(basedir, os.ModePerm); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(basedir, filename), data, 0600); err != nil {
		return err
	}
	return nil
}

func GetAppRef(d *v1beta1.Deployment) string {
	return fmt.Sprintf("%s-%s",
		d.Spec.DeploymentPackageRef.Name,
		d.Spec.DeploymentPackageRef.Version)
}

func IsDeployed(ctx context.Context, c client.Reader, d *v1beta1.Deployment) (bool, error) {
	appRef := GetAppRef(d)

	deploymentList := &v1beta1.DeploymentList{}
	if err := c.List(ctx, deploymentList, client.MatchingFields{appRefFilter: appRef}); err != nil {
		return false, err
	}

	dActiveProjectID := d.Labels[string(v1beta1.AppOrchActiveProjectID)]
	for _, i := range deploymentList.Items {
		iActiveProjectID := i.Labels[string(v1beta1.AppOrchActiveProjectID)]
		if dActiveProjectID == iActiveProjectID {
			if d.Name != i.Name {
				// returns true if there is another deployment using the same deployment package
				// and version as the given deployment.
				return true, nil
			}
		}
	}

	return false, nil
}

func HandleIsDeployed(ctx context.Context, c catalogclient.CatalogClient, vaultAuthClient auth.VaultAuth, d *v1beta1.Deployment, isDeployed bool) error {
	setOrUnset := "unset"
	if isDeployed {
		setOrUnset = "set"
	}

	activeProjectID := d.Labels[string(v1beta1.AppOrchActiveProjectID)]
	ctx, cancel, err := AddToOutgoingContext(ctx, vaultAuthClient, activeProjectID, true)
	if err != nil {
		return fmt.Errorf("failed to %s isDeployed (%v)", setOrUnset, err)
	}
	defer cancel()

	err = catalogclient.UpdateIsDeployed(ctx, c,
		d.Spec.DeploymentPackageRef.Name,
		d.Spec.DeploymentPackageRef.Version,
		isDeployed)
	if err != nil {
		return fmt.Errorf("failed to %s isDeployed (%v)", setOrUnset, err)
	}

	return nil
}

func GetGenericCondition(g *[]genericcondition.GenericCondition, cond string) (*genericcondition.GenericCondition, bool) {
	for i := range *g {
		c := &(*g)[i]
		if c.Type == cond {
			return c, true
		}
	}
	return nil, false
}

func GetState(bd *fleetv1alpha1.BundleDeployment) v1beta1.StateType {
	c, ok := GetGenericCondition(&bd.Status.Conditions, "Ready")
	if ok && bd.Spec.DeploymentID == bd.Status.AppliedDeploymentID {
		if c.Status == v1.ConditionTrue && bd.Status.Ready {
			return v1beta1.Running
		}
		// Map "Modified" state to "Ready"
		if bd.Status.Ready && !bd.Status.NonModified && bd.Status.Display.State == "Modified" {
			return v1beta1.Running
		}
	}
	return v1beta1.Down
}

func GetMessage(bd *fleetv1alpha1.BundleDeployment) string {
	message := ""
	if !bd.Status.Ready {
		for _, condition := range bd.Status.Conditions {
			if condition.Status == v1.ConditionFalse && condition.Message != "" {
				message = AppendMessage(message, condition.Message)
			}
		}
	}
	return message
}

func GetAppID(bd *fleetv1alpha1.BundleDeployment) string {
	return bd.Labels[string(v1beta1.LabelBundleName)]
}

func GetAppName(bd *fleetv1alpha1.BundleDeployment) string {
	return bd.Labels[string(v1beta1.LabelAppName)]
}

func GetDeploymentGeneration(bd *fleetv1alpha1.BundleDeployment) int64 {
	if genStr, ok := bd.Labels["deploymentGeneration"]; ok {
		if gen, err := strconv.ParseInt(genStr, 10, 64); err == nil {
			return gen
		} else {
			log.Error("failed to parse deploymentGeneration: %v", err)
		}
	}
	return 0
}

func AppendMessage(orig string, next string) string {
	if orig == "" {
		return next
	}
	return fmt.Sprintf("%s; %s", orig, next)
}

func MessageFromError(err error) string {
	if err != nil {
		return err.Error()
	}
	return ""
}

func CreateNamespace(ctx context.Context, s *kubernetes.Clientset, namespaceName string) error {
	namespace := &v1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: namespaceName,
		},
	}

	// Check if namespace found, if not then create
	_, err := s.CoreV1().Namespaces().Get(ctx, namespaceName, metav1.GetOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			_, err := s.CoreV1().Namespaces().Create(ctx, namespace, metav1.CreateOptions{})
			if err != nil {
				return k8serrors.K8sToTypedError(err)
			}

			log.Infof("namespace '%s' created successfully", namespace.Name)
			return nil
		}
		return k8serrors.K8sToTypedError(err)
	}

	return nil
}

func UpdateNsLabels(ctx context.Context, s *kubernetes.Clientset, namespaceName string) error {
	namespace, err := s.CoreV1().Namespaces().Get(ctx, namespaceName, metav1.GetOptions{})
	if err != nil {
		return k8serrors.K8sToTypedError(err)
	}

	if namespace.Labels == nil {
		namespace.Labels = make(map[string]string)
	}
	namespace.Labels[string(v1beta1.FleetRsProxy)] = "true"

	_, err = s.CoreV1().Namespaces().Update(ctx, namespace, metav1.UpdateOptions{})
	if err != nil {
		return k8serrors.K8sToTypedError(err)
	}

	log.Infof("added label '%s' to namespace '%s'", string(v1beta1.FleetRsProxy), namespaceName)
	return nil
}

// CreateRoleBinding will create a role binding using sa default from project id namespace
func CreateRoleBinding(ctx context.Context, s *kubernetes.Clientset, saNamespace string, name string, remoteNamespace string) error {
	roleBinding := &rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-%s", name, saNamespace),
			Namespace: remoteNamespace,
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:      "ServiceAccount",
				Name:      "default",
				Namespace: saNamespace,
			},
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "Role",
			Name:     name,
		},
	}

	_, err := s.RbacV1().RoleBindings(roleBinding.Namespace).Create(ctx, roleBinding, metav1.CreateOptions{})
	if err != nil {
		if apierrors.IsAlreadyExists(err) {
			log.Infof("rolebinding '%s' in namespace '%s' already exists", roleBinding.Name, roleBinding.Namespace)
			return nil
		}
		return k8serrors.K8sToTypedError(err)
	}

	log.Infof("rolebinding '%s' created in namespace '%s'", roleBinding.Name, roleBinding.Namespace)
	return nil
}

// Create secret.
func CreateSecret(ctx context.Context, s *kubernetes.Clientset, namespace string, secretName string, data map[string]string, basicAuth bool) error {
	secret := v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      secretName,
			Namespace: namespace,
			// Need to set owner reference so this Secret gets deleted
			OwnerReferences: []metav1.OwnerReference{},
		},
		Type: v1.SecretTypeOpaque,
	}

	if basicAuth {
		secret.Type = v1.SecretTypeBasicAuth
	}

	secret.StringData = data

	_, err := s.CoreV1().Secrets(namespace).Create(ctx, &secret, metav1.CreateOptions{})
	if err != nil {
		return err
	}

	return nil
}

// Delete secret.
func DeleteSecret(ctx context.Context, s *kubernetes.Clientset, namespace string, secretName string) error {
	err := s.CoreV1().Secrets(namespace).Delete(ctx, secretName, metav1.DeleteOptions{})
	if err != nil {
		return k8serrors.K8sToTypedError(err)
	}

	return nil
}

// Get secret.
func GetSecretValue(ctx context.Context, s *kubernetes.Clientset, namespace string, secretName string) (*v1.Secret, error) {
	value, err := s.CoreV1().Secrets(namespace).Get(ctx, secretName, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	return value, nil
}

func UpdateStatusCondition(conds []metav1.Condition, condType string, status metav1.ConditionStatus, reason string, err error) []metav1.Condition {
	condition := metav1.Condition{
		Type:               condType,
		LastTransitionTime: metav1.NewTime(Clock.Now()),
		Status:             status,
		Reason:             reason,
		Message:            MessageFromError(err),
	}

	for i, c := range conds {
		// Skip unrelated conditions
		if c.Type != condType {
			continue
		}
		// If this update doesn't contain a state transition, don't update
		// the conditions LastTransitionTime to Now()
		if c.Status == status {
			condition.LastTransitionTime = c.LastTransitionTime
		}
		// Overwrite the existing condition
		conds[i] = condition
		return conds
	}

	// If not found an existing condition of this type, simply insert
	// the new condition into the slice
	conds = append(conds, condition)
	return conds
}

func getCtxWithToken(ctx context.Context, vaultAuthClient auth.VaultAuth) (context.Context, error) {
	token, err := vaultAuthClient.GetM2MToken(ctx)
	if err != nil {
		return nil, err
	}
	if token == "" {
		return nil, fmt.Errorf("token is empty")
	}

	outCtx := metadata.AppendToOutgoingContext(ctx, "authorization", "Bearer "+token)
	err = vaultAuthClient.Logout(outCtx)
	return outCtx, err
}

// AddToOutgoingContext adds the ActiveProjectID and M2M token to context to pass to catalog service.
func AddToOutgoingContext(ctx context.Context, vaultAuthClient auth.VaultAuth, activeProjectID string, setTimeout bool) (context.Context, context.CancelFunc, error) {
	ctx, err := getCtxWithToken(ctx, vaultAuthClient)
	if err != nil {
		return ctx, nil, err
	}

	outCtx := metadata.AppendToOutgoingContext(ctx, "ActiveProjectID", activeProjectID)
	if setTimeout {
		ctx, cancel := context.WithTimeout(outCtx, 10*time.Second)
		return ctx, cancel, nil
	}
	return outCtx, nil, nil
}

// GetGitCaCert returns git ca cert
func GetGitCaCert() string {
	gitCaCertLock.RLock()
	defer gitCaCertLock.RUnlock()
	return string(gitCaCert)
}

// SetGitCaCert sets git ca cert
func SetGitCaCert(cert []byte) {
	gitCaCertLock.Lock()
	defer gitCaCertLock.Unlock()
	gitCaCert = cert
}

// WatchGitCaCertFile monitors for new Git CA Cert file. This is needed to handle for certificate rotation for secure webservers
func WatchGitCaCertFile(ctx context.Context, gitCaCertFolder, gitCaCertFile string) {
	absFilePath := filepath.Join(gitCaCertFolder, gitCaCertFile)
	// Check if the ca certificate exists in the first place. If it does not, it is because we are using
	// insecure git server or the certificate to be used with git server is already part of cert pool.
	// If the ca certificate does not exist, no more action is needed in this function and we can safely return.
	if _, err := os.Stat(absFilePath); err != nil && stdErr.Is(err, os.ErrNotExist) {
		log.Error("ca.crt file %v does not exist, not monitoring", absFilePath)
		return
	}
	// To start with read the initial ca.crt file. Then start the watcher for monitoring changes due to certificate rotation.
	cert, err := os.ReadFile(absFilePath)
	if err != nil {
		log.Fatal("unable to read the ca cert file: %v", err)
	}
	log.Infof("successfully loaded initial ca cert: %v", string(cert))
	SetGitCaCert(cert)

	// Start FS watcher
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatal("could not start file system watcher: %v", err)
	}
	defer watcher.Close()

	done := make(chan bool)
	go func() {
		log.Info("started watching git ca cert file")
		for {
			select {
			case event, ok := <-watcher.Events:
				log.Infof("received event with op: %v", event.Op)
				if !ok {
					log.Warnw("fs watcher closed")
					done <- true
					return
				}
				// Check for events for the ca.crt file
				if event.Op&fsnotify.Write == fsnotify.Write || event.Op&fsnotify.Create == fsnotify.Create {
					log.Info("new updates to cert file available, loading changes")
					// Reload the certificate or handle the change here
					cert, err := os.ReadFile(absFilePath) //nolint: govet
					if err != nil {
						log.Fatal("unable to read the ca cert file: %v", err)
					}
					log.Infof("successfully read new ca cert: %v", string(cert))
					SetGitCaCert(cert)
				}
			case err, ok := <-watcher.Errors: //nolint: govet
				if !ok {
					log.Warnw("fs watcher closed")
					done <- true
					return
				}
				log.Warnw("error received: %v", dazl.Error(err))
			case <-ctx.Done():
				log.Warnw("context cancelled, shutting down ca cert file watcher")
				done <- true
				return
			}
		}
	}()

	// Add the directory containing the ca.crt file to the watcher to monitor for changes
	if err = watcher.Add(gitCaCertFolder); err != nil {
		log.Fatal("unable to add watcher: %v", err)
	}

	// Block until a signal is received
	<-done
}

func ToInt32Clamped(i int) int32 {
	if i < 0 {
		return 0
	}
	if i > math.MaxInt32 {
		return math.MaxInt32
	}
	return int32(i)
}
