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

// Code generated by ack-generate. DO NOT EDIT.

package event_source_mapping

import (
	"context"
	"errors"
	"reflect"
	"strings"

	ackv1alpha1 "github.com/aws-controllers-k8s/runtime/apis/core/v1alpha1"
	ackcompare "github.com/aws-controllers-k8s/runtime/pkg/compare"
	ackcondition "github.com/aws-controllers-k8s/runtime/pkg/condition"
	ackerr "github.com/aws-controllers-k8s/runtime/pkg/errors"
	ackrtlog "github.com/aws-controllers-k8s/runtime/pkg/runtime/log"
	"github.com/aws/aws-sdk-go/aws"
	svcsdk "github.com/aws/aws-sdk-go/service/lambda"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	svcapitypes "github.com/aws-controllers-k8s/lambda-controller/apis/v1alpha1"
)

// Hack to avoid import errors during build...
var (
	_ = &metav1.Time{}
	_ = strings.ToLower("")
	_ = &aws.JSONValue{}
	_ = &svcsdk.Lambda{}
	_ = &svcapitypes.EventSourceMapping{}
	_ = ackv1alpha1.AWSAccountID("")
	_ = &ackerr.NotFound
	_ = &ackcondition.NotManagedMessage
	_ = &reflect.Value{}
)

