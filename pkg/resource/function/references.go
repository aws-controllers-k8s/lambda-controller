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
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	ec2apitypes "github.com/aws-controllers-k8s/ec2-controller/apis/v1alpha1"
	iamapitypes "github.com/aws-controllers-k8s/iam-controller/apis/v1alpha1"
	kmsapitypes "github.com/aws-controllers-k8s/kms-controller/apis/v1alpha1"
	ackv1alpha1 "github.com/aws-controllers-k8s/runtime/apis/core/v1alpha1"
	ackerr "github.com/aws-controllers-k8s/runtime/pkg/errors"
	acktypes "github.com/aws-controllers-k8s/runtime/pkg/types"
	s3apitypes "github.com/aws-controllers-k8s/s3-controller/apis/v1alpha1"

	svcapitypes "github.com/aws-controllers-k8s/lambda-controller/apis/v1alpha1"
)

// +kubebuilder:rbac:groups=s3.services.k8s.aws,resources=buckets,verbs=get;list
// +kubebuilder:rbac:groups=s3.services.k8s.aws,resources=buckets/status,verbs=get;list

// +kubebuilder:rbac:groups=kms.services.k8s.aws,resources=keys,verbs=get;list
// +kubebuilder:rbac:groups=kms.services.k8s.aws,resources=keys/status,verbs=get;list

// +kubebuilder:rbac:groups=iam.services.k8s.aws,resources=roles,verbs=get;list
// +kubebuilder:rbac:groups=iam.services.k8s.aws,resources=roles/status,verbs=get;list

// +kubebuilder:rbac:groups=ec2.services.k8s.aws,resources=securitygroups,verbs=get;list
// +kubebuilder:rbac:groups=ec2.services.k8s.aws,resources=securitygroups/status,verbs=get;list

// +kubebuilder:rbac:groups=ec2.services.k8s.aws,resources=subnets,verbs=get;list
// +kubebuilder:rbac:groups=ec2.services.k8s.aws,resources=subnets/status,verbs=get;list

// ClearResolvedReferences removes any reference values that were made
// concrete in the spec. It returns a copy of the input AWSResource which
// contains the original *Ref values, but none of their respective concrete
// values.
func (rm *resourceManager) ClearResolvedReferences(res acktypes.AWSResource) acktypes.AWSResource {
	ko := rm.concreteResource(res).ko.DeepCopy()

	if ko.Spec.Code != nil {
		if ko.Spec.Code.S3BucketRef != nil {
			ko.Spec.Code.S3Bucket = nil
		}
	}

	if ko.Spec.KMSKeyRef != nil {
		ko.Spec.KMSKeyARN = nil
	}

	if ko.Spec.RoleRef != nil {
		ko.Spec.Role = nil
	}

	if ko.Spec.VPCConfig != nil {
		if len(ko.Spec.VPCConfig.SecurityGroupRefs) > 0 {
			ko.Spec.VPCConfig.SecurityGroupIDs = nil
		}
	}

	if ko.Spec.VPCConfig != nil {
		if len(ko.Spec.VPCConfig.SubnetRefs) > 0 {
			ko.Spec.VPCConfig.SubnetIDs = nil
		}
	}

	return &resource{ko}
}

// ResolveReferences finds if there are any Reference field(s) present
// inside AWSResource passed in the parameter and attempts to resolve those
// reference field(s) into their respective target field(s). It returns a
// copy of the input AWSResource with resolved reference(s), a boolean which
// is set to true if the resource contains any references (regardless of if
// they are resolved successfully) and an error if the passed AWSResource's
// reference field(s) could not be resolved.
func (rm *resourceManager) ResolveReferences(
	ctx context.Context,
	apiReader client.Reader,
	res acktypes.AWSResource,
) (acktypes.AWSResource, bool, error) {
	namespace := res.MetaObject().GetNamespace()
	ko := rm.concreteResource(res).ko

	resourceHasReferences := false
	err := validateReferenceFields(ko)
	if fieldHasReferences, err := rm.resolveReferenceForCode_S3Bucket(ctx, apiReader, namespace, ko); err != nil {
		return &resource{ko}, (resourceHasReferences || fieldHasReferences), err
	} else {
		resourceHasReferences = resourceHasReferences || fieldHasReferences
	}

	if fieldHasReferences, err := rm.resolveReferenceForKMSKeyARN(ctx, apiReader, namespace, ko); err != nil {
		return &resource{ko}, (resourceHasReferences || fieldHasReferences), err
	} else {
		resourceHasReferences = resourceHasReferences || fieldHasReferences
	}

	if fieldHasReferences, err := rm.resolveReferenceForRole(ctx, apiReader, namespace, ko); err != nil {
		return &resource{ko}, (resourceHasReferences || fieldHasReferences), err
	} else {
		resourceHasReferences = resourceHasReferences || fieldHasReferences
	}

	if fieldHasReferences, err := rm.resolveReferenceForVPCConfig_SecurityGroupIDs(ctx, apiReader, namespace, ko); err != nil {
		return &resource{ko}, (resourceHasReferences || fieldHasReferences), err
	} else {
		resourceHasReferences = resourceHasReferences || fieldHasReferences
	}

	if fieldHasReferences, err := rm.resolveReferenceForVPCConfig_SubnetIDs(ctx, apiReader, namespace, ko); err != nil {
		return &resource{ko}, (resourceHasReferences || fieldHasReferences), err
	} else {
		resourceHasReferences = resourceHasReferences || fieldHasReferences
	}

	return &resource{ko}, resourceHasReferences, err
}

