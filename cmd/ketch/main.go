// Copyright 2016 F. Alan Jones.  All rights reserved.
// Use of this source code is governed by a Mozilla
// license that can be found in the LICENSE file.

package main

import (
	"net"
	"os"

	"github.com/Sirupsen/logrus"
	"github.com/hashicorp/memberlist"
	"gopkg.in/urfave/cli.v1"

	"github.com/watercraft/ketch"
	"github.com/watercraft/ketch/api"
)

// Create a new instance of the logger.
var log = logrus.New()

func main() {

	// Build CLI
	app := cli.NewApp()
	app.Name = "ketch"
	app.Version = "1.0.0"
	app.Usage = "Service for managing database replication. Once started, deploy and monitor databases with 'ketchctl'."
	app.Action = func(c *cli.Context) error {
		cli.ShowAppHelp(c)
		os.Exit(1)
		return nil
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
			Usage:  "IP or hostname to bind Ketch management service.",
			EnvVar: "KETCH_API_SERVER",
		},
		cli.UintFlag{
			Name:   "api-port",
			Value:  api.APIPort,
			Usage:  "Port to bind Ketch management service.",
			EnvVar: "KETCH_API_PORT",
		},
		cli.StringFlag{
			Name:   "member-server",
			Usage:  "IP or hostname to bind Ketch membership service. Must be unique among services deployed on the same port. (Default: API server)",
			EnvVar: "KETCH_MEMBER_SERVER",
		},
		cli.UintFlag{
			Name:   "member-port",
			Value:  api.MemberPort,
			Usage:  "Port to bind Ketch membership service on both TCP and UDP.",
			EnvVar: "KETCH_MEMBER_PORT",
		},
		cli.StringFlag{
			Name:   "member-list",
			Usage:  "Comma seperated list of IPs or hostnames to connect to.",
			EnvVar: "KETCH_MEMBER_LIST",
		},
		cli.StringFlag{
			Name:   "data-dir",
			Value:  "/var/lib/ketch",
			Usage:  "Directory to store all persisted data",
			EnvVar: "KETCH_DATA_DIR",
		},
		cli.StringFlag{
			Name:   "db-bin-dir",
			Value:  "/usr/lib/postgresql/9.5/bin",
			Usage:  "Directory for database executables",
			EnvVar: "KETCH_DB_BIN_DIR",
		},
	}

	app.Commands = []cli.Command{
		{
			Name:   "run",
			Usage:  "Run the Ketch service.",
			Action: runKetch,
		},
	}

	app.Run(os.Args)
}

// Crew is the global instence of the Ketch library class.
var Crew *ketch.Ketch

// runKetch
// runs the ketch service.
func runKetch(c *cli.Context) error {

	// Lookup server IP
	server := c.GlobalString("api-server")
	if c.GlobalIsSet("member-server") {
		server = c.GlobalString("member-server")
	}
	ips, err := net.LookupHost(server)
	if err != nil || len(ips) == 0 {
		log.WithFields(ketch.Locate(logrus.Fields{
			"server": server,
			"err":    err,
		})).Fatal("Failed to lookup IP")
	}

	// Configure Ketch using first IP from lookup
	var config ketch.Config
	config.Log = log
	config.DataDir = c.GlobalString("data-dir")
	config.DBBinDir = c.GlobalString("db-bin-dir")
	config.ListConfig = memberlist.DefaultLocalConfig()
	config.ListConfig.Name = server
	config.ListConfig.BindAddr = ips[0]
	config.ListConfig.BindPort = int(c.GlobalUint("member-port"))

	// Join crew
	Crew, err = JoinCrew(&config, c.GlobalString("member-list"))
	if err != nil || len(ips) == 0 {
		log.WithFields(ketch.Locate(logrus.Fields{
			"server":  config.ListConfig.Name,
			"address": config.ListConfig.BindAddr,
			"port":    config.ListConfig.BindPort,
			"err":     err,
		})).Fatal("Failed to join members")
	}
	ListenAndServe(c.GlobalString("api-server"), c.GlobalUint("api-port"))
	return nil
}
