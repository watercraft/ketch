// Copyright 2016 F. Alan Jones.  All rights reserved.
// Use of this source code is governed by a Mozilla
// license that can be found in the LICENSE file.

package ketch

import (
	"github.com/Sirupsen/logrus"

	"github.com/watercraft/ketch/msg"
)

// dispatchIncomingMsgs dispatches messages from incoming channel
func (k *Ketch) dispatchIncomingMsgs() {

	for {
		// Pull message from channel; blocking
		myMsg := <-k.incomingMsgCh

		k.log.WithFields(Locate(logrus.Fields{
			"msg": myMsg,
		})).Info("Dispatch")

		// Dispatch message
		var outMsgs msg.MsgList
		k.Lock()
		k.GetUptime()
		switch myMsg.GetCommon().Type {
		case msg.MsgTypeEpochSetupReq:
			k.onEpochSetupReq(myMsg.(*msg.MsgEpochSetupReq), &outMsgs)
		case msg.MsgTypeEpochSetupResp:
			k.onEpochSetupResp(myMsg.(*msg.MsgEpochSetupResp))
		case msg.MsgTypeEpochOpenReq:
			k.onEpochOpenReq(myMsg.(*msg.MsgEpochOpenReq), &outMsgs)
		case msg.MsgTypeEpochOpenResp:
			k.onEpochOpenResp(myMsg.(*msg.MsgEpochOpenResp))
		case msg.MsgTypeEpochCloseReq:
			k.onEpochCloseReq(myMsg.(*msg.MsgEpochCloseReq), &outMsgs)
		case msg.MsgTypeEpochCloseResp:
			k.onEpochCloseResp(myMsg.(*msg.MsgEpochCloseResp))
		case msg.MsgTypeEpochRevokeReq:
			k.onEpochRevokeReq(myMsg.(*msg.MsgEpochRevokeReq), &outMsgs)
		case msg.MsgTypeEpochRevokeResp:
			k.onEpochRevokeResp(myMsg.(*msg.MsgEpochRevokeResp))
		case msg.MsgTypeLeasePrepareReq:
			k.onLeasePrepareReq(myMsg.(*msg.MsgLeasePrepareReq), &outMsgs)
		case msg.MsgTypeLeasePrepareResp:
			k.onLeasePrepareResp(myMsg.(*msg.MsgLeasePrepareResp), &outMsgs)
		case msg.MsgTypeLeaseProposeReq:
			k.onLeaseProposeReq(myMsg.(*msg.MsgLeaseProposeReq), &outMsgs)
		case msg.MsgTypeLeaseProposeResp:
			k.onLeaseProposeResp(myMsg.(*msg.MsgLeaseProposeResp))
		case msg.MsgTypeReplicaCreateReq:
			k.onReplicaCreateReq(myMsg.(*msg.MsgReplicaCreateReq), &outMsgs)
		case msg.MsgTypeReplicaCreateResp:
			k.onReplicaCreateResp(myMsg.(*msg.MsgReplicaCreateResp))
		}
		k.Unlock()

		// Send response messages
		k.sendMsgs(outMsgs)
	}
}
