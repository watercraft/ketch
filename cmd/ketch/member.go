// Copyright 2016 F. Alan Jones.  All rights reserved.
// Use of this source code is governed by a Mozilla
// license that can be found in the LICENSE file.

package main

import (
	"strings"

	"github.com/Sirupsen/logrus"

	"github.com/watercraft/ketch"
)

func JoinCrew(config *ketch.Config, list string) (*ketch.Ketch, error) {

	// Create Member List
	crew, err := ketch.Create(config)
	if err != nil {
		log.WithFields(ketch.Locate(logrus.Fields{
			"err": err,
		})).Error("Failed to create membership list")
		return nil, err
	}

	// Extract non-empty peers from list
	peers := strings.Split(list, ",")
	for i := len(peers) - 1; i >= 0; i-- {
		if peers[i] == "" {
			peers = append(peers[:i], peers[i+1:]...)
		}
	}

	// Join other members
	numPeers, err := crew.Join(peers)
	if err != nil {
		log.WithFields(ketch.Locate(logrus.Fields{
			"peers": peers,
			"err":   err,
		})).Error("Failed to join peers")
		return nil, err
	}

	// Log success
	log.WithFields(ketch.Locate(logrus.Fields{
		"numMembers": numPeers + 1,
		"server":     config.ListConfig.Name,
		"address":    config.ListConfig.BindAddr,
		"port":       config.ListConfig.BindPort,
	})).Info("Joined members")

	return crew, nil
}
