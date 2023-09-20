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

// FunctionUrlConfigSpec defines the desired state of FunctionUrlConfig.
//
// Details about a Lambda function URL.
type FunctionURLConfigSpec struct {

// The type of authentication that your function URL uses. Set to AWS_IAM if
// you want to restrict access to authenticated IAM users only. Set to NONE
// if you want to bypass IAM authentication to create a public endpoint. For
// more information, see Security and auth model for Lambda function URLs (https://docs.aws.amazon.com/lambda/latest/dg/urls-auth.html).
// +kubebuilder:validation:Required
AuthType *string `json:"authType"`
// The cross-origin resource sharing (CORS) (https://developer.mozilla.org/en-US/docs/Web/HTTP/CORS)
// settings for your function URL.
CORS *CORS `json:"cors,omitempty"`
// The name of the Lambda function.
// 
// Name formats
// 
//    * Function name – my-function.
// 
//    * Function ARN – arn:aws:lambda:us-west-2:123456789012:function:my-function.
// 
//    * Partial ARN – 123456789012:function:my-function.
// 
// The length constraint applies only to the full ARN. If you specify only the
// function name, it is limited to 64 characters in length.
FunctionName *string `json:"functionName,omitempty"`
FunctionRef *ackv1alpha1.AWSResourceReferenceWrapper `json:"functionRef,omitempty"`
// The alias name.
Qualifier *string `json:"qualifier,omitempty"`
}

// FunctionURLConfigStatus defines the observed state of FunctionURLConfig
type FunctionURLConfigStatus struct {
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
	// When the function URL was created, in ISO-8601 format (https://www.w3.org/TR/NOTE-datetime)
// (YYYY-MM-DDThh:mm:ss.sTZD).
	// +kubebuilder:validation:Optional
	CreationTime *string `json:"creationTime,omitempty"`
	// The Amazon Resource Name (ARN) of your function.
	// +kubebuilder:validation:Optional
	FunctionARN *string `json:"functionARN,omitempty"`
	// The HTTP URL endpoint for your function.
	// +kubebuilder:validation:Optional
	FunctionURL *string `json:"functionURL,omitempty"`
}

// FunctionURLConfig is the Schema for the FunctionURLConfigs API
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
type FunctionURLConfig struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec   FunctionURLConfigSpec   `json:"spec,omitempty"`
	Status FunctionURLConfigStatus `json:"status,omitempty"`
}

// FunctionURLConfigList contains a list of FunctionURLConfig
// +kubebuilder:object:root=true
type FunctionURLConfigList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items []FunctionURLConfig `json:"items"`
}

func init() {
	SchemeBuilder.Register(&FunctionURLConfig{}, &FunctionURLConfigList{})
}
