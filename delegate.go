// Copyright 2016 F. Alan Jones.  All rights reserved.
// Use of this source code is governed by a Mozilla
// license that can be found in the LICENSE file.

package ketch

import (
	"github.com/Sirupsen/logrus"

	"github.com/watercraft/ketch/msg"
)

func (k *Ketch) NodeMeta(limit int) []byte {
	k.Lock()
	defer k.Unlock()

	k.log.WithFields(Locate(logrus.Fields{
		"limit": limit,
		"id":    k.runtime.ID,
	})).Info("NodeMeta")
	return k.runtime.ID.Bytes()
}

func (k *Ketch) NotifyMsg(buf []byte) {

	// Decode message
	msg, err := msg.FromBytes(buf)
	if err != nil {
		k.log.WithFields(Locate(logrus.Fields{
			"err":  err,
			"size": len(buf),
			"buf":  buf,
		})).Error("Failed to decode incoming message")
		return
	}

	select {
	case k.incomingMsgCh <- msg:
	default:
		k.log.WithFields(Locate(logrus.Fields{
			"size": len(buf),
			"msg":  msg,
		})).Error("Incomming message channel full, discarding message")
	}

	return
}

func (k *Ketch) GetBroadcasts(overhead, limit int) [][]byte {

	k.log.WithFields(Locate(logrus.Fields{
		"overhead": overhead,
		"limit":    limit,
	})).Debug("GetBroadcasts")
	return [][]byte{}
}

func (k *Ketch) LocalState(join bool) []byte {

	k.log.WithFields(Locate(logrus.Fields{
		"join": join,
	})).Debug("LocalState")
	return []byte{}
}

func (k *Ketch) MergeRemoteState(buf []byte, join bool) {

	k.log.WithFields(Locate(logrus.Fields{
		"buf":  buf,
		"join": join,
	})).Info("MergeRemoteState")
	return
}