// validateReferenceFields validates the reference field and corresponding
// identifier field.
func validateReferenceFields(ko *svcapitypes.Function) error {

	if ko.Spec.Code != nil {
		if ko.Spec.Code.S3BucketRef != nil && ko.Spec.Code.S3Bucket != nil {
			return ackerr.ResourceReferenceAndIDNotSupportedFor("Code.S3Bucket", "Code.S3BucketRef")
		}
	}

	if ko.Spec.KMSKeyRef != nil && ko.Spec.KMSKeyARN != nil {
		return ackerr.ResourceReferenceAndIDNotSupportedFor("KMSKeyARN", "KMSKeyRef")
	}

	if ko.Spec.RoleRef != nil && ko.Spec.Role != nil {
		return ackerr.ResourceReferenceAndIDNotSupportedFor("Role", "RoleRef")
	}
	if ko.Spec.RoleRef == nil && ko.Spec.Role == nil {
		return ackerr.ResourceReferenceOrIDRequiredFor("Role", "RoleRef")
	}

	if ko.Spec.VPCConfig != nil {
		if len(ko.Spec.VPCConfig.SecurityGroupRefs) > 0 && len(ko.Spec.VPCConfig.SecurityGroupIDs) > 0 {
			return ackerr.ResourceReferenceAndIDNotSupportedFor("VPCConfig.SecurityGroupIDs", "VPCConfig.SecurityGroupRefs")
		}
	}

	if ko.Spec.VPCConfig != nil {
		if len(ko.Spec.VPCConfig.SubnetRefs) > 0 && len(ko.Spec.VPCConfig.SubnetIDs) > 0 {
			return ackerr.ResourceReferenceAndIDNotSupportedFor("VPCConfig.SubnetIDs", "VPCConfig.SubnetRefs")
		}
	}
	return nil
}

// resolveReferenceForCode_S3Bucket reads the resource referenced
// from Code.S3BucketRef field and sets the Code.S3Bucket
// from referenced resource. Returns a boolean indicating whether a reference
// contains references, or an error
func (rm *resourceManager) resolveReferenceForCode_S3Bucket(
	ctx context.Context,
	apiReader client.Reader,
	namespace string,
	ko *svcapitypes.Function,
) (hasReferences bool, err error) {
	if ko.Spec.Code != nil {
		if ko.Spec.Code.S3BucketRef != nil && ko.Spec.Code.S3BucketRef.From != nil {
			hasReferences = true
			arr := ko.Spec.Code.S3BucketRef.From
			if arr.Name == nil || *arr.Name == "" {
				return hasReferences, fmt.Errorf("provided resource reference is nil or empty: Code.S3BucketRef")
			}
			obj := &s3apitypes.Bucket{}
			if err := getReferencedResourceState_Bucket(ctx, apiReader, obj, *arr.Name, namespace); err != nil {
				return hasReferences, err
			}
			ko.Spec.Code.S3Bucket = (*string)(obj.Spec.Name)
		}
	}

	return hasReferences, nil
}

