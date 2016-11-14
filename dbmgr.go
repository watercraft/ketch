// Copyright 2016 F. Alan Jones.  All rights reserved.
// Use of this source code is governed by a Mozilla
// license that can be found in the LICENSE file.

package ketch

import (
	"bufio"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"syscall"

	"github.com/Sirupsen/logrus"

	"github.com/watercraft/ketch/api"
)

const (
	DBDirMode  os.FileMode = 0700
	DBFileMode os.FileMode = 0600
	DBPWExt    string      = "pw"
)

func (k *Ketch) installDBMgrMgr() {
	k.installResourceMgr(&ResourceMgr{
		myType:          api.TypeDBMgr,
		assignIDs:       false,
		named:           true,
		persist:         false,
		init:            nil,
		getList:         nil,
		updateAfterLoad: nil,
	})
}

func (k *Ketch) handleSignals(sigCh chan os.Signal) {
	for _ = range sigCh {
		k.Lock()
		defer k.Unlock()
		for _, resource := range k.resourceMgr[api.TypeDBMgr].resource {
			dbmgr := resource.(*api.DBMgr)
			if (dbmgr.RunCmd != nil) && (dbmgr.RunCmd.Process != nil) {
				// "Fast" shutdown
				dbmgr.RunCmd.Process.Signal(syscall.SIGINT)
			}
		}
		os.Exit(1)
	}
}

func run(m *ResourceMgr, dbmgr *api.DBMgr, nextState api.State, stdin io.Reader, command string, args ...string) {
	m.k.log.WithFields(Locate(logrus.Fields{
		"cmd":  command,
		"args": args,
	})).Info("Start")
	dbmgr.RunCmd = exec.Command(path.Join(m.k.config.DBBinDir, command), args...)
	dbmgr.RunCmd.Env = dbmgr.RunEnv
	cmdOut, err := dbmgr.RunCmd.StdoutPipe()
	if err != nil {
		m.k.log.WithFields(Locate(logrus.Fields{
			"dbmgr": dbmgr,
			"err":   err,
			"cmd":   command,
			"args":  args,
		})).Error("Failed to initialize output pipe")
	}
	scanOut := bufio.NewScanner(cmdOut)
	go func() {
		for scanOut.Scan() {
			m.k.log.WithFields(Locate(logrus.Fields{
				"cmd": command,
			})).Info(scanOut.Text())
		}
	}()
	cmdErr, err := dbmgr.RunCmd.StderrPipe()
	if err != nil {
		m.k.log.WithFields(Locate(logrus.Fields{
			"dbmgr": dbmgr,
			"err":   err,
			"cmd":   command,
			"args":  args,
		})).Error("Failed to initialize error pipe")
	}
	scanErr := bufio.NewScanner(cmdErr)
	go func() {
		for scanErr.Scan() {
			// All postgres output goes to stderr; don't flag
			m.k.log.WithFields(Locate(logrus.Fields{
				"cmd": command,
			})).Error(scanErr.Text())
		}
	}()
	dbmgr.RunCmd.Stdin = stdin
	dbmgr.RunCmd.Start()
	go func() {
		err := dbmgr.RunCmd.Wait()
		m.k.Lock()
		defer m.k.Unlock()
		dbmgr.PendingState = ""
		if (err != nil) && (err.Error() != "exec: not started") {
			m.k.log.WithFields(Locate(logrus.Fields{
				"dbmgr": dbmgr,
				"err":   err,
				"cmd":   command,
				"args":  args,
			})).Error("Command exited with error")
			return
		}
		m.k.log.WithFields(Locate(logrus.Fields{
			"cmd":  command,
			"args": args,
		})).Info("Completed")
		dbmgr.State = nextState
	}()
}

