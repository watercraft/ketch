// Copyright 2016 F. Alan Jones.  All rights reserved.
// Use of this source code is governed by a Mozilla
// license that can be found in the LICENSE file.

package ketch

import (
	"net/http"

	"github.com/Sirupsen/logrus"
	"github.com/satori/go.uuid"

	"github.com/watercraft/ketch/api"
	"github.com/watercraft/ketch/msg"
)

type EpochMgr struct {
	ResourceMgr
}

func (k *Ketch) installEpochMgr() {
	m := &EpochMgr{
		ResourceMgr: ResourceMgr{
			myType:    api.TypeEpoch,
			assignIDs: false,
			named:     false,
			persist:   true,
		},
	}
	m.Init(k, m)
}

func (m *EpochMgr) InitResource(in api.Resource) (error, int) {
	return nil, http.StatusOK
}

func (m *EpochMgr) GetList() api.ResourceList {
	return nil
}

func (m *EpochMgr) UpdateAfterLoad(resource api.Resource) bool {
	return false
}

// Send setup requests for the current epoch.
// Returns true if all epoch members are open.
func sendEpochSetupReqs(m *ResourceMgr, replica *api.Replica, epochID *uuid.UUID, nextPeriod *uint16, outMsgs *msg.MsgList) bool {

	// Validate call
	epoch, ok := replica.Epochs[epochID.String()]
	if !ok {
		m.k.log.WithFields(Locate(logrus.Fields{
			"epochID": epochID,
			"replica": replica,
		})).Fatal("Send epoch setup requests for unknown epoch")
	}

	var count uint = 0
	for _, mbr := range epoch.Quorum {
		switch mbr.State {
		case api.StateUninitialized:
			msg := &msg.MsgEpochSetupReq{
				Common: msg.Common{
					Type:      msg.MsgTypeEpochSetupReq,
					DestID:    mbr.ID,
					SrcID:     m.k.runtime.ID,
					ReplicaID: replica.ID,
					EpochID:   *epochID,
				},
			}
			m.k.sendMsg(msg, outMsgs)
			if *nextPeriod > retransmitInterval {
				*nextPeriod = retransmitInterval
			}
		case api.StateNew, api.StateOpen, api.StateClosed:
			count++
		default:
			m.k.log.WithFields(Locate(logrus.Fields{
				"replica": replica,
				"mbr":     mbr,
			})).Error("Failed to send epoch setup request; unexpected member state")
		}
	}
	return count == replica.QuorumGroupSize
}

func (k *Ketch) onEpochSetupReq(req *msg.MsgEpochSetupReq, outMsgs *msg.MsgList) {

	// Create epoch resource in new state
	var epoch api.Epoch
	epoch.ID = req.EpochID
	epoch.State = api.StateNew
	epoch.ReplicaID = req.ReplicaID
	epoch.SuccessorEpochID = nil
	var list api.ResourceList
	list = append(list, &epoch)
	k.resourceMgr[api.TypeEpoch].CreateResources(list)

	// Send response
	var resp msg.MsgEpochSetupResp
	resp.Common = req.Common
	resp.SrcID = req.DestID
	resp.DestID = req.SrcID
	resp.Type = msg.MsgTypeEpochSetupResp
	k.sendMsg(&resp, outMsgs)
}

func (k *Ketch) onEpochSetupResp(resp *msg.MsgEpochSetupResp) {

	// Validate response
	mgr := k.resourceMgr[api.TypeReplica]
	replica, ok := mgr.resource[resp.ReplicaID].(*api.Replica)
	if !ok {
		k.log.WithFields(Locate(logrus.Fields{
			"resp": resp,
		})).Error("Epoch setup response for unknown replica")
		return
	}
	epoch, ok := replica.Epochs[resp.EpochID.String()]
	if !ok {
		k.log.WithFields(Locate(logrus.Fields{
			"resp":    resp,
			"replica": replica,
		})).Error("Epoch setup response for unknown epoch")
		return
	}

	// Mark epoch member open and save result
	for i, mbr := range epoch.Quorum {
		if mbr.ID == resp.SrcID {
			epoch.Quorum[i].State = api.StateNew
			break
		}
	}
	mgr.saveResource(replica.ID)
}

