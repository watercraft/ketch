// Copyright 2016 F. Alan Jones.  All rights reserved.
// Use of this source code is governed by a Mozilla
// license that can be found in the LICENSE file.

package ketch

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sort"

	"github.com/Sirupsen/logrus"
	"github.com/boltdb/bolt"
	"github.com/satori/go.uuid"

	"github.com/watercraft/ketch/api"
)

type initFunc func(*ResourceMgr, api.Resource) (error, int)
type getListFunc func(*ResourceMgr) api.ResourceList
type updateAfterLoadFunc func(*ResourceMgr, api.Resource) bool

type ResourceMgr struct {
	k               *Ketch
	myType          api.Type
	assignIDs       bool
	named           bool
	persist         bool
	resource        map[uuid.UUID]api.Resource
	resourceByName  map[string]uuid.UUID
	init            initFunc
	getList         getListFunc
	updateAfterLoad updateAfterLoadFunc
}

func (k *Ketch) installResourceMgr(mgr *ResourceMgr) {
	mgr.k = k
	mgr.resource = make(map[uuid.UUID]api.Resource)
	mgr.resourceByName = make(map[string]uuid.UUID)
	k.resourceMgr[mgr.myType] = mgr
	if mgr.persist {
		mgr.loadResources()
	}
}

// RefreshResources
// calls resource specific getList() to refresh resources.
func (m *ResourceMgr) RefreshResources() {
	if m.getList != nil {
		m.ClearResources()
		// TODO: Error handling
		_, _, _ = m.CreateResources(m.getList(m))
	}
}

// GetResources
// returns list of resources.
func (m *ResourceMgr) GetResources() api.ResourceList {
	m.RefreshResources()
	var list api.ResourceList
	for _, obj := range m.resource {
		// Copy object so the original is not modified
		list = append(list, obj.Clone())
	}
	sort.Sort(list)
	return list
}

// ClearResources
// clears all resources of the specified type.
func (m *ResourceMgr) ClearResources() {
	m.resource = make(map[uuid.UUID]api.Resource)
	m.resourceByName = make(map[string]uuid.UUID)
}

// CreateResources
// creates resources from a list.
// Returns the list created, error and http status
func (m *ResourceMgr) CreateResources(list api.ResourceList) (api.ResourceList, error, int) {

	// Validate list against ketch state
	for _, resource := range list {
		common := resource.GetCommon()
		if m.named && common.Name == "" {
			err := fmt.Errorf("Resource missing name")
			m.k.log.WithFields(Locate(logrus.Fields{
				"type": m.myType,
				"err":  err,
			})).Error("Failed to create resource")
			return nil, err, http.StatusBadRequest
		}
		if m.named {
			if _, ok := m.resourceByName[common.Name]; ok {
				err := fmt.Errorf("Resource already exists")
				m.k.log.WithFields(Locate(logrus.Fields{
					"type": m.myType,
					"err":  err,
					"name": common.Name,
				})).Error(err.Error())
				return nil, err, http.StatusConflict
			}
		}
		if m.assignIDs {
			// New ID if not coming from from outside
			common.ID = uuid.NewV4()
		}
		if _, ok := m.resource[common.ID]; ok {
			err := fmt.Errorf("Resource ID already exists")
			m.k.log.WithFields(Locate(logrus.Fields{
				"type": m.myType,
				"err":  err,
				"name": common.Name,
				"id":   common.ID,
			})).Error(err.Error())
			return nil, err, http.StatusInternalServerError
		}
		if m.init != nil {
			err, status := m.init(m, resource)
			if err != nil {
				m.k.log.WithFields(Locate(logrus.Fields{
					"type": m.myType,
					"err":  err,
					"name": common.Name,
					"id":   common.ID,
				})).Error("Failed to initialize resource")
				return nil, err, status
			}
		}
	}

	// Put resources into Ketch state
	for _, resource := range list {
		common := resource.GetCommon()
		m.resource[common.ID] = resource.Clone()
		m.resourceByName[common.Name] = common.ID
	}

	// Persist resources
	if !m.persist {
		return list, nil, http.StatusCreated
	}
	err := m.k.db.Update(func(tx *bolt.Tx) error {
		// Create Ketch bucket
		b, err := tx.CreateBucketIfNotExists([]byte(m.myType))
		if err != nil {
			m.k.log.WithFields(Locate(logrus.Fields{
				"type": m.myType,
				"err":  err,
			})).Error("Failed to create bucket")
			return err
		}

		// Put resources in bucket
		for _, resource := range list {
			common := resource.GetCommon()
			out, err := json.Marshal(resource)
			if err != nil {
				m.k.log.WithFields(Locate(logrus.Fields{
					"type": m.myType,
					"name": common.Name,
					"id":   common.ID,
					"err":  err,
				})).Error("Failed to marshal resource")
				return err
			}
			err = b.Put(common.ID.Bytes(), out)
			if err != nil {
				m.k.log.WithFields(Locate(logrus.Fields{
					"type": m.myType,
					"name": common.Name,
					"key":  common.ID,
					"out":  out,
					"err":  err,
				})).Error("Failed to put resource")
				return err
			}
		}
		return nil
	})
	if err != nil {
		return nil, err, http.StatusInternalServerError
	}

	return list, nil, http.StatusCreated
}

