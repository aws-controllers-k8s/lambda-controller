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

package alias

import (
	"context"

	svcapitypes "github.com/aws-controllers-k8s/lambda-controller/apis/v1alpha1"
	ackerr "github.com/aws-controllers-k8s/runtime/pkg/errors"
	ackrtlog "github.com/aws-controllers-k8s/runtime/pkg/runtime/log"
	"github.com/aws/aws-sdk-go-v2/aws"
	svcsdk "github.com/aws/aws-sdk-go-v2/service/lambda"
	svcsdktypes "github.com/aws/aws-sdk-go-v2/service/lambda/types"
)

// syncEventInvokeConfig calls `PutFunctionEventInvokeConfig` to update the fields
// or `DeleteFunctionEventInvokeConfig` if users removes the fields
func (rm *resourceManager) syncEventInvokeConfig(
	ctx context.Context,
	r *resource,
) (created *resource, err error) {
	rlog := ackrtlog.FromContext(ctx)
	exit := rlog.Trace("rm.syncEventInvokeConfig")
	defer exit(err)

	// Check if the user deleted the 'FunctionEventInvokeConfig' configuration
	// If yes, delete FunctionEventInvokeConfig
	if r.ko.Spec.FunctionEventInvokeConfig == nil {
		input_delete := &svcsdk.DeleteFunctionEventInvokeConfigInput{
			FunctionName: aws.String(*r.ko.Spec.FunctionName),
			Qualifier:    aws.String(*r.ko.Spec.Name),
		}
		_, err = rm.sdkapi.DeleteFunctionEventInvokeConfig(ctx, input_delete)
		rm.metrics.RecordAPICall("DELETE", "DeleteFunctionEventInvokeConfig", err)
		if err != nil {
			return nil, err
		}
		return r, nil
	}

	dspec := r.ko.Spec
	input := &svcsdk.PutFunctionEventInvokeConfigInput{
		FunctionName: aws.String(*dspec.FunctionName),
		Qualifier:    aws.String(*dspec.Name),
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
		input.MaximumEventAgeInSeconds = int32OrNil(dspec.FunctionEventInvokeConfig.MaximumEventAgeInSeconds)
	}
	if dspec.FunctionEventInvokeConfig.MaximumRetryAttempts != nil {
		input.MaximumRetryAttempts = int32OrNil(dspec.FunctionEventInvokeConfig.MaximumRetryAttempts)
	}

	_, err = rm.sdkapi.PutFunctionEventInvokeConfig(ctx, input)
	rm.metrics.RecordAPICall("UPDATE", "SyncEventInvokeConfig", err)
	if err != nil {
		return nil, err
	}
	return r, nil
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

	// Check if the user deleted the 'ProvisionedConcurrency' configuration
	// If yes, delete ProvisionedConcurrencyConfig
	if desired.ko.Spec.ProvisionedConcurrencyConfig == nil || desired.ko.Spec.ProvisionedConcurrencyConfig.ProvisionedConcurrentExecutions == nil {
		input_delete := &svcsdk.DeleteProvisionedConcurrencyConfigInput{
			FunctionName: aws.String(*desired.ko.Spec.FunctionName),
			Qualifier:    aws.String(*desired.ko.Spec.Name),
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
		FunctionName:                    aws.String(*dspec.FunctionName),
		Qualifier:                       aws.String(*dspec.Name),
		ProvisionedConcurrentExecutions: int32OrNil(dspec.ProvisionedConcurrencyConfig.ProvisionedConcurrentExecutions),
	}

	_, err = rm.sdkapi.PutProvisionedConcurrencyConfig(ctx, input)
	rm.metrics.RecordAPICall("UPDATE", "UpdateProvisionedConcurrency", err)
	if err != nil {
		return err
	}
	return nil
}

// setProvisionedConcurrencyConfig sets the Provisioned Concurrency
// for the Function's Alias
func (rm *resourceManager) setProvisionedConcurrencyConfig(
	ctx context.Context,
	ko *svcapitypes.Alias,
) (err error) {
	rlog := ackrtlog.FromContext(ctx)
	exit := rlog.Trace("rm.setProvisionedConcurrencyConfig")
	defer exit(err)

	var getProvisionedConcurrencyConfigOutput *svcsdk.GetProvisionedConcurrencyConfigOutput
	getProvisionedConcurrencyConfigOutput, err = rm.sdkapi.GetProvisionedConcurrencyConfig(
		ctx,
		&svcsdk.GetProvisionedConcurrencyConfigInput{
			FunctionName: ko.Spec.FunctionName,
			Qualifier:    ko.Spec.Name,
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
		cloudProvisionedConcurrency.ProvisionedConcurrentExecutions = int64OrNil(getProvisionedConcurrencyConfigOutput.RequestedProvisionedConcurrentExecutions)
		ko.Spec.ProvisionedConcurrencyConfig = cloudProvisionedConcurrency
	}

	return nil
}

func (rm *resourceManager) setFunctionEventInvokeConfigFromResponse(
	ko *svcapitypes.Alias,
	getFunctionEventInvokeConfigOutput *svcsdk.GetFunctionEventInvokeConfigOutput,
) {
	// creating FunctionEventInvokeConfig object to store the values returned from `Get` call
	cloudFunctionEventInvokeConfig := &svcapitypes.PutFunctionEventInvokeConfigInput{}
	cloudFunctionEventInvokeConfig.DestinationConfig = &svcapitypes.DestinationConfig{}
	cloudFunctionEventInvokeConfig.DestinationConfig.OnFailure = &svcapitypes.OnFailure{}
	cloudFunctionEventInvokeConfig.DestinationConfig.OnSuccess = &svcapitypes.OnSuccess{}
	cloudFunctionEventInvokeConfig.DestinationConfig.OnFailure.Destination = getFunctionEventInvokeConfigOutput.DestinationConfig.OnFailure.Destination
	cloudFunctionEventInvokeConfig.DestinationConfig.OnSuccess.Destination = getFunctionEventInvokeConfigOutput.DestinationConfig.OnSuccess.Destination
	cloudFunctionEventInvokeConfig.MaximumEventAgeInSeconds = int64OrNil(getFunctionEventInvokeConfigOutput.MaximumEventAgeInSeconds)
	cloudFunctionEventInvokeConfig.MaximumRetryAttempts = int64OrNil(getFunctionEventInvokeConfigOutput.MaximumRetryAttempts)
	ko.Spec.FunctionEventInvokeConfig = cloudFunctionEventInvokeConfig

}

// setFunctionEventInvokeConfig sets the fields to set asynchronous invocation
// for Function's Alias
func (rm *resourceManager) setFunctionEventInvokeConfig(
	ctx context.Context,
	ko *svcapitypes.Alias,
) (err error) {
	rlog := ackrtlog.FromContext(ctx)
	exit := rlog.Trace("rm.setFunctionEventInvokeConfig")
	defer exit(err)

	var getFunctionEventInvokeConfigOutput *svcsdk.GetFunctionEventInvokeConfigOutput
	getFunctionEventInvokeConfigOutput, err = rm.sdkapi.GetFunctionEventInvokeConfig(
		ctx,
		&svcsdk.GetFunctionEventInvokeConfigInput{
			FunctionName: ko.Spec.FunctionName,
			Qualifier:    ko.Spec.Name,
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

// setResourceAdditionalFields will describe the fields that are not return by the
// getFunctionConfiguration API call
func (rm *resourceManager) setResourceAdditionalFields(
	ctx context.Context,
	ko *svcapitypes.Alias,
) (err error) {
	rlog := ackrtlog.FromContext(ctx)
	exit := rlog.Trace("rm.setResourceAdditionalFields")
	defer exit(err)

	// To set Asynchronous Invocations for the function's alias
	err = rm.setFunctionEventInvokeConfig(ctx, ko)
	if err != nil {
		return err
	}

	// To set Provisioned Concurrency for the function's alias
	err = rm.setProvisionedConcurrencyConfig(ctx, ko)
	if err != nil {
		return err
	}

	return nil
}


func int32OrNil(val *int64) *int32 {
	if val != nil {
		return aws.Int32(int32(*val))
	}
	return nil
}

func int64OrNil(val *int32) *int64 {
	if val != nil {
		return aws.Int64(int64(*val))
	}
	return nil
}