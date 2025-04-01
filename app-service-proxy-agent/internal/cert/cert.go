// SPDX-FileCopyrightText: (C) 2024 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package cert

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"os"

	"github.com/sirupsen/logrus"
	kerrors "k8s.io/apimachinery/pkg/util/errors"
)

var (
	CertFilePath = "/etc/secrets/ca/cert" // #nosec G101
)

func GetTLSConfigs(insecure bool) *tls.Config {
	if insecure {
		return &tls.Config{
			InsecureSkipVerify: true, // #nosec G402
		}
	} else if _, err := os.Stat(CertFilePath); err == nil {
		caCertPEM, err := os.ReadFile(CertFilePath)
		if err != nil {
			errMsg := fmt.Errorf("Failed to read cert file: %v", err)
			logrus.Error(errMsg)
			panic(errMsg)
		}
		logrus.Infof("Validating CA at %s", CertFilePath)
		if err := ValidateCert(caCertPEM); err != nil {
			errMsg := fmt.Errorf("%s", "Failed to validate certificate")
			logrus.Error(errMsg)
			panic(errMsg)
		}

		certPool, err := x509.SystemCertPool()
		if err != nil {
			logrus.Errorf("Failed to configure certPool: %v", err)
			os.Exit(1)
		}

		ok := certPool.AppendCertsFromPEM(caCertPEM)
		if !ok {
			logrus.Errorf("Failed to parse root certificate")
			os.Exit(1)
		}

		return &tls.Config{
			RootCAs:    certPool,
			MinVersion: tls.VersionTLS13,
			MaxVersion: tls.VersionTLS13,
		}
	}
	return &tls.Config{
		MinVersion: tls.VersionTLS13,
		MaxVersion: tls.VersionTLS13,
	}
}

func ValidateCert(caPEM []byte) error {
	var blocks [][]byte
	for {
		var certDERBlock *pem.Block
		certDERBlock, caPEM = pem.Decode(caPEM)
		if certDERBlock == nil {
			break
		}
		if certDERBlock.Type == "CERTIFICATE" {
			blocks = append(blocks, certDERBlock.Bytes)
		}
	}

	logrus.Infof("Found %d certificates", len(blocks))
	if len(blocks) == 0 {
		return fmt.Errorf("No valid certificates found")
	} else if len(blocks) > 1 {
		logrus.Warnf("Found %d certificates, should be 1", len(blocks))
	}

	blockcount := 0
	errs := []error{}
	for _, block := range blocks {
		_, err := x509.ParseCertificate(block)
		if err != nil {
			logrus.Error(err)
			errs = append(errs, err)
			continue
		}
		logrus.Infof("Certificate #%d", blockcount)
		blockcount = blockcount + 1
	}

	return kerrors.NewAggregate(errs)
}
