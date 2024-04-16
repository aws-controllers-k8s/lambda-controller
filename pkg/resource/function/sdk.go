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

package function

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"

	ackv1alpha1 "github.com/aws-controllers-k8s/runtime/apis/core/v1alpha1"
	ackcompare "github.com/aws-controllers-k8s/runtime/pkg/compare"
	ackcondition "github.com/aws-controllers-k8s/runtime/pkg/condition"
	ackerr "github.com/aws-controllers-k8s/runtime/pkg/errors"
	ackrequeue "github.com/aws-controllers-k8s/runtime/pkg/requeue"
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
	_ = &svcapitypes.Function{}
	_ = ackv1alpha1.AWSAccountID("")
	_ = &ackerr.NotFound
	_ = &ackcondition.NotManagedMessage
	_ = &reflect.Value{}
	_ = fmt.Sprintf("")
	_ = &ackrequeue.NoRequeue{}
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

	var resp *svcsdk.GetFunctionOutput
	resp, err = rm.sdkapi.GetFunctionWithContext(ctx, input)
	rm.metrics.RecordAPICall("READ_ONE", "GetFunction", err)
	if err != nil {
		if reqErr, ok := ackerr.AWSRequestFailure(err); ok && reqErr.StatusCode() == 404 {
			return nil, ackerr.NotFound
		}
		if awsErr, ok := ackerr.AWSError(err); ok && awsErr.Code() == "ResourceNotFoundException" {
			return nil, ackerr.NotFound
		}
		return nil, err
	}

	// Merge in the information we read from the API call above to the copy of
	// the original Kubernetes object we passed to the function
	ko := r.ko.DeepCopy()

	if resp.Configuration.Architectures != nil {
		f0 := []*string{}
		for _, f0iter := range resp.Configuration.Architectures {
			var f0elem string
			f0elem = *f0iter
			f0 = append(f0, &f0elem)
		}
		ko.Spec.Architectures = f0
	} else {
		ko.Spec.Architectures = nil
	}
	if resp.Configuration.CodeSha256 != nil {
		ko.Status.CodeSHA256 = resp.Configuration.CodeSha256
	} else {
		ko.Status.CodeSHA256 = nil
	}
	if resp.Configuration.CodeSize != nil {
		ko.Status.CodeSize = resp.Configuration.CodeSize
	} else {
		ko.Status.CodeSize = nil
	}
	if resp.Configuration.DeadLetterConfig != nil {
		f3 := &svcapitypes.DeadLetterConfig{}
		if resp.Configuration.DeadLetterConfig.TargetArn != nil {
			f3.TargetARN = resp.Configuration.DeadLetterConfig.TargetArn
		}
		ko.Spec.DeadLetterConfig = f3
	} else {
		ko.Spec.DeadLetterConfig = nil
	}
	if resp.Configuration.Description != nil {
		ko.Spec.Description = resp.Configuration.Description
	} else {
		ko.Spec.Description = nil
	}
	if resp.Configuration.Environment != nil {
		f5 := &svcapitypes.Environment{}
		if resp.Configuration.Environment.Variables != nil {
			f5f1 := map[string]*string{}
			for f5f1key, f5f1valiter := range resp.Configuration.Environment.Variables {
				var f5f1val string
				f5f1val = *f5f1valiter
				f5f1[f5f1key] = &f5f1val
			}
			f5.Variables = f5f1
		}
		ko.Spec.Environment = f5
	} else {
		ko.Spec.Environment = nil
	}
	if resp.Configuration.EphemeralStorage != nil {
		f6 := &svcapitypes.EphemeralStorage{}
		if resp.Configuration.EphemeralStorage.Size != nil {
			f6.Size = resp.Configuration.EphemeralStorage.Size
		}
		ko.Spec.EphemeralStorage = f6
	} else {
		ko.Spec.EphemeralStorage = nil
	}
	if resp.Configuration.FileSystemConfigs != nil {
		f7 := []*svcapitypes.FileSystemConfig{}
		for _, f7iter := range resp.Configuration.FileSystemConfigs {
			f7elem := &svcapitypes.FileSystemConfig{}
			if f7iter.Arn != nil {
				f7elem.ARN = f7iter.Arn
			}
			if f7iter.LocalMountPath != nil {
				f7elem.LocalMountPath = f7iter.LocalMountPath
			}
			f7 = append(f7, f7elem)
		}
		ko.Spec.FileSystemConfigs = f7
	} else {
		ko.Spec.FileSystemConfigs = nil
	}
	if ko.Status.ACKResourceMetadata == nil {
		ko.Status.ACKResourceMetadata = &ackv1alpha1.ResourceMetadata{}
	}
	if resp.Configuration.FunctionArn != nil {
		arn := ackv1alpha1.AWSResourceName(*resp.Configuration.FunctionArn)
		ko.Status.ACKResourceMetadata.ARN = &arn
	}
	if resp.Configuration.FunctionName != nil {
		ko.Spec.Name = resp.Configuration.FunctionName
	} else {
		ko.Spec.Name = nil
	}
	if resp.Configuration.Handler != nil {
		ko.Spec.Handler = resp.Configuration.Handler
	} else {
		ko.Spec.Handler = nil
	}
	if resp.Configuration.ImageConfigResponse != nil {
		f11 := &svcapitypes.ImageConfigResponse{}
		if resp.Configuration.ImageConfigResponse.Error != nil {
			f11f0 := &svcapitypes.ImageConfigError{}
			if resp.Configuration.ImageConfigResponse.Error.ErrorCode != nil {
				f11f0.ErrorCode = resp.Configuration.ImageConfigResponse.Error.ErrorCode
			}
			if resp.Configuration.ImageConfigResponse.Error.Message != nil {
				f11f0.Message = resp.Configuration.ImageConfigResponse.Error.Message
			}
			f11.Error = f11f0
		}
		if resp.Configuration.ImageConfigResponse.ImageConfig != nil {
			f11f1 := &svcapitypes.ImageConfig{}
			if resp.Configuration.ImageConfigResponse.ImageConfig.Command != nil {
				f11f1f0 := []*string{}
				for _, f11f1f0iter := range resp.Configuration.ImageConfigResponse.ImageConfig.Command {
					var f11f1f0elem string
					f11f1f0elem = *f11f1f0iter
					f11f1f0 = append(f11f1f0, &f11f1f0elem)
				}
				f11f1.Command = f11f1f0
			}
			if resp.Configuration.ImageConfigResponse.ImageConfig.EntryPoint != nil {
				f11f1f1 := []*string{}
				for _, f11f1f1iter := range resp.Configuration.ImageConfigResponse.ImageConfig.EntryPoint {
					var f11f1f1elem string
					f11f1f1elem = *f11f1f1iter
					f11f1f1 = append(f11f1f1, &f11f1f1elem)
				}
				f11f1.EntryPoint = f11f1f1
			}
			if resp.Configuration.ImageConfigResponse.ImageConfig.WorkingDirectory != nil {
				f11f1.WorkingDirectory = resp.Configuration.ImageConfigResponse.ImageConfig.WorkingDirectory
			}
			f11.ImageConfig = f11f1
		}
		ko.Status.ImageConfigResponse = f11
	} else {
		ko.Status.ImageConfigResponse = nil
	}
	if resp.Configuration.KMSKeyArn != nil {
		ko.Spec.KMSKeyARN = resp.Configuration.KMSKeyArn
	} else {
		ko.Spec.KMSKeyARN = nil
	}
	if resp.Configuration.LastModified != nil {
		ko.Status.LastModified = resp.Configuration.LastModified
	} else {
		ko.Status.LastModified = nil
	}
	if resp.Configuration.LastUpdateStatus != nil {
		ko.Status.LastUpdateStatus = resp.Configuration.LastUpdateStatus
	} else {
		ko.Status.LastUpdateStatus = nil
	}
	if resp.Configuration.LastUpdateStatusReason != nil {
		ko.Status.LastUpdateStatusReason = resp.Configuration.LastUpdateStatusReason
	} else {
		ko.Status.LastUpdateStatusReason = nil
	}
	if resp.Configuration.LastUpdateStatusReasonCode != nil {
		ko.Status.LastUpdateStatusReasonCode = resp.Configuration.LastUpdateStatusReasonCode
	} else {
		ko.Status.LastUpdateStatusReasonCode = nil
	}
	if resp.Configuration.MasterArn != nil {
		ko.Status.MasterARN = resp.Configuration.MasterArn
	} else {
		ko.Status.MasterARN = nil
	}
	if resp.Configuration.MemorySize != nil {
		ko.Spec.MemorySize = resp.Configuration.MemorySize
	} else {
		ko.Spec.MemorySize = nil
	}
	if resp.Configuration.PackageType != nil {
		ko.Spec.PackageType = resp.Configuration.PackageType
	} else {
		ko.Spec.PackageType = nil
	}
	if resp.Configuration.RevisionId != nil {
		ko.Status.RevisionID = resp.Configuration.RevisionId
	} else {
		ko.Status.RevisionID = nil
	}
	if resp.Configuration.Role != nil {
		ko.Spec.Role = resp.Configuration.Role
	} else {
		ko.Spec.Role = nil
	}
	if resp.Configuration.Runtime != nil {
		ko.Spec.Runtime = resp.Configuration.Runtime
	} else {
		ko.Spec.Runtime = nil
	}
	if resp.Configuration.SigningJobArn != nil {
		ko.Status.SigningJobARN = resp.Configuration.SigningJobArn
	} else {
		ko.Status.SigningJobARN = nil
	}
	if resp.Configuration.SigningProfileVersionArn != nil {
		ko.Status.SigningProfileVersionARN = resp.Configuration.SigningProfileVersionArn
	} else {
		ko.Status.SigningProfileVersionARN = nil
	}
	if resp.Configuration.SnapStart != nil {
		f26 := &svcapitypes.SnapStart{}
		if resp.Configuration.SnapStart.ApplyOn != nil {
			f26.ApplyOn = resp.Configuration.SnapStart.ApplyOn
		}
		ko.Spec.SnapStart = f26
	} else {
		ko.Spec.SnapStart = nil
	}
	if resp.Configuration.State != nil {
		ko.Status.State = resp.Configuration.State
	} else {
		ko.Status.State = nil
	}
	if resp.Configuration.StateReason != nil {
		ko.Status.StateReason = resp.Configuration.StateReason
	} else {
		ko.Status.StateReason = nil
	}
	if resp.Configuration.StateReasonCode != nil {
		ko.Status.StateReasonCode = resp.Configuration.StateReasonCode
	} else {
		ko.Status.StateReasonCode = nil
	}
	if resp.Configuration.Timeout != nil {
		ko.Spec.Timeout = resp.Configuration.Timeout
	} else {
		ko.Spec.Timeout = nil
	}
	if resp.Configuration.TracingConfig != nil {
		f31 := &svcapitypes.TracingConfig{}
		if resp.Configuration.TracingConfig.Mode != nil {
			f31.Mode = resp.Configuration.TracingConfig.Mode
		}
		ko.Spec.TracingConfig = f31
	} else {
		ko.Spec.TracingConfig = nil
	}
	if resp.Configuration.Version != nil {
		ko.Status.Version = resp.Configuration.Version
	} else {
		ko.Status.Version = nil
	}
	if resp.Configuration.VpcConfig != nil {
		f33 := &svcapitypes.VPCConfig{}
		if resp.Configuration.VpcConfig.SecurityGroupIds != nil {
			f33f0 := []*string{}
			for _, f33f0iter := range resp.Configuration.VpcConfig.SecurityGroupIds {
				var f33f0elem string
				f33f0elem = *f33f0iter
				f33f0 = append(f33f0, &f33f0elem)
			}
			f33.SecurityGroupIDs = f33f0
		}
		if resp.Configuration.VpcConfig.SubnetIds != nil {
			f33f1 := []*string{}
			for _, f33f1iter := range resp.Configuration.VpcConfig.SubnetIds {
				var f33f1elem string
				f33f1elem = *f33f1iter
				f33f1 = append(f33f1, &f33f1elem)
			}
			f33.SubnetIDs = f33f1
		}
		ko.Spec.VPCConfig = f33
	} else {
		ko.Spec.VPCConfig = nil
	}

	rm.setStatusDefaults(ko)
	if resp.Code != nil {
		// We need to keep the desired .Code s3Bucket s3Key and s3ObjectVersion
		// part of the function's spec. So instead of setting Spec.Code to nil
		// we only set ImageURI
		//
		// When adopting a Function resource, Spec.Code field should be manually
		// initialised before injecting ImageURI.
		if ko.Spec.Code == nil {
			ko.Spec.Code = &svcapitypes.FunctionCode{}
		}
		if resp.Code.ImageUri != nil {
			ko.Spec.Code.ImageURI = resp.Code.ImageUri
		}
	}
	if resp.Configuration.Layers != nil {
		f16 := []*svcapitypes.Layer{}
		for _, f16iter := range resp.Configuration.Layers {
			f16elem := &svcapitypes.Layer{}
			if f16iter.Arn != nil {
				f16elem.ARN = f16iter.Arn
			}
			if f16iter.CodeSize != nil {
				f16elem.CodeSize = f16iter.CodeSize
			}
			if f16iter.SigningJobArn != nil {
				f16elem.SigningJobARN = f16iter.SigningJobArn
			}
			if f16iter.SigningProfileVersionArn != nil {
				f16elem.SigningProfileVersionARN = f16iter.SigningProfileVersionArn
			}
			f16 = append(f16, f16elem)
		}
		ko.Status.LayerStatuses = f16
	} else {
		ko.Status.LayerStatuses = nil
	}
	if resp.Tags != nil {
		expectedOutput := map[string]*string{}
		for expectedOutputKey, expectedOutputIter := range resp.Tags {
			var expectedOutputVal string
			expectedOutputVal = *expectedOutputIter
			expectedOutput[expectedOutputKey] = &expectedOutputVal
		}
		ko.Spec.Tags = expectedOutput
	}
	if err := rm.setResourceAdditionalFields(ctx, ko); err != nil {
		return nil, err
	}
	return &resource{ko}, nil
}