// getReferencedResourceState_Bucket looks up whether a referenced resource
// exists and is in a ACK.ResourceSynced=True state. If the referenced resource does exist and is
// in a Synced state, returns nil, otherwise returns `ackerr.ResourceReferenceTerminalFor` or
// `ResourceReferenceNotSyncedFor` depending on if the resource is in a Terminal state.
func getReferencedResourceState_Bucket(
	ctx context.Context,
	apiReader client.Reader,
	obj *s3apitypes.Bucket,
	name string, // the Kubernetes name of the referenced resource
	namespace string, // the Kubernetes namespace of the referenced resource
) error {
	namespacedName := types.NamespacedName{
		Namespace: namespace,
		Name:      name,
	}
	err := apiReader.Get(ctx, namespacedName, obj)
	if err != nil {
		return err
	}
	var refResourceSynced, refResourceTerminal bool
	for _, cond := range obj.Status.Conditions {
		if cond.Type == ackv1alpha1.ConditionTypeResourceSynced &&
			cond.Status == corev1.ConditionTrue {
			refResourceSynced = true
		}
		if cond.Type == ackv1alpha1.ConditionTypeTerminal &&
			cond.Status == corev1.ConditionTrue {
			return ackerr.ResourceReferenceTerminalFor(
				"Bucket",
				namespace, name)
		}
	}
	if refResourceTerminal {
		return ackerr.ResourceReferenceTerminalFor(
			"Bucket",
			namespace, name)
	}
	if !refResourceSynced {
		return ackerr.ResourceReferenceNotSyncedFor(
			"Bucket",
			namespace, name)
	}
	if obj.Spec.Name == nil {
		return ackerr.ResourceReferenceMissingTargetFieldFor(
			"Bucket",
			namespace, name,
			"Spec.Name")
	}
	return nil
}

// resolveReferenceForKMSKeyARN reads the resource referenced
// from KMSKeyRef field and sets the KMSKeyARN
// from referenced resource. Returns a boolean indicating whether a reference
// contains references, or an error
func (rm *resourceManager) resolveReferenceForKMSKeyARN(
	ctx context.Context,
	apiReader client.Reader,
	namespace string,
	ko *svcapitypes.Function,
) (hasReferences bool, err error) {
	if ko.Spec.KMSKeyRef != nil && ko.Spec.KMSKeyRef.From != nil {
		hasReferences = true
		arr := ko.Spec.KMSKeyRef.From
		if arr.Name == nil || *arr.Name == "" {
			return hasReferences, fmt.Errorf("provided resource reference is nil or empty: KMSKeyRef")
		}
		obj := &kmsapitypes.Key{}
		if err := getReferencedResourceState_Key(ctx, apiReader, obj, *arr.Name, namespace); err != nil {
			return hasReferences, err
		}
		ko.Spec.KMSKeyARN = (*string)(obj.Status.ACKResourceMetadata.ARN)
	}

	return hasReferences, nil
}

// getReferencedResourceState_Key looks up whether a referenced resource
// exists and is in a ACK.ResourceSynced=True state. If the referenced resource does exist and is
// in a Synced state, returns nil, otherwise returns `ackerr.ResourceReferenceTerminalFor` or
// `ResourceReferenceNotSyncedFor` depending on if the resource is in a Terminal state.
func getReferencedResourceState_Key(
	ctx context.Context,
	apiReader client.Reader,
	obj *kmsapitypes.Key,
	name string, // the Kubernetes name of the referenced resource
	namespace string, // the Kubernetes namespace of the referenced resource
) error {
	namespacedName := types.NamespacedName{
		Namespace: namespace,
		Name:      name,
	}
	err := apiReader.Get(ctx, namespacedName, obj)
	if err != nil {
		return err
	}
	var refResourceSynced, refResourceTerminal bool
	for _, cond := range obj.Status.Conditions {
		if cond.Type == ackv1alpha1.ConditionTypeResourceSynced &&
			cond.Status == corev1.ConditionTrue {
			refResourceSynced = true
		}
		if cond.Type == ackv1alpha1.ConditionTypeTerminal &&
			cond.Status == corev1.ConditionTrue {
			return ackerr.ResourceReferenceTerminalFor(
				"Key",
				namespace, name)
		}
	}
	if refResourceTerminal {
		return ackerr.ResourceReferenceTerminalFor(
			"Key",
			namespace, name)
	}
	if !refResourceSynced {
		return ackerr.ResourceReferenceNotSyncedFor(
			"Key",
			namespace, name)
	}
	if obj.Status.ACKResourceMetadata == nil || obj.Status.ACKResourceMetadata.ARN == nil {
		return ackerr.ResourceReferenceMissingTargetFieldFor(
			"Key",
			namespace, name,
			"Status.ACKResourceMetadata.ARN")
	}
	return nil
}

