// Copyright 2016 F. Alan Jones.  All rights reserved.
// Use of this source code is governed by a Mozilla
// license that can be found in the LICENSE file.

package api

import (
	"time"
)

// TypeRuntime is both the type and URL component for the runtime resource.
const TypeRuntime Type = "runtime"

// Runtime represents a runtime that is a member of the cluster.
type Runtime struct {
	// Common provides anonymous Name and ID fields.
	Common
	// Endpoint is the IP and port used to reach this server.
	Endpoint Endpoint `json:"endpoint"`
	// BootTime is the time reboot of the system hosting Ketch.
	BootTime time.Time `json:"boottime"`
}

func (r *Runtime) Clone() Resource {
	runtime := *r
	return &runtime
}

func (r *Runtime) GetCommon() *Common {
	return &r.Common
}
