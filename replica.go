// Copyright 2016 F. Alan Jones.  All rights reserved.
// Use of this source code is governed by a Mozilla
// license that can be found in the LICENSE file.

package ketch

import (
	"fmt"
	"net/http"

	"github.com/Sirupsen/logrus"
	"github.com/satori/go.uuid"

	"github.com/watercraft/ketch/api"
	"github.com/watercraft/ketch/msg"
)

const (
	maxQuorumGroupSize uint = 3
)

type ReplicaMgr struct {
	ResourceMgr
}

func (k *Ketch) installReplicaMgr() {
	m := &ReplicaMgr{
		ResourceMgr: ResourceMgr{
			myType:    api.TypeReplica,
			assignIDs: true,
			named:     true,
			persist:   true,
		},
	}
	m.Init(k, m)
}

func (m *ReplicaMgr) InitResource(in api.Resource) (error, int) {

	replica := in.(*api.Replica)
	if replica.QuorumGroupSize == 0 {
		replica.QuorumGroupSize = maxQuorumGroupSize
	}
	if replica.QuorumGroupSize > maxQuorumGroupSize {
		err := fmt.Errorf("Quorum group size must be less than or equal to %d", maxQuorumGroupSize)
		m.k.log.WithFields(Locate(logrus.Fields{
			"err":     err,
			"replica": replica,
		})).Error(err.Error())
		return err, http.StatusBadRequest
	}
	replica.State = api.StateNew
	replica.MemberType = api.ReplicaQuorumMemberTypeSync
	replica.DataState = api.DataStateInSync
	replica.HomeServerID = m.k.runtime.ID
	replica.PriorEpochID = nil
	replica.Epochs = make(map[string]*api.EpochSpec)

	return nil, http.StatusOK
}

func (m *ReplicaMgr) GetList() api.ResourceList {
	return nil
}

func (m *ReplicaMgr) UpdateAfterLoad(resource api.Resource) bool {
	return false
}

// Returns true if the replica's membership has not changed
func markReplicaPendingClosed(m *ResourceMgr, replica *api.Replica) bool {

	// If we don't have an epoch, return
	if replica.CurrentEpochID == nil {
		return true
	}

	// Review current membership and return true if all are up
	mbrDown := false
	for _, mbr := range replica.Epochs[replica.CurrentEpochID.String()].Quorum {
		_, ok := m.k.resourceMgr[api.TypeServer].resource[mbr.ID]
		if !ok {
			mbrDown = true
			m.k.log.WithFields(Locate(logrus.Fields{
				"replica": replica,
				"mbr":     mbr,
			})).Error("Member of replica epoch down")
			break
		}
	}
	if !mbrDown {
		// No member down, current epoch is good
		return true
	}

	// Mark replica for closing
	replica.PendingState = api.StateClosed
	replica.MasterServerID = nil
	m.saveResource(replica.ID)
	return true
}

// Returns true if the epoch is created
func createReplicaEpoch(m *ResourceMgr, replica *api.Replica) bool {

	// TODO: This assigns exactly 2 data member types; will need more for 5/7 group size

	// Already have epoch, return
	if replica.CurrentEpochID != nil {
		return true
	}

	// Only poplulate epoch if we have enough servers
	serverMgr := m.k.resourceMgr[api.TypeServer]
	list := serverMgr.GetResources()
	needed := replica.QuorumGroupSize
	if uint(len(list)) < needed {
		return false
	}

	// Create new epoch spec
	epoch := &api.EpochSpec{
		Common: api.Common{
			ID:    uuid.NewV4(),
			State: api.StateNew,
		},
		LeaseOwner:        false,
		LeaseExpireUptime: 0,
	}

	// Install epoch and save replica
	replica.CurrentEpochID = &epoch.ID
	replica.Epochs[epoch.ID.String()] = epoch
	defer m.saveResource(replica.ID)

	// Add this server
	dataType := api.ReplicaQuorumMemberTypeSync
	mbr := api.QuorumMember{
		Common: api.Common{
			Name:  m.k.runtime.Name,
			ID:    m.k.runtime.ID,
			State: api.StateUninitialized,
		},
		MemberType: dataType,
		DataState:  api.DataStateInSync,
	}
	epoch.Quorum = append(epoch.Quorum, mbr)
	if needed--; needed == 0 {
		return true
	}

	// Add home server if available
	resource, ok := serverMgr.resource[replica.HomeServerID]
	if ok && (replica.HomeServerID != m.k.runtime.ID) {
		server := resource.(*api.Server)
		mbr = api.QuorumMember{
			Common: api.Common{
				Name:  server.Name,
				ID:    server.ID,
				State: api.StateUninitialized,
			},
			MemberType: dataType,
			DataState:  api.DataStateCatchUp,
		}
		epoch.Quorum = append(epoch.Quorum, mbr)
		if needed--; needed == 0 {
			return true
		}
		dataType = api.ReplicaQuorumMemberTypeAsync
	}

	// Add other servers
	for _, resource := range list {
		server := resource.(*api.Server)
		if server.ID == m.k.runtime.ID || server.ID == replica.HomeServerID {
			continue
		}
		mbr = api.QuorumMember{
			Common: api.Common{
				Name:  server.Name,
				ID:    server.ID,
				State: api.StateUninitialized,
			},
			MemberType: dataType,
			DataState:  api.DataStateCatchUp,
		}
		epoch.Quorum = append(epoch.Quorum, mbr)
		if needed--; needed == 0 {
			return true
		}
		dataType = api.ReplicaQuorumMemberTypeAsync
	}

	m.k.log.WithFields(Locate(logrus.Fields{
		"replica": replica,
		"servers": list,
	})).Fatal("Ran out of servers to create epoch")
	return false
}