// resolveReferenceForRole reads the resource referenced
// from RoleRef field and sets the Role
// from referenced resource. Returns a boolean indicating whether a reference
// contains references, or an error
func (rm *resourceManager) resolveReferenceForRole(
	ctx context.Context,
	apiReader client.Reader,
	namespace string,
	ko *svcapitypes.Function,
) (hasReferences bool, err error) {
	if ko.Spec.RoleRef != nil && ko.Spec.RoleRef.From != nil {
		hasReferences = true
		arr := ko.Spec.RoleRef.From
		if arr.Name == nil || *arr.Name == "" {
			return hasReferences, fmt.Errorf("provided resource reference is nil or empty: RoleRef")
		}
		obj := &iamapitypes.Role{}
		if err := getReferencedResourceState_Role(ctx, apiReader, obj, *arr.Name, namespace); err != nil {
			return hasReferences, err
		}
		ko.Spec.Role = (*string)(obj.Status.ACKResourceMetadata.ARN)
	}

	return hasReferences, nil
}

// getReferencedResourceState_Role looks up whether a referenced resource
// exists and is in a ACK.ResourceSynced=True state. If the referenced resource does exist and is
// in a Synced state, returns nil, otherwise returns `ackerr.ResourceReferenceTerminalFor` or
// `ResourceReferenceNotSyncedFor` depending on if the resource is in a Terminal state.
func getReferencedResourceState_Role(
	ctx context.Context,
	apiReader client.Reader,
	obj *iamapitypes.Role,
	name string, // the Kubernetes name of the referenced resource
	namespace string, // the Kubernetes namespace of the referenced resource
) error {
	namespacedName := types.NamespacedName{
		Namespace: namespace,
		Name:      name,
	}
	err := apiReader.Get(ctx, namespacedName, obj)
	if err != nil {
		return err
	}
	var refResourceSynced, refResourceTerminal bool
	for _, cond := range obj.Status.Conditions {
		if cond.Type == ackv1alpha1.ConditionTypeResourceSynced &&
			cond.Status == corev1.ConditionTrue {
			refResourceSynced = true
		}
		if cond.Type == ackv1alpha1.ConditionTypeTerminal &&
			cond.Status == corev1.ConditionTrue {
			return ackerr.ResourceReferenceTerminalFor(
				"Role",
				namespace, name)
		}
	}
	if refResourceTerminal {
		return ackerr.ResourceReferenceTerminalFor(
			"Role",
			namespace, name)
	}
	if !refResourceSynced {
		return ackerr.ResourceReferenceNotSyncedFor(
			"Role",
			namespace, name)
	}
	if obj.Status.ACKResourceMetadata == nil || obj.Status.ACKResourceMetadata.ARN == nil {
		return ackerr.ResourceReferenceMissingTargetFieldFor(
			"Role",
			namespace, name,
			"Status.ACKResourceMetadata.ARN")
	}
	return nil
}

// resolveReferenceForVPCConfig_SecurityGroupIDs reads the resource referenced
// from VPCConfig.SecurityGroupRefs field and sets the VPCConfig.SecurityGroupIDs
// from referenced resource. Returns a boolean indicating whether a reference
// contains references, or an error
func (rm *resourceManager) resolveReferenceForVPCConfig_SecurityGroupIDs(
	ctx context.Context,
	apiReader client.Reader,
	namespace string,
	ko *svcapitypes.Function,
) (hasReferences bool, err error) {
	if ko.Spec.VPCConfig != nil {
		for _, f0iter := range ko.Spec.VPCConfig.SecurityGroupRefs {
			if f0iter != nil && f0iter.From != nil {
				hasReferences = true
				arr := f0iter.From
				if arr.Name == nil || *arr.Name == "" {
					return hasReferences, fmt.Errorf("provided resource reference is nil or empty: VPCConfig.SecurityGroupRefs")
				}
				obj := &ec2apitypes.SecurityGroup{}
				if err := getReferencedResourceState_SecurityGroup(ctx, apiReader, obj, *arr.Name, namespace); err != nil {
					return hasReferences, err
				}
				if ko.Spec.VPCConfig.SecurityGroupIDs == nil {
					ko.Spec.VPCConfig.SecurityGroupIDs = make([]*string, 0, 1)
				}
				ko.Spec.VPCConfig.SecurityGroupIDs = append(ko.Spec.VPCConfig.SecurityGroupIDs, (*string)(obj.Status.ID))
			}
		}
	}

	return hasReferences, nil
}