// sdkFind returns SDK-specific information about a supplied resource
func (rm *resourceManager) sdkFind(
	ctx context.Context,
	r *resource,
) (latest *resource, err error) {
	rlog := ackrtlog.FromContext(ctx)
	exit := rlog.Trace("rm.sdkFind")
	defer func() {
		exit(err)
	}()
	// If any required fields in the input shape are missing, AWS resource is
	// not created yet. Return NotFound here to indicate to callers that the
	// resource isn't yet created.
	if rm.requiredFieldsMissingFromReadOneInput(r) {
		return nil, ackerr.NotFound
	}

	input, err := rm.newDescribeRequestPayload(r)
	if err != nil {
		return nil, err
	}

	var resp *svcsdk.EventSourceMappingConfiguration
	resp, err = rm.sdkapi.GetEventSourceMappingWithContext(ctx, input)
	rm.metrics.RecordAPICall("READ_ONE", "GetEventSourceMapping", err)
	if err != nil {
		if awsErr, ok := ackerr.AWSError(err); ok && awsErr.Code() == "ResourceNotFoundException" {
			return nil, ackerr.NotFound
		}
		return nil, err
	}

	// Merge in the information we read from the API call above to the copy of
	// the original Kubernetes object we passed to the function
	ko := r.ko.DeepCopy()

	if resp.BatchSize != nil {
		ko.Spec.BatchSize = resp.BatchSize
	} else {
		ko.Spec.BatchSize = nil
	}
	if resp.BisectBatchOnFunctionError != nil {
		ko.Spec.BisectBatchOnFunctionError = resp.BisectBatchOnFunctionError
	} else {
		ko.Spec.BisectBatchOnFunctionError = nil
	}
	if resp.DestinationConfig != nil {
		f2 := &svcapitypes.DestinationConfig{}
		if resp.DestinationConfig.OnFailure != nil {
			f2f0 := &svcapitypes.OnFailure{}
			if resp.DestinationConfig.OnFailure.Destination != nil {
				f2f0.Destination = resp.DestinationConfig.OnFailure.Destination
			}
			f2.OnFailure = f2f0
		}
		if resp.DestinationConfig.OnSuccess != nil {
			f2f1 := &svcapitypes.OnSuccess{}
			if resp.DestinationConfig.OnSuccess.Destination != nil {
				f2f1.Destination = resp.DestinationConfig.OnSuccess.Destination
			}
			f2.OnSuccess = f2f1
		}
		ko.Spec.DestinationConfig = f2
	} else {
		ko.Spec.DestinationConfig = nil
	}
	if resp.EventSourceArn != nil {
		ko.Spec.EventSourceARN = resp.EventSourceArn
	} else {
		ko.Spec.EventSourceARN = nil
	}
	if resp.FunctionArn != nil {
		ko.Status.FunctionARN = resp.FunctionArn
	} else {
		ko.Status.FunctionARN = nil
	}
	if resp.FunctionResponseTypes != nil {
		f5 := []*string{}
		for _, f5iter := range resp.FunctionResponseTypes {
			var f5elem string
			f5elem = *f5iter
			f5 = append(f5, &f5elem)
		}
		ko.Spec.FunctionResponseTypes = f5
	} else {
		ko.Spec.FunctionResponseTypes = nil
	}
	if resp.LastModified != nil {
		ko.Status.LastModified = &metav1.Time{*resp.LastModified}
	} else {
		ko.Status.LastModified = nil
	}
	if resp.LastProcessingResult != nil {
		ko.Status.LastProcessingResult = resp.LastProcessingResult
	} else {
		ko.Status.LastProcessingResult = nil
	}
	if resp.MaximumBatchingWindowInSeconds != nil {
		ko.Spec.MaximumBatchingWindowInSeconds = resp.MaximumBatchingWindowInSeconds
	} else {
		ko.Spec.MaximumBatchingWindowInSeconds = nil
	}
	if resp.MaximumRecordAgeInSeconds != nil {
		ko.Spec.MaximumRecordAgeInSeconds = resp.MaximumRecordAgeInSeconds
	} else {
		ko.Spec.MaximumRecordAgeInSeconds = nil
	}
	if resp.MaximumRetryAttempts != nil {
		ko.Spec.MaximumRetryAttempts = resp.MaximumRetryAttempts
	} else {
		ko.Spec.MaximumRetryAttempts = nil
	}
	if resp.ParallelizationFactor != nil {
		ko.Spec.ParallelizationFactor = resp.ParallelizationFactor
	} else {
		ko.Spec.ParallelizationFactor = nil
	}
	if resp.Queues != nil {
		f12 := []*string{}
		for _, f12iter := range resp.Queues {
			var f12elem string
			f12elem = *f12iter
			f12 = append(f12, &f12elem)
		}
		ko.Spec.Queues = f12
	} else {
		ko.Spec.Queues = nil
	}
	if resp.SelfManagedEventSource != nil {
		f13 := &svcapitypes.SelfManagedEventSource{}
		if resp.SelfManagedEventSource.Endpoints != nil {
			f13f0 := map[string][]*string{}
			for f13f0key, f13f0valiter := range resp.SelfManagedEventSource.Endpoints {
				f13f0val := []*string{}
				for _, f13f0valiter := range f13f0valiter {
					var f13f0valelem string
					f13f0valelem = *f13f0valiter
					f13f0val = append(f13f0val, &f13f0valelem)
				}
				f13f0[f13f0key] = f13f0val
			}
			f13.Endpoints = f13f0
		}
		ko.Spec.SelfManagedEventSource = f13
	} else {
		ko.Spec.SelfManagedEventSource = nil
	}
	if resp.SourceAccessConfigurations != nil {
		f14 := []*svcapitypes.SourceAccessConfiguration{}
		for _, f14iter := range resp.SourceAccessConfigurations {
			f14elem := &svcapitypes.SourceAccessConfiguration{}
			if f14iter.Type != nil {
				f14elem.Type = f14iter.Type
			}
			if f14iter.URI != nil {
				f14elem.URI = f14iter.URI
			}
			f14 = append(f14, f14elem)
		}
		ko.Spec.SourceAccessConfigurations = f14
	} else {
		ko.Spec.SourceAccessConfigurations = nil
	}
	if resp.StartingPosition != nil {
		ko.Spec.StartingPosition = resp.StartingPosition
	} else {
		ko.Spec.StartingPosition = nil
	}
	if resp.StartingPositionTimestamp != nil {
		ko.Spec.StartingPositionTimestamp = &metav1.Time{*resp.StartingPositionTimestamp}
	} else {
		ko.Spec.StartingPositionTimestamp = nil
	}
	if resp.State != nil {
		ko.Status.State = resp.State
	} else {
		ko.Status.State = nil
	}
	if resp.StateTransitionReason != nil {
		ko.Status.StateTransitionReason = resp.StateTransitionReason
	} else {
		ko.Status.StateTransitionReason = nil
	}
	if resp.Topics != nil {
		f19 := []*string{}
		for _, f19iter := range resp.Topics {
			var f19elem string
			f19elem = *f19iter
			f19 = append(f19, &f19elem)
		}
		ko.Spec.Topics = f19
	} else {
		ko.Spec.Topics = nil
	}
	if resp.TumblingWindowInSeconds != nil {
		ko.Spec.TumblingWindowInSeconds = resp.TumblingWindowInSeconds
	} else {
		ko.Spec.TumblingWindowInSeconds = nil
	}
	if resp.UUID != nil {
		ko.Status.UUID = resp.UUID
	} else {
		ko.Status.UUID = nil
	}

	rm.setStatusDefaults(ko)
	return &resource{ko}, nil
}

// requiredFieldsMissingFromReadOneInput returns true if there are any fields
// for the ReadOne Input shape that are required but not present in the
// resource's Spec or Status
func (rm *resourceManager) requiredFieldsMissingFromReadOneInput(
	r *resource,
) bool {
	return r.ko.Status.UUID == nil

}

// newDescribeRequestPayload returns SDK-specific struct for the HTTP request
// payload of the Describe API call for the resource
func (rm *resourceManager) newDescribeRequestPayload(
	r *resource,
) (*svcsdk.GetEventSourceMappingInput, error) {
	res := &svcsdk.GetEventSourceMappingInput{}

	if r.ko.Status.UUID != nil {
		res.SetUUID(*r.ko.Status.UUID)
	}

	return res, nil
}

