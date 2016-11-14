// Copyright 2016 F. Alan Jones.  All rights reserved.
// Use of this source code is governed by a Mozilla
// license that can be found in the LICENSE file.

package msg

import (
	"bytes"

	"github.com/hashicorp/go-msgpack/codec"
)

// ToBytes returns a byte slice that encodes the message.
func ToBytes(msg Msg) ([]byte, error) {
	buf := bytes.NewBuffer(nil)
	buf.WriteByte(byte(msg.GetCommon().Type))
	enc := codec.NewEncoder(buf, &codec.MsgpackHandle{})
	err := enc.Encode(msg)
	return buf.Bytes(), err
}

// FromBytes returns message decoded from a byte slice.
func FromBytes(in []byte) (Msg, error) {
	myType := MsgType(in[0])
	in = in[1:]
	msg := NewMsgByType(myType)
	r := bytes.NewReader(in)
	dec := codec.NewDecoder(r, &codec.MsgpackHandle{})
	return msg, dec.Decode(msg)
}
