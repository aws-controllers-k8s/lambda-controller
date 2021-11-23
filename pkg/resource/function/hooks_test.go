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
	"reflect"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
)

func Test_compareMaps(t *testing.T) {
	type args struct {
		a map[string]*string
		b map[string]*string
	}
	tests := []struct {
		name        string
		args        args
		wantAdded   map[string]*string
		wantRemoved []string
		wantUpdated map[string]*string
	}{
		{
			name: "empty maps",
			args: args{
				a: map[string]*string{},
				b: map[string]*string{},
			},
			wantAdded:   map[string]*string{},
			wantRemoved: nil,
			wantUpdated: map[string]*string{},
		},
		{
			name: "new elements",
			args: args{
				a: map[string]*string{},
				b: map[string]*string{"k1": aws.String("v1")},
			},
			wantAdded:   map[string]*string{"k1": aws.String("v1")},
			wantRemoved: nil,
			wantUpdated: map[string]*string{},
		},
		{
			name: "updated elements",
			args: args{
				a: map[string]*string{"k1": aws.String("v1"), "k2": aws.String("v2")},
				b: map[string]*string{"k1": aws.String("v10"), "k2": aws.String("v20")},
			},
			wantAdded:   map[string]*string{},
			wantRemoved: nil,
			wantUpdated: map[string]*string{"k1": aws.String("v10"), "k2": aws.String("v20")},
		},
		{
			name: "removed elements",
			args: args{
				a: map[string]*string{"k1": aws.String("v1"), "k2": aws.String("v2")},
				b: map[string]*string{"k1": aws.String("v1")},
			},
			wantAdded:   map[string]*string{},
			wantRemoved: []string{"k2"},
			wantUpdated: map[string]*string{},
		},
		{
			name: "added, updated and removed elements",
			args: args{
				a: map[string]*string{"k1": aws.String("v1"), "k2": aws.String("v2")},
				b: map[string]*string{"k1": aws.String("v10"), "k3": aws.String("v3")},
			},
			wantAdded:   map[string]*string{"k3": aws.String("v3")},
			wantRemoved: []string{"k2"},
			wantUpdated: map[string]*string{"k1": aws.String("v10")},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotAdded, gotRemoved, gotUpdated := compareMaps(tt.args.a, tt.args.b)
			if !reflect.DeepEqual(gotAdded, tt.wantAdded) {
				t.Errorf("compareMaps() gotAdded = %v, want %v", gotAdded, tt.wantAdded)
			}
			if !reflect.DeepEqual(gotRemoved, tt.wantRemoved) {
				t.Errorf("compareMaps() gotRemoved = %v, want %v", gotRemoved, tt.wantRemoved)
			}
			if !reflect.DeepEqual(gotUpdated, tt.wantUpdated) {
				t.Errorf("compareMaps() gotUpdated = %v, want %v", gotUpdated, tt.wantUpdated)
			}
		})
	}
}
