// Copyright 2016 F. Alan Jones.  All rights reserved.
// Use of this source code is governed by a Mozilla
// license that can be found in the LICENSE file.

package ketch

import (
	"net/http"

	"github.com/watercraft/ketch/api"
)

type RuntimeMgr struct {
	ResourceMgr
}

func (k *Ketch) installRuntimeMgr() {
	m := &RuntimeMgr{
		ResourceMgr: ResourceMgr{
			myType:    api.TypeRuntime,
			assignIDs: true,
			named:     true,
			persist:   true,
		},
	}
	m.Init(k, m)
}

func (m *RuntimeMgr) InitResource(in api.Resource) (error, int) {
	return nil, http.StatusOK
}

func (m *RuntimeMgr) GetList() api.ResourceList {
	return nil
}

func (m *RuntimeMgr) UpdateAfterLoad(in api.Resource) bool {

	// If boottime or name changed, update boottime record
	m.k.runtime = in.(*api.Runtime)
	if m.k.runtime.Name != m.k.config.ListConfig.Name ||
		m.k.runtime.BootTime != *m.k.bootTime {

		// TODO: Renew leases here
		m.k.runtime.Name = m.k.config.ListConfig.Name
		m.k.runtime.BootTime = *m.k.bootTime
		return true
	}
	return false
}
