// Copyright 2016 F. Alan Jones.  All rights reserved.
// Use of this source code is governed by a Mozilla
// license that can be found in the LICENSE file.

package ketch

import (
	"net"
	"os"
	"os/signal"
	"path"
	"syscall"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/boltdb/bolt"
	"github.com/hashicorp/memberlist"

	"github.com/watercraft/ketch/api"
	"github.com/watercraft/ketch/msg"
)

const (
	// Name of Ketch database in data directory
	kDatabaseName string = "ketch.db"
	// Mode to create Ketch database directory
	kDatabaseDirMode os.FileMode = 0700
	// Mode to create Ketch database
	kDatabaseMode os.FileMode = 0600
)

// Config
// Configuration for Ketch service.
type Config struct {
	// Logger to send logs to
	Log *logrus.Logger

	// ListConfig is the HashiCorp Memberlist configuration to use.
	ListConfig *memberlist.Config

	// DataDir is the directory to store all data for this instance of Ketch.
	DataDir string

	// DBBinDir is the directory for database executables.
	DBBinDir string
}

// Create
// Build ketch Ketch object from config.
func Create(config *Config) (*Ketch, error) {

	// Initialize Ketch
	var k Ketch
	k.log = config.Log
	k.config = config

	// Create directory for Ketch database if it doesn't exist
	err := os.MkdirAll(k.config.DataDir, kDatabaseDirMode)
	if err != nil {
		k.log.WithFields(Locate(logrus.Fields{
			"datadir": k.config.DataDir,
			"mode":    kDatabaseDirMode,
			"err":     err,
		})).Error("Failed to open Ketch database")
		return nil, err
	}

	// Open Ketch database
	dbpath := path.Join(k.config.DataDir, kDatabaseName)
	k.db, err = bolt.Open(
		dbpath,
		kDatabaseMode,
		&bolt.Options{Timeout: 1 * time.Second})
	if err != nil {
		k.log.WithFields(Locate(logrus.Fields{
			"dbpath": dbpath,
			"mode":   kDatabaseMode,
			"err":    err,
		})).Error("Failed to open Ketch database")
		return nil, err
	}

	// Get time of the last system reboot
	k.bootTime, err = k.getBootTime()
	if err != nil {
		k.log.WithFields(Locate(logrus.Fields{
			"err": err,
		})).Error("Failed to get reboot time")
		return nil, err
	}

	// Install runtime manager
	k.resourceMgr = make(map[api.Type]*ResourceMgr)
	k.runtime = nil
	k.installRuntimeMgr()

	// Fill in first runtime if it does not exist
	if k.runtime == nil {
		k.runtime = &api.Runtime{
			Common: api.Common{
				Name: config.ListConfig.Name,
			},
			BootTime: *k.bootTime,
			Endpoint: api.Endpoint{
				Addr: net.ParseIP(config.ListConfig.BindAddr),
				Port: uint16(config.ListConfig.BindPort),
			},
		}
		list := api.ResourceList{k.runtime}
		_, err, _ := k.resourceMgr[api.TypeRuntime].CreateResources(list)
		if err != nil {
			k.log.WithFields(Locate(logrus.Fields{
				"err": err,
			})).Error("Failed to create runtime")
		}
	}

	// Initialize channels for incoming events
	k.incomingMsgCh = make(chan msg.Msg, 10)
	k.wakeServiceLoopCh = make(chan bool, 10)

	// Create Hashicorp Memberlist in memory object
	config.ListConfig.Delegate = &k
	config.ListConfig.DisableTcpPings = true
	k.list, err = memberlist.Create(config.ListConfig)
	if err != nil {
		k.log.WithFields(Locate(logrus.Fields{
			"err": err,
		})).Error("Failed to create memberlist")
		return nil, err
	}

	// Initialize other resource managers
	k.installServerMgr()
	k.installEpochMgr()
	k.installReplicaMgr()
	k.installDBMgrMgr()

	// Catch stop signals
	sigCh := make(chan os.Signal, 5)
	signal.Notify(sigCh, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		k.handleSignals(sigCh)
	}()

	// Launch service loop
	go k.dispatchIncomingMsgs()
	go k.serviceLoop()

	k.log.WithFields(Locate(logrus.Fields{
		"dbpath":  dbpath,
		"runtime": config.ListConfig.Name,
	})).Info("Open Ketch runtime")

	return &k, nil
}
