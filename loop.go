// Copyright 2016 F. Alan Jones.  All rights reserved.
// Use of this source code is governed by a Mozilla
// license that can be found in the LICENSE file.

package ketch

import (
	"time"

	"github.com/Sirupsen/logrus"

	"github.com/watercraft/ketch/api"
	"github.com/watercraft/ketch/msg"
)

const (
	// All intervals expressed in multiples of one second.
	// TODO: Consider moving these to Config.
	retransmitInterval uint16 = 1
	leaseRenewBefore   uint16 = 3 // Renew this many seconds before lease expires
	leasePeriod        uint16 = 9
	leaseGrace         uint16 = 1 // Added to period on acceptor for safty
)

func (k *Ketch) serviceLoop() {

	nextIteration := time.Now()
	for {
		// Block until next iteration
		select {
		case <-time.After(nextIteration.Sub(time.Now())):
		case <-k.wakeServiceLoopCh:
		}

		// Assure that processing after this statement doesn't delay next iteration
		nextIteration = time.Now()

		// Do everything
		nextPeriod, outMsgs := k.process()

		k.log.WithFields(Locate(logrus.Fields{
			"nextPeriod": nextPeriod,
			"outMsgs":    len(outMsgs),
		})).Info("Service")

		// Send messages (second arg returned from process())
		k.sendMsgs(outMsgs)

		// Next iteration is nextPeriod seconds after we started processing
		nextIteration = nextIteration.Add(time.Second * time.Duration(nextPeriod))
	}
}

func (k *Ketch) process() (uint16, msg.MsgList) {

	k.Lock()
	defer k.Unlock()

	// Initialize return values
	nextPeriod := leasePeriod
	var outMsgs msg.MsgList

	// Update server list
	k.resourceMgr[api.TypeServer].RefreshResources()

	// Review resident replicas
	k.GetUptime()
	replicaMgr := k.resourceMgr[api.TypeReplica]
	for _, resource := range replicaMgr.resource {
		replica := resource.(*api.Replica)

		// If membership changed mark replica for closing
		if !markReplicaPendingClosed(replicaMgr, replica) {
			continue
		}

		// If replica has a master and it is up, run as slave.
		// We clear this when the replica is closed.
		if replica.MasterServerID != nil {
			// Open replica as slave on closed port (restart database)
			// TODO: Toggle between open/closed
			if !runReplicaOnPort(replicaMgr, replica, api.DBStateSlave, replica.DBConfig.Port) {
				// Database not running yet
				continue
			}
			continue
		}

		// If epoch is marked for closing, close and replicate it
		if (replica.CurrentEpochID != nil) && (replica.PendingState == api.StateClosed) {
			// Setup epoch on quorum members
			if !sendEpochSetupReqs(replicaMgr, replica, replica.CurrentEpochID, &nextPeriod, &outMsgs) {
				// Continue to next replica if epoch isn't open
				continue
			}
			// Take lease
			if !sendLeasePrepareReqs(replicaMgr, replica, replica.CurrentEpochID, nil, &nextPeriod, &outMsgs) {
				// Continue if we do not yet have the lease
				continue
			}
			// Close epoch
			if !sendEpochCloseReqs(replicaMgr, replica, replica.CurrentEpochID, &nextPeriod, &outMsgs) {
				// Continue to next replica if epoch isn't open
				continue
			}
			// Open replica as master closed (restart database)
			if !runReplicaOnPort(replicaMgr, replica, api.DBStateMasterClosed, replica.DBConfig.ClosedPort) {
				// Database not running yet
				continue
			}
			// Close replica
			replica.State = api.StateClosed
			replica.PendingState = ""
			// Move current epoch to prior
			if replica.PriorEpochID == nil {
				// Save current epoch as prior
				replica.PriorEpochID = replica.CurrentEpochID
			} else {
				// Discard current epoch
				delete(replica.Epochs, replica.CurrentEpochID.String())
			}
			replica.CurrentEpochID = nil
			k.resourceMgr[api.TypeReplica].saveResource(replica.ID)
		}

		// Create a new epoch if members have failed
		if !createReplicaEpoch(replicaMgr, replica) {
			continue
		}

		// Setup epoch on quorum members
		if !sendEpochSetupReqs(replicaMgr, replica, replica.CurrentEpochID, &nextPeriod, &outMsgs) {
			// Continue to next replica if epoch isn't open
			continue
		}

		// Create peer replicas
		if !sendReplicaCreateReqs(replicaMgr, replica, &nextPeriod, &outMsgs) {
			// Continue if peer replicas are not yet in-sync
			continue
		}

		// If prior epoch exists, dispose of it
		if replica.PriorEpochID != nil {
			// Take lease for prior epoch
			if !sendLeasePrepareReqs(replicaMgr, replica, replica.PriorEpochID, replica.CurrentEpochID, &nextPeriod, &outMsgs) {
				// Continue if we do not yet have the lease
				continue
			}
			// Revoke prior epoch
			if !sendEpochRevokeReqs(replicaMgr, replica, replica.PriorEpochID, replica.CurrentEpochID, &nextPeriod, &outMsgs) {
				// Continue if prior epoch is not yet revoked
				continue
			}
			// Discard prior epoch
			delete(replica.Epochs, replica.PriorEpochID.String())
			replica.PriorEpochID = nil
			k.resourceMgr[api.TypeReplica].saveResource(replica.ID)
		}

		// Take lease for current epoch; important for new epochs
		if !sendLeasePrepareReqs(replicaMgr, replica, replica.CurrentEpochID, nil, &nextPeriod, &outMsgs) {
			// Continue if we do not yet have the lease
			continue
		}

		// TODO: Discover and mark replica epoch members in-sync

		// Open current epoch
		if !sendEpochOpenReqs(replicaMgr, replica, replica.CurrentEpochID, &nextPeriod, &outMsgs) {
			// Continue to next replica if epoch isn't open
			continue
		}

		// Open replica as master (start database)
		if !runReplicaOnPort(replicaMgr, replica, api.DBStateMaster, replica.DBConfig.Port) {
			// Database not running yet
			continue
		}
	}

	return nextPeriod, outMsgs
}