// Send open requests for given epoch.
// Returns true if all epoch members are open.
func sendEpochOpenReqs(m *ResourceMgr, replica *api.Replica, epochID *uuid.UUID, nextPeriod *uint16, outMsgs *msg.MsgList) bool {

	// Validate call
	epoch, ok := replica.Epochs[epochID.String()]
	if !ok {
		m.k.log.WithFields(Locate(logrus.Fields{
			"epochID": epochID,
			"replica": replica,
		})).Fatal("Send epoch open requests for unknown epoch")
	}

	var count uint = 0
	for _, mbr := range epoch.Quorum {
		switch mbr.State {
		case api.StateNew, api.StateClosed:
			msg := &msg.MsgEpochOpenReq{
				Common: msg.Common{
					Type:      msg.MsgTypeEpochOpenReq,
					DestID:    mbr.ID,
					SrcID:     m.k.runtime.ID,
					ReplicaID: replica.ID,
					EpochID:   *epochID,
				},
			}
			m.k.sendMsg(msg, outMsgs)
			if *nextPeriod > retransmitInterval {
				*nextPeriod = retransmitInterval
			}
		case api.StateOpen:
			count++
		default:
			m.k.log.WithFields(Locate(logrus.Fields{
				"replica": replica,
				"mbr":     mbr,
			})).Error("Failed to send epoch open request; unexpected member state")
		}
	}
	return count == replica.QuorumGroupSize
}

func (k *Ketch) onEpochOpenReq(req *msg.MsgEpochOpenReq, outMsgs *msg.MsgList) {

	mgr := k.resourceMgr[api.TypeEpoch]
	epoch, ok := mgr.resource[req.EpochID].(*api.Epoch)
	if !ok || epoch.ID != req.EpochID {
		k.log.WithFields(Locate(logrus.Fields{
			"req": req,
		})).Error("Epoch open request for unknown epoch")
		return
	}

	// Remove other revoked epochs for the same replica
	for _, resource := range mgr.resource {
		otherEpoch := resource.(*api.Epoch)
		if (otherEpoch.ID != epoch.ID) &&
			(otherEpoch.ReplicaID == epoch.ReplicaID) {
			delete(mgr.resource, otherEpoch.ID)
			mgr.saveResource(otherEpoch.ID)
		}
	}

	switch epoch.State {
	case api.StateNew, api.StateClosed:
		// Set epoch open
		epoch.State = api.StateOpen
		mgr.saveResource(epoch.ID)
	case api.StateOpen:
		// Already opend, send response
	default:
		k.log.WithFields(Locate(logrus.Fields{
			"req":   req,
			"epoch": epoch,
		})).Error("Epoch open request for epoch in unexpected state")
		return
	}

	// Send response
	var resp msg.MsgEpochOpenResp
	resp.Common = req.Common
	resp.SrcID = req.DestID
	resp.DestID = req.SrcID
	resp.Type = msg.MsgTypeEpochOpenResp
	k.sendMsg(&resp, outMsgs)
}

func (k *Ketch) onEpochOpenResp(resp *msg.MsgEpochOpenResp) {

	// Validate response
	mgr := k.resourceMgr[api.TypeReplica]
	replica, ok := mgr.resource[resp.ReplicaID].(*api.Replica)
	if !ok {
		k.log.WithFields(Locate(logrus.Fields{
			"resp": resp,
		})).Error("Epoch open response for unknown replica")
		return
	}
	epoch, ok := replica.Epochs[resp.EpochID.String()]
	if !ok {
		k.log.WithFields(Locate(logrus.Fields{
			"resp":    resp,
			"replica": replica,
		})).Error("Epoch open response for unknown epoch")
		return
	}

	// Mark epoch member opend and save result
	for i, mbr := range epoch.Quorum {
		if mbr.ID == resp.SrcID {
			epoch.Quorum[i].State = api.StateOpen
			break
		}
	}
	mgr.saveResource(replica.ID)
}