// getReferencedResourceState_SecurityGroup looks up whether a referenced resource
// exists and is in a ACK.ResourceSynced=True state. If the referenced resource does exist and is
// in a Synced state, returns nil, otherwise returns `ackerr.ResourceReferenceTerminalFor` or
// `ResourceReferenceNotSyncedFor` depending on if the resource is in a Terminal state.
func getReferencedResourceState_SecurityGroup(
	ctx context.Context,
	apiReader client.Reader,
	obj *ec2apitypes.SecurityGroup,
	name string, // the Kubernetes name of the referenced resource
	namespace string, // the Kubernetes namespace of the referenced resource
) error {
	namespacedName := types.NamespacedName{
		Namespace: namespace,
		Name:      name,
	}
	err := apiReader.Get(ctx, namespacedName, obj)
	if err != nil {
		return err
	}
	var refResourceSynced, refResourceTerminal bool
	for _, cond := range obj.Status.Conditions {
		if cond.Type == ackv1alpha1.ConditionTypeResourceSynced &&
			cond.Status == corev1.ConditionTrue {
			refResourceSynced = true
		}
		if cond.Type == ackv1alpha1.ConditionTypeTerminal &&
			cond.Status == corev1.ConditionTrue {
			return ackerr.ResourceReferenceTerminalFor(
				"SecurityGroup",
				namespace, name)
		}
	}
	if refResourceTerminal {
		return ackerr.ResourceReferenceTerminalFor(
			"SecurityGroup",
			namespace, name)
	}
	if !refResourceSynced {
		return ackerr.ResourceReferenceNotSyncedFor(
			"SecurityGroup",
			namespace, name)
	}
	if obj.Status.ID == nil {
		return ackerr.ResourceReferenceMissingTargetFieldFor(
			"SecurityGroup",
			namespace, name,
			"Status.ID")
	}
	return nil
}

// resolveReferenceForVPCConfig_SubnetIDs reads the resource referenced
// from VPCConfig.SubnetRefs field and sets the VPCConfig.SubnetIDs
// from referenced resource. Returns a boolean indicating whether a reference
// contains references, or an error
func (rm *resourceManager) resolveReferenceForVPCConfig_SubnetIDs(
	ctx context.Context,
	apiReader client.Reader,
	namespace string,
	ko *svcapitypes.Function,
) (hasReferences bool, err error) {
	if ko.Spec.VPCConfig != nil {
		for _, f0iter := range ko.Spec.VPCConfig.SubnetRefs {
			if f0iter != nil && f0iter.From != nil {
				hasReferences = true
				arr := f0iter.From
				if arr.Name == nil || *arr.Name == "" {
					return hasReferences, fmt.Errorf("provided resource reference is nil or empty: VPCConfig.SubnetRefs")
				}
				obj := &ec2apitypes.Subnet{}
				if err := getReferencedResourceState_Subnet(ctx, apiReader, obj, *arr.Name, namespace); err != nil {
					return hasReferences, err
				}
				if ko.Spec.VPCConfig.SubnetIDs == nil {
					ko.Spec.VPCConfig.SubnetIDs = make([]*string, 0, 1)
				}
				ko.Spec.VPCConfig.SubnetIDs = append(ko.Spec.VPCConfig.SubnetIDs, (*string)(obj.Status.SubnetID))
			}
		}
	}

	return hasReferences, nil
}

// getReferencedResourceState_Subnet looks up whether a referenced resource
// exists and is in a ACK.ResourceSynced=True state. If the referenced resource does exist and is
// in a Synced state, returns nil, otherwise returns `ackerr.ResourceReferenceTerminalFor` or
// `ResourceReferenceNotSyncedFor` depending on if the resource is in a Terminal state.
func getReferencedResourceState_Subnet(
	ctx context.Context,
	apiReader client.Reader,
	obj *ec2apitypes.Subnet,
	name string, // the Kubernetes name of the referenced resource
	namespace string, // the Kubernetes namespace of the referenced resource
) error {
	namespacedName := types.NamespacedName{
		Namespace: namespace,
		Name:      name,
	}
	err := apiReader.Get(ctx, namespacedName, obj)
	if err != nil {
		return err
	}
	var refResourceSynced, refResourceTerminal bool
	for _, cond := range obj.Status.Conditions {
		if cond.Type == ackv1alpha1.ConditionTypeResourceSynced &&
			cond.Status == corev1.ConditionTrue {
			refResourceSynced = true
		}
		if cond.Type == ackv1alpha1.ConditionTypeTerminal &&
			cond.Status == corev1.ConditionTrue {
			return ackerr.ResourceReferenceTerminalFor(
				"Subnet",
				namespace, name)
		}
	}
	if refResourceTerminal {
		return ackerr.ResourceReferenceTerminalFor(
			"Subnet",
			namespace, name)
	}
	if !refResourceSynced {
		return ackerr.ResourceReferenceNotSyncedFor(
			"Subnet",
			namespace, name)
	}
	if obj.Status.SubnetID == nil {
		return ackerr.ResourceReferenceMissingTargetFieldFor(
			"Subnet",
			namespace, name,
			"Status.SubnetID")
	}
	return nil
}
