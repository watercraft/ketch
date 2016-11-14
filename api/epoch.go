// Copyright 2016 F. Alan Jones.  All rights reserved.
// Use of this source code is governed by a Mozilla
// license that can be found in the LICENSE file.

package api

import (
	"github.com/satori/go.uuid"
)

// Type value for epoch
const TypeEpoch Type = "epoch"

// BallotNumber is the ballot number type for the Paxos Lease protocol.
// Because we persist generated sequence value we don't need a restart counter.
// When comparing ballot number, we use the sequence as the higher order field.
type BallotNumber struct {
	// Sequence is the auto increment value of ballots issued by Server.
	Sequence uint64 `json:"sequence"`
	// Server is the ID of the server issuing the ballot
	ServerID uuid.UUID `json:"serverID"`
}

// LessThan returns x < y for ballot numbers.
func (x *BallotNumber) LessThan(y *BallotNumber) bool {
	if x.Sequence == y.Sequence {
		return x.ServerID.String() < y.ServerID.String()
	}
	return x.Sequence < y.Sequence
}

// Equal returns x == y for ballot numbers.
func (x *BallotNumber) Equal(y *BallotNumber) bool {
	return uuid.Equal(x.ServerID, y.ServerID) &&
		x.Sequence == y.Sequence
}

// AcceptorState is the Paxos Lease acceptor state
type AcceptorState struct {
	// HighestPromised is the highest promised ballot number for a lease prepare request, nil if none
	HighestPromised BallotNumber `json:"highestPromised,omitempty"`
	// ProposalOwnerID is the server ID that owns the last accepted proposal
	ProposalOwnerID uuid.UUID `json:"proposalOwnerID"`
	// ProposalExpireUptime is the time since boot of host in seconds
	// when the accepted proposal expires and another can be accepted.
	ProposalExpireUptime int64 `json:"expireUptime"`
}

// Epoch provides state for the Epoch and Paxos Lease protocols.
type Epoch struct {
	Common
	// ReplicaID is replica context for the epoch.
	ReplicaID uuid.UUID `json:"replicaID"`
	// SuccessorEpochID is the ID of succesor epoch, nil if none
	SuccessorEpochID *uuid.UUID `json:"sucesserEpochID,omitempty"`
	// Acceptor is the Paxos Lease acceptor state
	Acceptor AcceptorState `json:"acceptor"`
}

func (e *Epoch) Clone() Resource {
	epoch := *e
	if epoch.SuccessorEpochID != nil {
		id := *e.SuccessorEpochID
		epoch.SuccessorEpochID = &id
	}
	return &epoch
}

func (e *Epoch) GetCommon() *Common {
	return &e.Common
}
