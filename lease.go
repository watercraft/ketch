// Copyright 2016 F. Alan Jones.  All rights reserved.
// Use of this source code is governed by a Mozilla
// license that can be found in the LICENSE file.

package ketch

import (
	"github.com/Sirupsen/logrus"
	"github.com/satori/go.uuid"

	"github.com/watercraft/ketch/api"
	"github.com/watercraft/ketch/msg"
)

// Returns true when we have the lease.
func sendLeasePrepareReqs(m *ResourceMgr, replica *api.Replica, epochID *uuid.UUID, successorEpochID *uuid.UUID, nextPeriod *uint16, outMsgs *msg.MsgList) bool {

	// Validate call
	epoch, ok := replica.Epochs[epochID.String()]
	if !ok {
		m.k.log.WithFields(Locate(logrus.Fields{
			"epochID": epochID,
			"replica": replica,
		})).Fatal("Send lease prepare requests for unknown epoch")
	}

	// If not time to renew, set nextPeriod and return
	// Note, LeaseExpireUptime starts with zero but is signed, so this test still works
	timeToRenew := epoch.LeaseExpireUptime - int64(leaseRenewBefore)
	if (timeToRenew > 0) && (m.k.uptime <= timeToRenew) {
		// Successfully renewed lease, set nextPeriod to renew a lease
		// Add extra to pass test for sending renewal above
		timeLeft := uint16(timeToRenew-m.k.uptime) + retransmitInterval
		if *nextPeriod > timeLeft {
			*nextPeriod = timeLeft
		}
		// Return value set in onLeaseProposeResp()
		return epoch.LeaseOwner
	}

	// Allocate ballot sequence number
	epoch.BallotSequence++
	epoch.BallotNumber = api.BallotNumber{
		ServerID: m.k.runtime.ID,
		Sequence: epoch.BallotSequence,
	}

	// Send pepare requests
	epoch.LeasePhase = api.LeasePhasePrepare
	for i, mbr := range epoch.Quorum {
		switch mbr.State {
		case api.StateNew, api.StateOpen, api.StateClosed:
		default:
			m.k.log.WithFields(Locate(logrus.Fields{
				"mbr":     mbr,
				"replica": replica,
			})).Error("Send lease prepare request for member in unexpected state")
			continue
		}
		epoch.Quorum[i].Accepted = false
		epoch.Quorum[i].LeaseOwned = false
		msg := &msg.MsgLeasePrepareReq{
			Common: msg.Common{
				Type:      msg.MsgTypeLeasePrepareReq,
				DestID:    mbr.ID,
				SrcID:     m.k.runtime.ID,
				ReplicaID: replica.ID,
				EpochID:   epoch.ID,
			},
			BallotNumber:     epoch.BallotNumber,
			SuccessorEpochID: successorEpochID,
		}
		m.k.sendMsg(msg, outMsgs)
	}
	m.k.resourceMgr[api.TypeReplica].saveResource(replica.ID)

	// Update period for next service cycle
	if *nextPeriod > retransmitInterval {
		*nextPeriod = retransmitInterval
	}

	return false
}

func (k *Ketch) onLeasePrepareReq(req *msg.MsgLeasePrepareReq, outMsgs *msg.MsgList) {

	// Prepare response response
	var resp msg.MsgLeasePrepareResp
	resp.Common = req.Common
	resp.SrcID = req.DestID
	resp.DestID = req.SrcID
	resp.Type = msg.MsgTypeLeasePrepareResp
	resp.BallotNumber = req.BallotNumber
	resp.ProposalOwnerID = nil
	resp.SuccessorMismatch = true

	// Send with mismatch for missing epoch
	defer k.sendMsg(&resp, outMsgs)

	mgr := k.resourceMgr[api.TypeEpoch]
	epoch, ok := mgr.resource[req.EpochID].(*api.Epoch)
	if !ok || epoch.ID != req.EpochID {
		k.log.WithFields(Locate(logrus.Fields{
			"req": req,
		})).Error("Lease prepare request for unknown epoch")
		return
	}
	switch epoch.State {
	case api.StateNew, api.StateOpen, api.StateClosed:
	default:
		k.log.WithFields(Locate(logrus.Fields{
			"req":   req,
			"epoch": epoch,
		})).Error("Lease prepare request for epoch in unexpected state")
		return
	}

	// Check for promised ballot with higher number
	if epoch.Acceptor.HighestPromised.LessThan(&req.BallotNumber) {
		epoch.Acceptor.HighestPromised = req.BallotNumber
		mgr.saveResource(epoch.ID)
	}
	resp.HighestPromised = epoch.Acceptor.HighestPromised

	// Check for accepted proposal
	if k.uptime < epoch.Acceptor.ProposalExpireUptime {
		resp.ProposalOwnerID = &epoch.Acceptor.ProposalOwnerID
	}

	// Check for revoked lease and return successor mismatch
	resp.SuccessorMismatch = false
	if epoch.SuccessorEpochID != nil {
		if (req.SuccessorEpochID == nil) ||
			(*epoch.SuccessorEpochID != *req.SuccessorEpochID) {
			k.log.WithFields(Locate(logrus.Fields{
				"req":   req,
				"epoch": epoch,
			})).Error("Lease prepare request successor doesn't match")
			resp.SuccessorMismatch = true
		}
	}
}

