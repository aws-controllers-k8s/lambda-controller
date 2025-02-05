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

package version

import (
	"context"
	"errors"
	"time"

	svcapitypes "github.com/aws-controllers-k8s/lambda-controller/apis/v1alpha1"
	ackcompare "github.com/aws-controllers-k8s/runtime/pkg/compare"
	ackerr "github.com/aws-controllers-k8s/runtime/pkg/errors"
	ackrequeue "github.com/aws-controllers-k8s/runtime/pkg/requeue"
	ackrtlog "github.com/aws-controllers-k8s/runtime/pkg/runtime/log"
	svcsdk "github.com/aws/aws-sdk-go-v2/service/lambda"
	svcsdktypes "github.com/aws/aws-sdk-go-v2/service/lambda/types"
	"github.com/aws/aws-sdk-go/aws"
)

var (
	ErrFunctionPending = errors.New("function in 'Pending' state, cannot be modified or deleted")
)

var (
	requeueWaitWhilePending = ackrequeue.NeededAfter(
		ErrFunctionPending,
		5*time.Second,
	)
)

// isVersionPending returns true if the supplied Function Version is in a pending
// state
func isVersionPending(r *resource) bool {
	if r.ko.Status.State == nil {
		return false
	}
	state := *r.ko.Status.State
	return state == string(svcapitypes.State_Pending)
}

func (rm *resourceManager) customUpdateVersion(
	ctx context.Context,
	desired *resource,
	latest *resource,
	delta *ackcompare.Delta,
) (*resource, error) {
	var err error
	rlog := ackrtlog.FromContext(ctx)
	exit := rlog.Trace("rm.customUpdateFunction")
	defer exit(err)

	if isVersionPending(desired) {
		return nil, requeueWaitWhilePending
	}

	if delta.DifferentAt("Spec.FunctionEventInvokeConfig") {
		err = rm.syncEventInvokeConfig(ctx, desired)
		if err != nil {
			return nil, err
		}
	}

	if delta.DifferentAt("Spec.ProvisionedConcurrencyConfig") {
		err = rm.updateProvisionedConcurrency(ctx, desired)
		if err != nil {
			return nil, err
		}
	}

	readOneLatest, err := rm.ReadOne(ctx, desired)
	if err != nil {
		return nil, err
	}
	return rm.concreteResource(readOneLatest), nil
}

// syncEventInvokeConfig calls `PutFunctionEventInvokeConfig` to update the fields
// or `DeleteFunctionEventInvokeConfig` if users removes the fields
func (rm *resourceManager) syncEventInvokeConfig(
	ctx context.Context,
	r *resource,
) (err error) {
	rlog := ackrtlog.FromContext(ctx)
	exit := rlog.Trace("rm.syncEventInvokeConfig")
	defer exit(err)

	if r.ko.Status.Version == nil {
		return nil
	}
	// Check if the user deleted the 'FunctionEventInvokeConfig' configuration
	// If yes, delete FunctionEventInvokeConfig
	if r.ko.Spec.FunctionEventInvokeConfig == nil {
		input_delete := &svcsdk.DeleteFunctionEventInvokeConfigInput{
			FunctionName: aws.String(*r.ko.Spec.FunctionName),
			Qualifier:    aws.String(*r.ko.Status.Version),
		}
		_, err = rm.sdkapi.DeleteFunctionEventInvokeConfig(ctx, input_delete)
		rm.metrics.RecordAPICall("DELETE", "DeleteFunctionEventInvokeConfig", err)
		if err != nil {
			return nil
		}
		return nil
	}

	dspec := r.ko.Spec
	input := &svcsdk.PutFunctionEventInvokeConfigInput{
		FunctionName: aws.String(*r.ko.Spec.FunctionName),
		Qualifier:    aws.String(*r.ko.Status.Version),
	}

	if dspec.FunctionEventInvokeConfig.DestinationConfig != nil {
		destinations := &svcsdktypes.DestinationConfig{}
		if dspec.FunctionEventInvokeConfig.DestinationConfig.OnFailure != nil {
			destinations.OnFailure = &svcsdktypes.OnFailure{}
			if dspec.FunctionEventInvokeConfig.DestinationConfig.OnFailure.Destination != nil {
				destinations.OnFailure.Destination = aws.String(*dspec.FunctionEventInvokeConfig.DestinationConfig.OnFailure.Destination)
			}
		}
		if dspec.FunctionEventInvokeConfig.DestinationConfig.OnSuccess != nil {
			destinations.OnSuccess = &svcsdktypes.OnSuccess{}
			if dspec.FunctionEventInvokeConfig.DestinationConfig.OnSuccess.Destination != nil {
				destinations.OnSuccess.Destination = aws.String(*dspec.FunctionEventInvokeConfig.DestinationConfig.OnSuccess.Destination)
			}
		}
		input.DestinationConfig = destinations
	}
	if dspec.FunctionEventInvokeConfig.MaximumEventAgeInSeconds != nil {
		input.MaximumEventAgeInSeconds = aws.Int32(int32(*dspec.FunctionEventInvokeConfig.MaximumEventAgeInSeconds))
	}
	if dspec.FunctionEventInvokeConfig.MaximumRetryAttempts != nil {
		input.MaximumRetryAttempts = aws.Int32(int32(*dspec.FunctionEventInvokeConfig.MaximumRetryAttempts))
	}

	_, err = rm.sdkapi.PutFunctionEventInvokeConfig(ctx, input)
	rm.metrics.RecordAPICall("UPDATE", "SyncEventInvokeConfig", err)
	if err != nil {
		return err
	}
	return nil
}

