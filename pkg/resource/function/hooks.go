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

package function

import (
	"context"
	"errors"
	"strings"
	"time"

	svcapitypes "github.com/aws-controllers-k8s/lambda-controller/apis/v1alpha1"
	ackcompare "github.com/aws-controllers-k8s/runtime/pkg/compare"
	ackerr "github.com/aws-controllers-k8s/runtime/pkg/errors"
	ackrequeue "github.com/aws-controllers-k8s/runtime/pkg/requeue"
	ackrtlog "github.com/aws-controllers-k8s/runtime/pkg/runtime/log"
	"github.com/aws/aws-sdk-go-v2/aws"
	svcsdk "github.com/aws/aws-sdk-go-v2/service/lambda"
	svcsdktypes "github.com/aws/aws-sdk-go-v2/service/lambda/types"
)

var (
	ErrFunctionPending           = errors.New("function in 'Pending' state, cannot be modified or deleted")
	ErrSourceImageDoesNotExist   = errors.New("source image does not exist")
	ErrCannotSetFunctionCSC      = errors.New("cannot set function code signing config when package type is Image")
	ErrCannotModifyTenancyConfig = errors.New("tenancy config cannot be modified after function creation")
)

var (
	requeueWaitWhilePending = ackrequeue.NeededAfter(
		ErrFunctionPending,
		5*time.Second,
	)
	requeueWaitWhileSourceImageDoesNotExist = ackrequeue.NeededAfter(
		ErrSourceImageDoesNotExist,
		1*time.Minute,
	)
)

// isFunctionPending returns true if the supplied Lambda Function is in a pending
// state
func isFunctionPending(r *resource) bool {
	if r.ko.Status.State == nil {
		return false
	}
	state := *r.ko.Status.State
	return state == string(svcapitypes.State_Pending)
}

// customUpdateFunction patches each of the resource properties in the backend AWS
// service API and returns a new resource with updated fields.
func (rm *resourceManager) customUpdateFunction(
	ctx context.Context,
	desired *resource,
	latest *resource,
	delta *ackcompare.Delta,
) (*resource, error) {
	var err error
	rlog := ackrtlog.FromContext(ctx)
	exit := rlog.Trace("rm.customUpdateFunction")
	defer exit(err)

	updatedStatusResource := rm.concreteResource(desired.DeepCopy())
	updatedStatusResource.SetStatus(latest)
	if isFunctionPending(latest) {
		return updatedStatusResource, requeueWaitWhilePending
	}

	if delta.DifferentAt("Spec.Tags") {
		err = rm.updateFunctionTags(ctx, latest, desired)
		if err != nil {
			return updatedStatusResource, err
		}
	}
	if delta.DifferentAt("Spec.ReservedConcurrentExecutions") {
		err = rm.updateFunctionConcurrency(ctx, desired)
		if err != nil {
			return updatedStatusResource, err
		}
	}
	if delta.DifferentAt("Spec.FunctionEventInvokeConfig") {
		err = rm.syncFunctionEventInvokeConfig(ctx, desired)
		if err != nil {
			return updatedStatusResource, err
		}
	}
	if delta.DifferentAt("Spec.CodeSigningConfigARN") {
		if desired.ko.Spec.PackageType != nil && *desired.ko.Spec.PackageType == "Image" &&
			desired.ko.Spec.CodeSigningConfigARN != nil && *desired.ko.Spec.CodeSigningConfigARN != "" {
			return updatedStatusResource, ackerr.NewTerminalError(ErrCannotSetFunctionCSC)
		} else {
			err = rm.updateFunctionCodeSigningConfig(ctx, desired)
			if err != nil {
				return updatedStatusResource, err
			}
		}
	}
	if delta.DifferentAt("Spec.TenancyConfig") {
		return updatedStatusResource, ackerr.NewTerminalError(ErrCannotModifyTenancyConfig)
	}

	// Only try to update Spec.Code or Spec.Configuration at once. It is
	// not correct to sequentially call UpdateFunctionConfiguration and
	// UpdateFunctionCode because both of them can put the function in a
	// Pending state.
	switch {
	case delta.DifferentAt("Spec.Code.ImageURI") || delta.DifferentAt("Spec.Code.SHA256") || delta.DifferentAt("Spec.Architectures"):
		err = rm.updateFunctionCode(ctx, desired, delta, latest)
		if err != nil {
			if strings.Contains(err.Error(), "Provide a valid source image.") {
				return updatedStatusResource, requeueWaitWhileSourceImageDoesNotExist
			} else {
				return updatedStatusResource, err
			}
		}
	case delta.DifferentExcept(
		"Spec.Code",
		"Spec.Tags",
		"Spec.ReservedConcurrentExecutions",
		"Spec.FunctionEventInvokeConfig",
		"Spec.CodeSigningConfigARN",
		"Spec.TenancyConfig"):
		err = rm.updateFunctionConfiguration(ctx, desired, delta)
		if err != nil {
			return updatedStatusResource, err
		}
	}

	readOneLatest, err := rm.ReadOne(ctx, desired)
	if err != nil {
		return updatedStatusResource, err
	}
	return rm.concreteResource(readOneLatest), nil
}

