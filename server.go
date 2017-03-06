// Copyright 2016 F. Alan Jones.  All rights reserved.
// Use of this source code is governed by a Mozilla
// license that can be found in the LICENSE file.

package ketch

import (
	"net/http"

	"github.com/satori/go.uuid"

	"github.com/watercraft/ketch/api"
)

type ServerMgr struct {
	ResourceMgr
}

func (k *Ketch) installServerMgr() {
	m := &ServerMgr{
		ResourceMgr: ResourceMgr{
			myType:    api.TypeServer,
			assignIDs: false,
			named:     true,
			persist:   false,
		},
	}
	m.Init(k, m)
}

func (m *ServerMgr) InitResource(in api.Resource) (error, int) {
	return nil, http.StatusOK
}

func (m *ServerMgr) GetList() api.ResourceList {
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

func (m *ServerMgr) UpdateAfterLoad(resource api.Resource) bool {
	return false
}
