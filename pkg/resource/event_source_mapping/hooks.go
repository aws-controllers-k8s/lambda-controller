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
	ackcompare "github.com/aws-controllers-k8s/runtime/pkg/compare"

	"github.com/aws-controllers-k8s/lambda-controller/apis/v1alpha1"
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
