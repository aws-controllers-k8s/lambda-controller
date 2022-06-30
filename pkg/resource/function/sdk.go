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
		if awsErr, ok := ackerr.AWSError(err); ok && awsErr.Code() == "ResourceNotFoundException" {
			return nil, ackerr.NotFound
		}
		return nil, err
	}

	// Merge in the information we read from the API call above to the copy of
	// the original Kubernetes object we passed to the function
	ko := r.ko.DeepCopy()

	if resp.Code != nil {
		f0 := &svcapitypes.FunctionCode{}
		if resp.Code.ImageUri != nil {
			f0.ImageURI = resp.Code.ImageUri
		}
		ko.Spec.Code = f0
	} else {
		ko.Spec.Code = nil
	}
	if resp.Tags != nil {
		f3 := map[string]*string{}
		for f3key, f3valiter := range resp.Tags {
			var f3val string
			f3val = *f3valiter
			f3[f3key] = &f3val
		}
		ko.Spec.Tags = f3
	} else {
		ko.Spec.Tags = nil
	}

	rm.setStatusDefaults(ko)
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
		f2 := &svcapitypes.DeadLetterConfig{}
		if resp.Configuration.DeadLetterConfig.TargetArn != nil {
			f2.TargetARN = resp.Configuration.DeadLetterConfig.TargetArn
		}
		ko.Spec.DeadLetterConfig = f2
	} else {
		ko.Spec.DeadLetterConfig = nil
	}
	if resp.Configuration.Description != nil {
		ko.Spec.Description = resp.Configuration.Description
	} else {
		ko.Spec.Description = nil
	}
	if resp.Configuration.Environment != nil {
		f4 := &svcapitypes.Environment{}
		if resp.Configuration.Environment.Variables != nil {
			f4f1 := map[string]*string{}
			for f4f1key, f4f1valiter := range resp.Configuration.Environment.Variables {
				var f4f1val string
				f4f1val = *f4f1valiter
				f4f1[f4f1key] = &f4f1val
			}
			f4.Variables = f4f1
		}
		ko.Spec.Environment = f4
	} else {
		ko.Spec.Environment = nil
	}
	if resp.Configuration.FileSystemConfigs != nil {
		f5 := []*svcapitypes.FileSystemConfig{}
		for _, f5iter := range resp.Configuration.FileSystemConfigs {
			f5elem := &svcapitypes.FileSystemConfig{}
			if f5iter.Arn != nil {
				f5elem.ARN = f5iter.Arn
			}
			if f5iter.LocalMountPath != nil {
				f5elem.LocalMountPath = f5iter.LocalMountPath
			}
			f5 = append(f5, f5elem)
		}
		ko.Spec.FileSystemConfigs = f5
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
		f9 := &svcapitypes.ImageConfigResponse{}
		if resp.Configuration.ImageConfigResponse.Error != nil {
			f9f0 := &svcapitypes.ImageConfigError{}
			if resp.Configuration.ImageConfigResponse.Error.ErrorCode != nil {
				f9f0.ErrorCode = resp.Configuration.ImageConfigResponse.Error.ErrorCode
			}
			if resp.Configuration.ImageConfigResponse.Error.Message != nil {
				f9f0.Message = resp.Configuration.ImageConfigResponse.Error.Message
			}
			f9.Error = f9f0
		}
		if resp.Configuration.ImageConfigResponse.ImageConfig != nil {
			f9f1 := &svcapitypes.ImageConfig{}
			if resp.Configuration.ImageConfigResponse.ImageConfig.Command != nil {
				f9f1f0 := []*string{}
				for _, f9f1f0iter := range resp.Configuration.ImageConfigResponse.ImageConfig.Command {
					var f9f1f0elem string
					f9f1f0elem = *f9f1f0iter
					f9f1f0 = append(f9f1f0, &f9f1f0elem)
				}
				f9f1.Command = f9f1f0
			}
			if resp.Configuration.ImageConfigResponse.ImageConfig.EntryPoint != nil {
				f9f1f1 := []*string{}
				for _, f9f1f1iter := range resp.Configuration.ImageConfigResponse.ImageConfig.EntryPoint {
					var f9f1f1elem string
					f9f1f1elem = *f9f1f1iter
					f9f1f1 = append(f9f1f1, &f9f1f1elem)
				}
				f9f1.EntryPoint = f9f1f1
			}
			if resp.Configuration.ImageConfigResponse.ImageConfig.WorkingDirectory != nil {
				f9f1.WorkingDirectory = resp.Configuration.ImageConfigResponse.ImageConfig.WorkingDirectory
			}
			f9.ImageConfig = f9f1
		}
		ko.Status.ImageConfigResponse = f9
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
		f28 := &svcapitypes.TracingConfig{}
		if resp.Configuration.TracingConfig.Mode != nil {
			f28.Mode = resp.Configuration.TracingConfig.Mode
		}
		ko.Spec.TracingConfig = f28
	} else {
		ko.Spec.TracingConfig = nil
	}
	if resp.Configuration.Version != nil {
		ko.Status.Version = resp.Configuration.Version
	} else {
		ko.Status.Version = nil
	}
	if resp.Configuration.VpcConfig != nil {
		f30 := &svcapitypes.VPCConfig{}
		if resp.Configuration.VpcConfig.SecurityGroupIds != nil {
			f30f0 := []*string{}
			for _, f30f0iter := range resp.Configuration.VpcConfig.SecurityGroupIds {
				var f30f0elem string
				f30f0elem = *f30f0iter
				f30f0 = append(f30f0, &f30f0elem)
			}
			f30.SecurityGroupIDs = f30f0
		}
		if resp.Configuration.VpcConfig.SubnetIds != nil {
			f30f1 := []*string{}
			for _, f30f1iter := range resp.Configuration.VpcConfig.SubnetIds {
				var f30f1elem string
				f30f1elem = *f30f1iter
				f30f1 = append(f30f1, &f30f1elem)
			}
			f30.SubnetIDs = f30f1
		}
		ko.Spec.VPCConfig = f30
	} else {
		ko.Spec.VPCConfig = nil
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
	if resp.FileSystemConfigs != nil {
		f6 := []*svcapitypes.FileSystemConfig{}
		for _, f6iter := range resp.FileSystemConfigs {
			f6elem := &svcapitypes.FileSystemConfig{}
			if f6iter.Arn != nil {
				f6elem.ARN = f6iter.Arn
			}
			if f6iter.LocalMountPath != nil {
				f6elem.LocalMountPath = f6iter.LocalMountPath
			}
			f6 = append(f6, f6elem)
		}
		ko.Spec.FileSystemConfigs = f6
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
		f10 := &svcapitypes.ImageConfigResponse{}
		if resp.ImageConfigResponse.Error != nil {
			f10f0 := &svcapitypes.ImageConfigError{}
			if resp.ImageConfigResponse.Error.ErrorCode != nil {
				f10f0.ErrorCode = resp.ImageConfigResponse.Error.ErrorCode
			}
			if resp.ImageConfigResponse.Error.Message != nil {
				f10f0.Message = resp.ImageConfigResponse.Error.Message
			}
			f10.Error = f10f0
		}
		if resp.ImageConfigResponse.ImageConfig != nil {
			f10f1 := &svcapitypes.ImageConfig{}
			if resp.ImageConfigResponse.ImageConfig.Command != nil {
				f10f1f0 := []*string{}
				for _, f10f1f0iter := range resp.ImageConfigResponse.ImageConfig.Command {
					var f10f1f0elem string
					f10f1f0elem = *f10f1f0iter
					f10f1f0 = append(f10f1f0, &f10f1f0elem)
				}
				f10f1.Command = f10f1f0
			}
			if resp.ImageConfigResponse.ImageConfig.EntryPoint != nil {
				f10f1f1 := []*string{}
				for _, f10f1f1iter := range resp.ImageConfigResponse.ImageConfig.EntryPoint {
					var f10f1f1elem string
					f10f1f1elem = *f10f1f1iter
					f10f1f1 = append(f10f1f1, &f10f1f1elem)
				}
				f10f1.EntryPoint = f10f1f1
			}
			if resp.ImageConfigResponse.ImageConfig.WorkingDirectory != nil {
				f10f1.WorkingDirectory = resp.ImageConfigResponse.ImageConfig.WorkingDirectory
			}
			f10.ImageConfig = f10f1
		}
		ko.Status.ImageConfigResponse = f10
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
		f29 := &svcapitypes.TracingConfig{}
		if resp.TracingConfig.Mode != nil {
			f29.Mode = resp.TracingConfig.Mode
		}
		ko.Spec.TracingConfig = f29
	} else {
		ko.Spec.TracingConfig = nil
	}
	if resp.Version != nil {
		ko.Status.Version = resp.Version
	} else {
		ko.Status.Version = nil
	}
	if resp.VpcConfig != nil {
		f31 := &svcapitypes.VPCConfig{}
		if resp.VpcConfig.SecurityGroupIds != nil {
			f31f0 := []*string{}
			for _, f31f0iter := range resp.VpcConfig.SecurityGroupIds {
				var f31f0elem string
				f31f0elem = *f31f0iter
				f31f0 = append(f31f0, &f31f0elem)
			}
			f31.SecurityGroupIDs = f31f0
		}
		if resp.VpcConfig.SubnetIds != nil {
			f31f1 := []*string{}
			for _, f31f1iter := range resp.VpcConfig.SubnetIds {
				var f31f1elem string
				f31f1elem = *f31f1iter
				f31f1 = append(f31f1, &f31f1elem)
			}
			f31.SubnetIDs = f31f1
		}
		ko.Spec.VPCConfig = f31
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
	if r.ko.Spec.FileSystemConfigs != nil {
		f6 := []*svcsdk.FileSystemConfig{}
		for _, f6iter := range r.ko.Spec.FileSystemConfigs {
			f6elem := &svcsdk.FileSystemConfig{}
			if f6iter.ARN != nil {
				f6elem.SetArn(*f6iter.ARN)
			}
			if f6iter.LocalMountPath != nil {
				f6elem.SetLocalMountPath(*f6iter.LocalMountPath)
			}
			f6 = append(f6, f6elem)
		}
		res.SetFileSystemConfigs(f6)
	}
	if r.ko.Spec.Name != nil {
		res.SetFunctionName(*r.ko.Spec.Name)
	}
	if r.ko.Spec.Handler != nil {
		res.SetHandler(*r.ko.Spec.Handler)
	}
	if r.ko.Spec.ImageConfig != nil {
		f9 := &svcsdk.ImageConfig{}
		if r.ko.Spec.ImageConfig.Command != nil {
			f9f0 := []*string{}
			for _, f9f0iter := range r.ko.Spec.ImageConfig.Command {
				var f9f0elem string
				f9f0elem = *f9f0iter
				f9f0 = append(f9f0, &f9f0elem)
			}
			f9.SetCommand(f9f0)
		}
		if r.ko.Spec.ImageConfig.EntryPoint != nil {
			f9f1 := []*string{}
			for _, f9f1iter := range r.ko.Spec.ImageConfig.EntryPoint {
				var f9f1elem string
				f9f1elem = *f9f1iter
				f9f1 = append(f9f1, &f9f1elem)
			}
			f9.SetEntryPoint(f9f1)
		}
		if r.ko.Spec.ImageConfig.WorkingDirectory != nil {
			f9.SetWorkingDirectory(*r.ko.Spec.ImageConfig.WorkingDirectory)
		}
		res.SetImageConfig(f9)
	}
	if r.ko.Spec.KMSKeyARN != nil {
		res.SetKMSKeyArn(*r.ko.Spec.KMSKeyARN)
	}
	if r.ko.Spec.Layers != nil {
		f11 := []*string{}
		for _, f11iter := range r.ko.Spec.Layers {
			var f11elem string
			f11elem = *f11iter
			f11 = append(f11, &f11elem)
		}
		res.SetLayers(f11)
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
	if r.ko.Spec.Tags != nil {
		f17 := map[string]*string{}
		for f17key, f17valiter := range r.ko.Spec.Tags {
			var f17val string
			f17val = *f17valiter
			f17[f17key] = &f17val
		}
		res.SetTags(f17)
	}
	if r.ko.Spec.Timeout != nil {
		res.SetTimeout(*r.ko.Spec.Timeout)
	}
	if r.ko.Spec.TracingConfig != nil {
		f19 := &svcsdk.TracingConfig{}
		if r.ko.Spec.TracingConfig.Mode != nil {
			f19.SetMode(*r.ko.Spec.TracingConfig.Mode)
		}
		res.SetTracingConfig(f19)
	}
	if r.ko.Spec.VPCConfig != nil {
		f20 := &svcsdk.VpcConfig{}
		if r.ko.Spec.VPCConfig.SecurityGroupIDs != nil {
			f20f0 := []*string{}
			for _, f20f0iter := range r.ko.Spec.VPCConfig.SecurityGroupIDs {
				var f20f0elem string
				f20f0elem = *f20f0iter
				f20f0 = append(f20f0, &f20f0elem)
			}
			f20.SetSecurityGroupIds(f20f0)
		}
		if r.ko.Spec.VPCConfig.SubnetIDs != nil {
			f20f1 := []*string{}
			for _, f20f1iter := range r.ko.Spec.VPCConfig.SubnetIDs {
				var f20f1elem string
				f20f1elem = *f20f1iter
				f20f1 = append(f20f1, &f20f1elem)
			}
			f20.SetSubnetIds(f20f1)
		}
		res.SetVpcConfig(f20)
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
	if err == nil {
		return false
	}
	awsErr, ok := ackerr.AWSError(err)
	if !ok {
		return false
	}
	switch awsErr.Code() {
	case "InvalidParameterValueException":
		return true
	default:
		return false
	}
}
