// SPDX-FileCopyrightText: (C) 2022 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package kubevirt

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"slices"

	grpc_auth "github.com/grpc-ecosystem/go-grpc-middleware/auth"
	resourcev2 "github.com/open-edge-platform/app-orch-deployment/app-resource-manager/api/nbi/v2/resource/v2"
	"github.com/open-edge-platform/app-orch-deployment/app-resource-manager/internal/adm"
	"github.com/open-edge-platform/app-orch-deployment/app-resource-manager/internal/model"
	"github.com/open-edge-platform/app-orch-deployment/app-resource-manager/internal/opa"
	"github.com/open-edge-platform/app-orch-deployment/app-resource-manager/internal/utils/k8serrors"
	"github.com/open-edge-platform/app-orch-deployment/app-resource-manager/internal/wsproxy"
	"github.com/open-edge-platform/orch-library/go/dazl"
	authlib "github.com/open-edge-platform/orch-library/go/pkg/grpc/auth"
	"github.com/open-edge-platform/orch-library/go/pkg/openpolicyagent"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/types/known/timestamppb"
	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"
)

const (
	AnnotationKeyForAppID         = "meta.helm.sh/release-name"
	AnnotationKeyForVMDescription = "resource.orchestrator.apis/description"
	VNCWebSocketPrefix            = "vnc"
	ContextMetadataTokenKey       = "Bearer"
	OIDCServerURL                 = "OIDC_SERVER_URL"
	HTTPHeaderOriginKey           = "Origin"
	HTTPHeaderForwardedHostKey    = "X-Forwarded-Host"
	HTTPHeaderForwardedForKey     = "X-Forwarded-For"

	CookieKeyCloakTokens      = "keycloak-tokens"
	CookieKeyCloakTokenPrefix = "keycloak-token-"
)

var log = dazl.GetPackageLogger()

func NewManager(configPath string, admClient adm.Client, tokenCheck bool) Manager {
	oidcURL := os.Getenv(OIDCServerURL)
	return &manager{
		configPath: configPath,
		admClient:  admClient,
		oidcURL:    oidcURL,
		tokenCheck: tokenCheck,
	}
}

//go:generate mockery --name KubevirtClient --filename kubevirtclient_mock.go --structname MockKubevirtClient --srcpkg=kubevirt.io/client-go/kubecli
//go:generate mockery --name VirtualMachineInterface --filename virtualmachineinterface_mock.go --structname MockVirtualMachineInterface --srcpkg=kubevirt.io/client-go/kubecli
//go:generate mockery --name VirtualMachineInstanceInterface --filename virtualmachineinstanceinterface_mock.go --structname MockVirtualMachineInstanceInterface --srcpkg=kubevirt.io/client-go/kubecli
//go:generate mockery --name StreamInterface --filename streaminterface_mock.go --structname MockStreamInterface --srcpkg=kubevirt.io/client-go/kubecli
//go:generate mockery --name Manager --filename kubevirt_manager_mock.go --structname MockKubevirtManager
type Manager interface {
	GetVMWorkloads(ctx context.Context, appID string, clusterID string) ([]*resourcev2.AppWorkload, error)
	StartVM(ctx context.Context, appID string, clusterID string, vmID string) error
	StopVM(ctx context.Context, appID string, clusterID string, vmID string) error
	RestartVM(ctx context.Context, appID string, clusterID string, vmID string) error
	GetVNCAddress(ctx context.Context, appID string, clusterID string, vmID string) (string, error)
	GetVNCWebSocketHandler(ctx context.Context, opaClient openpolicyagent.ClientWithResponsesInterface, ipSessionCounter wsproxy.Counter, accountSessionCounter wsproxy.Counter) func(w http.ResponseWriter, r *http.Request)
}

type manager struct {
	configPath string
	admClient  adm.Client
	oidcURL    string
	tokenCheck bool
}

