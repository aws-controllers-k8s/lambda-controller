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
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"reflect"
	"testing"

	svcapitypes "github.com/aws-controllers-k8s/lambda-controller/apis/v1alpha1"
	ackerr "github.com/aws-controllers-k8s/runtime/pkg/errors"
	ackmetrics "github.com/aws-controllers-k8s/runtime/pkg/metrics"
	"github.com/aws/aws-sdk-go-v2/aws"
	svcsdk "github.com/aws/aws-sdk-go-v2/service/lambda"
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

// fakeHTTPClient returns a canned HTTP response for every request, allowing us
// to drive the real svcsdk.Client (and thus the real
// setFunctionCodeSigningConfig code path) with a simulated AWS error response.
type fakeHTTPClient struct {
	statusCode int
	// errorType is returned in the X-Amzn-ErrorType header, which the AWS SDK
	// uses to populate the error code (e.g. "AccessDeniedException").
	errorType string
	// message is the JSON body's "message" field, surfaced as the error message.
	message string
}

func (f *fakeHTTPClient) Do(req *http.Request) (*http.Response, error) {
	body := `{"message":"` + f.message + `"}`
	header := http.Header{}
	header.Set("Content-Type", "application/json")
	if f.errorType != "" {
		header.Set("X-Amzn-ErrorType", f.errorType)
	}
	return &http.Response{
		StatusCode: f.statusCode,
		Header:     header,
		Body:       io.NopCloser(bytes.NewReader([]byte(body))),
	}, nil
}

// newTestResourceManager builds a resourceManager whose SDK client routes all
// requests through the supplied fake HTTP client.
func newTestResourceManager(httpClient *fakeHTTPClient) *resourceManager {
	sdkClient := svcsdk.New(svcsdk.Options{
		Region:      "us-west-2",
		Credentials: aws.AnonymousCredentials{},
		HTTPClient:  httpClient,
	})
	return &resourceManager{
		metrics: ackmetrics.NewMetrics("lambda"),
		sdkapi:  sdkClient,
	}
}

// Test_setFunctionCodeSigningConfig_errorHandling verifies how the controller
// classifies errors from GetFunctionCodeSigningConfig. In particular, a genuine
// IAM AccessDenied (which does NOT carry the "Unable to determine
// service/operation name to be authorized" message that regions without AWS
// Signer return) must NOT be converted into a terminal error.
func Test_setFunctionCodeSigningConfig_errorHandling(t *testing.T) {
	const regionUnsupportedMsg = "Unable to determine service/operation name to be authorized"
	const iamDeniedMsg = "User: arn:aws:iam::123456789012:role/example is not authorized to perform: lambda:GetFunctionCodeSigningConfig"

	tests := []struct {
		name       string
		httpClient *fakeHTTPClient
		// cscARN is the desired Spec.CodeSigningConfigARN
		cscARN *string
		// wantTerminal asserts the returned error is an ACK terminal error
		wantTerminal bool
		// wantErr asserts a (non-terminal) error is returned
		wantErr bool
	}{
		{
			name: "region without code signing, no CSC requested - suppressed",
			httpClient: &fakeHTTPClient{
				statusCode: 403,
				errorType:  "AccessDeniedException",
				message:    regionUnsupportedMsg,
			},
			cscARN:       nil,
			wantTerminal: false,
			wantErr:      false,
		},
		{
			name: "region without code signing, CSC requested - terminal",
			httpClient: &fakeHTTPClient{
				statusCode: 403,
				errorType:  "AccessDeniedException",
				message:    regionUnsupportedMsg,
			},
			cscARN:       aws.String("arn:aws:lambda:eu-central-2:123456789012:code-signing-config:csc-1"),
			wantTerminal: true,
			wantErr:      true,
		},
		{
			name: "genuine IAM AccessDenied, no CSC requested - not terminal",
			httpClient: &fakeHTTPClient{
				statusCode: 403,
				errorType:  "AccessDeniedException",
				message:    iamDeniedMsg,
			},
			cscARN:       nil,
			wantTerminal: false,
			wantErr:      true,
		},
		{
			name: "genuine IAM AccessDenied, CSC requested - not terminal",
			httpClient: &fakeHTTPClient{
				statusCode: 403,
				errorType:  "AccessDeniedException",
				message:    iamDeniedMsg,
			},
			cscARN:       aws.String("arn:aws:lambda:us-west-2:123456789012:code-signing-config:csc-1"),
			wantTerminal: false,
			wantErr:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rm := newTestResourceManager(tt.httpClient)
			ko := &svcapitypes.Function{}
			ko.Spec.Name = aws.String("test-function")
			ko.Spec.CodeSigningConfigARN = tt.cscARN

			err := rm.setFunctionCodeSigningConfig(context.Background(), ko)

			var terminalErr *ackerr.TerminalError
			gotTerminal := errors.As(err, &terminalErr)
			if gotTerminal != tt.wantTerminal {
				t.Errorf("setFunctionCodeSigningConfig() terminal = %v, want %v (err = %v)", gotTerminal, tt.wantTerminal, err)
			}
			if (err != nil) != tt.wantErr {
				t.Errorf("setFunctionCodeSigningConfig() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