func (k *Ketch) onLeasePrepareResp(resp *msg.MsgLeasePrepareResp, outMsgs *msg.MsgList) {

	// Validate prepare response
	mgr := k.resourceMgr[api.TypeReplica]
	replica, ok := mgr.resource[resp.ReplicaID].(*api.Replica)
	if !ok {
		k.log.WithFields(Locate(logrus.Fields{
			"resp": resp,
		})).Error("Lease prepare response for unknown replica")
		return
	}
	epoch, ok := replica.Epochs[resp.EpochID.String()]
	if !ok {
		k.log.WithFields(Locate(logrus.Fields{
			"resp":    resp,
			"replica": replica,
		})).Error("Lease prepare response for unknown epoch")
		return
	}
	if !epoch.BallotNumber.Equal(&resp.BallotNumber) {
		k.log.WithFields(Locate(logrus.Fields{
			"resp":    resp,
			"replica": replica,
			"epoch":   epoch,
		})).Error("Lease prepare response for unknown ballot")
		return
	}
	if epoch.LeasePhase != api.LeasePhasePrepare {
		// Since we progress to propose on a majority,
		// it is normal to see extra prepare responces.
		return
	}

	// If there is another accepted proposal or epoch is revoked
	// to a different successor, remove replica.
	defer mgr.saveResource(resp.ReplicaID)
	if ((resp.ProposalOwnerID != nil) && (*resp.ProposalOwnerID != k.runtime.ID)) ||
		resp.SuccessorMismatch {
		k.log.WithFields(Locate(logrus.Fields{
			"resp":    resp,
			"replica": replica,
		})).Error("Lease prepare response with conflicting proposal or successor")
		delete(mgr.resource, replica.ID)
		return
	}

	// Count prepare responses
	var count uint = 0
	leaseOwned := false
	for i, mbr := range epoch.Quorum {
		switch mbr.State {
		case api.StateNew, api.StateOpen, api.StateClosed:
		default:
			k.log.WithFields(Locate(logrus.Fields{
				"mbr":     mbr,
				"replica": replica,
				"resp":    resp,
			})).Error("Lease prepare response with epoch member in unexpected state")
			continue
		}
		if mbr.Accepted {
			count++
			continue
		}
		if mbr.ID == resp.SrcID {
			if (resp.ProposalOwnerID == nil) ||
				(*resp.ProposalOwnerID == k.runtime.ID) {
				// Report if lease proposal is owned
				if resp.ProposalOwnerID != nil {
					epoch.Quorum[i].LeaseOwned = true
				}
				if epoch.Quorum[i].LeaseOwned {
					leaseOwned = true
				}
				// Good response, count it
				epoch.Quorum[i].Accepted = true
				count++
			} else {
				// Update seq from highest promised for next retry
				if epoch.BallotSequence < resp.HighestPromised.Sequence {
					epoch.BallotSequence = resp.HighestPromised.Sequence
				}
			}
		}
	}
	if count <= (replica.QuorumGroupSize / 2) {
		return
	}

	// If acceptors report lease expired while we believe it is still in force, exit.
	// TODO: Cleanup this replica without affecting all
	if (!leaseOwned) && epoch.LeaseOwner && (k.uptime < epoch.LeaseExpireUptime) {
		k.log.WithFields(Locate(logrus.Fields{
			"replica": replica,
			"resp":    resp,
			"uptime":  k.uptime,
			"expire":  epoch.LeaseExpireUptime,
		})).Fatal("Lease expired unexpectedly")
	}

	// Send lease propose request
	epoch.LeasePhase = api.LeasePhasePropose
	for i, mbr := range epoch.Quorum {
		epoch.Quorum[i].Accepted = false
		msg := &msg.MsgLeaseProposeReq{
			Common: msg.Common{
				Type:      msg.MsgTypeLeaseProposeReq,
				DestID:    mbr.ID,
				SrcID:     k.runtime.ID,
				ReplicaID: replica.ID,
				EpochID:   epoch.ID,
			},
			BallotNumber:    epoch.BallotNumber,
			ProposedTimeout: leasePeriod,
		}
		k.sendMsg(msg, outMsgs)
	}

	// Update lease expire time
	epoch.LeaseExpireUptime = k.uptime + int64(leasePeriod)
}

