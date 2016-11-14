// Copyright 2016 F. Alan Jones.  All rights reserved.
// Use of this source code is governed by a Mozilla
// license that can be found in the LICENSE file.

package api

import (
	"github.com/satori/go.uuid"
)

// Type value for replica
const TypeReplica Type = "replica"

// DataState is the state of an replica: new, open or closed.
type TypeDataState string

const (
	// InSync means the associated data is up to date.
	DataStateInSync TypeDataState = "in-sync"
	// CatchUp means the associated data need to be brought up-to-date.
	DataStateCatchUp TypeDataState = "catch-up"
)

// ReplicaQuorumMemberType is the type of a replica quorum member.
type ReplicaQuorumMemberType string

const (
	// Data members that store in-sync copies of the managed data.
	ReplicaQuorumMemberTypeSync ReplicaQuorumMemberType = "sync"
	// Data members that store async copies of the managed data, with state "catch-up" above.
	ReplicaQuorumMemberTypeAsync ReplicaQuorumMemberType = "async"
	// Witness members participate in the epoch lease protocol
	// and can store an asynchronous copy of the managed data.
	ReplicaQuorumMemberTypeWitness ReplicaQuorumMemberType = "witness"
)

type LeasePhase string

const (
	// Prepare/Propose phase from Paxos Lease protocol
	LeasePhasePrepare = "prepare"
	LeasePhasePropose = "propose"
)

// QuorumMember tracks the quorum membership relation of this
// replica to servers.
type QuorumMember struct {
	// The Name and ID of the server at the time the replica was
	// created. The ID is used for all lookups and the name is
	// provided for debugging.
	Common
	// Type is the type of quorum member, either data or witness.
	MemberType ReplicaQuorumMemberType `json:"memberType,omitempty"`
	// State of the peer's data, either in-sync or catch-up
	DataState TypeDataState `json:"dataState,omitempty"`
	// Accepted tracks Paxos Lease request/reply status.
	// Used for both prepare and propose phases.
	Accepted bool `json:"accepted"`
	// LeaseOwned tracks if this member reported that the
	// lease proposal was still owned.  At least one member
	// must report the lease owned when renewing a lease.
	LeaseOwned bool `json:"leaseOwned"`
}

// EpochSpec provides the id and quorum membership of an epoch.
type EpochSpec struct {
	Common
	// EpochQuorum is this replica's quorum membership
	Quorum []QuorumMember `json:"quorum,omitempty"`
	// BallotSequence is an incrementing counter used to assign
	// ballot numbers in the lease protocol.
	BallotSequence uint64 `json:"ballotSequence"`
	// BallotNumber is the current ballot number for requests.
	BallotNumber BallotNumber `json:"ballotNumber"`
	// LeasePhase is the prepare/propose phase of the Paxos Lease protocol.
	LeasePhase LeasePhase `json:"leasePhase"`
	// LeaseOwner is true when we hold the lease
	LeaseOwner bool `json:"leaseOwner"`
	// LeaseExpireUptime is the time since boot of host in seconds
	// when the lease expires, if held
	LeaseExpireUptime int64 `json:"leaseExpireUptime"`
}

// DBSpec provides configuration for the managed database
type DBSpec struct {
	// Username/Password provide credentials to create on the
	// database after initialization.
	Username string `json:"username"`
	Password string `json:"password"`
	// Port is the service port for the database to listen on.
	Port uint16 `json:"port"`
	// ClosedPort is another port number that is configured when
	// the database is closed for access.  This is how we prevent
	// updates while synchronizing replication.
	ClosedPort uint16 `json:"closedPort"`
}

// Replica provides state for the Replica and Paxos Lease protocols.
type Replica struct {
	Common
	// Type is the type of quorum member, either data or witness.
	MemberType ReplicaQuorumMemberType `json:"memberType,omitempty"`
	// State of data in the replica, either in-sync or catch-up
	DataState TypeDataState `json:"dataState,omitempty"`
	// CurrentEpoch provides the id of the current epoch.
	CurrentEpochID *uuid.UUID `json:"currentEpochID,omitempty"`
	// PriorEpoch provides the id and quorum membership of the prior epoch.
	PriorEpochID *uuid.UUID `json:"priorEpochID,omitempty"`
	// Epochs is a map of current and prior epochs
	// The key is a string representation of the epoch ID (e.g. epochID.String())
	Epochs map[string]*EpochSpec `json:"epochs,omitempty"`
	// QuorumGroupSize sets the target size of the quorum group.  The
	// data quorum size will be a majority within the quorum.
	QuorumGroupSize uint `json:"quorumGroupSize"`
	// HomeServerID is the server ID where the replica originated from.
	// It is used to place the first synchoronous copy for the next failover.
	HomeServerID uuid.UUID `json:"homeServerID"`
	// MasterServerID is the server ID that created this replica.
	// Progress the state machine only if this server drops from the list.
	MasterServerID *uuid.UUID `json:"masterServerID,omitempty"`
	// DBConfig is configuration for the managed database.
	DBConfig DBSpec `json:"dbConfig"`
}

func (r *Replica) Clone() Resource {
	replica := *r
	if r.CurrentEpochID != nil {
		id := *r.CurrentEpochID
		replica.CurrentEpochID = &id
	}
	if r.PriorEpochID != nil {
		id := *r.PriorEpochID
		replica.PriorEpochID = &id
	}
	for key, value := range replica.Epochs {
		if value != nil {
			newvalue := *value
			replica.Epochs[key] = &newvalue
		}
	}
	if r.MasterServerID != nil {
		id := *r.MasterServerID
		replica.MasterServerID = &id
	}
	return &replica
}

func (r *Replica) GetCommon() *Common {
	return &r.Common
}
