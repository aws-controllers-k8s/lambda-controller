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

"""Integration tests for the Lambda function API.
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

RESOURCE_PLURAL = "functions"

CREATE_WAIT_AFTER_SECONDS = 10
UPDATE_WAIT_AFTER_SECONDS = 10
DELETE_WAIT_AFTER_SECONDS = 10

@pytest.fixture(scope="module")
def lambda_client():
    return boto3.client("lambda")

@service_marker
@pytest.mark.canary
class TestFunction:

    def get_function(self, lambda_client, function_name: str) -> dict:
        try:
            resp = lambda_client.get_function(
                FunctionName=function_name
            )
            return resp

        except Exception as e:
            logging.debug(e)
            return None

    def get_function_concurrency(self, lambda_client, function_name: str) -> int:
        try:
            resp = lambda_client.get_function_concurrency(
                FunctionName=function_name
            )
            return resp['ReservedConcurrentExecutions']

        except Exception as e:
            logging.debug(e)
            return None

    def function_exists(self, lambda_client, function_name: str) -> bool:
        return self.get_function(lambda_client, function_name) is not None

    def test_smoke(self, lambda_client):
        resource_name = random_suffix_name("lambda-function", 24)

        resources = get_bootstrap_resources()
        logging.debug(resources)

        replacements = REPLACEMENT_VALUES.copy()
        replacements["FUNCTION_NAME"] = resource_name
        replacements["BUCKET_NAME"] = resources.FunctionsBucketName
        replacements["LAMBDA_ROLE"] = resources.LambdaBasicRoleARN
        replacements["LAMBDA_FILE_NAME"] = resources.LambdaFunctionFileZip
        replacements["RESERVED_CONCURRENT_EXECUTIONS"] = "0"
        replacements["AWS_REGION"] = get_region()

        # Load Lambda CR
        resource_data = load_lambda_resource(
            "function",
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

        # Check Lambda function exists
        exists = self.function_exists(lambda_client, resource_name)
        assert exists

        # Update cr
        tags = {
            "v1": "k1",
            "v2": "k2",
            "v3": "k3",
        }
        cr["spec"]["description"] = "Updated description"
        cr["spec"]["tags"] = tags

        # Patch k8s resource
        k8s.patch_custom_resource(ref, cr)
        time.sleep(UPDATE_WAIT_AFTER_SECONDS)

        # Check function updated fields
        function = self.get_function(lambda_client, resource_name)
        assert function is not None
        assert function["Configuration"]["Description"] == "Updated description"
        assert function["Tags"] == tags

        # Delete k8s resource
        _, deleted = k8s.delete_custom_resource(ref)
        assert deleted is True

        time.sleep(DELETE_WAIT_AFTER_SECONDS)

        # Check Lambda function doesn't exist
        exists = self.function_exists(lambda_client, resource_name)
        assert not exists

    def test_reserved_concurrent_executions(self, lambda_client):
        resource_name = random_suffix_name("lambda-function", 24)

        resources = get_bootstrap_resources()
        logging.debug(resources)

        replacements = REPLACEMENT_VALUES.copy()
        replacements["FUNCTION_NAME"] = resource_name
        replacements["BUCKET_NAME"] = resources.FunctionsBucketName
        replacements["LAMBDA_ROLE"] = resources.LambdaBasicRoleARN
        replacements["LAMBDA_FILE_NAME"] = resources.LambdaFunctionFileZip
        replacements["RESERVED_CONCURRENT_EXECUTIONS"] = "2"
        replacements["AWS_REGION"] = get_region()

        # Load Lambda CR
        resource_data = load_lambda_resource(
            "function",
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

        # Check Lambda function exists
        exists = self.function_exists(lambda_client, resource_name)
        assert exists

        reservedConcurrentExecutions = self.get_function_concurrency(lambda_client, resource_name)
        assert reservedConcurrentExecutions == 2

        # Update cr
        cr["spec"]["reservedConcurrentExecutions"] = 0

        # Patch k8s resource
        k8s.patch_custom_resource(ref, cr)
        time.sleep(UPDATE_WAIT_AFTER_SECONDS)

        # Check function updated fields
        reservedConcurrentExecutions = self.get_function_concurrency(lambda_client, resource_name)
        assert reservedConcurrentExecutions == 0

        # Delete k8s resource
        _, deleted = k8s.delete_custom_resource(ref)
        assert deleted is True

        time.sleep(DELETE_WAIT_AFTER_SECONDS)

        # Check Lambda function doesn't exist
        exists = self.function_exists(lambda_client, resource_name)
        assert not exists