// sdkCreate creates the supplied resource in the backend AWS service API and
// returns a copy of the resource with resource fields (in both Spec and
// Status) filled in with values from the CREATE API operation's Output shape.
func (rm *resourceManager) sdkCreate(
	ctx context.Context,
	desired *resource,
) (created *resource, err error) {
	rlog := ackrtlog.FromContext(ctx)
	exit := rlog.Trace("rm.sdkCreate")
	defer func() {
		exit(err)
	}()
	input, err := rm.newCreateRequestPayload(ctx, desired)
	if err != nil {
		return nil, err
	}

	var resp *svcsdk.EventSourceMappingConfiguration
	_ = resp
	resp, err = rm.sdkapi.CreateEventSourceMappingWithContext(ctx, input)
	rm.metrics.RecordAPICall("CREATE", "CreateEventSourceMapping", err)
	if err != nil {
		return nil, err
	}
	// Merge in the information we read from the API call above to the copy of
	// the original Kubernetes object we passed to the function
	ko := desired.ko.DeepCopy()

	if resp.BatchSize != nil {
		ko.Spec.BatchSize = resp.BatchSize
	} else {
		ko.Spec.BatchSize = nil
	}
	if resp.BisectBatchOnFunctionError != nil {
		ko.Spec.BisectBatchOnFunctionError = resp.BisectBatchOnFunctionError
	} else {
		ko.Spec.BisectBatchOnFunctionError = nil
	}
	if resp.DestinationConfig != nil {
		f2 := &svcapitypes.DestinationConfig{}
		if resp.DestinationConfig.OnFailure != nil {
			f2f0 := &svcapitypes.OnFailure{}
			if resp.DestinationConfig.OnFailure.Destination != nil {
				f2f0.Destination = resp.DestinationConfig.OnFailure.Destination
			}
			f2.OnFailure = f2f0
		}
		if resp.DestinationConfig.OnSuccess != nil {
			f2f1 := &svcapitypes.OnSuccess{}
			if resp.DestinationConfig.OnSuccess.Destination != nil {
				f2f1.Destination = resp.DestinationConfig.OnSuccess.Destination
			}
			f2.OnSuccess = f2f1
		}
		ko.Spec.DestinationConfig = f2
	} else {
		ko.Spec.DestinationConfig = nil
	}
	if resp.EventSourceArn != nil {
		ko.Spec.EventSourceARN = resp.EventSourceArn
	} else {
		ko.Spec.EventSourceARN = nil
	}
	if resp.FunctionArn != nil {
		ko.Status.FunctionARN = resp.FunctionArn
	} else {
		ko.Status.FunctionARN = nil
	}
	if resp.FunctionResponseTypes != nil {
		f5 := []*string{}
		for _, f5iter := range resp.FunctionResponseTypes {
			var f5elem string
			f5elem = *f5iter
			f5 = append(f5, &f5elem)
		}
		ko.Spec.FunctionResponseTypes = f5
	} else {
		ko.Spec.FunctionResponseTypes = nil
	}
	if resp.LastModified != nil {
		ko.Status.LastModified = &metav1.Time{*resp.LastModified}
	} else {
		ko.Status.LastModified = nil
	}
	if resp.LastProcessingResult != nil {
		ko.Status.LastProcessingResult = resp.LastProcessingResult
	} else {
		ko.Status.LastProcessingResult = nil
	}
	if resp.MaximumBatchingWindowInSeconds != nil {
		ko.Spec.MaximumBatchingWindowInSeconds = resp.MaximumBatchingWindowInSeconds
	} else {
		ko.Spec.MaximumBatchingWindowInSeconds = nil
	}
	if resp.MaximumRecordAgeInSeconds != nil {
		ko.Spec.MaximumRecordAgeInSeconds = resp.MaximumRecordAgeInSeconds
	} else {
		ko.Spec.MaximumRecordAgeInSeconds = nil
	}
	if resp.MaximumRetryAttempts != nil {
		ko.Spec.MaximumRetryAttempts = resp.MaximumRetryAttempts
	} else {
		ko.Spec.MaximumRetryAttempts = nil
	}
	if resp.ParallelizationFactor != nil {
		ko.Spec.ParallelizationFactor = resp.ParallelizationFactor
	} else {
		ko.Spec.ParallelizationFactor = nil
	}
	if resp.Queues != nil {
		f12 := []*string{}
		for _, f12iter := range resp.Queues {
			var f12elem string
			f12elem = *f12iter
			f12 = append(f12, &f12elem)
		}
		ko.Spec.Queues = f12
	} else {
		ko.Spec.Queues = nil
	}
	if resp.SelfManagedEventSource != nil {
		f13 := &svcapitypes.SelfManagedEventSource{}
		if resp.SelfManagedEventSource.Endpoints != nil {
			f13f0 := map[string][]*string{}
			for f13f0key, f13f0valiter := range resp.SelfManagedEventSource.Endpoints {
				f13f0val := []*string{}
				for _, f13f0valiter := range f13f0valiter {
					var f13f0valelem string
					f13f0valelem = *f13f0valiter
					f13f0val = append(f13f0val, &f13f0valelem)
				}
				f13f0[f13f0key] = f13f0val
			}
			f13.Endpoints = f13f0
		}
		ko.Spec.SelfManagedEventSource = f13
	} else {
		ko.Spec.SelfManagedEventSource = nil
	}
	if resp.SourceAccessConfigurations != nil {
		f14 := []*svcapitypes.SourceAccessConfiguration{}
		for _, f14iter := range resp.SourceAccessConfigurations {
			f14elem := &svcapitypes.SourceAccessConfiguration{}
			if f14iter.Type != nil {
				f14elem.Type = f14iter.Type
			}
			if f14iter.URI != nil {
				f14elem.URI = f14iter.URI
			}
			f14 = append(f14, f14elem)
		}
		ko.Spec.SourceAccessConfigurations = f14
	} else {
		ko.Spec.SourceAccessConfigurations = nil
	}
	if resp.StartingPosition != nil {
		ko.Spec.StartingPosition = resp.StartingPosition
	} else {
		ko.Spec.StartingPosition = nil
	}
	if resp.StartingPositionTimestamp != nil {
		ko.Spec.StartingPositionTimestamp = &metav1.Time{*resp.StartingPositionTimestamp}
	} else {
		ko.Spec.StartingPositionTimestamp = nil
	}
	if resp.State != nil {
		ko.Status.State = resp.State
	} else {
		ko.Status.State = nil
	}
	if resp.StateTransitionReason != nil {
		ko.Status.StateTransitionReason = resp.StateTransitionReason
	} else {
		ko.Status.StateTransitionReason = nil
	}
	if resp.Topics != nil {
		f19 := []*string{}
		for _, f19iter := range resp.Topics {
			var f19elem string
			f19elem = *f19iter
			f19 = append(f19, &f19elem)
		}
		ko.Spec.Topics = f19
	} else {
		ko.Spec.Topics = nil
	}
	if resp.TumblingWindowInSeconds != nil {
		ko.Spec.TumblingWindowInSeconds = resp.TumblingWindowInSeconds
	} else {
		ko.Spec.TumblingWindowInSeconds = nil
	}
	if resp.UUID != nil {
		ko.Status.UUID = resp.UUID
	} else {
		ko.Status.UUID = nil
	}

	rm.setStatusDefaults(ko)
	return &resource{ko}, nil
}

