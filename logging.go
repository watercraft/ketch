// Copyright 2016 F. Alan Jones.  All rights reserved.
// Use of this source code is governed by a Mozilla
// license that can be found in the LICENSE file.

package ketch

import (
	"path/filepath"
	"runtime"

	"github.com/Sirupsen/logrus"
)

// Locate() Add file and line to logrus fields
func Locate(fields logrus.Fields) logrus.Fields {
	_, path, line, ok := runtime.Caller(1)
	if ok {
		_, file := filepath.Split(path)
		fields["file"] = file
		fields["line"] = line
	}
	return fields
}
