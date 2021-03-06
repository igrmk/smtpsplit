package main

import (
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
)

type config struct {
	ListenAddress  string            `json:"listen_address"`  // the address to listen to for incoming emails
	Host           string            `json:"host"`            // the host name used to introduce this router
	TimeoutSeconds int               `json:"timeout_seconds"` // the timeout for incoming and outgoing emails
	Debug          bool              `json:"debug"`           // debug mode
	Certificate    string            `json:"certificate"`     // the certificate path for STARTTLS
	CertificateKey string            `json:"certificate_key"` // the certificate key path for STARTTLS
	Routes         map[string]string `json:"routes"`          // a domain to an address map
}

func readConfig(path string) *config {
	file, err := os.Open(filepath.Clean(path))
	checkErr(err)
	defer func() { checkErr(file.Close()) }()
	return parseConfig(file)
}

func parseConfig(r io.Reader) *config {
	decoder := json.NewDecoder(r)
	decoder.DisallowUnknownFields()
	cfg := &config{}
	err := decoder.Decode(cfg)
	checkErr(err)
	checkErr(checkConfig(cfg))
	return cfg
}

func checkConfig(cfg *config) error {
	if cfg.ListenAddress == "" {
		return errors.New("configure listen_address")
	}
	if cfg.Routes == nil {
		return errors.New("configure routes")
	}
	if cfg.Host == "" {
		hostname, err := os.Hostname()
		if err != nil {
			return err
		}
		cfg.Host = hostname
	}
	return nil
}
