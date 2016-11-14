// Copyright 2016 F. Alan Jones.  All rights reserved.
// Use of this source code is governed by a Mozilla
// license that can be found in the LICENSE file.

package ketch

import (
	"github.com/watercraft/ketch/api"
)

func (k *Ketch) installRuntimeMgr() {
	k.installResourceMgr(&ResourceMgr{
		myType:          api.TypeRuntime,
		assignIDs:       true,
		named:           true,
		persist:         true,
		init:            nil,
		getList:         nil,
		updateAfterLoad: updateRuntimeAfterLoad,
	})
}

func updateRuntimeAfterLoad(m *ResourceMgr, in api.Resource) bool {

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
