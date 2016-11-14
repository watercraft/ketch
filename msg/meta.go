// Copyright 2016 F. Alan Jones.  All rights reserved.
// Use of this source code is governed by a Mozilla
// license that can be found in the LICENSE file.

package msg

import (
	"github.com/satori/go.uuid"

	"github.com/watercraft/ketch/api"
)

// MsgType is an integer ID of a type of message that can be received
// on network channels from other members.
type MsgType byte

// The list of available message types.
const (
	MsgTypeNoOp                 = iota
	MsgTypeEpochSetupReq        // 1
	MsgTypeEpochSetupResp       // 2
	MsgTypeEpochOpenReq         // 3
	MsgTypeEpochOpenResp        // 4
	MsgTypeEpochCloseReq        // 5
	MsgTypeEpochCloseResp       // 6
	MsgTypeEpochRevokeReq       // 7
	MsgTypeEpochRevokeResp      // 8
	MsgTypeLeasePrepareReq      // 9
	MsgTypeLeasePrepareResp     // 10
	MsgTypeLeaseProposeReq      // 11
	MsgTypeLeaseProposeResp     // 12
	MsgTypeReplicaCreateReq     // 13
	MsgTypeReplicaCreateResp    // 14
	MsgTypeReplicaSetInSyncReq  // 15
	MsgTypeReplicaSetInSyncResp // 16
)

func NewMsgByType(myType MsgType) Msg {
	switch myType {
	case MsgTypeEpochSetupReq:
		return new(MsgEpochSetupReq)
	case MsgTypeEpochSetupResp:
		return new(MsgEpochSetupResp)
	case MsgTypeEpochOpenReq:
		return new(MsgEpochOpenReq)
	case MsgTypeEpochOpenResp:
		return new(MsgEpochOpenResp)
	case MsgTypeEpochCloseReq:
		return new(MsgEpochCloseReq)
	case MsgTypeEpochCloseResp:
		return new(MsgEpochCloseResp)
	case MsgTypeEpochRevokeReq:
		return new(MsgEpochRevokeReq)
	case MsgTypeEpochRevokeResp:
		return new(MsgEpochRevokeResp)
	case MsgTypeLeasePrepareReq:
		return new(MsgLeasePrepareReq)
	case MsgTypeLeasePrepareResp:
		return new(MsgLeasePrepareResp)
	case MsgTypeLeaseProposeReq:
		return new(MsgLeaseProposeReq)
	case MsgTypeLeaseProposeResp:
		return new(MsgLeaseProposeResp)
	case MsgTypeReplicaCreateReq:
		return new(MsgReplicaCreateReq)
	case MsgTypeReplicaCreateResp:
		return new(MsgReplicaCreateResp)
	}
	return nil
}

type Msg interface {
	GetCommon() *Common
}

type MsgList []Msg

// Common provides common attributes for all messages.
type Common struct {
	// Type is the type of message
	Type MsgType
	// Dest is the server IP and port for the destination.
	Dest api.Endpoint
	// DestID is the server ID for the destination.
	DestID uuid.UUID
	// SrcID is the server ID for the source.
	SrcID uuid.UUID
	// ReplicaID is replica context for the message.
	ReplicaID uuid.UUID
	// EpochID identifies the epoch context for the message.
	EpochID uuid.UUID
}
