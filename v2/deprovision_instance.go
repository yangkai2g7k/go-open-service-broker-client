/*
Copyright 2019 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v2

import (
	"fmt"
	"net/http"
)

func (c *client) DeprovisionInstance(r *DeprovisionRequest) (*DeprovisionResponse, error) {
	if err := validateDeprovisionRequest(r); err != nil {
		return nil, err
	}

	fullURL := fmt.Sprintf(serviceInstanceURLFmt, c.URL, r.InstanceID)

	params := map[string]string{
		VarKeyServiceID: string(r.ServiceID),
		VarKeyPlanID:    string(r.PlanID),
	}
	if r.AcceptsIncomplete {
		params[AcceptsIncomplete] = "true"
	}
	if r.Force {
		params[Force] = "true"
	}

	response, err := c.prepareAndDo(http.MethodDelete, fullURL, params, nil, r.OriginatingIdentity)
	if err != nil {
		return nil, err
	}

	defer func() {
		drainReader(response.Body)
		response.Body.Close()
	}()

	switch response.StatusCode {
	case http.StatusOK:
		return &DeprovisionResponse{}, nil
	case http.StatusGone:
		return &DeprovisionResponse{}, c.handleFailureResponse(response)
	case http.StatusAccepted:
		if !r.AcceptsIncomplete {
			// If the client did not signify that it could handle asynchronous
			// operations, a '202 Accepted' response should be treated as an error.
			return nil, c.handleFailureResponse(response)
		}

		responseBodyObj := &asyncSuccessResponseBody{}
		if err := c.unmarshalResponse(response, responseBodyObj); err != nil {
			return nil, err
		}

		var opPtr *OperationKey
		if responseBodyObj.Operation != nil {
			opStr := *responseBodyObj.Operation
			op := OperationKey(opStr)
			opPtr = &op
		}

		userResponse := &DeprovisionResponse{
			Async:        true,
			OperationKey: opPtr,
		}

		return userResponse, nil
	default:
		return nil, c.handleFailureResponse(response)
	}
}

func validateDeprovisionRequest(request *DeprovisionRequest) error {
	if request.InstanceID == "" {
		return required("instanceID")
	}

	if request.ServiceID == "" {
		return required("serviceID")
	}

	if request.PlanID == "" {
		return required("planID")
	}

	return nil
}