// Returns true if all sync members are in-sync.
func sendReplicaCreateReqs(m *ResourceMgr, replica *api.Replica, nextPeriod *uint16, outMsgs *msg.MsgList) bool {

	reqSent := false
	for _, mbr := range replica.Epochs[replica.CurrentEpochID.String()].Quorum {

		// If member is myself, witness or data already in-sync, continue
		if (mbr.ID == m.k.runtime.ID) ||
			(mbr.MemberType == api.ReplicaQuorumMemberTypeWitness) ||
			(mbr.DataState == api.DataStateInSync) {
			continue
		}

		msg := &msg.MsgReplicaCreateReq{
			Common: msg.Common{
				Type:      msg.MsgTypeReplicaCreateReq,
				DestID:    mbr.ID,
				SrcID:     m.k.runtime.ID,
				ReplicaID: replica.ID,
				EpochID:   *replica.CurrentEpochID,
			},
			Replica: *replica,
		}
		m.k.sendMsg(msg, outMsgs)
		reqSent = true
	}

	// If no requests sent, we are done
	if !reqSent {
		return true
	}

	// Update period for next service cycle
	if *nextPeriod > retransmitInterval {
		*nextPeriod = retransmitInterval
	}
	return false
}

func (k *Ketch) onReplicaCreateReq(req *msg.MsgReplicaCreateReq, outMsgs *msg.MsgList) {

	// TODO: create replica DB

	replica := req.Replica
	replica.MasterServerID = &req.SrcID
	replica.DataState = api.DataStateCatchUp

	// Slave replica doesn't have a lease
	for _, epoch := range replica.Epochs {
		epoch.LeaseOwner = false
		epoch.LeaseExpireUptime = 0
	}

	found := false
	for _, mbr := range replica.Epochs[replica.CurrentEpochID.String()].Quorum {

		// Set slave replica member type and data state
		if mbr.ID != k.runtime.ID {
			continue
		}
		found = true
		// Set member type and data state
		replica.MemberType = mbr.MemberType
	}
	if found {
		// Install replica, possibly replacing existing one
		mgr := k.resourceMgr[api.TypeReplica]
		mgr.resource[replica.ID] = &replica
		mgr.resourceByName[replica.Name] = replica.ID
	} else {
		k.log.WithFields(Locate(logrus.Fields{
			"req": req,
		})).Error("Replica create request with epoch that doesn't have this server")
	}

	// Send response
	var resp msg.MsgReplicaCreateResp
	resp.Common = req.Common
	resp.SrcID = req.DestID
	resp.DestID = req.SrcID
	resp.Type = msg.MsgTypeReplicaCreateResp
	k.sendMsg(&resp, outMsgs)
}

func (k *Ketch) onReplicaCreateResp(resp *msg.MsgReplicaCreateResp) {

	// Validate prepare response
	mgr := k.resourceMgr[api.TypeReplica]
	replica, ok := mgr.resource[resp.ReplicaID].(*api.Replica)
	if !ok {
		k.log.WithFields(Locate(logrus.Fields{
			"resp": resp,
		})).Error("Replica create response for unknown replica")
		return
	}
	epoch, ok := replica.Epochs[resp.EpochID.String()]
	if !ok {
		k.log.WithFields(Locate(logrus.Fields{
			"resp":    resp,
			"replica": replica,
		})).Error("Replica create response for unknown epoch")
		return
	}

	// Set quorum member in-sync
	found := false
	for i, mbr := range epoch.Quorum {
		if mbr.ID == resp.SrcID {
			epoch.Quorum[i].DataState = api.DataStateInSync
			found = true
			break
		}
	}
	if !found {
		k.log.WithFields(Locate(logrus.Fields{
			"resp":    resp,
			"replica": replica,
		})).Error("Replica create response from unknown quorum member")
		return
	}

	mgr.saveResource(resp.ReplicaID)
}
