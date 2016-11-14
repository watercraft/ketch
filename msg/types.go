// Copyright 2016 F. Alan Jones.  All rights reserved.
// Use of this source code is governed by a Mozilla
// license that can be found in the LICENSE file.

package msg

import (
	"github.com/satori/go.uuid"

	"github.com/watercraft/ketch/api"
)

type MsgEpochSetupReq struct {
	Common
}

func (m *MsgEpochSetupReq) GetCommon() *Common {
	return &m.Common
}

type MsgEpochSetupResp struct {
	Common
}

func (m *MsgEpochSetupResp) GetCommon() *Common {
	return &m.Common
}

type MsgEpochOpenReq struct {
	Common
}

func (m *MsgEpochOpenReq) GetCommon() *Common {
	return &m.Common
}

type MsgEpochOpenResp struct {
	Common
}

func (m *MsgEpochOpenResp) GetCommon() *Common {
	return &m.Common
}

type MsgEpochCloseReq struct {
	Common
}

func (m *MsgEpochCloseReq) GetCommon() *Common {
	return &m.Common
}

type MsgEpochCloseResp struct {
	Common
}

func (m *MsgEpochCloseResp) GetCommon() *Common {
	return &m.Common
}

type MsgEpochRevokeReq struct {
	Common
	SuccessorEpochID uuid.UUID
}

func (m *MsgEpochRevokeReq) GetCommon() *Common {
	return &m.Common
}

type MsgEpochRevokeResp struct {
	Common
}

func (m *MsgEpochRevokeResp) GetCommon() *Common {
	return &m.Common
}

type MsgLeasePrepareReq struct {
	Common
	BallotNumber     api.BallotNumber
	SuccessorEpochID *uuid.UUID
}

func (m *MsgLeasePrepareReq) GetCommon() *Common {
	return &m.Common
}

type MsgLeasePrepareResp struct {
	Common
	BallotNumber      api.BallotNumber
	HighestPromised   api.BallotNumber
	ProposalOwnerID   *uuid.UUID
	SuccessorMismatch bool
}

func (m *MsgLeasePrepareResp) GetCommon() *Common {
	return &m.Common
}

type MsgLeaseProposeReq struct {
	Common
	BallotNumber    api.BallotNumber
	ProposedTimeout uint16
}

func (m *MsgLeaseProposeReq) GetCommon() *Common {
	return &m.Common
}

type MsgLeaseProposeResp struct {
	Common
	BallotNumber        api.BallotNumber
	HasAcceptedProposal bool
	ProposalOwnerID     uuid.UUID
	EpochState          api.State
}

func (m *MsgLeaseProposeResp) GetCommon() *Common {
	return &m.Common
}

type MsgReplicaCreateReq struct {
	Common
	Replica api.Replica
}

func (m *MsgReplicaCreateReq) GetCommon() *Common {
	return &m.Common
}

type MsgReplicaCreateResp struct {
	Common
}

func (m *MsgReplicaCreateResp) GetCommon() *Common {
	return &m.Common
}
