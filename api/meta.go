// Copyright 2016 F. Alan Jones.  All rights reserved.
// Use of this source code is governed by a Mozilla
// license that can be found in the LICENSE file.

package api

import (
	"encoding/json"
	"net"

	"github.com/satori/go.uuid"
)

// These structures are basic building blocks of a JSON API.

// URLBase is the base string from which URLs are constructed.
// Using Type here to conviently add resource types.
const URLBase Type = "/api/v1/"

// APIPort is the default management API port.
const APIPort uint = 7460

// MemberPort is the default port for the Ketch protocol.
const MemberPort uint = 7459

// Type is a type for the type field of objects.
type Type string

// State is an option lifecycle state: new, open, closed
type State string

const (
	StateUninitialized State = "uninitialized"
	StateNew           State = "new"
	StateOpen          State = "open"
	StateClosed        State = "closed"
	StateDelete        State = "delete"
)

// Common provides common attributes for all objects.
type Common struct {
	// Name is a human readable identifier for the object
	Name string `json:"name,omitempty"`
	// ID is an optional unique identifier
	ID uuid.UUID `json:"id,omitempty"`
	// State is an option lifecycle state: new, open, closed
	State State `json:"state,omitempty"`
	// PendingState is a request to change state that is in progress
	PendingState State `json:"pendingState,omitempty"`
}

// Endpoint is an instance of the Ketch membership service.
// This is shared by the Runtime and Server types.
type Endpoint struct {
	Addr net.IP `json:"addr"`
	Port uint16 `json:"port"`
}

// Data provides the data portion of an API response.
type Data struct {
	// Type describes the type of object.
	Type Type `json:"type"`
	// ID is an optional unique identifier
	// This is duplicated in the common attributes.
	ID uuid.UUID `json:"id,omitempty"`
	// Attributes is unique attributes of an object.
	Attributes json.RawMessage `json:"attributes,omitempty"`
}

// Error provides an error for the errors portion of an  response.
// Note, may contain informational messages in a successful reponse.
type Error struct {
	// Status is an HTTP status code (see net/http/status.go)
	Status int `json:"status"`
	// Title is the short text matching the status, see http.StatusText()
	Title string `json:"title"`
	// Detail provides additional information
	Detail string `json:"detail,omitempty"`
}

// ResourceBody provides the structure of a request or response.
type ResourceBody struct {
	Data []Data `json:"data,omitempty"`
	// Errors is omitted on a successful response unless there is additional information.
	Errors []Error `json:"errors,omitempty"`
}