// updateFunctionConfiguration calls the UpdateFunctionConfiguration to edit a
// specific lambda function configuration.
func (rm *resourceManager) updateFunctionConfiguration(
	ctx context.Context,
	desired *resource,
	delta *ackcompare.Delta,
) error {
	var err error
	rlog := ackrtlog.FromContext(ctx)
	exit := rlog.Trace("rm.updateFunctionConfiguration")
	defer exit(err)

	dspec := desired.ko.Spec
	input := &svcsdk.UpdateFunctionConfigurationInput{
		FunctionName: aws.String(*dspec.Name),
	}

	if delta.DifferentAt("Spec.DeadLetterConfig") {
		deadLetterConfig := &svcsdktypes.DeadLetterConfig{}
		if dspec.DeadLetterConfig != nil {
			deadLetterConfigCopy := dspec.DeadLetterConfig.DeepCopy()
			deadLetterConfig.TargetArn = deadLetterConfigCopy.TargetARN
		}
		input.DeadLetterConfig = deadLetterConfig
	}

	if delta.DifferentAt("Spec.Description") {
		if dspec.Description != nil {
			input.Description = aws.String(*dspec.Description)
		} else {
			input.Description = aws.String("")
		}
	}

	if delta.DifferentAt("Spec.Environment") {
		environment := &svcsdktypes.Environment{}
		if dspec.Environment != nil {
			environmentCopy := dspec.Environment.DeepCopy()
			environment.Variables = make(map[string]string)
			for k, v := range environmentCopy.Variables {
				environment.Variables[k] = *v
			}
		}
		input.Environment = environment
	}

	if delta.DifferentAt("Spec.EphemeralStorage") {
		ephemeralStorage := &svcsdktypes.EphemeralStorage{}
		if dspec.EphemeralStorage != nil {
			ephemeralStorageCopy := dspec.EphemeralStorage.DeepCopy()
			ephemeralStorage.Size = aws.Int32(int32(*ephemeralStorageCopy.Size))
		}
		input.EphemeralStorage = ephemeralStorage
	}

	if delta.DifferentAt("Spec.FileSystemConfigs") {
		fileSystemConfigs := []svcsdktypes.FileSystemConfig{}
		if len(dspec.FileSystemConfigs) > 0 {
			for _, elem := range dspec.FileSystemConfigs {
				elemCopy := elem.DeepCopy()
				fscElem := &svcsdktypes.FileSystemConfig{
					Arn:            elemCopy.ARN,
					LocalMountPath: elemCopy.LocalMountPath,
				}
				fileSystemConfigs = append(fileSystemConfigs, *fscElem)
			}
			input.FileSystemConfigs = fileSystemConfigs
		}
	}

	if delta.DifferentAt("Spec.Handler") {
		if dspec.Handler != nil {
			input.Handler = aws.String(*dspec.Handler)
		} else {
			input.Handler = aws.String("")
		}
	}

	if delta.DifferentAt("Spec.ImageConfig") {
		if dspec.ImageConfig != nil && dspec.Code.ImageURI != nil && *dspec.Code.ImageURI != "" {
			imageConfig := &svcsdktypes.ImageConfig{}
			if dspec.ImageConfig != nil {
				imageConfigCopy := dspec.ImageConfig.DeepCopy()
				imageConfig.Command = make([]string, len(imageConfigCopy.Command))
				for i, elem := range imageConfigCopy.Command {
					imageConfig.Command[i] = *elem
				}
				imageConfig.EntryPoint = make([]string, len(imageConfigCopy.EntryPoint))
				for i, elem := range imageConfigCopy.EntryPoint {
					imageConfig.EntryPoint[i] = *elem
				}
				imageConfig.WorkingDirectory = imageConfigCopy.WorkingDirectory
			}
			input.ImageConfig = imageConfig
		}
	}

	if delta.DifferentAt("Spec.KMSKeyARN") {
		if dspec.KMSKeyARN != nil {
			input.KMSKeyArn = aws.String(*dspec.KMSKeyARN)
		} else {
			input.KMSKeyArn = aws.String("")
		}
	}

	if delta.DifferentAt("Spec.Layers") {
		layers := []string{}
		if len(dspec.Layers) > 0 {
			for _, iter := range dspec.Layers {
				var elem string = *iter
				layers = append(layers, elem)
			}
			input.Layers = layers
		}
	}

	if delta.DifferentAt("Spec.LoggingConfig") {
		if dspec.LoggingConfig != nil {
			logConfig := &svcsdktypes.LoggingConfig{}
			if dspec.LoggingConfig.ApplicationLogLevel != nil {
				logConfig.ApplicationLogLevel = svcsdktypes.ApplicationLogLevel(*dspec.LoggingConfig.ApplicationLogLevel)
			}
			if dspec.LoggingConfig.LogFormat != nil {
				logConfig.LogFormat = svcsdktypes.LogFormat(*dspec.LoggingConfig.LogFormat)
			}
			if dspec.LoggingConfig.LogGroup != nil {
				logConfig.LogGroup = dspec.LoggingConfig.LogGroup
			}
			if dspec.LoggingConfig.SystemLogLevel != nil {
				logConfig.SystemLogLevel = svcsdktypes.SystemLogLevel(*dspec.LoggingConfig.SystemLogLevel)
			}
			input.LoggingConfig = logConfig
		}
	}

	if delta.DifferentAt("Spec.MemorySize") {
		if dspec.MemorySize != nil {
			input.MemorySize = aws.Int32(int32(*dspec.MemorySize))
		} else {
			input.MemorySize = aws.Int32(0)
		}
	}

	if delta.DifferentAt("Spec.Role") {
		if dspec.Role != nil {
			input.Role = aws.String(*dspec.Role)
		} else {
			input.Role = aws.String("")
		}
	}

	if delta.DifferentAt("Spec.Runtime") {
		if dspec.Runtime != nil {
			input.Runtime = svcsdktypes.Runtime(*dspec.Runtime)
		} else {
			input.Runtime = svcsdktypes.Runtime("")
		}
	}

	if delta.DifferentAt(("Spec.SnapStart")) {
		snapStart := &svcsdktypes.SnapStart{}
		if dspec.SnapStart != nil {
			snapStartCopy := dspec.SnapStart.DeepCopy()
			snapStart.ApplyOn = svcsdktypes.SnapStartApplyOn(*snapStartCopy.ApplyOn)
		}
		input.SnapStart = snapStart
	}

	if delta.DifferentAt("Spec.Timeout") {
		if dspec.Timeout != nil {
			input.Timeout = aws.Int32(int32(*dspec.Timeout))
		} else {
			input.Timeout = aws.Int32(0)
		}
	}

	if delta.DifferentAt("Spec.TracingConfig") {
		tracingConfig := &svcsdktypes.TracingConfig{}
		if dspec.TracingConfig != nil {
			tracingConfig.Mode = svcsdktypes.TracingMode(*dspec.TracingConfig.Mode)
		}
		input.TracingConfig = tracingConfig
	}

	if delta.DifferentAt("Spec.VPCConfig") {
		VPCConfig := &svcsdktypes.VpcConfig{}
		if dspec.VPCConfig != nil {
			vpcConfigCopy := dspec.VPCConfig.DeepCopy()
			VPCConfig.SubnetIds = make([]string, len(vpcConfigCopy.SubnetIDs))
			for i, elem := range vpcConfigCopy.SubnetIDs {
				VPCConfig.SubnetIds[i] = *elem
			}
			VPCConfig.SecurityGroupIds = make([]string, len(vpcConfigCopy.SecurityGroupIDs))
			for i, elem := range vpcConfigCopy.SecurityGroupIDs {
				VPCConfig.SecurityGroupIds[i] = *elem
			}
		}
		input.VpcConfig = VPCConfig
	}

	_, err = rm.sdkapi.UpdateFunctionConfiguration(ctx, input)
	rm.metrics.RecordAPICall("UPDATE", "UpdateFunctionConfiguration", err)
	if err != nil {
		return err
	}

	return nil
}