// newCreateRequestPayload returns an SDK-specific struct for the HTTP request
// payload of the Create API call for the resource
func (rm *resourceManager) newCreateRequestPayload(
	ctx context.Context,
	r *resource,
) (*svcsdk.CreateEventSourceMappingInput, error) {
	res := &svcsdk.CreateEventSourceMappingInput{}

	if r.ko.Spec.BatchSize != nil {
		res.SetBatchSize(*r.ko.Spec.BatchSize)
	}
	if r.ko.Spec.BisectBatchOnFunctionError != nil {
		res.SetBisectBatchOnFunctionError(*r.ko.Spec.BisectBatchOnFunctionError)
	}
	if r.ko.Spec.DestinationConfig != nil {
		f2 := &svcsdk.DestinationConfig{}
		if r.ko.Spec.DestinationConfig.OnFailure != nil {
			f2f0 := &svcsdk.OnFailure{}
			if r.ko.Spec.DestinationConfig.OnFailure.Destination != nil {
				f2f0.SetDestination(*r.ko.Spec.DestinationConfig.OnFailure.Destination)
			}
			f2.SetOnFailure(f2f0)
		}
		if r.ko.Spec.DestinationConfig.OnSuccess != nil {
			f2f1 := &svcsdk.OnSuccess{}
			if r.ko.Spec.DestinationConfig.OnSuccess.Destination != nil {
				f2f1.SetDestination(*r.ko.Spec.DestinationConfig.OnSuccess.Destination)
			}
			f2.SetOnSuccess(f2f1)
		}
		res.SetDestinationConfig(f2)
	}
	if r.ko.Spec.Enabled != nil {
		res.SetEnabled(*r.ko.Spec.Enabled)
	}
	if r.ko.Spec.EventSourceARN != nil {
		res.SetEventSourceArn(*r.ko.Spec.EventSourceARN)
	}
	if r.ko.Spec.FunctionName != nil {
		res.SetFunctionName(*r.ko.Spec.FunctionName)
	}
	if r.ko.Spec.FunctionResponseTypes != nil {
		f6 := []*string{}
		for _, f6iter := range r.ko.Spec.FunctionResponseTypes {
			var f6elem string
			f6elem = *f6iter
			f6 = append(f6, &f6elem)
		}
		res.SetFunctionResponseTypes(f6)
	}
	if r.ko.Spec.MaximumBatchingWindowInSeconds != nil {
		res.SetMaximumBatchingWindowInSeconds(*r.ko.Spec.MaximumBatchingWindowInSeconds)
	}
	if r.ko.Spec.MaximumRecordAgeInSeconds != nil {
		res.SetMaximumRecordAgeInSeconds(*r.ko.Spec.MaximumRecordAgeInSeconds)
	}
	if r.ko.Spec.MaximumRetryAttempts != nil {
		res.SetMaximumRetryAttempts(*r.ko.Spec.MaximumRetryAttempts)
	}
	if r.ko.Spec.ParallelizationFactor != nil {
		res.SetParallelizationFactor(*r.ko.Spec.ParallelizationFactor)
	}
	if r.ko.Spec.Queues != nil {
		f11 := []*string{}
		for _, f11iter := range r.ko.Spec.Queues {
			var f11elem string
			f11elem = *f11iter
			f11 = append(f11, &f11elem)
		}
		res.SetQueues(f11)
	}
	if r.ko.Spec.SelfManagedEventSource != nil {
		f12 := &svcsdk.SelfManagedEventSource{}
		if r.ko.Spec.SelfManagedEventSource.Endpoints != nil {
			f12f0 := map[string][]*string{}
			for f12f0key, f12f0valiter := range r.ko.Spec.SelfManagedEventSource.Endpoints {
				f12f0val := []*string{}
				for _, f12f0valiter := range f12f0valiter {
					var f12f0valelem string
					f12f0valelem = *f12f0valiter
					f12f0val = append(f12f0val, &f12f0valelem)
				}
				f12f0[f12f0key] = f12f0val
			}
			f12.SetEndpoints(f12f0)
		}
		res.SetSelfManagedEventSource(f12)
	}
	if r.ko.Spec.SourceAccessConfigurations != nil {
		f13 := []*svcsdk.SourceAccessConfiguration{}
		for _, f13iter := range r.ko.Spec.SourceAccessConfigurations {
			f13elem := &svcsdk.SourceAccessConfiguration{}
			if f13iter.Type != nil {
				f13elem.SetType(*f13iter.Type)
			}
			if f13iter.URI != nil {
				f13elem.SetURI(*f13iter.URI)
			}
			f13 = append(f13, f13elem)
		}
		res.SetSourceAccessConfigurations(f13)
	}
	if r.ko.Spec.StartingPosition != nil {
		res.SetStartingPosition(*r.ko.Spec.StartingPosition)
	}
	if r.ko.Spec.StartingPositionTimestamp != nil {
		res.SetStartingPositionTimestamp(r.ko.Spec.StartingPositionTimestamp.Time)
	}
	if r.ko.Spec.Topics != nil {
		f16 := []*string{}
		for _, f16iter := range r.ko.Spec.Topics {
			var f16elem string
			f16elem = *f16iter
			f16 = append(f16, &f16elem)
		}
		res.SetTopics(f16)
	}
	if r.ko.Spec.TumblingWindowInSeconds != nil {
		res.SetTumblingWindowInSeconds(*r.ko.Spec.TumblingWindowInSeconds)
	}

	return res, nil
}

