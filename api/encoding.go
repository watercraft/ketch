// Copyright 2016 F. Alan Jones.  All rights reserved.
// Use of this source code is governed by a Mozilla
// license that can be found in the LICENSE file.

package api

import (
	"encoding/json"
	"net/http"
)

func NewByType(myType Type) Resource {

	switch myType {
	case TypeRuntime:
		return new(Runtime)
	case TypeServer:
		return new(Server)
	case TypeEpoch:
		return new(Epoch)
	case TypeReplica:
		return new(Replica)
	}
	return nil
}

func MarshalList(myType Type, list ResourceList) ([]byte, error) {

	var resc ResourceBody
	for _, item := range list {
		data := Data{
			Type: myType,
			ID:   item.GetCommon().ID,
		}
		var err error
		data.Attributes, err = json.Marshal(item)
		if err != nil {
			return nil, err
		}
		resc.Data = append(resc.Data, data)
	}
	out, err := json.Marshal(resc)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func UnmarshalList(myType Type, req *http.Request) (ResourceList, error) {

	// Unmarshal resouce struct
	var resc ResourceBody
	decoder := json.NewDecoder(req.Body)
	err := decoder.Decode(&resc)
	if err != nil {
		return nil, err
	}
	defer req.Body.Close()

	// Extract data items
	var list ResourceList
	for _, item := range resc.Data {
		dst := NewByType(myType)
		err = json.Unmarshal(item.Attributes, dst)
		if err != nil {
			return nil, err
		}
		list = append(list, dst)
	}

	return list, nil
}