// loadResources
// loads existing resources from database into Ketch.
// Called locked.
func (m *ResourceMgr) loadResources() {

	// Iterate over resources
	err := m.k.db.Update(func(tx *bolt.Tx) error {
		// Create Ketch bucket if it doesn't exist
		bk, err := tx.CreateBucketIfNotExists([]byte(m.myType))
		if err != nil {
			m.k.log.WithFields(Locate(logrus.Fields{
				"type": m.myType,
				"err":  err,
			})).Error("Failed to create bucket")
			return err
		}
		// Look for keys in the bucket
		cur := bk.Cursor()
		for key, value := cur.First(); key != nil; key, value = cur.Next() {
			resource := api.NewByType(m.myType)
			err := json.Unmarshal(value, &resource)
			if err != nil {
				m.k.log.WithFields(Locate(logrus.Fields{
					"type":  m.myType,
					"err":   err,
					"key":   key,
					"value": string(value),
				})).Fatal("Failed to unmarshal resource")
			}
			common := resource.GetCommon()
			if id, err := uuid.FromBytes(key); err != nil || id != common.ID {
				m.k.log.WithFields(Locate(logrus.Fields{
					"type": m.myType,
					"err":  err,
					"name": common.Name,
					"id":   common.ID,
				})).Fatal("Failed to match resource key")
			}
			m.resource[common.ID] = resource
			m.resourceByName[common.Name] = common.ID
			if m.updateAfterLoad != nil {
				if m.updateAfterLoad(m, resource) {
					out, err := json.Marshal(resource)
					if err != nil {
						m.k.log.WithFields(Locate(logrus.Fields{
							"type": m.myType,
							"name": common.Name,
							"id":   common.ID,
							"err":  err,
						})).Error("Failed to marshal resource")
						return err
					}
					err = bk.Put(common.ID.Bytes(), out)
					if err != nil {
						m.k.log.WithFields(Locate(logrus.Fields{
							"type": m.myType,
							"name": common.Name,
							"key":  common.ID,
							"out":  out,
							"err":  err,
						})).Error("Failed to put resource")
						return err
					}
				}
			}
		}
		return nil
	})
	if err != nil {
		m.k.log.WithFields(Locate(logrus.Fields{
			"type": m.myType,
			"err":  err,
		})).Fatal("Failed to load resources")
	}
}

// saveResource
// update an existing resource in database from memory.
// Called locked.
// TODO: Return error for API PATCH; fatal for now.
func (m *ResourceMgr) saveResource(id uuid.UUID) {

	err := m.k.db.Update(func(tx *bolt.Tx) error {
		// Create Ketch bucket if it doesn't exist
		bk, err := tx.CreateBucketIfNotExists([]byte(m.myType))
		if err != nil {
			m.k.log.WithFields(Locate(logrus.Fields{
				"type": m.myType,
				"err":  err,
			})).Error("Failed to create bucket")
			return err
		}
		// Marshal resource
		resource, ok := m.resource[id]
		if !ok {
			err = bk.Delete(id.Bytes())
			if err != nil {
				m.k.log.WithFields(Locate(logrus.Fields{
					"type": m.myType,
					"key":  id,
					"err":  err,
				})).Error("Failed to delete resource")
				return err
			}
			return nil
		}
		common := resource.GetCommon()
		out, err := json.Marshal(resource)
		if err != nil {
			m.k.log.WithFields(Locate(logrus.Fields{
				"type": m.myType,
				"name": common.Name,
				"id":   common.ID,
				"err":  err,
			})).Error("Failed to marshal resource")
			return err
		}
		err = bk.Put(common.ID.Bytes(), out)
		if err != nil {
			m.k.log.WithFields(Locate(logrus.Fields{
				"type": m.myType,
				"name": common.Name,
				"key":  common.ID,
				"out":  out,
				"err":  err,
			})).Error("Failed to put resource")
			return err
		}
		return nil
	})
	if err != nil {
		m.k.log.WithFields(Locate(logrus.Fields{
			"type": m.myType,
			"err":  err,
		})).Fatal("Failed to update resources")
	}
}