// Send close requests for given epoch.
// Returns true if all epoch members are open.
func sendEpochCloseReqs(m *ResourceMgr, replica *api.Replica, epochID *uuid.UUID, nextPeriod *uint16, outMsgs *msg.MsgList) bool {

	// Validate call
	epoch, ok := replica.Epochs[epochID.String()]
	if !ok {
		m.k.log.WithFields(Locate(logrus.Fields{
			"epochID": epochID,
			"replica": replica,
		})).Fatal("Send epoch close requests for unknown epoch")
	}

	var count uint = 0
	for _, mbr := range epoch.Quorum {
		switch mbr.State {
		case api.StateNew, api.StateOpen:
			msg := &msg.MsgEpochCloseReq{
				Common: msg.Common{
					Type:      msg.MsgTypeEpochCloseReq,
					DestID:    mbr.ID,
					SrcID:     m.k.runtime.ID,
					ReplicaID: replica.ID,
					EpochID:   *epochID,
				},
			}
			m.k.sendMsg(msg, outMsgs)
			if *nextPeriod > retransmitInterval {
				*nextPeriod = retransmitInterval
			}
		case api.StateClosed:
			count++
		default:
			m.k.log.WithFields(Locate(logrus.Fields{
				"replica": replica,
				"mbr":     mbr,
			})).Error("Failed to send epoch close request; unexpected member state")
		}
	}
	return count >= (replica.QuorumGroupSize / 2)
}

func (k *Ketch) onEpochCloseReq(req *msg.MsgEpochCloseReq, outMsgs *msg.MsgList) {

	mgr := k.resourceMgr[api.TypeEpoch]
	epoch, ok := mgr.resource[req.EpochID].(*api.Epoch)
	if !ok || epoch.ID != req.EpochID {
		k.log.WithFields(Locate(logrus.Fields{
			"req": req,
		})).Error("Epoch close request for unknown epoch")
		return
	}

	switch epoch.State {
	case api.StateNew, api.StateOpen:
		// Set epoch closed
		epoch.State = api.StateClosed
		epoch.PendingState = ""
		mgr.saveResource(epoch.ID)
	case api.StateClosed:
		// Already closed, send response
	default:
		k.log.WithFields(Locate(logrus.Fields{
			"req":   req,
			"epoch": epoch,
		})).Error("Epoch close request for epoch in unexpected state")
		// Close anyway; it is the safe thing to do
		epoch.State = api.StateClosed
		epoch.PendingState = ""
		mgr.saveResource(epoch.ID)
	}

	// Send response
	var resp msg.MsgEpochCloseResp
	resp.Common = req.Common
	resp.SrcID = req.DestID
	resp.DestID = req.SrcID
	resp.Type = msg.MsgTypeEpochCloseResp
	k.sendMsg(&resp, outMsgs)
}

func (k *Ketch) onEpochCloseResp(resp *msg.MsgEpochCloseResp) {

	// Validate response
	mgr := k.resourceMgr[api.TypeReplica]
	replica, ok := mgr.resource[resp.ReplicaID].(*api.Replica)
	if !ok {
		k.log.WithFields(Locate(logrus.Fields{
			"resp": resp,
		})).Error("Epoch close response for unknown replica")
		return
	}
	epoch, ok := replica.Epochs[resp.EpochID.String()]
	if !ok {
		k.log.WithFields(Locate(logrus.Fields{
			"resp":    resp,
			"replica": replica,
		})).Error("Epoch close response for unknown epoch")
		return
	}

	// Mark epoch member closed and save result
	for i, mbr := range epoch.Quorum {
		if mbr.ID == resp.SrcID {
			epoch.Quorum[i].State = api.StateClosed
			epoch.PendingState = ""
			break
		}
	}
	mgr.saveResource(replica.ID)
}

