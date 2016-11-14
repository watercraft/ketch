// Copyright 2016 F. Alan Jones.  All rights reserved.
// Use of this source code is governed by a Mozilla
// license that can be found in the LICENSE file.

package main

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strconv"

	"github.com/ghodss/yaml"
	"gopkg.in/urfave/cli.v1"

	"github.com/watercraft/ketch/api"
)

func main() {

	// Build CLI
	app := cli.NewApp()
	app.Name = "ketchctl"
	app.Version = "1.0.0"
	app.Usage = "Command line tool for managing Ketch service to deploy and monitor databases."
	app.Action = func(c *cli.Context) error {
		cli.ShowAppHelp(c)
		return cli.NewExitError("", 1)
	}

	// Provide default for servers, if available
	hostname, err := os.Hostname()
	if err != nil {
		hostname = "localhost"
	}

	// Options
	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:   "api-server",
			Value:  hostname,
			Usage:  "IP or hostname to connect to Ketch management service.",
			EnvVar: "KETCH_API_SERVER",
		},
		cli.UintFlag{
			Name:   "api-port",
			Value:  api.APIPort,
			Usage:  "Port to bind Ketch management service.",
			EnvVar: "KETCH_API_PORT",
		},
	}

	app.Commands = []cli.Command{
		{
			Name:   "login",
			Usage:  "Saves the server and port specified for subsequent commands. Must specify --api-server.",
			Action: patchConfig,
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:  "server, s",
					Value: "localhost",
					Usage: "IP or hostname to connect to Ketch management service.",
				},
				cli.UintFlag{
					Name:  "port, p",
					Value: api.APIPort,
					Usage: "Port to bind Ketch management service.",
				},
			},
		},
		{
			Name:  "get",
			Usage: "Display resources.",
			Subcommands: []cli.Command{
				{
					Name:   "runtime",
					Usage:  "Get local server details.",
					Action: getCmd,
				},
				{
					Name:   "server",
					Usage:  "Get list of connected servers.",
					Action: getCmd,
				},
				{
					Name:   "epoch",
					Usage:  "Get list of local epochs.",
					Action: getCmd,
				},
				{
					Name:   "replica",
					Usage:  "Get list of local database replicas.",
					Action: getCmd,
				},
				{
					Name:   "dbmgr",
					Usage:  "Get list of local database managers.",
					Action: getCmd,
				},
			},
		},
		{
			Name:  "create",
			Usage: "Creates a resources.",
			Subcommands: []cli.Command{
				{
					Name:   "replica",
					Usage:  "Create a replica resource.",
					Action: createCmd,
					Flags: []cli.Flag{
						cli.StringFlag{
							Name:  "filename, f",
							Value: "-",
							Usage: "Filename to use as input for create.",
						},
					},
				},
			},
		},
	}

	app.Run(os.Args)
}

// outputResponse
// will output an http response in yaml
func outputResponse(resp *http.Response) error {

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return cli.NewExitError(fmt.Sprintf("Failed to read response body, error: %v", err), 1)
	}
	out, err := yaml.JSONToYAML(body)
	if err != nil {
		return cli.NewExitError(fmt.Sprintf("Failed to parse response body %v, error: %v", body, err), 1)
	}
	fmt.Println(string(out))

	return nil
}

// getCmd
// displays the resouce spcified in the subcommand name.
func getCmd(c *cli.Context) error {

	// Read configuration
	configPath := filepath.Join(configDir, configFile)
	config, err := readConfig(c, configPath)
	if err != nil {
		return err
	}

	// Make request
	url := "http://" + config.Server + ":" + strconv.Itoa(int(config.Port)) + string(api.URLBase) + c.Command.Name
	resp, err := http.Get(url)
	if resp == nil {
		err = fmt.Errorf("No response from server")
	}
	if err != nil {
		return cli.NewExitError(fmt.Sprintf("Failed request to %s, error: %v", url, err), 1)
	}
	defer resp.Body.Close()

	// Output response
	return outputResponse(resp)
}

// createCmd
// create the resouce spcified in the subcommand name using the file as input.
func createCmd(c *cli.Context) error {

	// Read configuration
	configPath := filepath.Join(configDir, configFile)
	config, err := readConfig(c, configPath)
	if err != nil {
		return err
	}

	// Read resource to create
	in := os.Stdin
	path := c.String("filename")
	if path != "-" {
		in, err = os.Open(path)
		if err != nil {
			return cli.NewExitError(fmt.Sprintf("Failed to open file %s, error: %v", path, err), 1)
		}
	}
	defer in.Close()
	buf := bytes.NewBuffer(nil)
	io.Copy(buf, in)
	body, err := yaml.YAMLToJSON(buf.Bytes())
	if err != nil {
		return cli.NewExitError(fmt.Sprintf("Failed to parse resource from %s, error: %v", path, err), 1)
	}

	// Make request
	url := "http://" + config.Server + ":" + strconv.Itoa(int(config.Port)) + string(api.URLBase) + c.Command.Name
	resp, err := http.Post(url, "application/json", bytes.NewReader(body))
	if resp == nil {
		err = fmt.Errorf("No response from server")
	}
	if err != nil {
		return cli.NewExitError(fmt.Sprintf("Failed request to %s, error: %v", url, err), 1)
	}
	defer resp.Body.Close()

	// Output response
	return outputResponse(resp)
}