// updateFunctionsTags uses TagResource and UntagResource to add, remove and update
// a lambda function tags.
func (rm *resourceManager) updateFunctionTags(
	ctx context.Context,
	latest *resource,
	desired *resource,
) error {
	var err error
	rlog := ackrtlog.FromContext(ctx)
	exit := rlog.Trace("rm.updateFunctionTags")
	defer exit(err)

	added, removed, updated := compareMaps(latest.ko.Spec.Tags, desired.ko.Spec.Tags)

	// There is no api call to update tags, so we need to remove them and add them later
	// with their new values.
	if len(removed)+len(updated) > 0 {
		removeTags := []string{}
		for k := range updated {
			removeTags = append(removeTags, k)
		}
		for _, k := range removed {
			removeTags = append(removeTags, k)
		}
		input := &svcsdk.UntagResourceInput{
			Resource: (*string)(desired.ko.Status.ACKResourceMetadata.ARN),
			TagKeys:  removeTags,
		}
		_, err = rm.sdkapi.UntagResource(ctx, input)
		rm.metrics.RecordAPICall("UPDATE", "UntagResource", err)
		if err != nil {
			return err
		}
	}

	if len(updated)+len(added) > 0 {
		addedTags := map[string]string{}
		for k, v := range added {
			addedTags[k] = *v
		}
		for k, v := range updated {
			addedTags[k] = *v
		}

		input := &svcsdk.TagResourceInput{
			Resource: (*string)(desired.ko.Status.ACKResourceMetadata.ARN),
			Tags:     addedTags,
		}
		_, err = rm.sdkapi.TagResource(ctx, input)
		rm.metrics.RecordAPICall("UPDATE", "TagResource", err)
		if err != nil {
			return err
		}
	}

	return nil
}