// sdkUpdate patches the supplied resource in the backend AWS service API and
// returns a new resource with updated fields.
func (rm *resourceManager) sdkUpdate(
	ctx context.Context,
	desired *resource,
	latest *resource,
	delta *ackcompare.Delta,
) (updated *resource, err error) {
	rlog := ackrtlog.FromContext(ctx)
	exit := rlog.Trace("rm.sdkUpdate")
	defer func() {
		exit(err)
	}()
	input, err := rm.newUpdateRequestPayload(ctx, desired)
	if err != nil {
		return nil, err
	}

	var resp *svcsdk.EventSourceMappingConfiguration
	_ = resp
	resp, err = rm.sdkapi.UpdateEventSourceMappingWithContext(ctx, input)
	rm.metrics.RecordAPICall("UPDATE", "UpdateEventSourceMapping", err)
	if err != nil {
		return nil, err
	}
	// Merge in the information we read from the API call above to the copy of
	// the original Kubernetes object we passed to the function
	ko := desired.ko.DeepCopy()

	if resp.BatchSize != nil {
		ko.Spec.BatchSize = resp.BatchSize
	} else {
		ko.Spec.BatchSize = nil
	}
	if resp.BisectBatchOnFunctionError != nil {
		ko.Spec.BisectBatchOnFunctionError = resp.BisectBatchOnFunctionError
	} else {
		ko.Spec.BisectBatchOnFunctionError = nil
	}
	if resp.DestinationConfig != nil {
		f2 := &svcapitypes.DestinationConfig{}
		if resp.DestinationConfig.OnFailure != nil {
			f2f0 := &svcapitypes.OnFailure{}
			if resp.DestinationConfig.OnFailure.Destination != nil {
				f2f0.Destination = resp.DestinationConfig.OnFailure.Destination
			}
			f2.OnFailure = f2f0
		}
		if resp.DestinationConfig.OnSuccess != nil {
			f2f1 := &svcapitypes.OnSuccess{}
			if resp.DestinationConfig.OnSuccess.Destination != nil {
				f2f1.Destination = resp.DestinationConfig.OnSuccess.Destination
			}
			f2.OnSuccess = f2f1
		}
		ko.Spec.DestinationConfig = f2
	} else {
		ko.Spec.DestinationConfig = nil
	}
	if resp.EventSourceArn != nil {
		ko.Spec.EventSourceARN = resp.EventSourceArn
	} else {
		ko.Spec.EventSourceARN = nil
	}
	if resp.FunctionArn != nil {
		ko.Status.FunctionARN = resp.FunctionArn
	} else {
		ko.Status.FunctionARN = nil
	}
	if resp.FunctionResponseTypes != nil {
		f5 := []*string{}
		for _, f5iter := range resp.FunctionResponseTypes {
			var f5elem string
			f5elem = *f5iter
			f5 = append(f5, &f5elem)
		}
		ko.Spec.FunctionResponseTypes = f5
	} else {
		ko.Spec.FunctionResponseTypes = nil
	}
	if resp.LastModified != nil {
		ko.Status.LastModified = &metav1.Time{*resp.LastModified}
	} else {
		ko.Status.LastModified = nil
	}
	if resp.LastProcessingResult != nil {
		ko.Status.LastProcessingResult = resp.LastProcessingResult
	} else {
		ko.Status.LastProcessingResult = nil
	}
	if resp.MaximumBatchingWindowInSeconds != nil {
		ko.Spec.MaximumBatchingWindowInSeconds = resp.MaximumBatchingWindowInSeconds
	} else {
		ko.Spec.MaximumBatchingWindowInSeconds = nil
	}
	if resp.MaximumRecordAgeInSeconds != nil {
		ko.Spec.MaximumRecordAgeInSeconds = resp.MaximumRecordAgeInSeconds
	} else {
		ko.Spec.MaximumRecordAgeInSeconds = nil
	}
	if resp.MaximumRetryAttempts != nil {
		ko.Spec.MaximumRetryAttempts = resp.MaximumRetryAttempts
	} else {
		ko.Spec.MaximumRetryAttempts = nil
	}
	if resp.ParallelizationFactor != nil {
		ko.Spec.ParallelizationFactor = resp.ParallelizationFactor
	} else {
		ko.Spec.ParallelizationFactor = nil
	}
	if resp.Queues != nil {
		f12 := []*string{}
		for _, f12iter := range resp.Queues {
			var f12elem string
			f12elem = *f12iter
			f12 = append(f12, &f12elem)
		}
		ko.Spec.Queues = f12
	} else {
		ko.Spec.Queues = nil
	}
	if resp.SelfManagedEventSource != nil {
		f13 := &svcapitypes.SelfManagedEventSource{}
		if resp.SelfManagedEventSource.Endpoints != nil {
			f13f0 := map[string][]*string{}
			for f13f0key, f13f0valiter := range resp.SelfManagedEventSource.Endpoints {
				f13f0val := []*string{}
				for _, f13f0valiter := range f13f0valiter {
					var f13f0valelem string
					f13f0valelem = *f13f0valiter
					f13f0val = append(f13f0val, &f13f0valelem)
				}
				f13f0[f13f0key] = f13f0val
			}
			f13.Endpoints = f13f0
		}
		ko.Spec.SelfManagedEventSource = f13
	} else {
		ko.Spec.SelfManagedEventSource = nil
	}
	if resp.SourceAccessConfigurations != nil {
		f14 := []*svcapitypes.SourceAccessConfiguration{}
		for _, f14iter := range resp.SourceAccessConfigurations {
			f14elem := &svcapitypes.SourceAccessConfiguration{}
			if f14iter.Type != nil {
				f14elem.Type = f14iter.Type
			}
			if f14iter.URI != nil {
				f14elem.URI = f14iter.URI
			}
			f14 = append(f14, f14elem)
		}
		ko.Spec.SourceAccessConfigurations = f14
	} else {
		ko.Spec.SourceAccessConfigurations = nil
	}
	if resp.StartingPosition != nil {
		ko.Spec.StartingPosition = resp.StartingPosition
	} else {
		ko.Spec.StartingPosition = nil
	}
	if resp.StartingPositionTimestamp != nil {
		ko.Spec.StartingPositionTimestamp = &metav1.Time{*resp.StartingPositionTimestamp}
	} else {
		ko.Spec.StartingPositionTimestamp = nil
	}
	if resp.State != nil {
		ko.Status.State = resp.State
	} else {
		ko.Status.State = nil
	}
	if resp.StateTransitionReason != nil {
		ko.Status.StateTransitionReason = resp.StateTransitionReason
	} else {
		ko.Status.StateTransitionReason = nil
	}
	if resp.Topics != nil {
		f19 := []*string{}
		for _, f19iter := range resp.Topics {
			var f19elem string
			f19elem = *f19iter
			f19 = append(f19, &f19elem)
		}
		ko.Spec.Topics = f19
	} else {
		ko.Spec.Topics = nil
	}
	if resp.TumblingWindowInSeconds != nil {
		ko.Spec.TumblingWindowInSeconds = resp.TumblingWindowInSeconds
	} else {
		ko.Spec.TumblingWindowInSeconds = nil
	}
	if resp.UUID != nil {
		ko.Status.UUID = resp.UUID
	} else {
		ko.Status.UUID = nil
	}

	rm.setStatusDefaults(ko)
	return &resource{ko}, nil
}

