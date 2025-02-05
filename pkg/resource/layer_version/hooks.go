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

package layer_version

import (
	"context"
	"fmt"
	"sort"

	ackrtlog "github.com/aws-controllers-k8s/runtime/pkg/runtime/log"
	svcsdk "github.com/aws/aws-sdk-go-v2/service/lambda"
)

// customPreDelete deletes all the previous versions of a
// LayerVersion except the latest version
// This function is used as a sdk_delete hook, to delete all the previous versions of a LayerVersion when delete API call is made
func customPreDelete(
	r *resource,
	rm *resourceManager,
	ctx context.Context,
) error {
	// Getting the list of all the versions of a LayerVersion
	input := &svcsdk.ListLayerVersionsInput{
		LayerName: r.ko.Spec.LayerName,
	}
	response, err := rm.sdkapi.ListLayerVersions(ctx, input)
	if err != nil {
		return err
	}

	log := ackrtlog.FromContext(ctx)
	log.Debug("Deleting previous versions of LayerVersion")

	// The above API call returns output containing list of versions as LayerVersions and a pagination token as NextMarker

	// Extracting the list of versions and assigning it to a new variable
	versionList := response.LayerVersions

	// sorting the list in ascending order
	sort.Slice(versionList, func(i, j int) bool {
		return versionList[i].Version < versionList[j].Version
	})

	for i := 0; i < len(versionList)-1; i++ {
		input := &svcsdk.DeleteLayerVersionInput{
			LayerName:     r.ko.Spec.LayerName,
			VersionNumber: &versionList[i].Version,
		}
		// Delete API call to delete the versions one by one
		logMessage := fmt.Sprintf("Deleting version %v of %v", *input.VersionNumber, *input.LayerName)
		log.Debug(logMessage)

		_, err = rm.sdkapi.DeleteLayerVersion(ctx, input)
		rm.metrics.RecordAPICall("DELETE", "DeleteLayerVersion", err)
		if err != nil {
			return err
		}
	}

	return nil
}
