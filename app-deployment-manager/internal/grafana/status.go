// SPDX-FileCopyrightText: (C) 2023 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package grafana

import (
	"context"
	"encoding/json"
	"fmt"
	grafanaapi "github.com/grafana/grafana-api-golang-client"
	"github.com/open-edge-platform/orch-library/go/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"net/url"
)

const (
	GrafanaURL                    string = "http://kube-prometheus-stack-grafana.orch-platform"
	GrafanaSecretName             string = "kube-prometheus-stack-grafana"
	GrafanaSecretKeyAdminUsername string = "admin-user"
	GrafanaSecretKeyAdminPassword string = "admin-password"
)

//go:generate mockery --name SecretInterface --filename mockery_secretinterface.go --structname MockerySecretInterface --output mockery --srcpkg=k8s.io/client-go/kubernetes/typed/core/v1
//go:generate mockery --name CoreV1Interface --filename mockery_corev1interface.go --structname MockeryCoreV1Interface --output mockery --srcpkg=k8s.io/client-go/kubernetes/typed/core/v1

func getGrafanaURL(_ context.Context) (string, error) {
	// todo: update this function to get URL from Helm chart or somewhere, not hardcoded value
	return GrafanaURL, nil
}

func getGrafanaCredentials(ctx context.Context) (string, string, error) {
	// todo: update this function to get credential from Keycloak
	config, err := rest.InClusterConfig()
	if err != nil {
		return "", "", err
	}

	cs, err := kubernetes.NewForConfig(config)
	if err != nil {
		return "", "", err
	}

	secret, err := cs.CoreV1().Secrets("").Get(ctx, GrafanaSecretName, metav1.GetOptions{})
	if err != nil {
		return "", "", err
	}

	username := string(secret.Data[GrafanaSecretKeyAdminUsername])
	password := string(secret.Data[GrafanaSecretKeyAdminPassword])

	return username, password, nil
}

func GetGrafanaDashboardUID(_ context.Context, dashboardJSONString string) (string, error) {
	var objMap map[string]json.RawMessage
	err := json.Unmarshal([]byte(dashboardJSONString), &objMap)
	if err != nil {
		return "", err
	}

	if uid, ok := objMap["uid"]; ok {
		// delete quotes at the beginning and end of uid field
		return string(uid)[1 : len(string(uid))-1], nil
	}

	return "", errors.NewNotFound("uid not found in dashboard JSON file")
}

func SetGrafanaDashboardUID(_ context.Context, dashboardJSONString string, uid string) (string, error) {
	var objMap map[string]json.RawMessage
	err := json.Unmarshal([]byte(dashboardJSONString), &objMap)
	if err != nil {
		return "", err
	}

	objMap["uid"] = json.RawMessage(fmt.Sprintf("\"%s\"", uid))
	newJSON, err := json.Marshal(objMap)
	return string(newJSON), err
}

func IsGrafanaDashboardReady(ctx context.Context, uid string) error {
	gURL, err := getGrafanaURL(ctx)
	if err != nil {
		return err
	}

	username, password, err := getGrafanaCredentials(ctx)
	if err != nil {
		return err
	}

	gcli, err := grafanaapi.New(gURL, grafanaapi.Config{
		BasicAuth: url.UserPassword(username, password),
	})

	if err != nil {
		return err
	}

	_, err = gcli.DashboardByUID(uid)
	if err != nil {
		return err
	}

	return nil
}
