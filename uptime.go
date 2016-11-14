// Copyright 2016 F. Alan Jones.  All rights reserved.
// Use of this source code is governed by a Mozilla
// license that can be found in the LICENSE file.

package ketch

import (
	"syscall"

	"github.com/Sirupsen/logrus"
)

// GetUptime sets uptime seconds since boot.
// We use this value for leases as it is monotonically increasing.
// On system restart all acceptor lease values are reset, compromising availability for safety.
func (k *Ketch) GetUptime() {

	var info syscall.Sysinfo_t
	err := syscall.Sysinfo(&info)
	if err != nil {
		k.log.WithFields(Locate(logrus.Fields{
			"err": err,
		})).Fatal("Failed to retrieve uptime")
	}
	k.uptime = info.Uptime
}
