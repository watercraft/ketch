// Copyright 2016 F. Alan Jones.  All rights reserved.
// Use of this source code is governed by a Mozilla
// license that can be found in the LICENSE file.

package ketch

import (
	"github.com/Sirupsen/logrus"
	"github.com/hashicorp/memberlist"

	"github.com/watercraft/ketch/api"
	"github.com/watercraft/ketch/msg"
)

func (k *Ketch) sendMsgs(msgs msg.MsgList) {
	for _, myMsg := range msgs {
		buf, err := msg.ToBytes(myMsg)
		if err != nil {
			k.log.WithFields(Locate(logrus.Fields{
				"err": err,
				"msg": myMsg,
			})).Info("Failed to encode message")
		}
		dest := myMsg.GetCommon().Dest
		node := &memberlist.Node{
			Addr: dest.Addr,
			Port: dest.Port,
		}
		k.list.SendToUDP(node, buf)
	}
}

func (k *Ketch) sendMsg(msg msg.Msg, outMsgs *msg.MsgList) {

	common := msg.GetCommon()
	resource, ok := k.resourceMgr[api.TypeServer].resource[common.DestID]
	if !ok {
		k.log.WithFields(Locate(logrus.Fields{
			"msg": msg,
		})).Info("Attempt to send to unknown server")
		return
	}
	common.Dest = resource.(*api.Server).Endpoint
	*outMsgs = append(*outMsgs, msg)
}