// Returns true when the database is up on the service port
func runReplicaOnPort(m *ResourceMgr, replica *api.Replica, dbState api.DBState, port uint16) bool {

	// Create dbmgr if it does not exist
	var dbmgr *api.DBMgr
	resource, ok := m.k.resourceMgr[api.TypeDBMgr].resource[replica.ID]
	if ok {
		dbmgr = resource.(*api.DBMgr)
	} else {
		dbmgr = &api.DBMgr{
			Common:  replica.Common,
			DBState: dbState,
		}
		dbmgr.State = api.StateUninitialized
		dbmgr.RunEnv = []string{fmt.Sprintf("PGPASSWORD=%s", replica.DBConfig.Password)}
		m.k.resourceMgr[api.TypeDBMgr].resource[dbmgr.ID] = dbmgr
	}

	// If database already running...
	if dbmgr.State == api.StateOpen {
		// Already on correct port, return
		if dbmgr.Port == port {
			return true
		}
		if (dbmgr.RunCmd == nil) || (dbmgr.RunCmd.Process == nil) {
			m.k.log.WithFields(Locate(logrus.Fields{
				"dbmgr": dbmgr,
			})).Error("Database manager open without command")
			return false
		}
		// "Fast" shutdown to restart on correct port
		dbmgr.RunCmd.Process.Signal(syscall.SIGINT)
		return false
	}

	// Create directory if it does not exist
	dbmgr.DBDir = path.Join(m.k.config.DataDir, replica.ID.String())
	info, err := os.Stat(dbmgr.DBDir)
	if (err != nil && !os.IsNotExist(err)) || (err == nil && !info.IsDir()) {
		m.k.log.WithFields(Locate(logrus.Fields{
			"dbmgr": dbmgr,
			"err":   err,
		})).Error("Failed to stat database directory")
		return false
	}

	pwFile := dbmgr.DBDir + "pw"
	var master string
	if replica.MasterServerID != nil {
		resource, ok := m.k.resourceMgr[api.TypeServer].resource[*replica.MasterServerID]
		if !ok {
			m.k.log.WithFields(Locate(logrus.Fields{
				"dbmgr": dbmgr,
				"err":   err,
			})).Info("Attempt to start slave for unknown master")
			return false
		}
		master = resource.(*api.Server).Endpoint.Addr.String()
	}

	switch dbmgr.State {

	case api.StateUninitialized:
		if dbmgr.PendingState != "" {
			// Still working on something
			return false
		}
		// dbmgr just created
		if os.IsNotExist(err) {
			// Create and init
			err = os.MkdirAll(dbmgr.DBDir, DBDirMode)
			if err != nil {
				m.k.log.WithFields(Locate(logrus.Fields{
					"dbmgr": dbmgr,
					"err":   err,
				})).Error("Failed to create database directory")
				return false
			}
		}
		info, err = os.Stat(path.Join(dbmgr.DBDir, "PG_VERSION"))
		if (err != nil && !os.IsNotExist(err)) || (err == nil && info.IsDir()) {
			m.k.log.WithFields(Locate(logrus.Fields{
				"dbmgr": dbmgr,
				"err":   err,
			})).Error("Failed to stat database version file")
			return false
		}
		if os.IsNotExist(err) {
			dbmgr.PendingState = api.StateClosed
			dbmgr.Port = port
			if dbState == api.DBStateSlave {
				run(m, dbmgr, api.StateClosed, nil, "pg_basebackup",
					"--pgdata", dbmgr.DBDir,
					"--host", master,
					"--port", fmt.Sprintf("%d", dbmgr.Port),
					"--username", replica.DBConfig.Username,
					"-X", "stream", "-P")
			} else {
				ioutil.WriteFile(pwFile, []byte(replica.DBConfig.Password), DBFileMode)
				run(m, dbmgr, api.StateNew, nil, "initdb",
					"--pgdata", dbmgr.DBDir,
					"--auth", "md5",
					"--username", replica.DBConfig.Username,
					"--pwfile", pwFile)
			}
			return false
		} else if dbState == api.DBStateSlave {
			run(m, dbmgr, api.StateClosed, nil, "pg_rewind",
				"--target-pgdata", dbmgr.DBDir,
				"--source-server", fmt.Sprintf("'host=%s port=%d user=%s application_name=%s'", master, dbmgr.Port, replica.DBConfig.Username, m.k.runtime.ID))
		}
		dbmgr.State = api.StateClosed

	case api.StateNew:
		// After initdb, create database named after replica
		// TODO: Figure out how to signal no terminal to postgres; using this script for now.
		// #!/bin/sh
		// export PGDATA=$1
		// export PGDATABASE=$2
		// /bin/echo create database ${PGDATABASE} | /usr/lib/postgresql/9.5/bin/postgres --single -D ${PGDATA} postgres
		run(m, dbmgr, api.StateClosed, nil, "alan.sh",
			dbmgr.DBDir,
			replica.Name)

	case api.StateClosed:
		// Start server
		dbmgr.State = api.StateOpen
		dbmgr.Port = port
		err = os.Remove(pwFile)
		if err != nil && !os.IsNotExist(err) {
			m.k.log.WithFields(Locate(logrus.Fields{
				"dbmgr":  dbmgr,
				"err":    err,
				"pwFile": pwFile,
			})).Error("Failed to remove password file")
			return false
		}
		if dbState == api.DBStateSlave {
			recoveryConf := path.Join(dbmgr.DBDir, "recovery.conf")
			out := []byte("standby_mode='on'\n" +
				fmt.Sprintf("primary_conninfo='host=%s port=%d user=%s application_name=%s'\n",
					master, dbmgr.Port, replica.DBConfig.Username, m.k.runtime.ID) +
				"recovery_target_timeline='latest'\n" +
				fmt.Sprintf("trigger_file='%s'\n", path.Join(dbmgr.DBDir, "trigger_file")))
			err = ioutil.WriteFile(recoveryConf, out, DBFileMode)
			if err != nil {
				m.k.log.WithFields(Locate(logrus.Fields{
					"dbmgr":        dbmgr,
					"err":          err,
					"recoveryConf": recoveryConf,
				})).Error("Failed to write to postgres recovery config file")
			}
			run(m, dbmgr, api.StateClosed, nil, "postgres",
				"-D", dbmgr.DBDir,
				"-c", "listen_addresses=",
				"-c", fmt.Sprintf("unix_socket_directories=%s", m.k.config.DataDir))
		} else {
			hbaConf := path.Join(dbmgr.DBDir, "pg_hba.conf")
			out := []byte(fmt.Sprintf("host all %s 0.0.0.0/0 md5\nhost replication %s 0.0.0.0/0 md5\n",
				replica.DBConfig.Username, replica.DBConfig.Username))
			err = ioutil.WriteFile(hbaConf, out, DBFileMode)
			if err != nil {
				m.k.log.WithFields(Locate(logrus.Fields{
					"dbmgr":   dbmgr,
					"err":     err,
					"hbaConf": hbaConf,
				})).Error("Failed to write to postgres authentication config file")
			}
			run(m, dbmgr, api.StateClosed, nil, "postgres",
				"-D", dbmgr.DBDir,
				"-c", fmt.Sprintf("unix_socket_directories=%s", m.k.config.DataDir),
				"-c", fmt.Sprintf("port=%d", dbmgr.Port),
				"-c", fmt.Sprintf("listen_addresses=%s", m.k.runtime.Endpoint.Addr.String()),
				"-c", "wal_level=hot_standby",
				"-c", "synchronous_commit=on",
				"-c", "max_wal_senders=3",
				"-c", "synchronous_standby_names=''")
		}
		return true
	}

	// Never reached
	return false
}