func (rm *resourceManager) setFunctionEventInvokeConfigFromResponse(
	ko *svcapitypes.Version,
	getFunctionEventInvokeConfigOutput *svcsdk.GetFunctionEventInvokeConfigOutput,
) {
	// creating FunctionEventInvokeConfig object to store the values returned from `Get` call
	cloudFunctionEventInvokeConfig := &svcapitypes.PutFunctionEventInvokeConfigInput{}
	cloudFunctionEventInvokeConfig.DestinationConfig = &svcapitypes.DestinationConfig{}
	cloudFunctionEventInvokeConfig.DestinationConfig.OnFailure = &svcapitypes.OnFailure{}
	cloudFunctionEventInvokeConfig.DestinationConfig.OnSuccess = &svcapitypes.OnSuccess{}
	cloudFunctionEventInvokeConfig.DestinationConfig.OnFailure.Destination = getFunctionEventInvokeConfigOutput.DestinationConfig.OnFailure.Destination
	cloudFunctionEventInvokeConfig.DestinationConfig.OnSuccess.Destination = getFunctionEventInvokeConfigOutput.DestinationConfig.OnSuccess.Destination
	cloudFunctionEventInvokeConfig.MaximumEventAgeInSeconds = aws.Int64(int64(*getFunctionEventInvokeConfigOutput.MaximumEventAgeInSeconds))
	cloudFunctionEventInvokeConfig.MaximumRetryAttempts = aws.Int64(int64(*getFunctionEventInvokeConfigOutput.MaximumRetryAttempts))
	ko.Spec.FunctionEventInvokeConfig = cloudFunctionEventInvokeConfig

}

// setFunctionEventInvokeConfig sets the fields to set asynchronous invocation
// for Function's Version
func (rm *resourceManager) setFunctionEventInvokeConfig(
	ctx context.Context,
	ko *svcapitypes.Version,
) (err error) {
	rlog := ackrtlog.FromContext(ctx)
	exit := rlog.Trace("rm.setFunctionEventInvokeConfig")
	defer exit(err)

	var getFunctionEventInvokeConfigOutput *svcsdk.GetFunctionEventInvokeConfigOutput
	getFunctionEventInvokeConfigOutput, err = rm.sdkapi.GetFunctionEventInvokeConfig(
		ctx,
		&svcsdk.GetFunctionEventInvokeConfigInput{
			FunctionName: ko.Spec.FunctionName,
			Qualifier:    ko.Status.Version,
		},
	)
	rm.metrics.RecordAPICall("GET", "GetFunctionEventInvokeConfig", err)

	if err != nil {
		if awserr, ok := ackerr.AWSError(err); ok && (awserr.ErrorCode() == "EventInvokeConfigNotFoundException" || awserr.ErrorCode() == "ResourceNotFoundException") {
			ko.Spec.FunctionEventInvokeConfig = nil
		} else {
			return err
		}
	} else {
		rm.setFunctionEventInvokeConfigFromResponse(ko, getFunctionEventInvokeConfigOutput)
	}

	return nil
}

