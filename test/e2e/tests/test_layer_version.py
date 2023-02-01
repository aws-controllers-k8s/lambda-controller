# Copyright Amazon.com Inc. or its affiliates. All Rights Reserved.
#
# Licensed under the Apache License, Version 2.0 (the "License"). You may
# not use this file except in compliance with the License. A copy of the
# License is located at
#
# 	 http://aws.amazon.com/apache2.0/
#
# or in the "license" file accompanying this file. This file is distributed
# on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either
# express or implied. See the License for the specific language governing
# permissions and limitations under the License.

"""Integration tests for the Lambda layer version API.
"""

import pytest
import time
import logging

from acktest.resources import random_suffix_name
from acktest.aws.identity import get_region
from acktest.k8s import resource as k8s

from e2e import service_marker, CRD_GROUP, CRD_VERSION, load_lambda_resource
from e2e.replacement_values import REPLACEMENT_VALUES
from e2e.bootstrap_resources import get_bootstrap_resources
from e2e.service_bootstrap import LAMBDA_FUNCTION_FILE_ZIP
from e2e.tests.helper import LambdaValidator

RESOURCE_PLURAL = "layerversions"

CREATE_WAIT_AFTER_SECONDS = 10
UPDATE_WAIT_AFTER_SECONDS = 10
DELETE_WAIT_AFTER_SECONDS = 10

@service_marker
@pytest.mark.canary
class TestLayerVersion:

    def test_smoke(self, lambda_client):
        resource_name = random_suffix_name("lambda-lv", 24)

        resources = get_bootstrap_resources()
        logging.debug(resources)

        replacements = REPLACEMENT_VALUES.copy()
        replacements["AWS_REGION"] = get_region()
        replacements["LAYER_VERSION"] = resource_name
        replacements["BUCKET_NAME"] = resources.FunctionsBucket.name
        replacements["LAMBDA_FILE_NAME"] = LAMBDA_FUNCTION_FILE_ZIP

        # Load Lambda CR
        resource_data = load_lambda_resource(
            "layer_version",
            additional_replacements=replacements,
        )
        logging.debug(resource_data)

        # Create k8s resource
        ref = k8s.CustomResourceReference(
            CRD_GROUP, CRD_VERSION, RESOURCE_PLURAL,
            resource_name, namespace="default",
        )
        k8s.create_custom_resource(ref, resource_data)
        cr = k8s.wait_resource_consumed_by_controller(ref)

        assert cr is not None
        assert k8s.get_resource_exists(ref)

        time.sleep(CREATE_WAIT_AFTER_SECONDS)

        cr = k8s.wait_resource_consumed_by_controller(ref)

        lambda_validator = LambdaValidator(lambda_client)

        version_number = cr['status']['versionNumber']

        # Check layer version exists
        assert lambda_validator.layer_version_exists(resource_name, version_number)

        # Update cr
        new_description = "new description"
        updates = {
            "spec": {
                "description": new_description
            },
        }

        #Patch k8s resource
        k8s.patch_custom_resource(ref, updates)
        time.sleep(UPDATE_WAIT_AFTER_SECONDS)

        cr = k8s.wait_resource_consumed_by_controller(ref)
        version_number = cr['status']['versionNumber']

        #Check layer version description
        layer_version = lambda_validator.get_layer_version(resource_name, version_number)
        assert layer_version is not None
        assert layer_version['Description'] == 'new description'

        # Delete k8s resource
        _, deleted = k8s.delete_custom_resource(ref)
        assert deleted is True

        # Check if all versions are deleted
        layer_name = cr['spec']['layerName']
        list = lambda_validator.list_layer_versions(layer_name)
        assert len(list["LayerVersions"]) == 0

        time.sleep(DELETE_WAIT_AFTER_SECONDS)

        # Check layer version doesn't exist
        assert not lambda_validator.layer_version_exists(resource_name, version_number)