// newUpdateRequestPayload returns an SDK-specific struct for the HTTP request
// payload of the Update API call for the resource
func (rm *resourceManager) newUpdateRequestPayload(
	ctx context.Context,
	r *resource,
) (*svcsdk.UpdateEventSourceMappingInput, error) {
	res := &svcsdk.UpdateEventSourceMappingInput{}

	if r.ko.Spec.BatchSize != nil {
		res.SetBatchSize(*r.ko.Spec.BatchSize)
	}
	if r.ko.Spec.BisectBatchOnFunctionError != nil {
		res.SetBisectBatchOnFunctionError(*r.ko.Spec.BisectBatchOnFunctionError)
	}
	if r.ko.Spec.DestinationConfig != nil {
		f2 := &svcsdk.DestinationConfig{}
		if r.ko.Spec.DestinationConfig.OnFailure != nil {
			f2f0 := &svcsdk.OnFailure{}
			if r.ko.Spec.DestinationConfig.OnFailure.Destination != nil {
				f2f0.SetDestination(*r.ko.Spec.DestinationConfig.OnFailure.Destination)
			}
			f2.SetOnFailure(f2f0)
		}
		if r.ko.Spec.DestinationConfig.OnSuccess != nil {
			f2f1 := &svcsdk.OnSuccess{}
			if r.ko.Spec.DestinationConfig.OnSuccess.Destination != nil {
				f2f1.SetDestination(*r.ko.Spec.DestinationConfig.OnSuccess.Destination)
			}
			f2.SetOnSuccess(f2f1)
		}
		res.SetDestinationConfig(f2)
	}
	if r.ko.Spec.Enabled != nil {
		res.SetEnabled(*r.ko.Spec.Enabled)
	}
	if r.ko.Spec.FunctionName != nil {
		res.SetFunctionName(*r.ko.Spec.FunctionName)
	}
	if r.ko.Spec.FunctionResponseTypes != nil {
		f5 := []*string{}
		for _, f5iter := range r.ko.Spec.FunctionResponseTypes {
			var f5elem string
			f5elem = *f5iter
			f5 = append(f5, &f5elem)
		}
		res.SetFunctionResponseTypes(f5)
	}
	if r.ko.Spec.MaximumBatchingWindowInSeconds != nil {
		res.SetMaximumBatchingWindowInSeconds(*r.ko.Spec.MaximumBatchingWindowInSeconds)
	}
	if r.ko.Spec.MaximumRecordAgeInSeconds != nil {
		res.SetMaximumRecordAgeInSeconds(*r.ko.Spec.MaximumRecordAgeInSeconds)
	}
	if r.ko.Spec.MaximumRetryAttempts != nil {
		res.SetMaximumRetryAttempts(*r.ko.Spec.MaximumRetryAttempts)
	}
	if r.ko.Spec.ParallelizationFactor != nil {
		res.SetParallelizationFactor(*r.ko.Spec.ParallelizationFactor)
	}
	if r.ko.Spec.SourceAccessConfigurations != nil {
		f10 := []*svcsdk.SourceAccessConfiguration{}
		for _, f10iter := range r.ko.Spec.SourceAccessConfigurations {
			f10elem := &svcsdk.SourceAccessConfiguration{}
			if f10iter.Type != nil {
				f10elem.SetType(*f10iter.Type)
			}
			if f10iter.URI != nil {
				f10elem.SetURI(*f10iter.URI)
			}
			f10 = append(f10, f10elem)
		}
		res.SetSourceAccessConfigurations(f10)
	}
	if r.ko.Spec.TumblingWindowInSeconds != nil {
		res.SetTumblingWindowInSeconds(*r.ko.Spec.TumblingWindowInSeconds)
	}
	if r.ko.Status.UUID != nil {
		res.SetUUID(*r.ko.Status.UUID)
	}

	return res, nil
}