// requiredFieldsMissingFromReadOneInput returns true if there are any fields
// for the ReadOne Input shape that are required but not present in the
// resource's Spec or Status
func (rm *resourceManager) requiredFieldsMissingFromReadOneInput(
	r *resource,
) bool {
	return r.ko.Spec.Name == nil

}

// newDescribeRequestPayload returns SDK-specific struct for the HTTP request
// payload of the Describe API call for the resource
func (rm *resourceManager) newDescribeRequestPayload(
	r *resource,
) (*svcsdk.GetFunctionInput, error) {
	res := &svcsdk.GetFunctionInput{}

	if r.ko.Spec.Name != nil {
		res.SetFunctionName(*r.ko.Spec.Name)
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
	if desired.ko.Spec.CodeSigningConfigARN != nil && *desired.ko.Spec.CodeSigningConfigARN == "" {
		input.CodeSigningConfigArn = nil
	}

	var resp *svcsdk.FunctionConfiguration
	_ = resp
	resp, err = rm.sdkapi.CreateFunctionWithContext(ctx, input)
	rm.metrics.RecordAPICall("CREATE", "CreateFunction", err)
	if err != nil {
		return nil, err
	}
	// Merge in the information we read from the API call above to the copy of
	// the original Kubernetes object we passed to the function
	ko := desired.ko.DeepCopy()

	if resp.Architectures != nil {
		f0 := []*string{}
		for _, f0iter := range resp.Architectures {
			var f0elem string
			f0elem = *f0iter
			f0 = append(f0, &f0elem)
		}
		ko.Spec.Architectures = f0
	} else {
		ko.Spec.Architectures = nil
	}
	if resp.CodeSha256 != nil {
		ko.Status.CodeSHA256 = resp.CodeSha256
	} else {
		ko.Status.CodeSHA256 = nil
	}
	if resp.CodeSize != nil {
		ko.Status.CodeSize = resp.CodeSize
	} else {
		ko.Status.CodeSize = nil
	}
	if resp.DeadLetterConfig != nil {
		f3 := &svcapitypes.DeadLetterConfig{}
		if resp.DeadLetterConfig.TargetArn != nil {
			f3.TargetARN = resp.DeadLetterConfig.TargetArn
		}
		ko.Spec.DeadLetterConfig = f3
	} else {
		ko.Spec.DeadLetterConfig = nil
	}
	if resp.Description != nil {
		ko.Spec.Description = resp.Description
	} else {
		ko.Spec.Description = nil
	}
	if resp.Environment != nil {
		f5 := &svcapitypes.Environment{}
		if resp.Environment.Variables != nil {
			f5f1 := map[string]*string{}
			for f5f1key, f5f1valiter := range resp.Environment.Variables {
				var f5f1val string
				f5f1val = *f5f1valiter
				f5f1[f5f1key] = &f5f1val
			}
			f5.Variables = f5f1
		}
		ko.Spec.Environment = f5
	} else {
		ko.Spec.Environment = nil
	}
	if resp.EphemeralStorage != nil {
		f6 := &svcapitypes.EphemeralStorage{}
		if resp.EphemeralStorage.Size != nil {
			f6.Size = resp.EphemeralStorage.Size
		}
		ko.Spec.EphemeralStorage = f6
	} else {
		ko.Spec.EphemeralStorage = nil
	}
	if resp.FileSystemConfigs != nil {
		f7 := []*svcapitypes.FileSystemConfig{}
		for _, f7iter := range resp.FileSystemConfigs {
			f7elem := &svcapitypes.FileSystemConfig{}
			if f7iter.Arn != nil {
				f7elem.ARN = f7iter.Arn
			}
			if f7iter.LocalMountPath != nil {
				f7elem.LocalMountPath = f7iter.LocalMountPath
			}
			f7 = append(f7, f7elem)
		}
		ko.Spec.FileSystemConfigs = f7
	} else {
		ko.Spec.FileSystemConfigs = nil
	}
	if ko.Status.ACKResourceMetadata == nil {
		ko.Status.ACKResourceMetadata = &ackv1alpha1.ResourceMetadata{}
	}
	if resp.FunctionArn != nil {
		arn := ackv1alpha1.AWSResourceName(*resp.FunctionArn)
		ko.Status.ACKResourceMetadata.ARN = &arn
	}
	if resp.FunctionName != nil {
		ko.Spec.Name = resp.FunctionName
	} else {
		ko.Spec.Name = nil
	}
	if resp.Handler != nil {
		ko.Spec.Handler = resp.Handler
	} else {
		ko.Spec.Handler = nil
	}
	if resp.ImageConfigResponse != nil {
		f11 := &svcapitypes.ImageConfigResponse{}
		if resp.ImageConfigResponse.Error != nil {
			f11f0 := &svcapitypes.ImageConfigError{}
			if resp.ImageConfigResponse.Error.ErrorCode != nil {
				f11f0.ErrorCode = resp.ImageConfigResponse.Error.ErrorCode
			}
			if resp.ImageConfigResponse.Error.Message != nil {
				f11f0.Message = resp.ImageConfigResponse.Error.Message
			}
			f11.Error = f11f0
		}
		if resp.ImageConfigResponse.ImageConfig != nil {
			f11f1 := &svcapitypes.ImageConfig{}
			if resp.ImageConfigResponse.ImageConfig.Command != nil {
				f11f1f0 := []*string{}
				for _, f11f1f0iter := range resp.ImageConfigResponse.ImageConfig.Command {
					var f11f1f0elem string
					f11f1f0elem = *f11f1f0iter
					f11f1f0 = append(f11f1f0, &f11f1f0elem)
				}
				f11f1.Command = f11f1f0
			}
			if resp.ImageConfigResponse.ImageConfig.EntryPoint != nil {
				f11f1f1 := []*string{}
				for _, f11f1f1iter := range resp.ImageConfigResponse.ImageConfig.EntryPoint {
					var f11f1f1elem string
					f11f1f1elem = *f11f1f1iter
					f11f1f1 = append(f11f1f1, &f11f1f1elem)
				}
				f11f1.EntryPoint = f11f1f1
			}
			if resp.ImageConfigResponse.ImageConfig.WorkingDirectory != nil {
				f11f1.WorkingDirectory = resp.ImageConfigResponse.ImageConfig.WorkingDirectory
			}
			f11.ImageConfig = f11f1
		}
		ko.Status.ImageConfigResponse = f11
	} else {
		ko.Status.ImageConfigResponse = nil
	}
	if resp.KMSKeyArn != nil {
		ko.Spec.KMSKeyARN = resp.KMSKeyArn
	} else {
		ko.Spec.KMSKeyARN = nil
	}
	if resp.LastModified != nil {
		ko.Status.LastModified = resp.LastModified
	} else {
		ko.Status.LastModified = nil
	}
	if resp.LastUpdateStatus != nil {
		ko.Status.LastUpdateStatus = resp.LastUpdateStatus
	} else {
		ko.Status.LastUpdateStatus = nil
	}
	if resp.LastUpdateStatusReason != nil {
		ko.Status.LastUpdateStatusReason = resp.LastUpdateStatusReason
	} else {
		ko.Status.LastUpdateStatusReason = nil
	}
	if resp.LastUpdateStatusReasonCode != nil {
		ko.Status.LastUpdateStatusReasonCode = resp.LastUpdateStatusReasonCode
	} else {
		ko.Status.LastUpdateStatusReasonCode = nil
	}
	if resp.MasterArn != nil {
		ko.Status.MasterARN = resp.MasterArn
	} else {
		ko.Status.MasterARN = nil
	}
	if resp.MemorySize != nil {
		ko.Spec.MemorySize = resp.MemorySize
	} else {
		ko.Spec.MemorySize = nil
	}
	if resp.PackageType != nil {
		ko.Spec.PackageType = resp.PackageType
	} else {
		ko.Spec.PackageType = nil
	}
	if resp.RevisionId != nil {
		ko.Status.RevisionID = resp.RevisionId
	} else {
		ko.Status.RevisionID = nil
	}
	if resp.Role != nil {
		ko.Spec.Role = resp.Role
	} else {
		ko.Spec.Role = nil
	}
	if resp.Runtime != nil {
		ko.Spec.Runtime = resp.Runtime
	} else {
		ko.Spec.Runtime = nil
	}
	if resp.SigningJobArn != nil {
		ko.Status.SigningJobARN = resp.SigningJobArn
	} else {
		ko.Status.SigningJobARN = nil
	}
	if resp.SigningProfileVersionArn != nil {
		ko.Status.SigningProfileVersionARN = resp.SigningProfileVersionArn
	} else {
		ko.Status.SigningProfileVersionARN = nil
	}
	if resp.SnapStart != nil {
		f26 := &svcapitypes.SnapStart{}
		if resp.SnapStart.ApplyOn != nil {
			f26.ApplyOn = resp.SnapStart.ApplyOn
		}
		ko.Spec.SnapStart = f26
	} else {
		ko.Spec.SnapStart = nil
	}
	if resp.State != nil {
		ko.Status.State = resp.State
	} else {
		ko.Status.State = nil
	}
	if resp.StateReason != nil {
		ko.Status.StateReason = resp.StateReason
	} else {
		ko.Status.StateReason = nil
	}
	if resp.StateReasonCode != nil {
		ko.Status.StateReasonCode = resp.StateReasonCode
	} else {
		ko.Status.StateReasonCode = nil
	}
	if resp.Timeout != nil {
		ko.Spec.Timeout = resp.Timeout
	} else {
		ko.Spec.Timeout = nil
	}
	if resp.TracingConfig != nil {
		f31 := &svcapitypes.TracingConfig{}
		if resp.TracingConfig.Mode != nil {
			f31.Mode = resp.TracingConfig.Mode
		}
		ko.Spec.TracingConfig = f31
	} else {
		ko.Spec.TracingConfig = nil
	}
	if resp.Version != nil {
		ko.Status.Version = resp.Version
	} else {
		ko.Status.Version = nil
	}
	if resp.VpcConfig != nil {
		f33 := &svcapitypes.VPCConfig{}
		if resp.VpcConfig.SecurityGroupIds != nil {
			f33f0 := []*string{}
			for _, f33f0iter := range resp.VpcConfig.SecurityGroupIds {
				var f33f0elem string
				f33f0elem = *f33f0iter
				f33f0 = append(f33f0, &f33f0elem)
			}
			f33.SecurityGroupIDs = f33f0
		}
		if resp.VpcConfig.SubnetIds != nil {
			f33f1 := []*string{}
			for _, f33f1iter := range resp.VpcConfig.SubnetIds {
				var f33f1elem string
				f33f1elem = *f33f1iter
				f33f1 = append(f33f1, &f33f1elem)
			}
			f33.SubnetIDs = f33f1
		}
		ko.Spec.VPCConfig = f33
	} else {
		ko.Spec.VPCConfig = nil
	}

	rm.setStatusDefaults(ko)
	if resp.Layers != nil {
		f16 := []*svcapitypes.Layer{}
		for _, f16iter := range resp.Layers {
			f16elem := &svcapitypes.Layer{}
			if f16iter.Arn != nil {
				f16elem.ARN = f16iter.Arn
			}
			if f16iter.CodeSize != nil {
				f16elem.CodeSize = f16iter.CodeSize
			}
			if f16iter.SigningJobArn != nil {
				f16elem.SigningJobARN = f16iter.SigningJobArn
			}
			if f16iter.SigningProfileVersionArn != nil {
				f16elem.SigningProfileVersionARN = f16iter.SigningProfileVersionArn
			}
			f16 = append(f16, f16elem)
		}
		ko.Status.LayerStatuses = f16
	} else {
		ko.Status.LayerStatuses = nil
	}
	return &resource{ko}, nil
}

// newCreateRequestPayload returns an SDK-specific struct for the HTTP request
// payload of the Create API call for the resource
func (rm *resourceManager) newCreateRequestPayload(
	ctx context.Context,
	r *resource,
) (*svcsdk.CreateFunctionInput, error) {
	res := &svcsdk.CreateFunctionInput{}

	if r.ko.Spec.Architectures != nil {
		f0 := []*string{}
		for _, f0iter := range r.ko.Spec.Architectures {
			var f0elem string
			f0elem = *f0iter
			f0 = append(f0, &f0elem)
		}
		res.SetArchitectures(f0)
	}
	if r.ko.Spec.Code != nil {
		f1 := &svcsdk.FunctionCode{}
		if r.ko.Spec.Code.ImageURI != nil {
			f1.SetImageUri(*r.ko.Spec.Code.ImageURI)
		}
		if r.ko.Spec.Code.S3Bucket != nil {
			f1.SetS3Bucket(*r.ko.Spec.Code.S3Bucket)
		}
		if r.ko.Spec.Code.S3Key != nil {
			f1.SetS3Key(*r.ko.Spec.Code.S3Key)
		}
		if r.ko.Spec.Code.S3ObjectVersion != nil {
			f1.SetS3ObjectVersion(*r.ko.Spec.Code.S3ObjectVersion)
		}
		if r.ko.Spec.Code.ZipFile != nil {
			f1.SetZipFile(r.ko.Spec.Code.ZipFile)
		}
		res.SetCode(f1)
	}
	if r.ko.Spec.CodeSigningConfigARN != nil {
		res.SetCodeSigningConfigArn(*r.ko.Spec.CodeSigningConfigARN)
	}
	if r.ko.Spec.DeadLetterConfig != nil {
		f3 := &svcsdk.DeadLetterConfig{}
		if r.ko.Spec.DeadLetterConfig.TargetARN != nil {
			f3.SetTargetArn(*r.ko.Spec.DeadLetterConfig.TargetARN)
		}
		res.SetDeadLetterConfig(f3)
	}
	if r.ko.Spec.Description != nil {
		res.SetDescription(*r.ko.Spec.Description)
	}
	if r.ko.Spec.Environment != nil {
		f5 := &svcsdk.Environment{}
		if r.ko.Spec.Environment.Variables != nil {
			f5f0 := map[string]*string{}
			for f5f0key, f5f0valiter := range r.ko.Spec.Environment.Variables {
				var f5f0val string
				f5f0val = *f5f0valiter
				f5f0[f5f0key] = &f5f0val
			}
			f5.SetVariables(f5f0)
		}
		res.SetEnvironment(f5)
	}
	if r.ko.Spec.EphemeralStorage != nil {
		f6 := &svcsdk.EphemeralStorage{}
		if r.ko.Spec.EphemeralStorage.Size != nil {
			f6.SetSize(*r.ko.Spec.EphemeralStorage.Size)
		}
		res.SetEphemeralStorage(f6)
	}
	if r.ko.Spec.FileSystemConfigs != nil {
		f7 := []*svcsdk.FileSystemConfig{}
		for _, f7iter := range r.ko.Spec.FileSystemConfigs {
			f7elem := &svcsdk.FileSystemConfig{}
			if f7iter.ARN != nil {
				f7elem.SetArn(*f7iter.ARN)
			}
			if f7iter.LocalMountPath != nil {
				f7elem.SetLocalMountPath(*f7iter.LocalMountPath)
			}
			f7 = append(f7, f7elem)
		}
		res.SetFileSystemConfigs(f7)
	}
	if r.ko.Spec.Name != nil {
		res.SetFunctionName(*r.ko.Spec.Name)
	}
	if r.ko.Spec.Handler != nil {
		res.SetHandler(*r.ko.Spec.Handler)
	}
	if r.ko.Spec.ImageConfig != nil {
		f10 := &svcsdk.ImageConfig{}
		if r.ko.Spec.ImageConfig.Command != nil {
			f10f0 := []*string{}
			for _, f10f0iter := range r.ko.Spec.ImageConfig.Command {
				var f10f0elem string
				f10f0elem = *f10f0iter
				f10f0 = append(f10f0, &f10f0elem)
			}
			f10.SetCommand(f10f0)
		}
		if r.ko.Spec.ImageConfig.EntryPoint != nil {
			f10f1 := []*string{}
			for _, f10f1iter := range r.ko.Spec.ImageConfig.EntryPoint {
				var f10f1elem string
				f10f1elem = *f10f1iter
				f10f1 = append(f10f1, &f10f1elem)
			}
			f10.SetEntryPoint(f10f1)
		}
		if r.ko.Spec.ImageConfig.WorkingDirectory != nil {
			f10.SetWorkingDirectory(*r.ko.Spec.ImageConfig.WorkingDirectory)
		}
		res.SetImageConfig(f10)
	}
	if r.ko.Spec.KMSKeyARN != nil {
		res.SetKMSKeyArn(*r.ko.Spec.KMSKeyARN)
	}
	if r.ko.Spec.Layers != nil {
		f12 := []*string{}
		for _, f12iter := range r.ko.Spec.Layers {
			var f12elem string
			f12elem = *f12iter
			f12 = append(f12, &f12elem)
		}
		res.SetLayers(f12)
	}
	if r.ko.Spec.MemorySize != nil {
		res.SetMemorySize(*r.ko.Spec.MemorySize)
	}
	if r.ko.Spec.PackageType != nil {
		res.SetPackageType(*r.ko.Spec.PackageType)
	}
	if r.ko.Spec.Publish != nil {
		res.SetPublish(*r.ko.Spec.Publish)
	}
	if r.ko.Spec.Role != nil {
		res.SetRole(*r.ko.Spec.Role)
	}
	if r.ko.Spec.Runtime != nil {
		res.SetRuntime(*r.ko.Spec.Runtime)
	}
	if r.ko.Spec.SnapStart != nil {
		f18 := &svcsdk.SnapStart{}
		if r.ko.Spec.SnapStart.ApplyOn != nil {
			f18.SetApplyOn(*r.ko.Spec.SnapStart.ApplyOn)
		}
		res.SetSnapStart(f18)
	}
	if r.ko.Spec.Tags != nil {
		f19 := map[string]*string{}
		for f19key, f19valiter := range r.ko.Spec.Tags {
			var f19val string
			f19val = *f19valiter
			f19[f19key] = &f19val
		}
		res.SetTags(f19)
	}
	if r.ko.Spec.Timeout != nil {
		res.SetTimeout(*r.ko.Spec.Timeout)
	}
	if r.ko.Spec.TracingConfig != nil {
		f21 := &svcsdk.TracingConfig{}
		if r.ko.Spec.TracingConfig.Mode != nil {
			f21.SetMode(*r.ko.Spec.TracingConfig.Mode)
		}
		res.SetTracingConfig(f21)
	}
	if r.ko.Spec.VPCConfig != nil {
		f22 := &svcsdk.VpcConfig{}
		if r.ko.Spec.VPCConfig.SecurityGroupIDs != nil {
			f22f0 := []*string{}
			for _, f22f0iter := range r.ko.Spec.VPCConfig.SecurityGroupIDs {
				var f22f0elem string
				f22f0elem = *f22f0iter
				f22f0 = append(f22f0, &f22f0elem)
			}
			f22.SetSecurityGroupIds(f22f0)
		}
		if r.ko.Spec.VPCConfig.SubnetIDs != nil {
			f22f1 := []*string{}
			for _, f22f1iter := range r.ko.Spec.VPCConfig.SubnetIDs {
				var f22f1elem string
				f22f1elem = *f22f1iter
				f22f1 = append(f22f1, &f22f1elem)
			}
			f22.SetSubnetIds(f22f1)
		}
		res.SetVpcConfig(f22)
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
) (*resource, error) {
	return rm.customUpdateFunction(ctx, desired, latest, delta)
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
	var resp *svcsdk.DeleteFunctionOutput
	_ = resp
	resp, err = rm.sdkapi.DeleteFunctionWithContext(ctx, input)
	rm.metrics.RecordAPICall("DELETE", "DeleteFunction", err)
	return nil, err
}

// newDeleteRequestPayload returns an SDK-specific struct for the HTTP request
// payload of the Delete API call for the resource
func (rm *resourceManager) newDeleteRequestPayload(
	r *resource,
) (*svcsdk.DeleteFunctionInput, error) {
	res := &svcsdk.DeleteFunctionInput{}

	if r.ko.Spec.Name != nil {
		res.SetFunctionName(*r.ko.Spec.Name)
	}

	return res, nil
}

// setStatusDefaults sets default properties into supplied custom resource
func (rm *resourceManager) setStatusDefaults(
	ko *svcapitypes.Function,
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