// updateFunctionsCode calls UpdateFunctionCode to update a specific lambda
// function code.
func (rm *resourceManager) updateFunctionCode(
	ctx context.Context,
	desired *resource,
	delta *ackcompare.Delta,
	latest *resource,
) error {
	var err error
	rlog := ackrtlog.FromContext(ctx)
	exit := rlog.Trace("rm.updateFunctionCode")
	defer exit(err)

	dspec := desired.ko.Spec
	input := &svcsdk.UpdateFunctionCodeInput{
		FunctionName: aws.String(*dspec.Name),
	}

	if dspec.Architectures != nil {
		input.Architectures = make([]svcsdktypes.Architecture, len(dspec.Architectures))
		for i, elem := range dspec.Architectures {
			input.Architectures[i] = svcsdktypes.Architecture(*elem)
		}
	} else {
		input.Architectures = nil
	}

	if dspec.Code != nil {
		if delta.DifferentAt("Spec.Code.SHA256") && dspec.Code.SHA256 != nil {
			if dspec.Code.S3Key != nil {
				input.S3Key = aws.String(*dspec.Code.S3Key)
			}
			if dspec.Code.S3Bucket != nil {
				input.S3Bucket = aws.String(*dspec.Code.S3Bucket)
			}
			if dspec.Code.S3ObjectVersion != nil {
				input.S3ObjectVersion = aws.String(*dspec.Code.S3ObjectVersion)
			}
			if dspec.Code.ZipFile != nil {
				input.ZipFile = dspec.Code.ZipFile
			}
		} else if delta.DifferentAt("Spec.Code.ImageURI") && dspec.Code.ImageURI != nil {
			if dspec.Code.ImageURI != nil {
				input.ImageUri = aws.String(*dspec.Code.ImageURI)
			}

		} else { // We need to pass the latest code to Update API call,
			//if there is change in architecture and no change in Code
			if latest.ko.Spec.PackageType != nil && *latest.ko.Spec.PackageType == "Image" {
				input.ImageUri = latest.ko.Spec.Code.ImageURI
			} else if latest.ko.Spec.PackageType != nil && *latest.ko.Spec.PackageType == "Zip" {
				input.S3Bucket = latest.ko.Spec.Code.S3Bucket
				input.S3Key = latest.ko.Spec.Code.S3Key
			}
		}
	}

	_, err = rm.sdkapi.UpdateFunctionCode(ctx, input)
	rm.metrics.RecordAPICall("UPDATE", "UpdateFunctionCode", err)
	if err != nil {
		return err
	}

	return nil
}

