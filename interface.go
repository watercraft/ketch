// Copyright 2016 F. Alan Jones.  All rights reserved.
// Use of this source code is governed by a Mozilla
// license that can be found in the LICENSE file.

package ketch

import (
	"github.com/watercraft/ketch/api"
)

// GetResources
// returns list of resources.
func (k *Ketch) GetResources(myType api.Type) api.ResourceList {
	k.Lock()
	defer k.Unlock()
	return k.resourceMgr[myType].GetResources()
}

// CreateResources
// creates resources from a list.
// Returns the list created, error and http status
func (k *Ketch) CreateResources(myType api.Type, list api.ResourceList) (api.ResourceList, error, int) {
	k.Lock()
	defer k.Unlock()
	list, err, status := k.resourceMgr[myType].CreateResources(list)
	k.wakeServiceLoopCh <- true // Wake service loop to service new resource
	return list, err, status
}