// Send revoke requests for given epoch.
// Returns true if all epoch members are revoked.
func sendEpochRevokeReqs(m *ResourceMgr, replica *api.Replica, epochID *uuid.UUID, successorEpochID *uuid.UUID, nextPeriod *uint16, outMsgs *msg.MsgList) bool {

	// Validate call
	epoch, ok := replica.Epochs[epochID.String()]
	if !ok || (successorEpochID == nil) {
		m.k.log.WithFields(Locate(logrus.Fields{
			"epochID":          epochID,
			"replica":          replica,
			"successorEpochID": successorEpochID,
		})).Fatal("Send epoch revoke requests for unknown epoch")
	}

	var count uint = 0
	for _, mbr := range epoch.Quorum {
		switch mbr.State {
		case api.StateNew, api.StateOpen, api.StateClosed:
			if mbr.PendingState == api.StateDelete {
				count++
			} else {
				msg := &msg.MsgEpochRevokeReq{
					Common: msg.Common{
						Type:      msg.MsgTypeEpochRevokeReq,
						DestID:    mbr.ID,
						SrcID:     m.k.runtime.ID,
						ReplicaID: replica.ID,
						EpochID:   *epochID,
					},
					SuccessorEpochID: *successorEpochID,
				}
				m.k.sendMsg(msg, outMsgs)
				if *nextPeriod > retransmitInterval {
					*nextPeriod = retransmitInterval
				}
			}
		default:
			m.k.log.WithFields(Locate(logrus.Fields{
				"replica": replica,
				"mbr":     mbr,
			})).Error("Failed to send epoch revoke request; unexpected member state")
		}
	}

	return count > (replica.QuorumGroupSize / 2)
}

func (k *Ketch) onEpochRevokeReq(req *msg.MsgEpochRevokeReq, outMsgs *msg.MsgList) {

	mgr := k.resourceMgr[api.TypeEpoch]
	epoch, ok := mgr.resource[req.EpochID].(*api.Epoch)
	if !ok || epoch.ID != req.EpochID {
		k.log.WithFields(Locate(logrus.Fields{
			"req": req,
		})).Error("Epoch revoke request for unknown epoch")
		return
	}

	switch epoch.State {
	case api.StateClosed:
		// Set epoch revoked
		epoch.PendingState = api.StateDelete
		epoch.SuccessorEpochID = &req.SuccessorEpochID
		mgr.saveResource(epoch.ID)
	default:
		k.log.WithFields(Locate(logrus.Fields{
			"req":   req,
			"epoch": epoch,
		})).Error("Epoch revoke request for epoch in unexpected state")
		// Revoke anyway; it is the safe thing to do
		epoch.PendingState = api.StateDelete
		epoch.SuccessorEpochID = &req.SuccessorEpochID
		mgr.saveResource(epoch.ID)
	}

	// Send response
	var resp msg.MsgEpochRevokeResp
	resp.Common = req.Common
	resp.SrcID = req.DestID
	resp.DestID = req.SrcID
	resp.Type = msg.MsgTypeEpochRevokeResp
	k.sendMsg(&resp, outMsgs)
}

func (k *Ketch) onEpochRevokeResp(resp *msg.MsgEpochRevokeResp) {

	// Validate response
	mgr := k.resourceMgr[api.TypeReplica]
	replica, ok := mgr.resource[resp.ReplicaID].(*api.Replica)
	if !ok {
		k.log.WithFields(Locate(logrus.Fields{
			"resp": resp,
		})).Error("Epoch revoke response for unknown replica")
		return
	}
	epoch, ok := replica.Epochs[resp.EpochID.String()]
	if !ok {
		k.log.WithFields(Locate(logrus.Fields{
			"resp":    resp,
			"replica": replica,
		})).Error("Epoch revoke response for unknown epoch")
		return
	}

	// Mark epoch member revoked and save result
	for i, mbr := range epoch.Quorum {
		if mbr.ID == resp.SrcID {
			epoch.Quorum[i].PendingState = api.StateDelete
		}
	}

	mgr.saveResource(replica.ID)
}