func (m *manager) GetVMWorkloads(ctx context.Context, appID string, clusterID string) ([]*resourcev2.AppWorkload, error) {
	results := make([]*resourcev2.AppWorkload, 0)

	kubevirtClient, err := m.getKubevirtClient(ctx, clusterID)
	if err != nil {
		log.Warnw("Failed to get kubevirt client", dazl.String("ClusterID", clusterID), dazl.Error(err))
		return nil, err
	}

	vmList, err := m.getKubevirtVirtualMachineList(ctx, kubevirtClient, appID)
	if err != nil {
		log.Warnw("Failed to list virtual machines", dazl.String("AppID", appID), dazl.Error(err))
		return nil, err
	}

	for _, vm := range vmList {

		results = append(results, &resourcev2.AppWorkload{
			Type:          resourcev2.AppWorkload_TYPE_VIRTUAL_MACHINE,
			Id:            string(vm.ObjectMeta.UID),
			Name:          vm.ObjectMeta.Name,
			Namespace:     vm.Namespace,
			CreateTime:    timestamppb.New(vm.ObjectMeta.CreationTimestamp.Time),
			WorkloadReady: vm.Status.Ready,
			Workload: &resourcev2.AppWorkload_VirtualMachine{
				VirtualMachine: &resourcev2.VirtualMachine{
					Status:      convertVMStatusV2(vm.Status.PrintableStatus),
					AdminStatus: convertAdminStatusV2(vm.Spec.Running),
				},
			},
		})
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].Id < results[j].Id
	})

	return results, nil

}

func (m *manager) StartVM(ctx context.Context, appID string, clusterID string, vmID string) error {
	kubevirtClient, err := m.getKubevirtClient(ctx, clusterID)
	if err != nil {
		log.Warnw("Failed to get kubevirt client", dazl.String("ClusterID", clusterID), dazl.Error(err))
		return err
	}

	vm, err := m.getKubevirtVirtualMachine(ctx, kubevirtClient, appID, vmID)
	if err != nil {
		log.Warnw("Failed to get application virtual machine", dazl.String("AppID", appID), dazl.Error(err))
		return err
	}
	err = kubevirtClient.VirtualMachine(vm.Namespace).Start(ctx, vm.Name, &v1.StartOptions{})
	if err != nil {
		log.Warnw("Failed to start application virtual machine", dazl.String("AppID", appID), dazl.Error(err))
		return k8serrors.K8sToTypedError(err)
	}
	return nil
}

func (m *manager) StopVM(ctx context.Context, appID string, clusterID string, vmID string) error {
	kubevirtClient, err := m.getKubevirtClient(ctx, clusterID)
	if err != nil {
		log.Warnw("Failed to get kubevirt client", dazl.String("ClusterID", clusterID), dazl.Error(err))
		return err
	}

	vm, err := m.getKubevirtVirtualMachine(ctx, kubevirtClient, appID, vmID)
	if err != nil {
		log.Warnw("Failed to get application virtual machine", dazl.String("AppID", appID), dazl.Error(err))
		return err
	}
	err = kubevirtClient.VirtualMachine(vm.Namespace).Stop(ctx, vm.Name, &v1.StopOptions{})
	if err != nil {
		log.Warnw("Failed to stop application virtual machine", dazl.String("AppID", appID), dazl.Error(err))
		return k8serrors.K8sToTypedError(err)
	}
	return nil
}

func (m *manager) RestartVM(ctx context.Context, appID string, clusterID string, vmID string) error {
	kubevirtClient, err := m.getKubevirtClient(ctx, clusterID)
	if err != nil {
		log.Warnw("Failed to get kubevirt client", dazl.String("ClusterID", clusterID), dazl.Error(err))
		return err
	}

	vm, err := m.getKubevirtVirtualMachine(ctx, kubevirtClient, appID, vmID)
	if err != nil {
		log.Warnw("Failed to get application virtual machine", dazl.String("AppID", appID), dazl.Error(err))
		return err
	}
	err = kubevirtClient.VirtualMachine(vm.Namespace).Restart(ctx, vm.Name, &v1.RestartOptions{})
	if err != nil {
		log.Warnw("Failed to restart application virtual machine", dazl.String("AppID", appID), dazl.Error(err))
		return k8serrors.K8sToTypedError(err)
	}
	return nil
}

