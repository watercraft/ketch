// Copyright 2016 F. Alan Jones.  All rights reserved.
// Use of this source code is governed by a Mozilla
// license that can be found in the LICENSE file.

package api

// TypeServer is bothe the type and URL component for the server resource.
const TypeServer Type = "server"

// Server represents a server that is a member of the cluster.
type Server struct {
	Common // Anonymous Name and ID fields
	// Endpoint is the IP and port used to reach this server.
	Endpoint Endpoint `json:"endpoint"`
}

func (s *Server) Clone() Resource {
	server := *s
	return &server
}

func (s *Server) GetCommon() *Common {
	return &s.Common
}
