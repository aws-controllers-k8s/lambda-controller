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
	"encoding/json"
	"fmt"

	ackcompare "github.com/aws-controllers-k8s/runtime/pkg/compare"
	ackerr "github.com/aws-controllers-k8s/runtime/pkg/errors"
	ackrtlog "github.com/aws-controllers-k8s/runtime/pkg/runtime/log"
	"github.com/aws/aws-sdk-go-v2/aws"
	svcsdk "github.com/aws/aws-sdk-go-v2/service/lambda"
	"github.com/aws/aws-sdk-go-v2/service/lambda/types"
	svcsdktypes "github.com/aws/aws-sdk-go-v2/service/lambda/types"
	"github.com/micahhausler/aws-iam-policy/policy"

	svcapitypes "github.com/aws-controllers-k8s/lambda-controller/apis/v1alpha1"
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

// permissionEqual compares two AddPermissionInput structs to check if they're functionally equivalent
func permissionEqual(a, b *svcapitypes.AddPermissionInput) bool {
	if a == nil || b == nil {
		return a == b
	}
	if !stringPtrEquals(a.StatementID, b.StatementID) {
		return false
	}

	if !stringPtrEquals(a.Action, b.Action) ||
		!stringPtrEquals(a.Principal, b.Principal) ||
		!stringPtrEquals(a.SourceARN, b.SourceARN) ||
		!stringPtrEquals(a.SourceAccount, b.SourceAccount) ||
		!stringPtrEquals(a.EventSourceToken, b.EventSourceToken) ||
		!stringPtrEquals(a.PrincipalOrgID, b.PrincipalOrgID) {
		return false
	}

	return true
}

// Helper function to compare string pointers
func stringPtrEquals(a, b *string) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	return *a == *b
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

	// Set the permissions for the alias
	err = rm.setPermissions(ctx, ko)
	if err != nil {
		return err
	}

	return nil
}

func (rm *resourceManager) setPermissions(ctx context.Context, ko *svcapitypes.Alias) error {
	rlog := ackrtlog.FromContext(ctx)
	exit := rlog.Trace("rm.setPermissions")
	var err error
	defer func() { exit(err) }()

	functionName := fmt.Sprintf("%s:%s", *ko.Spec.FunctionName, *ko.Spec.Name)
	// get the policy for the function using the alias name as the qualifier. now we don't
	// have to worry about function versions..
	input := &svcsdk.GetPolicyInput{
		FunctionName: aws.String(functionName),
	}

	output, err := rm.sdkapi.GetPolicy(ctx, input)
	rm.metrics.RecordAPICall("GET", "GetPolicy", err)
	if err != nil {
		// Yes, believe it or not, the API returns a ResourceNotFoundException if the policy is empty
		// so we need to handle this case.
		if awsErr, ok := ackerr.AWSError(err); ok && awsErr.ErrorCode() == "ResourceNotFoundException" {
			ko.Spec.Permissions = []*svcapitypes.AddPermissionInput{}
			// set err to nil so we don't log an error
			err = nil
			return nil
		}
		return err
	}

	policyDoc := &policy.Policy{}
	err = json.Unmarshal([]byte(*output.Policy), policyDoc)
	if err != nil {
		return err
	}

	// Convert policy statements to permissions
	var permissions []*svcapitypes.AddPermissionInput
	if policyDoc.Statements != nil {
		for _, stmt := range policyDoc.Statements.Values() {
			if stmt.Sid == "" { // skip empty SID statements
				continue
			}

			permission := &svcapitypes.AddPermissionInput{
				StatementID: aws.String(stmt.Sid),
			}

			if stmt.Action != nil && len(stmt.Action.Values()) > 0 {
				permission.Action = aws.String(stmt.Action.Values()[0])
			}

			if stmt.Principal != nil {
				if stmt.Principal.Service() != nil && len(stmt.Principal.Service().Values()) > 0 {
					permission.Principal = aws.String(stmt.Principal.Service().Values()[0])
				} else if stmt.Principal.AWS() != nil && len(stmt.Principal.AWS().Values()) > 0 {
					permission.Principal = aws.String(stmt.Principal.AWS().Values()[0])
				}
			}

			if stmt.Condition != nil {
				// ArnLike condition
				if arnCond, ok := stmt.Condition["ArnLike"]; ok {
					if sourceArn, ok := arnCond["AWS:SourceArn"]; ok {
						strValues, _, _ := sourceArn.Values()
						if len(strValues) > 0 {
							permission.SourceARN = aws.String(strValues[0])
						}
					}
				}

				// StringEquals condition
				if stringCond, ok := stmt.Condition["StringEquals"]; ok {
					if sourceAcct, ok := stringCond["AWS:SourceAccount"]; ok {
						strValues, _, _ := sourceAcct.Values()
						if len(strValues) > 0 {
							permission.SourceAccount = aws.String(strValues[0])
						}
					}

					// Extract EventSourceToken
					if token, ok := stringCond["lambda:EventSourceToken"]; ok {
						strValues, _, _ := token.Values()
						if len(strValues) > 0 {
							permission.EventSourceToken = aws.String(strValues[0])
						}
					}

					// Extract PrincipalOrgID
					if orgID, ok := stringCond["aws:PrincipalOrgID"]; ok {
						strValues, _, _ := orgID.Values()
						if len(strValues) > 0 {
							permission.PrincipalOrgID = aws.String(strValues[0])
						}
					}
				}
			}

			permissions = append(permissions, permission)
		}
	}

	ko.Spec.Permissions = permissions
	return nil
}

