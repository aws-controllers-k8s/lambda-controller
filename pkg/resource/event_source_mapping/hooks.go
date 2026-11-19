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

package event_source_mapping

import (
	"context"

	ackcompare "github.com/aws-controllers-k8s/runtime/pkg/compare"
	svcsdktypes "github.com/aws/aws-sdk-go-v2/service/lambda/types"

	"github.com/aws-controllers-k8s/lambda-controller/apis/v1alpha1"
	acktags "github.com/aws-controllers-k8s/lambda-controller/pkg/resource/tags"
)

func customPreCompare(
	delta *ackcompare.Delta,
	a *resource,
	b *resource,
) {
	if ackcompare.HasNilDifference(a.ko.Spec.FilterCriteria, b.ko.Spec.FilterCriteria) {
		delta.Add("Spec.FilterCriteria", a.ko.Spec.FilterCriteria, b.ko.Spec.FilterCriteria)
	} else if a.ko.Spec.FilterCriteria != nil && b.ko.Spec.FilterCriteria != nil {
		if !equalFilterSlices(a.ko.Spec.FilterCriteria.Filters, b.ko.Spec.FilterCriteria.Filters) {
			delta.Add("Spec.FilterCriteria.Filters", a.ko.Spec.FilterCriteria, b.ko.Spec.FilterCriteria)
		}
	}
	compareDestinationConfig(
		delta,
		normalizeDestinationConfig(a.ko.Spec.DestinationConfig),
		normalizeDestinationConfig(b.ko.Spec.DestinationConfig),
	)
}

// normalizeDestinationConfig returns dc with empty OnFailure/OnSuccess
// entries (no Destination) removed, and returns nil if nothing is left.
//
// For stream event sources (DynamoDB, Kinesis, Kafka, ...) Lambda returns
// `DestinationConfig: {OnFailure: {}}` even when no destination was
// configured, and rejects that same value on Create/Update with
// "The Destination field is required for an OnFailure configuration".
// An empty entry carries no meaning, so we treat it as unset everywhere.
//
// The input is never mutated; a new struct is returned when something is
// stripped.
func normalizeDestinationConfig(
	dc *v1alpha1.DestinationConfig,
) *v1alpha1.DestinationConfig {
	if dc == nil {
		return nil
	}
	onFailure := dc.OnFailure
	if onFailure != nil && onFailure.Destination == nil {
		onFailure = nil
	}
	onSuccess := dc.OnSuccess
	if onSuccess != nil && onSuccess.Destination == nil {
		onSuccess = nil
	}
	if onFailure == nil && onSuccess == nil {
		return nil
	}
	if onFailure == dc.OnFailure && onSuccess == dc.OnSuccess {
		return dc
	}
	return &v1alpha1.DestinationConfig{
		OnFailure: onFailure,
		OnSuccess: onSuccess,
	}
}

// cleanEmptyDestinationConfig normalizes ko.Spec.DestinationConfig in place
// on an object built from an AWS API response, so that the empty
// `{onFailure: {}}` Lambda returns for stream sources is never written back
// into the CR spec.
func cleanEmptyDestinationConfig(ko *v1alpha1.EventSourceMapping) {
	ko.Spec.DestinationConfig = normalizeDestinationConfig(ko.Spec.DestinationConfig)
}

// cleanEmptyDestinationConfigInput is the SDK-side equivalent of
// normalizeDestinationConfig. It is applied to Create/Update request
// payloads so that a spec still holding `{onFailure: {}}` (persisted by
// older controller versions, or copied from `kubectl get -o yaml`) does not
// make Lambda reject the call.
func cleanEmptyDestinationConfigInput(
	dc *svcsdktypes.DestinationConfig,
) *svcsdktypes.DestinationConfig {
	if dc == nil {
		return nil
	}
	if dc.OnFailure != nil && dc.OnFailure.Destination == nil {
		dc.OnFailure = nil
	}
	if dc.OnSuccess != nil && dc.OnSuccess.Destination == nil {
		dc.OnSuccess = nil
	}
	if dc.OnFailure == nil && dc.OnSuccess == nil {
		return nil
	}
	return dc
}

