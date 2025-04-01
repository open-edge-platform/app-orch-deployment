// SPDX-FileCopyrightText: (C) 2023 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package model

import (
	"github.com/go-playground/validator/v10"
	"github.com/open-edge-platform/orch-library/go/dazl"
	"gopkg.in/yaml.v2"
	"os"
)

var log = dazl.GetPackageLogger()

type ConfigModel struct {
	AppDeploymentManager AppDeploymentManager `yaml:"appDeploymentManager" validate:"required"`
	WebSocketServer      WebSocketServer      `yaml:"webSocketServer" validate:"required"`
}

type AppDeploymentManager struct {
	Endpoint string `yaml:"endpoint" validate:"required"`
}

type WebSocketServer struct {
	Protocol               string   `yaml:"protocol" validate:"required"`
	HostName               string   `yaml:"hostName" validate:"required"`
	SessionLimitPerIP      int      `yaml:"sessionLimitPerIP" validate:"min=0"`
	SessionLimitPerAccount int      `yaml:"sessionLimitPerAccount" validate:"min=0"`
	ReadLimitByte          int      `yaml:"readLimitByte" validate:"min=0"`
	DlIdleTimeoutMin       int      `yaml:"dlIdleTimeoutMin" validate:"min=0"`
	UlIdleTimeoutMin       int      `yaml:"ulIdleTimeoutMin" validate:"min=0"`
	AllowedOrigins         []string `yaml:"allowedOrigins" validate:"required"`
}

func GetConfigModel(configPath string) (*ConfigModel, error) {
	rawYamlFile, err := os.ReadFile(configPath)
	if err != nil {
		log.Errorw("Failed to read config.yaml file", dazl.Error(err))
		return nil, err
	}

	cfgModel := &ConfigModel{}
	err = yaml.Unmarshal(rawYamlFile, cfgModel)
	if err != nil {
		log.Errorw("Failed to unmarshal config.yaml file", dazl.Error(err))
		return nil, err
	}

	validate := validator.New()
	err = validate.Struct(cfgModel)
	if err != nil {
		log.Errorw("Failed to validate config model", dazl.Error(err))
		return nil, err
	}

	log.Debugf("Received config.yaml file %v", cfgModel)
	return cfgModel, nil
}
