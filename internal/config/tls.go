package config

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"
)

func SetupTLSConfig(cfg TLSConfig) (*tls.Config, error) {
	var err error
	tlsConfig := &tls.Config{}

	// Check if the server is configured with a certificate and key.
	// If so, load the certificate and key.
	if cfg.CertFile != "" && cfg.KeyFile != "" {
		tlsConfig.Certificates = make([]tls.Certificate, 1)
		// load the certificate and key
		tlsConfig.Certificates[0], err = tls.LoadX509KeyPair(
			cfg.CertFile,
			cfg.KeyFile,
		)
		// check whether the certificate and key were loaded without error
		if err != nil {
			return nil, err
		}
	}

	// Check if the server is configured with a CA certificate.
	if cfg.CAFile != "" {
		// try read the CA certificate
		b, err := ioutil.ReadFile(cfg.CAFile)
		// check whether the CA certificate was read without error
		if err != nil {
			return nil, err
		}

		// create a new certificate pool
		ca := x509.NewCertPool()
		// append the CA certificate to the certificate pool
		ok := ca.AppendCertsFromPEM([]byte(b))
		if !ok {
			// if the CA certificate was not appended to the certificate pool, return an error with a formatted message
			return nil, fmt.Errorf(
				"failed to parse root certificate: %q",
				cfg.CAFile,
			)
		}

		// check if the server is configured as a server
		if cfg.Server {
			tlsConfig.ClientCAs = ca
			tlsConfig.ClientAuth = tls.RequireAndVerifyClientCert
		} else {
			tlsConfig.RootCAs = ca
		}
		// set the server name to the server address
		tlsConfig.ServerName = cfg.ServerAddress
	}
	return tlsConfig, nil
}

type TLSConfig struct {
	CertFile      string
	KeyFile       string
	CAFile        string
	ServerAddress string
	Server        bool
}