// updateProvisionedConcurrency calls `PutProvisionedConcurrencyConfig` to update the fields
// or `DeleteProvisionedConcurrencyConfig` if users removes the fields
func (rm *resourceManager) updateProvisionedConcurrency(
	ctx context.Context,
	desired *resource,
) error {
	var err error
	rlog := ackrtlog.FromContext(ctx)
	exit := rlog.Trace("rm.updateProvisionedConcurrency")
	defer exit(err)

	if desired.ko.Status.Version == nil {
		return nil
	}

	// Check if the user deleted the 'ProvisionedConcurrency' configuration
	// If yes, delete ProvisionedConcurrencyConfig
	if desired.ko.Spec.ProvisionedConcurrencyConfig == nil || desired.ko.Spec.ProvisionedConcurrencyConfig.ProvisionedConcurrentExecutions == nil {
		input_delete := &svcsdk.DeleteProvisionedConcurrencyConfigInput{
			FunctionName: aws.String(*desired.ko.Spec.FunctionName),
			Qualifier:    aws.String(*desired.ko.Status.Version),
		}
		_, err = rm.sdkapi.DeleteProvisionedConcurrencyConfig(ctx, input_delete)
		rm.metrics.RecordAPICall("DELETE", "DeleteProvisionedConcurrency", err)
		if err != nil {
			return err
		}
		return nil
	}

	dspec := desired.ko.Spec
	input := &svcsdk.PutProvisionedConcurrencyConfigInput{
		FunctionName:                    aws.String(*desired.ko.Spec.FunctionName),
		Qualifier:                       aws.String(*desired.ko.Status.Version),
		ProvisionedConcurrentExecutions: aws.Int32(int32(*dspec.ProvisionedConcurrencyConfig.ProvisionedConcurrentExecutions)),
	}

	_, err = rm.sdkapi.PutProvisionedConcurrencyConfig(ctx, input)
	rm.metrics.RecordAPICall("UPDATE", "UpdateProvisionedConcurrency", err)
	if err != nil {
		return err
	}
	return nil
}

// setProvisionedConcurrencyConfig sets the Provisioned Concurrency
// for the Function's Version
func (rm *resourceManager) setProvisionedConcurrencyConfig(
	ctx context.Context,
	ko *svcapitypes.Version,
) (err error) {
	rlog := ackrtlog.FromContext(ctx)
	exit := rlog.Trace("rm.setProvisionedConcurrencyConfig")
	defer exit(err)

	var getProvisionedConcurrencyConfigOutput *svcsdk.GetProvisionedConcurrencyConfigOutput
	getProvisionedConcurrencyConfigOutput, err = rm.sdkapi.GetProvisionedConcurrencyConfig(
		ctx,
		&svcsdk.GetProvisionedConcurrencyConfigInput{
			FunctionName: ko.Spec.FunctionName,
			Qualifier:    ko.Status.Version,
		},
	)
	rm.metrics.RecordAPICall("GET", "GetProvisionedConcurrencyConfig", err)

	if err != nil {
		if awserr, ok := ackerr.AWSError(err); ok && (awserr.ErrorCode() == "ProvisionedConcurrencyConfigNotFoundException" || awserr.ErrorCode() == "ResourceNotFoundException") {
			ko.Spec.ProvisionedConcurrencyConfig = nil
		} else {
			return err
		}
	} else {
		// creating ProvisionedConcurrency object to store the values returned from `Get` call
		cloudProvisionedConcurrency := &svcapitypes.PutProvisionedConcurrencyConfigInput{}
		cloudProvisionedConcurrency.ProvisionedConcurrentExecutions = aws.Int64(int64(*getProvisionedConcurrencyConfigOutput.RequestedProvisionedConcurrentExecutions))
		ko.Spec.ProvisionedConcurrencyConfig = cloudProvisionedConcurrency
	}

	return nil
}

// setResourceAdditionalFields will describe the fields that are not return by the
// getFunctionConfiguration API call
func (rm *resourceManager) setResourceAdditionalFields(
	ctx context.Context,
	ko *svcapitypes.Version,
) (err error) {
	rlog := ackrtlog.FromContext(ctx)
	exit := rlog.Trace("rm.setResourceAdditionalFields")
	defer exit(err)

	// To set Asynchronous Invocations for the function's version
	err = rm.setFunctionEventInvokeConfig(ctx, ko)
	if err != nil {
		return err
	}

	// To set Provisioned Concurrency for the function's version
	err = rm.setProvisionedConcurrencyConfig(ctx, ko)
	if err != nil {
		return err
	}

	return nil
}
