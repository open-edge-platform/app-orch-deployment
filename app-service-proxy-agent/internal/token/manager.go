// SPDX-FileCopyrightText: (C) 2024 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package token

import (
	"context"
	"fmt"
	"github.com/open-edge-platform/app-orch-deployment/app-service-proxy-agent/internal/cert"
	"github.com/sirupsen/logrus"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

var (
	iterationPeriod = 1 * time.Minute
	retryPeriod     = 10 * time.Second
	maxIterations   = 100
)

func NewManager() Manager {
	return &manager{}
}

type Manager interface {
	Start(ctx context.Context)
}

type manager struct {
	token       string
	ttlHours    time.Duration
	updatedTime time.Time

	insecureSkipVerify bool
	agentID            string

	serverURL string
}

func (m *manager) Start(ctx context.Context) {
	go m.run(ctx)
}

func (m *manager) run(_ context.Context) {
	logrus.Info("start token manager")
	for {
		logrus.Infof("(re)triggered token refresh - triggered after %s from the previous trial", iterationPeriod.String())

		// wait before run
		time.Sleep(iterationPeriod)

		var err error
		// prep
		err = m.retryFunc(m.tokenRefreshPrep, retryPeriod)
		if err != nil {
			logrus.Errorf("failed to prepare for token refresh request - err %+v", err)
			continue
		}

		// it shouldn't be retried directly - need to run prep func again and then retry after retry period
		err = m.requestTokenRefresh()
		if err != nil {
			logrus.Errorf("failed to request token refresh - err %+v", err)
			continue
		}

		logrus.Infof("waiting new token is delievered")
		err = m.retryFunc(m.verifyTokenRefresh, retryPeriod)
		if err != nil {
			logrus.Errorf("verificiation failed for token refresh - err %+v", err)
			continue
		}
		logrus.Infof("token is successfully delivered")
	}
}

func (m *manager) retryFunc(f func() error, period time.Duration) error {
	var err error
	for i := 0; i < maxIterations; i++ {
		err = f()
		if err == nil {
			return nil
		}
		time.Sleep(period)
	}
	return err
}

// tokenRefreshPrep is to prepare token refresh before request
func (m *manager) tokenRefreshPrep() error {
	var err error
	m.token, err = m.getToken()
	if err != nil {
		logrus.Errorf("failed to get token from secret file - err %+v", err)
		return err
	}

	m.ttlHours, err = m.getTTLHours()
	if err != nil {
		logrus.Errorf("failed to get TTL hours from secret file - err %+v", err)
		return err
	}

	m.updatedTime, err = m.getUpdatedTime()
	if err != nil {
		logrus.Errorf("failed to get updated time from secret file - err %+v", err)
		return err
	}

	m.insecureSkipVerify = true
	skip, ok := os.LookupEnv("INSECURE_SKIP_VERIFY")
	if !ok || skip == "" || skip == "false" {
		m.insecureSkipVerify = false
	}

	m.agentID, ok = os.LookupEnv("AGENT_ID")
	if !ok || m.agentID == "" {
		logrus.Errorf("failed to get AGENT_ID")
		return fmt.Errorf("AGENT_ID is not set")
	}

	serverURL, ok := os.LookupEnv("PROXY_SERVER_URL")
	if !ok || serverURL == "" {
		logrus.Errorf("failed to get PROXY_SERVER_URL")
		return fmt.Errorf("PROXY_SERVER_URL is not set")
	}
	m.serverURL = serverURL
	m.serverURL = strings.Replace(m.serverURL, "wss://", "https://", 1)
	m.serverURL = strings.Replace(m.serverURL, "ws://", "http://", 1)

	m.serverURL = fmt.Sprintf("%s/renew/%s", m.serverURL, m.agentID)

	return nil
}

// requestTokenRefresh requests token refresh to proxy
func (m *manager) requestTokenRefresh() error {
	var err error

	logrus.Info("waiting until token is expired")

	// set timer
	remainingExpTime := m.ttlHours - time.Since(m.updatedTime)
	if remainingExpTime < 0 {
		remainingExpTime = 0
	}
	timer := time.NewTimer(remainingExpTime)

	logrus.Infof("remaining timer - %s; will call renew API after timer expired", remainingExpTime.String())

	// wait until timer expired
	<-timer.C

	// start requesting
	logrus.Info("start requesting token refresh")
	client := m.getHTTPClient()

	req, err := http.NewRequest("GET", m.serverURL, nil)
	if err != nil {
		logrus.Errorf("failed to create http request message - err %+v", err)
		return err
	}

	// set token
	req.Header.Set("Authorization", m.token)

	// call /renew API
	logrus.Infof("calling renew API URL: %s", m.serverURL)
	resp, err := client.Do(req)
	if err != nil {
		logrus.Errorf("failed to get response from proxy - err %+v", err)
		return err
	}

	msg, err := io.ReadAll(resp.Body)
	if err != nil {
		logrus.Warnf("failed to read response body - %+v", err)
	}

	switch resp.StatusCode {
	case http.StatusOK:
		// renew done
		logrus.Infof("renew request is successfully delivered")
	case http.StatusUnauthorized:
		// token is not matched
		err = fmt.Errorf("token is invalid - err %s", msg)
		logrus.Error(err)
		return err
	case http.StatusTooEarly:
		// token renew requested too early
		err = fmt.Errorf("token renew requested too early - err %s", msg)
		logrus.Error(err)
		return err
	case http.StatusInternalServerError:
		// other types errors
		err = fmt.Errorf("renew failed - err %s", msg)
		logrus.Error(err)
		return err
	default:
		// unexpected status codes
		err = fmt.Errorf("unknown status code captured - code: %d / body: %s", resp.StatusCode, msg)
		logrus.Error(err)
		return err
	}

	return nil
}

func (m *manager) verifyTokenRefresh() error {
	// wait until the secret files are changed - with backoff timer
	newToken, err := m.getToken()
	if err != nil {
		logrus.Errorf("failed to get token from secret file - err %+v", err)
		return err
	}

	newUpdatedTime, err := m.getUpdatedTime()
	if err != nil {
		logrus.Errorf("failed to get updated time from secret file - err %+v", err)
		return err
	}

	if m.token == newToken || m.updatedTime == newUpdatedTime {
		err := fmt.Errorf("new token is not installed yet, waiting new tokens")
		logrus.Error(err)
		return err
	}
	return nil
}

func (m *manager) getHTTPClient() *http.Client {
	tr := &http.Transport{
		TLSClientConfig: cert.GetTLSConfigs(m.insecureSkipVerify),
	}

	return &http.Client{Transport: tr}
}

func (m *manager) getToken() (string, error) {
	raw, err := os.ReadFile(TokenFilePath)
	if err != nil {
		logrus.Errorf("failed to read token secret file - err %+v", err)
		return "", err
	}
	return string(raw), nil
}

func (m *manager) getTTLHours() (time.Duration, error) {
	raw, err := os.ReadFile(TTLHoursFilePath)
	if err != nil {
		logrus.Errorf("failed to read ttl hours secret file - err %+v", err)
		return 0, err
	}

	ttlHours, err := strconv.Atoi(string(raw))
	if err != nil {
		logrus.Errorf("failed to convert string type ttlHours to integer - err %+v", err)
		return 0, err
	}

	return time.Duration(ttlHours) * time.Hour, nil
}

func (m *manager) getUpdatedTime() (time.Time, error) {
	raw, err := os.ReadFile(UpdatedTimeFilePath)
	if err != nil {
		logrus.Errorf("failed to read updated time secret file")
		return time.Time{}, err
	}

	updatedTime, err := time.Parse(time.DateTime, string(raw))
	if err != nil {
		logrus.Errorf("failed to parse string type updated time to time.Time - err %+v", err)
		return time.Time{}, err
	}

	return updatedTime, nil
}