func (k *Ketch) onLeaseProposeReq(req *msg.MsgLeaseProposeReq, outMsgs *msg.MsgList) {

	mgr := k.resourceMgr[api.TypeEpoch]
	epoch, ok := mgr.resource[req.EpochID].(*api.Epoch)
	if !ok || epoch.ID != req.EpochID {
		k.log.WithFields(Locate(logrus.Fields{
			"req": req,
		})).Error("Lease propose request for unknown epoch")
		return
	}
	switch epoch.State {
	case api.StateNew, api.StateOpen, api.StateClosed:
	default:
		k.log.WithFields(Locate(logrus.Fields{
			"req":   req,
			"epoch": epoch,
		})).Error("Lease propose request for epoch in unexpected state")
		return
	}

	// If the req ballot >= highest promised, accept proposal
	if !req.BallotNumber.LessThan(&epoch.Acceptor.HighestPromised) {
		epoch.Acceptor.ProposalExpireUptime = k.uptime + int64(req.ProposedTimeout) + int64(leaseGrace)
		epoch.Acceptor.ProposalOwnerID = req.SrcID
		mgr.saveResource(req.EpochID)
	}

	// Send response
	var resp msg.MsgLeaseProposeResp
	resp.Common = req.Common
	resp.SrcID = req.DestID
	resp.DestID = req.SrcID
	resp.Type = msg.MsgTypeLeaseProposeResp
	resp.BallotNumber = req.BallotNumber
	resp.HasAcceptedProposal = false
	if k.uptime < epoch.Acceptor.ProposalExpireUptime {
		resp.HasAcceptedProposal = true
		resp.ProposalOwnerID = epoch.Acceptor.ProposalOwnerID
	}
	resp.EpochState = epoch.State
	k.sendMsg(&resp, outMsgs)
}

func (k *Ketch) onLeaseProposeResp(resp *msg.MsgLeaseProposeResp) {

	// Validate propose response
	mgr := k.resourceMgr[api.TypeReplica]
	replica, ok := mgr.resource[resp.ReplicaID].(*api.Replica)
	if !ok {
		k.log.WithFields(Locate(logrus.Fields{
			"resp": resp,
		})).Error("Lease propose response for unknown replica")
		return
	}
	epoch, ok := replica.Epochs[resp.EpochID.String()]
	if !ok {
		k.log.WithFields(Locate(logrus.Fields{
			"resp":    resp,
			"replica": replica,
		})).Error("Lease propose response for unknown epoch")
		return
	}
	if !epoch.BallotNumber.Equal(&resp.BallotNumber) {
		k.log.WithFields(Locate(logrus.Fields{
			"resp":    resp,
			"replica": replica,
			"epoch":   epoch,
		})).Error("Lease propose response for unknown ballot")
		return
	}
	if epoch.LeasePhase != api.LeasePhasePropose {
		// Since we progress to propose on a majority,
		// it is normal to see extra propose responces.
		// Usually we hold the lease with this phase setting,
		// so the responses drain which allows us to print
		// an info message here.
		k.log.WithFields(Locate(logrus.Fields{
			"resp":    resp,
			"replica": replica,
			"epoch":   epoch,
		})).Info("Lease propose response in unexpected phase")
		return
	}

	// Count propose responses
	defer mgr.saveResource(resp.ReplicaID)
	var count uint = 0
	for i, mbr := range epoch.Quorum {
		switch mbr.State {
		case api.StateNew, api.StateOpen, api.StateClosed:
		default:
			k.log.WithFields(Locate(logrus.Fields{
				"mbr":     mbr,
				"replica": replica,
			})).Error("Lease propose response with epoch member in unexpected state")
			continue
		}
		if !mbr.Accepted {
			count++
			continue
		}
		if mbr.ID == resp.SrcID && resp.ProposalOwnerID == k.runtime.ID {
			// If epoch not open, close replica
			if (resp.EpochState != api.StateOpen) && (replica.State != api.StateClosed) {
				replica.PendingState = api.StateClosed
				k.resourceMgr[api.TypeReplica].saveResource(replica.ID)
			}
			// Count response
			epoch.Quorum[i].Accepted = false
			count++
		}
	}
	if count <= (replica.QuorumGroupSize / 2) {
		return
	}

	// We have the lease until expire uptime set in prepare response
	epoch.LeaseOwner = true
}
