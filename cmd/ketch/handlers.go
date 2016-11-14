// Copyright 2016 F. Alan Jones.  All rights reserved.
// Use of this source code is governed by a Mozilla
// license that can be found in the LICENSE file.

package main

import (
	"encoding/json"
	"net/http"

	"github.com/Sirupsen/logrus"

	"github.com/watercraft/ketch"
	"github.com/watercraft/ketch/api"
)

func WriteError(w http.ResponseWriter, err error, status int) {
	resp := api.ResourceBody{
		Errors: []api.Error{
			{
				Status: status,
				Title:  http.StatusText(status),
				Detail: err.Error(),
			},
		},
	}
	out, err2 := json.Marshal(resp)
	if err2 != nil {
		log.WithFields(ketch.Locate(logrus.Fields{
			"err": err2,
		})).Info("Failed marshal error")
		// Override error with internal
		out = []byte{}
		status = http.StatusInternalServerError
	}
	w.WriteHeader(status)
	_, err = w.Write(out)
	if err != nil {
		log.WithFields(ketch.Locate(logrus.Fields{
			"err": err,
		})).Info("Failed to write error response")
	}
}

func writeResourceBody(w http.ResponseWriter, myType api.Type, list api.ResourceList) {
	out, err := api.MarshalList(myType, list)
	if err != nil {
		WriteError(w, err, http.StatusInternalServerError)
		return
	}
	_, err = w.Write(out)
	if err != nil {
		log.WithFields(ketch.Locate(logrus.Fields{
			"err": err,
			"out": out,
		})).Error("Failed to write create response")
	}
}

func HandleGetRuntime(w http.ResponseWriter, req *http.Request) {
	list := Crew.GetResources(api.TypeRuntime)
	writeResourceBody(w, api.TypeRuntime, list)
}

func HandleGetServer(w http.ResponseWriter, req *http.Request) {
	list := Crew.GetResources(api.TypeServer)
	writeResourceBody(w, api.TypeServer, list)
}

func HandleGetEpoch(w http.ResponseWriter, req *http.Request) {
	list := Crew.GetResources(api.TypeEpoch)
	writeResourceBody(w, api.TypeEpoch, list)
}

func HandleGetReplica(w http.ResponseWriter, req *http.Request) {
	list := Crew.GetResources(api.TypeReplica)
	writeResourceBody(w, api.TypeReplica, list)
}

func HandlePostReplica(w http.ResponseWriter, req *http.Request) {

	// Unmarshal resouce list
	list, err := api.UnmarshalList(api.TypeReplica, req)
	if err != nil {
		WriteError(w, err, http.StatusInternalServerError)
		return
	}

	// Add replicas
	list, err, status := Crew.CreateResources(api.TypeReplica, list)
	if err != nil {
		WriteError(w, err, status)
		return
	}
	log.WithFields(ketch.Locate(logrus.Fields{
		"count": len(list),
	})).Info("Created replicas")

	// Return replicas created
	writeResourceBody(w, api.TypeReplica, list)
}

func HandleGetDBmgr(w http.ResponseWriter, req *http.Request) {
	list := Crew.GetResources(api.TypeDBMgr)
	writeResourceBody(w, api.TypeDBMgr, list)
}
