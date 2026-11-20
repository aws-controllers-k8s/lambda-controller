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
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	svcsdktypes "github.com/aws/aws-sdk-go-v2/service/lambda/types"

	svcapitypes "github.com/aws-controllers-k8s/lambda-controller/apis/v1alpha1"
)

const testQueueARN = "arn:aws:sqs:us-west-2:123456789012:dlq"

func esmWithDestinationConfig(dc *svcapitypes.DestinationConfig) *resource {
	return &resource{ko: &svcapitypes.EventSourceMapping{
		Spec: svcapitypes.EventSourceMappingSpec{
			FunctionName:      aws.String("fn"),
			DestinationConfig: dc,
		},
	}}
}

// pathString renders an ackcompare.Path as "A.B.C". Path does not expose its
// parts, so go through its JSON encoding.
func pathString(t *testing.T, p json.Marshaler) string {
	t.Helper()
	b, err := p.MarshalJSON()
	if err != nil {
		t.Fatalf("marshalling path: %v", err)
	}
	var v struct{ Parts []string }
	if err := json.Unmarshal(b, &v); err != nil {
		t.Fatalf("unmarshalling path: %v", err)
	}
	return strings.Join(v.Parts, ".")
}

func Test_normalizeDestinationConfig(t *testing.T) {
	tests := []struct {
		name string
		in   *svcapitypes.DestinationConfig
		want *svcapitypes.DestinationConfig
	}{
		{
			name: "nil",
			in:   nil,
			want: nil,
		},
		{
			name: "empty struct",
			in:   &svcapitypes.DestinationConfig{},
			want: nil,
		},
		{
			name: "empty OnFailure returned for stream sources",
			in: &svcapitypes.DestinationConfig{
				OnFailure: &svcapitypes.OnFailure{},
			},
			want: nil,
		},
		{
			name: "empty OnFailure and OnSuccess",
			in: &svcapitypes.DestinationConfig{
				OnFailure: &svcapitypes.OnFailure{},
				OnSuccess: &svcapitypes.OnSuccess{},
			},
			want: nil,
		},
		{
			name: "OnFailure with destination is kept",
			in: &svcapitypes.DestinationConfig{
				OnFailure: &svcapitypes.OnFailure{Destination: aws.String(testQueueARN)},
			},
			want: &svcapitypes.DestinationConfig{
				OnFailure: &svcapitypes.OnFailure{Destination: aws.String(testQueueARN)},
			},
		},
		{
			name: "OnSuccess with destination is kept, empty OnFailure dropped",
			in: &svcapitypes.DestinationConfig{
				OnFailure: &svcapitypes.OnFailure{},
				OnSuccess: &svcapitypes.OnSuccess{Destination: aws.String(testQueueARN)},
			},
			want: &svcapitypes.DestinationConfig{
				OnSuccess: &svcapitypes.OnSuccess{Destination: aws.String(testQueueARN)},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var before *svcapitypes.DestinationConfig
			if tt.in != nil {
				before = tt.in.DeepCopy()
			}
			got := normalizeDestinationConfig(tt.in)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("normalizeDestinationConfig() = %+v, want %+v", got, tt.want)
			}
			if !reflect.DeepEqual(tt.in, before) {
				t.Errorf("normalizeDestinationConfig() mutated its input: got %+v, want %+v", tt.in, before)
			}
		})
	}
}

func Test_cleanEmptyDestinationConfig(t *testing.T) {
	r := esmWithDestinationConfig(&svcapitypes.DestinationConfig{
		OnFailure: &svcapitypes.OnFailure{},
	})
	cleanEmptyDestinationConfig(r.ko)
	if r.ko.Spec.DestinationConfig != nil {
		t.Errorf("expected nil DestinationConfig, got %+v", r.ko.Spec.DestinationConfig)
	}

	r = esmWithDestinationConfig(&svcapitypes.DestinationConfig{
		OnFailure: &svcapitypes.OnFailure{Destination: aws.String(testQueueARN)},
	})
	cleanEmptyDestinationConfig(r.ko)
	if r.ko.Spec.DestinationConfig == nil ||
		r.ko.Spec.DestinationConfig.OnFailure == nil ||
		*r.ko.Spec.DestinationConfig.OnFailure.Destination != testQueueARN {
		t.Errorf("expected OnFailure destination to be kept, got %+v", r.ko.Spec.DestinationConfig)
	}
}

