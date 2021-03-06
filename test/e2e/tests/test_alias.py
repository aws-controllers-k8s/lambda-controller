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

"""Integration tests for the Lambda alias API.
"""

import boto3
import pytest
import time
import logging
from typing import Dict, Tuple

from acktest.resources import random_suffix_name
from acktest.aws.identity import get_region
from acktest.k8s import resource as k8s
from e2e import service_marker, CRD_GROUP, CRD_VERSION, load_lambda_resource
from e2e.replacement_values import REPLACEMENT_VALUES
from e2e.bootstrap_resources import get_bootstrap_resources

RESOURCE_PLURAL = "aliases"

CREATE_WAIT_AFTER_SECONDS = 10
UPDATE_WAIT_AFTER_SECONDS = 10
DELETE_WAIT_AFTER_SECONDS = 10

@pytest.fixture(scope="module")
def lambda_client():
    return boto3.client("lambda")

@pytest.fixture(scope="module")
def lambda_function():
        resource_name = random_suffix_name("lambda-function", 24)
        resources = get_bootstrap_resources()

        replacements = REPLACEMENT_VALUES.copy()
        replacements["FUNCTION_NAME"] = resource_name
        replacements["BUCKET_NAME"] = resources.FunctionsBucketName
        replacements["LAMBDA_ROLE"] = resources.LambdaBasicRoleARN
        replacements["LAMBDA_FILE_NAME"] = resources.LambdaFunctionFileZip
        replacements["RESERVED_CONCURRENT_EXECUTIONS"] = "0"
        replacements["CODE_SIGNING_CONFIG_ARN"] = ""
        replacements["AWS_REGION"] = get_region()

        # Load function CR
        resource_data = load_lambda_resource(
            "function",
            additional_replacements=replacements,
        )
        logging.debug(resource_data)

        # Create k8s resource
        function_reference = k8s.CustomResourceReference(
            CRD_GROUP, CRD_VERSION, "functions",
            resource_name, namespace="default",
        )

        # Create lambda function
        k8s.create_custom_resource(function_reference, resource_data)
        function_resource = k8s.wait_resource_consumed_by_controller(function_reference)

        assert function_resource is not None
        assert k8s.get_resource_exists(function_reference)

        time.sleep(CREATE_WAIT_AFTER_SECONDS)

        yield (function_reference, function_resource)

        _, deleted = k8s.delete_custom_resource(function_reference)
        assert deleted

@service_marker
@pytest.mark.canary
class TestAlias:
    def get_alias(self, lambda_client, alias_name: str, function_name: str) -> dict:
        try:
            resp = lambda_client.get_alias(
                Name=alias_name,
                FunctionName=function_name
            )
            return resp

        except Exception as e:
            logging.debug(e)
            return None

    def alias_exist(self, lambda_client, alias_name: str, function_name: str) -> bool:
        return self.get_alias(lambda_client, alias_name, function_name) is not None

    def test_smoke(self, lambda_client, lambda_function):
        (_, function_resource) = lambda_function
        lambda_function_name = function_resource["spec"]["name"]

        resource_name = random_suffix_name("lambda-alias", 24)

        replacements = REPLACEMENT_VALUES.copy()
        replacements["AWS_REGION"] = get_region()
        replacements["ALIAS_NAME"] = resource_name
        replacements["FUNCTION_NAME"] = lambda_function_name
        replacements["FUNCTION_VERSION"] = "$LATEST"

        # Load alias CR
        resource_data = load_lambda_resource(
            "alias",
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

        # Check alias exists
        alias = self.alias_exist(lambda_client, resource_name, lambda_function_name)
        assert alias is not None

        # Update cr
        cr["spec"]["description"] = ""

        # Patch k8s resource
        k8s.patch_custom_resource(ref, cr)
        time.sleep(UPDATE_WAIT_AFTER_SECONDS)

        # Check alias description
        alias = self.get_alias(lambda_client, resource_name, lambda_function_name)
        assert alias is not None
        assert alias["Description"] == ""

        # Delete k8s resource
        _, deleted = k8s.delete_custom_resource(ref)
        assert deleted

        time.sleep(DELETE_WAIT_AFTER_SECONDS)

        # Check alias doesn't exist
        exists = self.get_alias(lambda_client, resource_name, lambda_function_name)
        assert not exists

