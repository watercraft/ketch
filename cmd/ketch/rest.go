// Copyright 2016 F. Alan Jones.  All rights reserved.
// Use of this source code is governed by a Mozilla
// license that can be found in the LICENSE file.

package main

import (
	"net"
	"net/http"
	"strconv"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/gorilla/mux"
	"github.com/urfave/negroni"

	"github.com/watercraft/ketch"
	"github.com/watercraft/ketch/api"
)

// Logger() is a middleware handler that logs the request as it goes in and the response as it goes out.
type Logger struct {
	// Logger inherits from log.Logger used to log messages with the Logger middleware
	*logrus.Logger
}

// ServeHTTP() provides middleware function for Logger, logging in logrus style
func (log *Logger) ServeHTTP(rw http.ResponseWriter, r *http.Request, next http.HandlerFunc) {

	start := time.Now()
	log.WithFields(ketch.Locate(logrus.Fields{
		"method": r.Method,
		"url":    r.URL.Path,
	})).Info("Started")

	next(rw, r)

	res := rw.(negroni.ResponseWriter)
	log.WithFields(ketch.Locate(logrus.Fields{
		"code":   res.Status(),
		"status": http.StatusText(res.Status()),
		"time":   time.Since(start),
	})).Info("Completed")
}

// NewLogger() returns a new Logger instance
func NewLogger() *Logger {
	return &Logger{logrus.New()}
}

func ListenAndServe(server string, port uint) {

	mux := mux.NewRouter()
	mux.HandleFunc(string(api.URLBase+api.TypeRuntime), HandleGetRuntime).Methods("GET")
	mux.HandleFunc(string(api.URLBase+api.TypeServer), HandleGetServer).Methods("GET")
	mux.HandleFunc(string(api.URLBase+api.TypeEpoch), HandleGetEpoch).Methods("GET")
	mux.HandleFunc(string(api.URLBase+api.TypeReplica), HandleGetReplica).Methods("GET")
	mux.HandleFunc(string(api.URLBase+api.TypeReplica), HandlePostReplica).Methods("POST")
	mux.HandleFunc(string(api.URLBase+api.TypeDBMgr), HandleGetDBmgr).Methods("GET")

	n := negroni.New(
		negroni.NewRecovery(),
		NewLogger(), // log to logrus
		negroni.NewStatic(http.Dir("public")),
	)
	n.UseHandler(mux)

	url := net.JoinHostPort(server, strconv.Itoa(int(port)))
	log.WithFields(ketch.Locate(logrus.Fields{
		"url": url,
	})).Info("Starting management service")
	http.ListenAndServe(url, n)
}
