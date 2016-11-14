// Copyright 2016 F. Alan Jones.  All rights reserved.
// Use of this source code is governed by a Mozilla
// license that can be found in the LICENSE file.

package ketch

import (
	"encoding/binary"
	"io"
	"os"
	"time"

	"github.com/Sirupsen/logrus"
)

const (
	// Location of utmp file
	kUtmpFile string = "/var/run/utmp"
	// Type of reboot record
	kTypeBootTime int16 = 2
)

// Structures for reading utmp
type ExitStatus struct {
	X__e_termination int16
	X__e_exit        int16
}

type TimeVal struct {
	Sec  int32
	Usec int32
}

type Utmp struct {
	Type              int16
	Pad_cgo_0         [2]byte
	Pid               int32
	Line              [32]byte
	Id                [4]byte
	User              [32]byte
	Host              [256]byte
	Exit              ExitStatus
	Session           int32
	Tv                TimeVal
	Addr_v6           [4]int32
	X__glibc_reserved [20]byte
}

// bootTime returns the last reboot time
func (k *Ketch) getBootTime() (*time.Time, error) {

	// Open utmp file
	file, err := os.Open(kUtmpFile)
	if err != nil {
		k.log.WithFields(Locate(logrus.Fields{
			"utmpFile": kUtmpFile,
			"err":      err,
		})).Error("Failed to open utmp file")
		return nil, err
	}
	defer func() {
		err := file.Close()
		if err != nil {
			k.log.WithFields(Locate(logrus.Fields{
				"utmpFile": kUtmpFile,
				"err":      err,
			})).Info("Failed to close utmp file")
		}
	}()

	// Read until we find a reboot record
	var data Utmp
	for {
		err = binary.Read(file, binary.LittleEndian, &data)
		if err != nil {
			var msg string
			if err == io.EOF {
				msg = "Failed to find reboot record"
			} else {
				msg = "Failed read"
			}
			k.log.WithFields(Locate(logrus.Fields{
				"utmpFile": kUtmpFile,
				"err":      err,
			})).Error(msg)
			return nil, err
		}
		if data.Type == kTypeBootTime {
			bootTime := time.Unix(int64(data.Tv.Sec), int64(data.Tv.Usec))
			return &bootTime, nil
		}
	}
}
