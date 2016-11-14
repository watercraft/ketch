// Copyright 2016 F. Alan Jones.  All rights reserved.
// Use of this source code is governed by a Mozilla
// license that can be found in the LICENSE file.

package ketch

import (
	"github.com/satori/go.uuid"

	"github.com/watercraft/ketch/api"
)

func (k *Ketch) installServerMgr() {
	k.installResourceMgr(&ResourceMgr{
		myType:          api.TypeServer,
		assignIDs:       false,
		named:           true,
		persist:         false,
		init:            nil,
		getList:         getServerList,
		updateAfterLoad: nil,
	})
}

func getServerList(m *ResourceMgr) api.ResourceList {
	nodes := m.k.list.Members()
	var list api.ResourceList
	for _, node := range nodes {
		server := &api.Server{
			Common: api.Common{
				Name: node.Name,
				ID:   uuid.FromBytesOrNil(node.Meta),
			},
			Endpoint: api.Endpoint{
				Addr: node.Addr,
				Port: node.Port,
			},
		}
		list = append(list, server)
	}
	return list
}