// compares the desired and latest permissions and  returns two slices:
// permissions to remove and permissions to add. Updates are represented
// as a permission to remove and a permission to add.
//
// Yes the API doesn't support updating permissions directly... yikes.
func comparePermissions(
	desired []*svcapitypes.AddPermissionInput,
	latest []*svcapitypes.AddPermissionInput,
) (toRemove []*svcapitypes.AddPermissionInput, toAdd []*svcapitypes.AddPermissionInput) {
	// create maps for fast lookup by StatementID
	latestMap := make(map[string]*svcapitypes.AddPermissionInput)
	for _, p := range latest {
		if p.StatementID != nil {
			latestMap[*p.StatementID] = p
		}
	}
	desiredMap := make(map[string]*svcapitypes.AddPermissionInput)
	for _, p := range desired {
		if p.StatementID != nil {
			desiredMap[*p.StatementID] = p
		}
	}

	// Find permissions to add or update
	for statementID, desiredPermission := range desiredMap {
		latestPerm, exists := latestMap[statementID]
		if !exists {
			toAdd = append(toAdd, desiredPermission)
		} else if !permissionEqual(desiredPermission, latestPerm) {
			// Permission exists but needs update (remove then add)
			toRemove = append(toRemove, latestPerm)
			toAdd = append(toAdd, desiredPermission)
		}
	}

	// Find permissions to remove
	for statementID, latestPerm := range latestMap {
		if _, exists := desiredMap[statementID]; !exists {
			toRemove = append(toRemove, latestPerm)
		}
	}

	return toRemove, toAdd
}

// syncPermissions examines the permissions in the desired and latest resources
// and calls the AddPermission and RemovePermission APIs to ensure that the set
// of permissions stays in sync with the desired state.
func (rm *resourceManager) syncPermissions(
	ctx context.Context,
	desired *resource,
	latest *resource,
) (err error) {
	rlog := ackrtlog.FromContext(ctx)
	exit := rlog.Trace("rm.syncPermissions")
	defer func() { exit(err) }()

	toRemove, toAdd := comparePermissions(desired.ko.Spec.Permissions, latest.ko.Spec.Permissions)

	// Process removals first to avoid conflicts
	for _, p := range toRemove {
		if p.StatementID == nil {
			continue
		}
		rlog.Debug("removing permission", "statement_id", p.StatementID)
		if err = rm.removePermission(ctx, desired, p.StatementID); err != nil {
			return err
		}
	}

	// Then process additions
	for _, p := range toAdd {
		if p.StatementID == nil {
			continue
		}
		rlog.Debug("adding permission", "statement_id", p.StatementID)
		if err = rm.addPermission(ctx, desired, p); err != nil {
			return err
		}
	}

	return nil
}

// removePermission removes a permission from the Lambda alias
func (rm *resourceManager) removePermission(
	ctx context.Context,
	r *resource,
	statementID *string,
) error {
	rlog := ackrtlog.FromContext(ctx)
	exit := rlog.Trace("rm.removePermission")
	var err error
	defer func() { exit(err) }()

	input := &svcsdk.RemovePermissionInput{
		FunctionName: r.ko.Spec.FunctionName,
		Qualifier:    r.ko.Spec.Name,
		StatementId:  statementID,
	}

	_, err = rm.sdkapi.RemovePermission(ctx, input)
	rm.metrics.RecordAPICall("DELETE", "RemovePermission", err)
	return err
}

// addPermission adds a permission to the Lambda alias
func (rm *resourceManager) addPermission(
	ctx context.Context,
	r *resource,
	permission *svcapitypes.AddPermissionInput,
) error {
	rlog := ackrtlog.FromContext(ctx)
	exit := rlog.Trace("rm.addPermission")
	var err error
	defer func() { exit(err) }()

	functionName := fmt.Sprintf("%s:%s", *r.ko.Spec.FunctionName, *r.ko.Spec.Name)
	input := &svcsdk.AddPermissionInput{
		FunctionName:     aws.String(functionName),
		Qualifier:        nil, // We do not need to set the qualifier for aliases
		Action:           permission.Action,
		Principal:        permission.Principal,
		SourceAccount:    permission.SourceAccount,
		SourceArn:        permission.SourceARN,
		StatementId:      permission.StatementID,
		EventSourceToken: permission.EventSourceToken,
		PrincipalOrgID:   permission.PrincipalOrgID,
		RevisionId:       nil, // Avoid setting revisionId, since they are mainly related to functions.

	}
	if permission.FunctionURLAuthType != nil {
		input.FunctionUrlAuthType = types.FunctionUrlAuthType(*permission.FunctionURLAuthType)
	}

	_, err = rm.sdkapi.AddPermission(ctx, input)
	rm.metrics.RecordAPICall("PUT", "AddPermission", err)
	return err
}

func customPreCompare(
	delta *ackcompare.Delta,
	a *resource,
	b *resource,
) {
	if len(a.ko.Spec.Permissions) != len(b.ko.Spec.Permissions) {
		delta.Add("Spec.Permissions", a.ko.Spec.Permissions, b.ko.Spec.Permissions)
	} else if len(a.ko.Spec.Permissions) > 0 {
		toAdd, toRemove := comparePermissions(a.ko.Spec.Permissions, b.ko.Spec.Permissions)
		if len(toAdd) > 0 || len(toRemove) > 0 {
			delta.Add("Spec.Permissions", a.ko.Spec.Permissions, b.ko.Spec.Permissions)
		}
	}
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
