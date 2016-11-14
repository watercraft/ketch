// Copyright 2016 F. Alan Jones.  All rights reserved.
// Use of this source code is governed by a Mozilla
// license that can be found in the LICENSE file.

package api

// Resource is an interface for resource types
type Resource interface {
	Clone() Resource
	GetCommon() *Common
}

// ResourceList is a sortable slice of Resource
type ResourceList []Resource

func (l ResourceList) Len() int {
	return len(l)
}

func (l ResourceList) Swap(i, j int) {
	l[i], l[j] = l[j], l[i]
}

func (l ResourceList) Less(i, j int) bool {
	if l[i].GetCommon().Name == l[j].GetCommon().Name {
		return l[i].GetCommon().ID.String() < l[j].GetCommon().ID.String()
	}
	return l[i].GetCommon().Name < l[j].GetCommon().Name
}
