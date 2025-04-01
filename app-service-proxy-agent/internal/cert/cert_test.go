// SPDX-FileCopyrightText: (C) 2024 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package cert

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"github.com/stretchr/testify/assert"
	"math/big"
	"os"
	"path/filepath"
	"testing"
)

func testGenCert() ([]byte, error) {
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, err
	}
	template := &x509.Certificate{
		SerialNumber: new(big.Int).SetInt64(0),
	}

	raw, err := x509.CreateCertificate(rand.Reader, template, template, &priv.PublicKey, priv)
	if err != nil {
		return nil, err
	}

	cert := make([]byte, 0)
	buf := bytes.NewBuffer(cert)

	err = pem.Encode(buf, &pem.Block{Type: "CERTIFICATE", Bytes: raw})
	if err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func TestGetTLSConfigs_Insecure(t *testing.T) {
	tls := GetTLSConfigs(true)
	assert.NotNil(t, tls)
}

func TestGetTLSConfigs_Secure_FailedGetCertFilePath(t *testing.T) {
	tls := GetTLSConfigs(false)
	assert.NotNil(t, tls)
}

func TestGetTLSConfigs_Secure_FailedWrongCert(t *testing.T) {
	cert := []byte("test")

	origCertFilePath := CertFilePath
	CertFilePath = filepath.Join(t.TempDir(), "cert")
	defer func() {
		// need to be reverted back to original value for other tests
		CertFilePath = origCertFilePath
	}()

	err := os.WriteFile(CertFilePath, cert, 0600)
	assert.NoError(t, err)

	defer func() {
		if r := recover(); r == nil {
			t.Error("call agent.Run succeed - expected: failed")
		}
	}()
	_ = GetTLSConfigs(false)

}

func TestGetTLSConfigs_Secure(t *testing.T) {
	cert, err := testGenCert()
	assert.NoError(t, err)

	origCertFilePath := CertFilePath
	CertFilePath = filepath.Join(t.TempDir(), "cert")
	defer func() {
		// need to be reverted back to original value for other tests
		CertFilePath = origCertFilePath
	}()

	err = os.WriteFile(CertFilePath, cert, 0600)
	assert.NoError(t, err)

	tls := GetTLSConfigs(false)

	assert.NotNil(t, tls)
}

func TestValidateCert_NilCaPEM(t *testing.T) {
	err := ValidateCert(nil)
	assert.NotNil(t, err)
}