// compareMaps compares two string to string maps and returns three outputs: a
// map of the new key/values observed, a list of the keys of the removed values
// and a map containing the updated keys and their new values.
func compareMaps(
	a map[string]*string,
	b map[string]*string,
) (added map[string]*string, removed []string, updated map[string]*string) {
	added = map[string]*string{}
	updated = map[string]*string{}
	visited := make(map[string]bool, len(a))
	for keyA, valueA := range a {
		valueB, found := b[keyA]
		if !found {
			removed = append(removed, keyA)
			continue
		}
		if *valueA != *valueB {
			updated[keyA] = valueB
		}
		visited[keyA] = true
	}
	for keyB, valueB := range b {
		_, found := a[keyB]
		if !found {
			added[keyB] = valueB
		}
	}
	return
}

func customPreCompare(
	delta *ackcompare.Delta,
	a *resource,
	b *resource,
) {
	// No need to compare difference in S3 Key/Bucket/ObjectVersion. As in sdkFind() there is a copy 'ko := r.ko.DeepCopy()'
	// of S3 Key/Bucket/ObjectVersion passed. This 'ko' then stores the values of latest S3 fields which API returns
	// and compares it with desired field values. Since the API doesn't return values of S3 fields, it doesn't
	// notice any changes between desired and latest, hence fails to recognize the update in the values.

	// To solve this we created a new field 'Code.SHA256' to store the hash value of deployment package. Any change
	// in hash value refers to change in S3 Key/Bucket/ObjectVersion and controller can recognize the change in
	// desired and latest value of 'Code.SHA256' and hence calls the update function.

	if ackcompare.HasNilDifference(a.ko.Spec.Code, b.ko.Spec.Code) {
		delta.Add("Spec.Code", a.ko.Spec.Code, b.ko.Spec.Code)
	} else if a.ko.Spec.Code != nil && b.ko.Spec.Code != nil {
		if a.ko.Spec.PackageType != nil && *a.ko.Spec.PackageType == "Zip" {
			if a.ko.Spec.Code.SHA256 != nil {
				if ackcompare.HasNilDifference(a.ko.Spec.Code.SHA256, b.ko.Status.CodeSHA256) {
					delta.Add("Spec.Code.SHA256", a.ko.Spec.Code.SHA256, b.ko.Status.CodeSHA256)
				} else if a.ko.Spec.Code.SHA256 != nil && b.ko.Status.CodeSHA256 != nil {
					if *a.ko.Spec.Code.SHA256 != *b.ko.Status.CodeSHA256 {
						delta.Add("Spec.Code.SHA256", a.ko.Spec.Code.SHA256, b.ko.Status.CodeSHA256)
					}
				}
			}
		}
	}
}

// updateFunctionConcurrency calls `PutFunctionConcurrency` to update the fields
func (rm *resourceManager) updateFunctionConcurrency(
	ctx context.Context,
	desired *resource,
) error {
	var err error
	rlog := ackrtlog.FromContext(ctx)
	exit := rlog.Trace("rm.updateFunctionConcurrency")
	defer exit(err)

	dspec := desired.ko.Spec
	input := &svcsdk.PutFunctionConcurrencyInput{
		FunctionName: aws.String(*dspec.Name),
	}

	if desired.ko.Spec.ReservedConcurrentExecutions != nil {
		input.ReservedConcurrentExecutions = aws.Int32(int32(*desired.ko.Spec.ReservedConcurrentExecutions))
	} else {
		// Delete concurrency config if value is nil
		_, err = rm.sdkapi.DeleteFunctionConcurrency(ctx, &svcsdk.DeleteFunctionConcurrencyInput{
			FunctionName: aws.String(*dspec.Name),
		})
		rm.metrics.RecordAPICall("DELETE", "DeleteFunctionConcurrency", err)
		return err
	}

	_, err = rm.sdkapi.PutFunctionConcurrency(ctx, input)
	rm.metrics.RecordAPICall("UPDATE", "PutFunctionConcurrency", err)
	return err
}