func Test_cleanEmptyDestinationConfigInput(t *testing.T) {
	tests := []struct {
		name string
		in   *svcsdktypes.DestinationConfig
		want *svcsdktypes.DestinationConfig
	}{
		{
			name: "nil",
			in:   nil,
			want: nil,
		},
		{
			name: "empty OnFailure is dropped",
			in: &svcsdktypes.DestinationConfig{
				OnFailure: &svcsdktypes.OnFailure{},
			},
			want: nil,
		},
		{
			name: "empty OnSuccess dropped, OnFailure kept",
			in: &svcsdktypes.DestinationConfig{
				OnFailure: &svcsdktypes.OnFailure{Destination: aws.String(testQueueARN)},
				OnSuccess: &svcsdktypes.OnSuccess{},
			},
			want: &svcsdktypes.DestinationConfig{
				OnFailure: &svcsdktypes.OnFailure{Destination: aws.String(testQueueARN)},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := cleanEmptyDestinationConfigInput(tt.in)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("cleanEmptyDestinationConfigInput() = %+v, want %+v", got, tt.want)
			}
		})
	}
}

func Test_newResourceDelta_DestinationConfig(t *testing.T) {
	tests := []struct {
		name      string
		desired   *svcapitypes.DestinationConfig
		latest    *svcapitypes.DestinationConfig
		wantDiffs []string
	}{
		{
			// A CR created by an older controller version has the empty
			// `onFailure: {}` Lambda returned for a stream source persisted in
			// its spec. The (cleaned) observed state is nil. This must not
			// produce a diff, otherwise the controller sends an invalid
			// UpdateEventSourceMapping call on every reconcile.
			name:    "upgrade: desired empty OnFailure vs latest nil",
			desired: &svcapitypes.DestinationConfig{OnFailure: &svcapitypes.OnFailure{}},
			latest:  nil,
		},
		{
			name:    "desired nil vs latest empty OnFailure",
			desired: nil,
			latest:  &svcapitypes.DestinationConfig{OnFailure: &svcapitypes.OnFailure{}},
		},
		{
			name:    "desired empty struct vs latest nil",
			desired: &svcapitypes.DestinationConfig{},
			latest:  nil,
		},
		{
			name:    "both nil",
			desired: nil,
			latest:  nil,
		},
		{
			name: "same destination",
			desired: &svcapitypes.DestinationConfig{
				OnFailure: &svcapitypes.OnFailure{Destination: aws.String(testQueueARN)},
			},
			latest: &svcapitypes.DestinationConfig{
				OnFailure: &svcapitypes.OnFailure{Destination: aws.String(testQueueARN)},
			},
		},
		{
			name: "destination added",
			desired: &svcapitypes.DestinationConfig{
				OnFailure: &svcapitypes.OnFailure{Destination: aws.String(testQueueARN)},
			},
			latest:    nil,
			wantDiffs: []string{"Spec.DestinationConfig"},
		},
		{
			name: "destination added next to an empty OnSuccess",
			desired: &svcapitypes.DestinationConfig{
				OnFailure: &svcapitypes.OnFailure{Destination: aws.String(testQueueARN)},
				OnSuccess: &svcapitypes.OnSuccess{},
			},
			latest:    &svcapitypes.DestinationConfig{OnFailure: &svcapitypes.OnFailure{}},
			wantDiffs: []string{"Spec.DestinationConfig"},
		},
		{
			name: "destination changed",
			desired: &svcapitypes.DestinationConfig{
				OnFailure: &svcapitypes.OnFailure{Destination: aws.String(testQueueARN)},
			},
			latest: &svcapitypes.DestinationConfig{
				OnFailure: &svcapitypes.OnFailure{Destination: aws.String(testQueueARN + "-old")},
			},
			wantDiffs: []string{"Spec.DestinationConfig.OnFailure.Destination"},
		},
		{
			name: "OnSuccess added while OnFailure unchanged",
			desired: &svcapitypes.DestinationConfig{
				OnFailure: &svcapitypes.OnFailure{Destination: aws.String(testQueueARN)},
				OnSuccess: &svcapitypes.OnSuccess{Destination: aws.String(testQueueARN)},
			},
			latest: &svcapitypes.DestinationConfig{
				OnFailure: &svcapitypes.OnFailure{Destination: aws.String(testQueueARN)},
			},
			wantDiffs: []string{"Spec.DestinationConfig.OnSuccess"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			desired := esmWithDestinationConfig(tt.desired)
			latest := esmWithDestinationConfig(tt.latest)
			delta := newResourceDelta(desired, latest)
			var got []string
			for _, d := range delta.Differences {
				got = append(got, pathString(t, d.Path))
			}
			if !reflect.DeepEqual(got, tt.wantDiffs) {
				t.Errorf("newResourceDelta() differences = %v, want %v", got, tt.wantDiffs)
			}
			// The comparison must not rewrite the desired spec.
			if !reflect.DeepEqual(desired.ko.Spec.DestinationConfig, tt.desired) {
				t.Errorf("newResourceDelta() mutated desired DestinationConfig: %+v", desired.ko.Spec.DestinationConfig)
			}
		})
	}
}
