// Copyright 2016 F. Alan Jones.  All rights reserved.
// Use of this source code is governed by a Mozilla
// license that can be found in the LICENSE file.

package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/ghodss/yaml"
	"gopkg.in/urfave/cli.v1"
)

var configDir string = filepath.Join(os.Getenv("HOME"), ".ketchctl.d")

const (
	configDirMode  os.FileMode = 0775
	configFile     string      = "config"
	configFileMode os.FileMode = 0664
)

// Config is the ketchctl configuration object
type Config struct {
	Server string `json:"server"`
	Port   uint   `json:"port"`
}

// readConfig
// read the ketchctl config from a file
func readConfig(c *cli.Context, path string) (*Config, error) {

	// Read configuration
	var config Config
	var err error
	if _, err = os.Stat(path); os.IsNotExist(err) {
		// No config; use defaults
		config.Server = c.GlobalString("api-server")
		config.Port = c.GlobalUint("api-port")
	} else if err != nil {
		// Error returned from stat()
		return nil, cli.NewExitError(fmt.Sprintf("Failed to read configuration %s, error: %v", path, err), 1)
	} else {
		// Read config from file
		buf, err := ioutil.ReadFile(path)
		if err != nil {
			return nil, cli.NewExitError(fmt.Sprintf("Failed to read configuration %s, error: %v", path, err), 1)
		}
		err = yaml.Unmarshal(buf, &config)
		if err != nil {
			return nil, cli.NewExitError(fmt.Sprintf("Failed to parse configuration %s, error: %v", path, err), 1)
		}
	}

	return &config, nil
}

// patchConfig
// updates ketchctl configuration.
func patchConfig(c *cli.Context) error {

	// Read configuration
	configPath := filepath.Join(configDir, configFile)
	config, err := readConfig(c, configPath)
	if err != nil {
		return err
	}

	// Patch configuration
	if c.IsSet("server") {
		config.Server = c.String("server")
	}
	if c.IsSet("port") {
		config.Port = c.Uint("port")
	}

	// Write configuration
	err = os.MkdirAll(configDir, configDirMode)
	if err != nil {
		return cli.NewExitError(fmt.Sprintf("Failed to create directory for configuration %s, error: %v", configDir, err), 1)
	}
	buf, err := yaml.Marshal(config)
	err = ioutil.WriteFile(configPath, buf, configFileMode)
	if err != nil {
		return cli.NewExitError(fmt.Sprintf("Failed to write configuration file %s, error: %v", configPath, err), 1)
	}

	return nil
}
