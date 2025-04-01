// SPDX-FileCopyrightText: (C) 2022 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package wsproxy

import (
	"fmt"
	"github.com/open-edge-platform/app-orch-deployment/app-resource-manager/internal/model"
	"github.com/open-edge-platform/orch-library/go/pkg/errors"
	"sync"
)

type CounterType int

const (
	CounterTypeUnknown = iota
	CounterTypeIP
	CounterTypeAccount
)

func (t CounterType) String() string {
	return [...]string{"CounterTypeUnknown", "CounterTypeIP", "CounterTypeAccount"}[t]
}

func NewCounter(configPath string, counterType CounterType) Counter {
	return &counter{
		counterType: counterType,
		configPath:  configPath,
		store:       make(map[string]int),
	}
}

//go:generate mockery --name Counter --filename counter_mock.go --structname MockCounter
type Counter interface {
	Increase(key string) error
	Decrease(key string) error
	Print() string
}

type counter struct {
	counterType CounterType
	configPath  string
	store       map[string]int
	mu          sync.RWMutex
}

func (c *counter) Print() string {
	result := ""
	c.mu.Lock()
	defer c.mu.Unlock()
	for k, v := range c.store {
		result += fmt.Sprintf("%s: %d / ", k, v)
	}
	return result
}

func (c *counter) Increase(key string) error {
	limit, err := c.getMaxLimit()
	if err != nil {
		msg := fmt.Sprintf("failed to retrieve limit value in config file: err %v", err)
		return errors.NewNotFound(msg)
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	// store does not have count value for the received key - add new entry
	v, ok := c.store[key]
	if !ok {
		c.store[key] = 1
	}

	// store has the counter for the received key but value exceeds the max limit
	if limit > 0 && (v+1) > limit {
		msg := fmt.Sprintf("exceeding count limit: limit %v, key %v", limit, key)
		return errors.NewForbidden(msg)
	}

	// store has the counter for the received key but value does not exceed the max limit or no limit is set
	c.store[key] = v + 1

	return nil
}

func (c *counter) Decrease(key string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	v, ok := c.store[key]
	if !ok {
		msg := fmt.Sprintf("failed to find the value from counter: key %v", key)
		return errors.NewNotFound(msg)
	}

	if v < 1 {
		msg := fmt.Sprintf("failed to decrease value since the decreased value will be negative: key %v, value %v", key, v)
		return errors.NewInvalid(msg)
	}

	c.store[key] = v - 1

	return nil
}

func (c *counter) getMaxLimit() (int, error) {
	switch c.counterType {
	case CounterTypeIP:
		return c.getMaxLimitPerIP()
	case CounterTypeAccount:
		return c.getMaxLimitPerAccount()
	default:
		msg := fmt.Sprintf("received counter type is invalid - not defined: counterType %v", c.counterType.String())
		return 0, errors.NewInvalid(msg)
	}
}

func (c *counter) getMaxLimitPerIP() (int, error) {
	configModel, err := model.GetConfigModel(c.configPath)
	if err != nil {
		return 0, err
	}

	return configModel.WebSocketServer.SessionLimitPerIP, nil
}

func (c *counter) getMaxLimitPerAccount() (int, error) {
	configModel, err := model.GetConfigModel(c.configPath)
	if err != nil {
		return 0, err
	}

	return configModel.WebSocketServer.SessionLimitPerAccount, nil
}
