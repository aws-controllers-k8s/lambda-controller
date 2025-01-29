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

package v1alpha1

import (
	ackv1alpha1 "github.com/aws-controllers-k8s/runtime/apis/core/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// LayerVersionSpec defines the desired state of LayerVersion.
type LayerVersionSpec struct {

	// A list of compatible instruction set architectures (https://docs.aws.amazon.com/lambda/latest/dg/foundation-arch.html).
	CompatibleArchitectures []*string `json:"compatibleArchitectures,omitempty"`
	// A list of compatible function runtimes (https://docs.aws.amazon.com/lambda/latest/dg/lambda-runtimes.html).
	// Used for filtering with ListLayers and ListLayerVersions.
	//
	// The following list includes deprecated runtimes. For more information, see
	// Runtime deprecation policy (https://docs.aws.amazon.com/lambda/latest/dg/lambda-runtimes.html#runtime-support-policy).
	CompatibleRuntimes []*string `json:"compatibleRuntimes,omitempty"`
	// The function layer archive.
	// +kubebuilder:validation:Required
	Content *LayerVersionContentInput `json:"content"`
	// The description of the version.
	Description *string `json:"description,omitempty"`
	// The name or Amazon Resource Name (ARN) of the layer.
	// +kubebuilder:validation:Required
	LayerName *string `json:"layerName"`
	// The layer's software license. It can be any of the following:
	//
	//   - An SPDX license identifier (https://spdx.org/licenses/). For example,
	//     MIT.
	//
	//   - The URL of a license hosted on the internet. For example, https://opensource.org/licenses/MIT.
	//
	//   - The full text of the license.
	LicenseInfo *string `json:"licenseInfo,omitempty"`
}

// LayerVersionStatus defines the observed state of LayerVersion
type LayerVersionStatus struct {
	// All CRs managed by ACK have a common `Status.ACKResourceMetadata` member
	// that is used to contain resource sync state, account ownership,
	// constructed ARN for the resource
	// +kubebuilder:validation:Optional
	ACKResourceMetadata *ackv1alpha1.ResourceMetadata `json:"ackResourceMetadata"`
	// All CRS managed by ACK have a common `Status.Conditions` member that
	// contains a collection of `ackv1alpha1.Condition` objects that describe
	// the various terminal states of the CR and its backend AWS service API
	// resource
	// +kubebuilder:validation:Optional
	Conditions []*ackv1alpha1.Condition `json:"conditions"`
	// The date that the layer version was created, in ISO-8601 format (https://www.w3.org/TR/NOTE-datetime)
	// (YYYY-MM-DDThh:mm:ss.sTZD).
	// +kubebuilder:validation:Optional
	CreatedDate *string `json:"createdDate,omitempty"`
	// The ARN of the layer.
	// +kubebuilder:validation:Optional
	LayerARN *string `json:"layerARN,omitempty"`
	// The version number.
	// +kubebuilder:validation:Optional
	VersionNumber *int64 `json:"versionNumber,omitempty"`
}

// LayerVersion is the Schema for the LayerVersions API
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
type LayerVersion struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              LayerVersionSpec   `json:"spec,omitempty"`
	Status            LayerVersionStatus `json:"status,omitempty"`
}

// LayerVersionList contains a list of LayerVersion
// +kubebuilder:object:root=true
type LayerVersionList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []LayerVersion `json:"items"`
}

func init() {
	SchemeBuilder.Register(&LayerVersion{}, &LayerVersionList{})
}