// sdkDelete deletes the supplied resource in the backend AWS service API
func (rm *resourceManager) sdkDelete(
	ctx context.Context,
	r *resource,
) (latest *resource, err error) {
	rlog := ackrtlog.FromContext(ctx)
	exit := rlog.Trace("rm.sdkDelete")
	defer func() {
		exit(err)
	}()
	input, err := rm.newDeleteRequestPayload(r)
	if err != nil {
		return nil, err
	}
	var resp *svcsdk.EventSourceMappingConfiguration
	_ = resp
	resp, err = rm.sdkapi.DeleteEventSourceMappingWithContext(ctx, input)
	rm.metrics.RecordAPICall("DELETE", "DeleteEventSourceMapping", err)
	return nil, err
}

// newDeleteRequestPayload returns an SDK-specific struct for the HTTP request
// payload of the Delete API call for the resource
func (rm *resourceManager) newDeleteRequestPayload(
	r *resource,
) (*svcsdk.DeleteEventSourceMappingInput, error) {
	res := &svcsdk.DeleteEventSourceMappingInput{}

	if r.ko.Status.UUID != nil {
		res.SetUUID(*r.ko.Status.UUID)
	}

	return res, nil
}

// setStatusDefaults sets default properties into supplied custom resource
func (rm *resourceManager) setStatusDefaults(
	ko *svcapitypes.EventSourceMapping,
) {
	if ko.Status.ACKResourceMetadata == nil {
		ko.Status.ACKResourceMetadata = &ackv1alpha1.ResourceMetadata{}
	}
	if ko.Status.ACKResourceMetadata.Region == nil {
		ko.Status.ACKResourceMetadata.Region = &rm.awsRegion
	}
	if ko.Status.ACKResourceMetadata.OwnerAccountID == nil {
		ko.Status.ACKResourceMetadata.OwnerAccountID = &rm.awsAccountID
	}
	if ko.Status.Conditions == nil {
		ko.Status.Conditions = []*ackv1alpha1.Condition{}
	}
}

