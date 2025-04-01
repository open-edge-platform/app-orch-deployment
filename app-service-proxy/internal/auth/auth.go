// SPDX-FileCopyrightText: (C) 2023 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package auth

import (
	"context"
	"crypto/rand"
	"fmt"
	"net/http"

	"math/big"
	"os"
	"strconv"
	"time"

	"github.com/open-edge-platform/app-orch-deployment/app-service-proxy/internal/vault"
	"github.com/sirupsen/logrus"
	//_ "k8s.io/client-go/plugin/pkg/client/auth"
)

const (
	DefaultTokenTTLHours = 100
	TokenTTLHoursEnvKey  = "TOKEN_TTL_HOURS" // #nosec G101
)

var ConnectAuthorizer = func(req *http.Request) (string, bool, error) {
	//var client admclient.AdmClient
	ctx := context.Background()
	id := req.Header.Get("x-tunnel-id")
	token := req.Header.Get("x-api-tunnel-token")
	logrus.Infof("Websocket connection request, id=%s", id)

	vaultManager := vault.NewManager()
	tokenV, err := vaultManager.GetToken(ctx, id)
	if err != nil {
		logrus.Errorf("Error getting token from vault - err %+v", err)
		return "", false, fmt.Errorf("could not get token")
	}

	if time.Since(tokenV.UpdatedTime) > time.Duration(tokenV.TTLHours)*time.Hour {
		logrus.Errorf("token is expired - token %+v for cluster %s", *tokenV, id)
		return "", false, fmt.Errorf("token expired - %+v", *tokenV)
	}

	if token != tokenV.Value {
		logrus.Errorf("Invalid token, id=%s", id)
		return "", false, fmt.Errorf("invalid token")
	}

	logrus.Infof("Cluster %s is authenticated", id)
	return id, true, nil
}

var RenewTokenAuthorizer = func(req *http.Request, id string) (bool, error) {
	ctx := context.Background()

	token := req.Header.Get("Authorization")

	vaultManager := vault.NewManager()
	tokenV, err := vaultManager.GetToken(ctx, id)
	if err != nil {
		logrus.Errorf("Error getting token from vault - err %+v", err)
		return false, err
	}

	if token != tokenV.Value {
		return false, fmt.Errorf("invalid token")
	}

	logrus.Infof("Cluster %s is authenticated to renew token", id)
	return true, nil
}

func GenerateToken() (string, error) {
	tokenLength := 54
	characters := "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	charsLength := big.NewInt(int64(len(characters)))
	token := make([]byte, tokenLength)
	for i := range token {
		r, err := rand.Int(rand.Reader, charsLength)
		if err != nil {
			return "", err
		}
		token[i] = characters[r.Int64()]
	}
	return string(token), nil
}

func GetTokenTTLHours() (int, error) {
	ttlStr := os.Getenv(TokenTTLHoursEnvKey)
	if ttlStr == "" {
		logrus.Errorf("failed to get Token TTL Hours from env variables %s; set default token TTL Hours %dh", TokenTTLHoursEnvKey, DefaultTokenTTLHours)
		return DefaultTokenTTLHours, nil
	}

	ttlHour, err := strconv.Atoi(ttlStr)
	if err != nil {
		logrus.Errorf("failed to convert string type ttlHour %v to integer; set default token TTL hours %dh - err: %+v", ttlStr, DefaultTokenTTLHours, err)
		return DefaultTokenTTLHours, nil
	}

	return ttlHour, nil
}
