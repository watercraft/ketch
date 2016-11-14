// Copyright 2016 F. Alan Jones.  All rights reserved.
// Use of this source code is governed by a Mozilla
// license that can be found in the LICENSE file.

package api

import (
	"os/exec"
)

// TypeDBmgr is both the type and URL component for the dbmgr resource.
const TypeDBMgr Type = "dbmgr"

// DBState is the logical state of the currently running database
type DBState string

const (
	DBStateMaster       DBState = "master"
	DBStateMasterClosed DBState = "master-closed" // on closed port
	DBStateSlave        DBState = "slave"
	DBStateDown         DBState = "down"
)

// DBMgr represents a dbmgr that is used to manage the running state of the database.
// This object is not persisted in the saved configuration.
type DBMgr struct {
	Common
	// DBState is the logical state of the currently running database
	DBState DBState `json:"dbState,omitempty"`
	// DBDir is the full path directory for the database
	DBDir string `json:"dbDir,omitempty"`
	// Port is the port number that the running database is listening on
	Port uint16 `json:"port,omitempty"`
	// RunCmd is the command object for the running database.
	RunCmd *exec.Cmd `json:"-"`
	// RunEnv is a set of environment variables for the command
	RunEnv []string `json:"-"`
}

func (d *DBMgr) Clone() Resource {
	dbmgr := *d
	return &dbmgr
}

func (d *DBMgr) GetCommon() *Common {
	return &d.Common
}