// updateConditions returns updated resource, true; if conditions were updated
// else it returns nil, false
func (rm *resourceManager) updateConditions(
	r *resource,
	onSuccess bool,
	err error,
) (*resource, bool) {
	ko := r.ko.DeepCopy()
	rm.setStatusDefaults(ko)

	// Terminal condition
	var terminalCondition *ackv1alpha1.Condition = nil
	var recoverableCondition *ackv1alpha1.Condition = nil
	var syncCondition *ackv1alpha1.Condition = nil
	for _, condition := range ko.Status.Conditions {
		if condition.Type == ackv1alpha1.ConditionTypeTerminal {
			terminalCondition = condition
		}
		if condition.Type == ackv1alpha1.ConditionTypeRecoverable {
			recoverableCondition = condition
		}
		if condition.Type == ackv1alpha1.ConditionTypeResourceSynced {
			syncCondition = condition
		}
	}
	var termError *ackerr.TerminalError
	if rm.terminalAWSError(err) || err == ackerr.SecretTypeNotSupported || err == ackerr.SecretNotFound || errors.As(err, &termError) {
		if terminalCondition == nil {
			terminalCondition = &ackv1alpha1.Condition{
				Type: ackv1alpha1.ConditionTypeTerminal,
			}
			ko.Status.Conditions = append(ko.Status.Conditions, terminalCondition)
		}
		var errorMessage = ""
		if err == ackerr.SecretTypeNotSupported || err == ackerr.SecretNotFound || errors.As(err, &termError) {
			errorMessage = err.Error()
		} else {
			awsErr, _ := ackerr.AWSError(err)
			errorMessage = awsErr.Error()
		}
		terminalCondition.Status = corev1.ConditionTrue
		terminalCondition.Message = &errorMessage
	} else {
		// Clear the terminal condition if no longer present
		if terminalCondition != nil {
			terminalCondition.Status = corev1.ConditionFalse
			terminalCondition.Message = nil
		}
		// Handling Recoverable Conditions
		if err != nil {
			if recoverableCondition == nil {
				// Add a new Condition containing a non-terminal error
				recoverableCondition = &ackv1alpha1.Condition{
					Type: ackv1alpha1.ConditionTypeRecoverable,
				}
				ko.Status.Conditions = append(ko.Status.Conditions, recoverableCondition)
			}
			recoverableCondition.Status = corev1.ConditionTrue
			awsErr, _ := ackerr.AWSError(err)
			errorMessage := err.Error()
			if awsErr != nil {
				errorMessage = awsErr.Error()
			}
			recoverableCondition.Message = &errorMessage
		} else if recoverableCondition != nil {
			recoverableCondition.Status = corev1.ConditionFalse
			recoverableCondition.Message = nil
		}
	}
	// Required to avoid the "declared but not used" error in the default case
	_ = syncCondition
	if terminalCondition != nil || recoverableCondition != nil || syncCondition != nil {
		return &resource{ko}, true // updated
	}
	return nil, false // not updated
}

// terminalAWSError returns awserr, true; if the supplied error is an aws Error type
// and if the exception indicates that it is a Terminal exception
// 'Terminal' exception are specified in generator configuration
func (rm *resourceManager) terminalAWSError(err error) bool {
	// No terminal_errors specified for this resource in generator config
	return false
}