// compareDestinationConfig compares two (already normalized)
// DestinationConfig values. Spec.DestinationConfig is marked as
// compare.is_ignored in generator.yaml so this replaces the generated
// comparison, which would otherwise report a difference between
// `{onFailure: {}}` and nil.
func compareDestinationConfig(
	delta *ackcompare.Delta,
	a *v1alpha1.DestinationConfig,
	b *v1alpha1.DestinationConfig,
) {
	if ackcompare.HasNilDifference(a, b) {
		delta.Add("Spec.DestinationConfig", a, b)
		return
	}
	if a == nil {
		return
	}
	if ackcompare.HasNilDifference(a.OnFailure, b.OnFailure) {
		delta.Add("Spec.DestinationConfig.OnFailure", a.OnFailure, b.OnFailure)
	} else if a.OnFailure != nil &&
		!equalStrings(a.OnFailure.Destination, b.OnFailure.Destination) {
		delta.Add("Spec.DestinationConfig.OnFailure.Destination", a.OnFailure.Destination, b.OnFailure.Destination)
	}
	if ackcompare.HasNilDifference(a.OnSuccess, b.OnSuccess) {
		delta.Add("Spec.DestinationConfig.OnSuccess", a.OnSuccess, b.OnSuccess)
	} else if a.OnSuccess != nil &&
		!equalStrings(a.OnSuccess.Destination, b.OnSuccess.Destination) {
		delta.Add("Spec.DestinationConfig.OnSuccess.Destination", a.OnSuccess.Destination, b.OnSuccess.Destination)
	}
}

// equalFilterSlices returns whether two Filter arrays are
// equal or not.
func equalFilterSlices(a, b []*v1alpha1.Filter) bool {
	if len(a) != len(b) {
		return false
	}

	// The Lambda control plane doesn't change the order of
	// submitted filters.
	for x, aVal := range a {
		bVal := b[x]
		if ackcompare.HasNilDifference(aVal, bVal) ||
			!equalStrings(aVal.Pattern, bVal.Pattern) {
			return false
		}
	}
	return true
}

// filterCriteriasDeleted return true if a user deleted the filter
// criterias by deleting the spec.filterCriteria field or the
// spec.filterCriteria.filters field, false otherwise.
//
// This function is used as a sdk_update_post_build_request hook, to
// properly build an update call that will delete ESM filters.
func filterCriteriasDeleted(
	observed *resource,
	desired *resource,
	delta *ackcompare.Delta,
) bool {
	if delta.DifferentAt("Spec.FilterCriteria") ||
		delta.DifferentAt("Spec.FilterCriteria.Filters") {
		// If the observed resource doesn't have any filters, nothing
		// has been deleted from the CR's filters.
		if observed.ko.Spec.FilterCriteria == nil ||
			len(observed.ko.Spec.FilterCriteria.Filters) == 0 {
			return false
		}
		// Observing that the resource have at least one non-nil filter
		// and the desired one have a nil `FilterCriteria` or nil
		// `FilterCriteria.Filters`, means that the user wants to delete
		// their filters.
		if desired.ko.Spec.FilterCriteria == nil ||
			len(desired.ko.Spec.FilterCriteria.Filters) == 0 {
			return true
		}
	}
	return false
}

func equalStrings(a, b *string) bool {
	if a == nil {
		return b == nil || *b == ""
	}
	return (*a == "" && b == nil) || *a == *b
}

func (rm *resourceManager) getTags(
	ctx context.Context,
	resourceARN string,
) (map[string]*string, error) {
	return acktags.GetTags(ctx, rm.sdkapi, rm.metrics, resourceARN)
}

func (rm *resourceManager) syncTags(
	ctx context.Context,
	desired *resource,
	latest *resource,
) error {
	return acktags.SyncTags(
		ctx, rm.sdkapi, rm.metrics,
		string(*latest.ko.Status.ACKResourceMetadata.ARN),
		desired.ko.Spec.Tags, latest.ko.Spec.Tags,
	)
}