// syncFunctionEventInvokeConfig calls `PutFunctionEventInvokeConfig` to update the fields
// or `DeleteFunctionEventInvokeConfig` is users removes the fields
func (rm *resourceManager) syncFunctionEventInvokeConfig(
	ctx context.Context,
	desired *resource,
) error {
	var err error
	rlog := ackrtlog.FromContext(ctx)
	exit := rlog.Trace("rm.syncEventInvokeConfig")
	defer exit(err)

	// Check if the user deleted the 'FunctionEventInvokeConfig' configuration
	// If yes, delete FunctionEventInvokeConfig
	if desired.ko.Spec.FunctionEventInvokeConfig == nil {
		input_delete := &svcsdk.DeleteFunctionEventInvokeConfigInput{
			FunctionName: aws.String(*desired.ko.Spec.Name),
		}
		_, err = rm.sdkapi.DeleteFunctionEventInvokeConfig(ctx, input_delete)
		rm.metrics.RecordAPICall("DELETE", "DeleteFunctionEventInvokeConfig", err)
		if err != nil {
			return err
		}
		return nil
	}

	dspec := desired.ko.Spec
	input := &svcsdk.PutFunctionEventInvokeConfigInput{
		FunctionName: aws.String(*dspec.Name),
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

// updateFunctionCodeSigningConfig calls PutFunctionCodeSigningConfig to update
// the code signing configuration
func (rm *resourceManager) updateFunctionCodeSigningConfig(
	ctx context.Context,
	desired *resource,
) error {
	var err error
	rlog := ackrtlog.FromContext(ctx)
	exit := rlog.Trace("rm.updateFunctionCodeSigningConfig")
	defer exit(err)

	if desired.ko.Spec.CodeSigningConfigARN == nil || *desired.ko.Spec.CodeSigningConfigARN == "" {
		return rm.deleteFunctionCodeSigningConfig(ctx, desired)
	}

	dspec := desired.ko.Spec
	input := &svcsdk.PutFunctionCodeSigningConfigInput{
		FunctionName:         aws.String(*dspec.Name),
		CodeSigningConfigArn: aws.String(*dspec.CodeSigningConfigARN),
	}

	_, err = rm.sdkapi.PutFunctionCodeSigningConfig(ctx, input)
	rm.metrics.RecordAPICall("UPDATE", "PutFunctionCodeSigningConfig", err)
	if err != nil {
		return err
	}
	return nil
}

// deleteFunctionCodeSigningConfig calls deleteFunctionCodeSigningConfig to update
// the code signing configuration
func (rm *resourceManager) deleteFunctionCodeSigningConfig(
	ctx context.Context,
	desired *resource,
) error {
	var err error
	rlog := ackrtlog.FromContext(ctx)
	exit := rlog.Trace("rm.deleteFunctionCodeSigningConfig")
	defer exit(err)

	dspec := desired.ko.Spec
	input := &svcsdk.DeleteFunctionCodeSigningConfigInput{
		FunctionName: aws.String(*dspec.Name),
	}

	_, err = rm.sdkapi.DeleteFunctionCodeSigningConfig(ctx, input)
	rm.metrics.RecordAPICall("UPDATE", "DeleteFunctionCodeSigningConfig", err)
	if err != nil {
		return err
	}
	return nil
}

// setFunctionConcurrency sets the concurrency fields
// for the Function resource
func (rm *resourceManager) setFunctionConcurrency(
	ctx context.Context,
	ko *svcapitypes.Function,
) (err error) {
	rlog := ackrtlog.FromContext(ctx)
	exit := rlog.Trace("rm.setFunctionConcurrency")
	defer exit(err)

	// Add nil checks for required fields
	if ko == nil || ko.Spec.Name == nil {
		return nil // Skip if required fields are not set
	}

	var getFunctionConcurrencyOutput *svcsdk.GetFunctionConcurrencyOutput
	getFunctionConcurrencyOutput, err = rm.sdkapi.GetFunctionConcurrency(
		ctx,
		&svcsdk.GetFunctionConcurrencyInput{
			FunctionName: ko.Spec.Name,
		},
	)
	rm.metrics.RecordAPICall("GET", "GetFunctionConcurrency", err)
	if err != nil {
		// If the concurrency config doesn't exist, don't return error
		var notFound *svcsdktypes.ResourceNotFoundException
		if errors.As(err, &notFound) {
			ko.Spec.ReservedConcurrentExecutions = nil
			return nil
		}
		return err
	}

	if getFunctionConcurrencyOutput != nil && getFunctionConcurrencyOutput.ReservedConcurrentExecutions != nil {
		ko.Spec.ReservedConcurrentExecutions = aws.Int64(int64(*getFunctionConcurrencyOutput.ReservedConcurrentExecutions))
	} else {
		ko.Spec.ReservedConcurrentExecutions = nil
	}

	return nil
}

// setFunctionCodeSigningConfig sets the code signing
// fields for the Function resource
func (rm *resourceManager) setFunctionCodeSigningConfig(
	ctx context.Context,
	ko *svcapitypes.Function,
) (err error) {
	rlog := ackrtlog.FromContext(ctx)
	exit := rlog.Trace("rm.setFunctionCodeSigningConfig")
	defer exit(err)

	// Add nil checks for required fields
	if ko == nil || ko.Spec.Name == nil {
		return nil // Skip if required fields are not set
	}

	var getFunctionCodeSigningConfigOutput *svcsdk.GetFunctionCodeSigningConfigOutput
	getFunctionCodeSigningConfigOutput, err = rm.sdkapi.GetFunctionCodeSigningConfig(
		ctx,
		&svcsdk.GetFunctionCodeSigningConfigInput{
			FunctionName: ko.Spec.Name,
		},
	)
	rm.metrics.RecordAPICall("GET", "GetFunctionCodeSigningConfig", err)
	if err != nil {
		return err
	}

	if getFunctionCodeSigningConfigOutput != nil && getFunctionCodeSigningConfigOutput.CodeSigningConfigArn != nil {
		ko.Spec.CodeSigningConfigARN = getFunctionCodeSigningConfigOutput.CodeSigningConfigArn
	}

	return nil
}

func (rm *resourceManager) setFunctionEventInvokeConfigFromResponse(
	ko *svcapitypes.Function,
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
// for Function resource
func (rm *resourceManager) setFunctionEventInvokeConfig(
	ctx context.Context,
	ko *svcapitypes.Function,
) (err error) {
	rlog := ackrtlog.FromContext(ctx)
	exit := rlog.Trace("rm.setFunctionEventInvokeConfig")
	defer exit(err)

	var getFunctionEventInvokeConfigOutput *svcsdk.GetFunctionEventInvokeConfigOutput
	getFunctionEventInvokeConfigOutput, err = rm.sdkapi.GetFunctionEventInvokeConfig(
		ctx,
		&svcsdk.GetFunctionEventInvokeConfigInput{
			FunctionName: ko.Spec.Name,
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

// setResourceAdditionalFields will describe the fields that are not return by
// API calls
func (rm *resourceManager) setResourceAdditionalFields(
	ctx context.Context,
	ko *svcapitypes.Function,
) (err error) {
	rlog := ackrtlog.FromContext(ctx)
	exit := rlog.Trace("rm.setResourceAdditionalFields")
	defer exit(err)

	// To set Function Concurrency for the function
	err = rm.setFunctionConcurrency(ctx, ko)
	if err != nil {
		return err
	}

	// To set Asynchronous Invocations for the function
	err = rm.setFunctionEventInvokeConfig(ctx, ko)
	if err != nil {
		return err
	}

	// To set Code Signing Config based on the PackageType for the function
	if ko.Spec.PackageType != nil && *ko.Spec.PackageType == "Zip" {
		err = rm.setFunctionCodeSigningConfig(ctx, ko)
		if err != nil {
			return err
		}
	}
	if ko.Spec.PackageType != nil && *ko.Spec.PackageType == "Image" &&
		ko.Spec.CodeSigningConfigARN != nil && *ko.Spec.CodeSigningConfigARN != "" {
		return ackerr.NewTerminalError(ErrCannotSetFunctionCSC)
	}

	return nil
}