func (m *manager) GetVNCAddress(ctx context.Context, appID string, clusterID string, vmID string) (string, error) {
	cfgModel, err := model.GetConfigModel(m.configPath)
	if err != nil {
		log.Warnw("Failed to get configuration information", dazl.Error(err))
		return "", err
	}

	// validation
	// cluster ID is valid
	kubevirtClient, err := m.getKubevirtClient(ctx, clusterID)
	if err != nil {
		log.Warnw("Failed to get kubevirt client", dazl.String("ClusterID", clusterID), dazl.Error(err))
		return "", err
	}

	// app ID and VM ID is valid
	_, err = m.getKubevirtVirtualMachine(ctx, kubevirtClient, appID, vmID)
	if err != nil {
		log.Warnw("Failed to get application virtual machine", dazl.String("AppID", appID), dazl.Error(err))
		return "", err
	}

	// e.g., wss://vnc.kind.internal/vnc/appID/clusterID/vmID
	tokenString := ""
	if m.oidcURL != "" {
		tokenString, err = grpc_auth.AuthFromMD(ctx, ContextMetadataTokenKey)
		if err != nil {
			return "", err
		}
	}

	activeProjectID, err := opa.GetActiveProjectID(ctx)
	if err != nil {
		log.Errorf("failed to get active project ID, error: %v", err)
		activeProjectID = "nil"
	}

	if tokenString == "" {
		addr := fmt.Sprintf("%s://%s/%s/%s/%s/%s/%s", cfgModel.WebSocketServer.Protocol, cfgModel.WebSocketServer.HostName, VNCWebSocketPrefix, activeProjectID, appID, clusterID, vmID)
		return addr, nil
	}

	addr := fmt.Sprintf("%s://%s/%s/%s/%s/%s/%s", cfgModel.WebSocketServer.Protocol, cfgModel.WebSocketServer.HostName, VNCWebSocketPrefix, activeProjectID, appID, clusterID, vmID)
	return addr, nil
}

