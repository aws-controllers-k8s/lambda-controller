// Copyright Amazon.com Inc. or its affiliates. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License"). You may
// not use this file except in compliance with the License. A copy of the
// License is located at
//
//     http://aws.amazon.com/apache2.0/
//
// or in the "license" file accompanying this file. This file is distributed
// on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either
// express or implied. See the License for the specific language governing
// permissions and limitations under the License.

package broker

import (
	svcapitypes "github.com/aws-controllers-k8s/mq-controller/apis/v1alpha1"
)

// brokerCreateFailed returns whether the supplied AmazonMQ broker is marked as
// CREATE_FAILED
func brokerCreateFailed(r *resource) bool {
	if r.ko.Status.BrokerState == nil {
		return false
	}
	bs := *r.ko.Status.BrokerState
	return bs == string(svcapitypes.BrokerState_CREATION_FAILED)
}

// brokerCreateInProgress returns true if the supplied AmazonMQ broker is marked as
// CREATION_IN_PROGRESS
func brokerCreateInProgress(r *resource) bool {
	if r.ko.Status.BrokerState == nil {
		return false
	}
	bs := *r.ko.Status.BrokerState
	return bs == string(svcapitypes.BrokerState_CREATION_IN_PROGRESS)
}

// brokerDeleteInProgress returns true if the supplied AmazonMQ broker is marked as
// DELETE_IN_PROGRESS
func brokerDeleteInProgress(r *resource) bool {
	if r.ko.Status.BrokerState == nil {
		return false
	}
	bs := *r.ko.Status.BrokerState
	return bs == string(svcapitypes.BrokerState_DELETION_IN_PROGRESS)
}