func (m *manager) GetVNCWebSocketHandler(ctx context.Context, opaClient openpolicyagent.ClientWithResponsesInterface,
	ipSessionCounter wsproxy.Counter, accountSessionCounter wsproxy.Counter) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		log.Infow("vnc access request received", dazl.String("path", r.URL.Path))
		cfgModel, err := model.GetConfigModel(m.configPath)
		if err != nil {
			msg := fmt.Sprintf("failed to get config model, err: %v", err)
			err := setErrorStatus(w, http.StatusInternalServerError, msg)
			if err != nil {
				log.Errorw("failed to encode error status message", dazl.Error(err))
			}
			return
		}

		// validation
		// http origin validation
		// directly accessing ARM for VNC should be blocked: (i) no origins; (2) wrong origins; (3) not forwarded from Traefik or Nginx
		log.Debugf("received header info %+v", *r)

		origin := r.Header.Get(HTTPHeaderOriginKey)
		forwardedHostName := r.Header.Get(HTTPHeaderForwardedHostKey)
		remoteIP := strings.Split(r.Header.Get(HTTPHeaderForwardedForKey), ",")[0]

		// 1. validate origin in HTTP/HTTPS request header
		if !slices.Contains(cfgModel.WebSocketServer.AllowedOrigins, origin) {
			msg := fmt.Sprintf("failed to validate http origin due to no or wrong origins in http header, received origins: %v, allowed origins: %v", origin, cfgModel.WebSocketServer.AllowedOrigins)
			err := setErrorStatus(w, http.StatusForbidden, msg)
			if err != nil {
				log.Errorw("failed to encode error status message", dazl.Error(err))
			}
			return
		}
		// 2. validate forwarded host name
		// since VNC WS is coming through Traefik or Nginx, it should be set
		if forwardedHostName != cfgModel.WebSocketServer.HostName {
			msg := fmt.Sprintf("failed to validate http origin due to no or wrong forwarded host info, received forwarded host: %v, expected forwarded host: %v", forwardedHostName, cfgModel.WebSocketServer.HostName)
			err := setErrorStatus(w, http.StatusForbidden, msg)
			if err != nil {
				log.Errorw("failed to encode error status message", dazl.Error(err))
			}
			return
		}

		// token validation
		tokenString := ""
		if m.tokenCheck {
			tokenString, err = getCombinedJWTToken(r)
			if err != nil {
				msg := fmt.Sprintf("cannot get JWT token to Authenticate: %v", err)
				err := setErrorStatus(w, http.StatusBadRequest, msg)
				if err != nil {
					log.Errorw("failed to encode error status message", dazl.Error(err))
				}
				return
			}
		}

		path, err := newVNCPath(r.URL.Path)
		if err != nil {
			msg := fmt.Sprintf("failed to parse VNC path, error: %v", err)
			err := setErrorStatus(w, http.StatusNotFound, msg)
			if err != nil {
				log.Errorw("failed to encode error status message", dazl.Error(err))
			}
			return
		}

		accounts := make([]string, 0)

		outCtx := context.Background()

		if opa.IsOPAEnabled() {
			log.Infow("processing authorization for VNC websocket")
			// put Token to context

			md := metadata.New(map[string]string{"authorization": fmt.Sprintf("Bearer %s", tokenString), "activeprojectid": path.projectID})
			outCtx = metadata.NewIncomingContext(ctx, md)

			outCtx, err = authlib.AuthenticationInterceptor(outCtx)
			if err != nil {
				msg := fmt.Sprintf("failed to parse token string, error: %v", err)
				err := setErrorStatus(w, http.StatusUnauthorized, msg)
				if err != nil {
					log.Errorw("failed to encode error status message", dazl.Error(err))
				}
				return
			}

			// authorization with JWT/RBAC
			vncReq := &resourcev2.GetVNCRequest{
				AppId:            path.appID,
				ClusterId:        path.clusterID,
				VirtualMachineId: path.vmID,
			}
			if err := opa.IsAuthorized(outCtx, vncReq, opaClient); err != nil {
				msg := fmt.Sprintf("access denied by OPA rules, error: %v", err)
				err := setErrorStatus(w, http.StatusUnauthorized, msg)
				if err != nil {
					log.Errorw("failed to encode error status message", dazl.Error(err))
				}
				return
			}

			md, ok := metadata.FromIncomingContext(outCtx)
			if !ok {
				msg := fmt.Sprintf("failed to get username from token, error: %v", err)
				err := setErrorStatus(w, http.StatusUnauthorized, msg)
				if err != nil {
					log.Errorw("failed to encode error status message", dazl.Error(err))
				}
				return
			}

			accounts = md.Get("preferred_username")
		}

		// limit sessions
		// per IP address
		err = ipSessionCounter.Increase(remoteIP)
		if err != nil {
			msg := fmt.Sprintf("session request per IP exceeds the limit, ip: %v, error: %v", remoteIP, err)
			err := setErrorStatus(w, http.StatusTooManyRequests, msg)
			if err != nil {
				log.Errorw("failed to encode error status message", dazl.Error(err))
			}
			return
		}
		defer func() {
			err = ipSessionCounter.Decrease(remoteIP)
			if err != nil {
				log.Warnw("ignorable; failed to decrease the counter", dazl.String("ip", remoteIP), dazl.Error(err))
			}
		}()
		log.Debugw("ip session counter increased", dazl.String("ip", remoteIP), dazl.String("counter", ipSessionCounter.Print()))

		// per account
		for _, account := range accounts {
			err = accountSessionCounter.Increase(account)
			if err != nil {
				msg := fmt.Sprintf("session request per account exceeds the limit, account: %v, err: %v", account, err)
				err := setErrorStatus(w, http.StatusTooManyRequests, msg)
				if err != nil {
					log.Errorw("failed to encode error status message", dazl.Error(err))
				}
				return
			}
			log.Debugw("ip session counter decreased", dazl.String("ip", remoteIP), dazl.String("counter", ipSessionCounter.Print()))
		}
		log.Debugw("account session counter increased", dazl.Strings("accounts", accounts), dazl.String("counter", accountSessionCounter.Print()))

		defer func() {
			for _, account := range accounts {
				err = accountSessionCounter.Decrease(account)
				if err != nil {
					log.Warnw("ignorable; failed to decrease the counter", dazl.String("account", account), dazl.Error(err))
				}
			}
			log.Debugw("account session counter decreased", dazl.Strings("accounts", accounts), dazl.String("counter", accountSessionCounter.Print()))
		}()

		kubevirtClient, err := m.getKubevirtClient(outCtx, path.clusterID)
		if err != nil {
			msg := fmt.Sprintf("failed to get kubevirt client, error: %v", err)
			err := setErrorStatus(w, http.StatusNotFound, msg)
			if err != nil {
				log.Errorw("failed to encode error status message", dazl.Error(err))
			}
			return
		}

		vncStream, err := m.getVNCStream(outCtx, kubevirtClient, path.appID, path.vmID)
		if err != nil {
			msg := fmt.Sprintf("failed to get VNC stream, error: %v", err)
			err := setErrorStatus(w, http.StatusNotFound, msg)
			if err != nil {
				log.Errorw("failed to encode error status message", dazl.Error(err))
			}
			return
		}

		upgrader := m.getVNCWebSocketUpgrader()
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			msg := fmt.Sprintf("failed to call websocket upgrade, error: %v", err)
			err := setErrorStatus(w, http.StatusBadRequest, msg)
			if err != nil {
				log.Errorw("failed to encode error status message", dazl.Error(err))
			}
			return
		}
		// set read limit
		conn.SetReadLimit(int64(cfgModel.WebSocketServer.ReadLimitByte))

		pipeInReader, pipeInWriter := io.Pipe()
		pipeOutReader, pipeOutWriter := io.Pipe()

		k8ResChan := make(chan error)
		writeStop := make(chan error)
		readStop := make(chan error)

		defer pipeInReader.Close()
		defer pipeInWriter.Close()
		defer pipeOutReader.Close()
		defer pipeOutWriter.Close()
		defer conn.Close()

		go func() {
			k8ResChan <- vncStream.Stream(kubecli.StreamOptions{
				In:  pipeInReader,
				Out: pipeOutWriter,
			})
		}()

		// write to WebSocket <- pipeOutReader
		dlIdleTimeout := time.Duration(cfgModel.WebSocketServer.DlIdleTimeoutMin) * time.Minute
		ulIdleTimeout := time.Duration(cfgModel.WebSocketServer.UlIdleTimeoutMin) * time.Minute
		go func() {
			var err error
			if cfgModel.WebSocketServer.DlIdleTimeoutMin == 0 {
				_, err = io.Copy(WebSocketWriter{conn}, pipeOutReader)
			} else {
				_, err = wsproxy.CopyBufferWithIdleTimeout(WebSocketWriter{conn}, pipeOutReader, nil, dlIdleTimeout)
			}
			readStop <- err
		}()

		// read from WebSocket -> pipeInWriter
		go func() {
			var err error
			if cfgModel.WebSocketServer.UlIdleTimeoutMin == 0 {
				_, err = io.Copy(pipeInWriter, WebSocketReader{conn})
			} else {
				_, err = wsproxy.CopyBufferWithIdleTimeout(pipeInWriter, WebSocketReader{conn}, nil, ulIdleTimeout)
			}
			writeStop <- err
		}()

		select {
		case err = <-readStop:
			log.Debugw("VNC out reader stopped", dazl.Error(err))
		case err = <-writeStop:
			log.Debugw("VNC in writer stopped", dazl.Error(err))
		case err = <-k8ResChan:
			log.Debugw("VNC disconnected from KubeVirt side", dazl.Error(err))
		}

		if err != nil {
			return
		}
	}
}

// getCombinedJWTToken extracts all cookies with the name "token" and combines their values to form a JWT token
func getCombinedJWTToken(req *http.Request) (string, error) {
	var tokenParts []string

	numTokenCookiesStr, err := req.Cookie(CookieKeyCloakTokens)
	if err != nil {
		log.Errorf("Error retrieving keycloak-tokens cookie: %v", err)
		return "", err
	}

	// Iterate over all cookies in the request
	numTokenCookies, err := strconv.Atoi(numTokenCookiesStr.Value)
	if err != nil {
		log.Errorf("Error parsing keycloak-tokens cookie value: %v", err)
		return "", err
	}
	for num := 0; num < numTokenCookies; num++ {
		tokenName := CookieKeyCloakTokenPrefix + strconv.Itoa(num)
		log.Infof("tokenName : %s", tokenName)
		tokenSubPart, err := req.Cookie(tokenName)
		if err != nil {
			log.Errorf("Error retrieving token cookie: %v", err)
			return "", err
		}

		tokenParts = append(tokenParts, tokenSubPart.Value)
	}

	// Combine the token parts to form the JWT token
	jwtToken := strings.Join(tokenParts, "")
	return jwtToken, nil
}
