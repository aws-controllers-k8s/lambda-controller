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

"""Integration tests for the Lambda version API.
"""

import pytest
import time
import logging
import hashlib
import base64

from acktest.resources import random_suffix_name
from acktest.aws.identity import get_region
from acktest.k8s import resource as k8s

from e2e import service_marker, CRD_GROUP, CRD_VERSION, load_lambda_resource
from e2e.replacement_values import REPLACEMENT_VALUES
from e2e.bootstrap_resources import get_bootstrap_resources
from e2e.service_bootstrap import LAMBDA_FUNCTION_FILE_ZIP, LAMBDA_FUNCTION_FILE_PATH_ZIP
from e2e.tests.helper import LambdaValidator

RESOURCE_PLURAL = "versions"

CREATE_WAIT_AFTER_SECONDS = 10
UPDATE_WAIT_AFTER_SECONDS = 10
DELETE_WAIT_AFTER_SECONDS = 10

@pytest.fixture(scope="module")
def lambda_function():
        resource_name = random_suffix_name("lambda-function", 24)
        resources = get_bootstrap_resources()

        replacements = REPLACEMENT_VALUES.copy()
        replacements["FUNCTION_NAME"] = resource_name
        replacements["BUCKET_NAME"] = resources.FunctionsBucket.name
        replacements["LAMBDA_ROLE"] = resources.EICRole.arn
        replacements["LAMBDA_FILE_NAME"] = LAMBDA_FUNCTION_FILE_ZIP
        replacements["RESERVED_CONCURRENT_EXECUTIONS"] = "3"
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
class TestVersion:
    def test_smoke(self, lambda_client, lambda_function):
        (_, function_resource) = lambda_function
        lambda_function_name = function_resource["spec"]["name"]

        resource_name = random_suffix_name("lambda-version", 24)

        replacements = REPLACEMENT_VALUES.copy()
        replacements["AWS_REGION"] = get_region()
        replacements["FUNCTION_NAME"] = lambda_function_name
        replacements["VERSION_NAME"] = resource_name
        
        # Load Lambda CR
        resource_data = load_lambda_resource(
            "version",
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

        version_number = cr['status']['version']

        # Check version exists
        assert lambda_validator.version_exists(lambda_function_name, version_number)

        # Delete k8s resource
        _, deleted = k8s.delete_custom_resource(ref)
        assert deleted is True

        time.sleep(DELETE_WAIT_AFTER_SECONDS)

        # Check function version doesn't exist
        assert not lambda_validator.version_exists(lambda_function_name, version_number)

    def test_version_with_check(self, lambda_client, lambda_function):
        (_, function_resource) = lambda_function
        lambda_function_name = function_resource["spec"]["name"]

        resource_name = random_suffix_name("lambda-version", 24)

        archive_1 = open(LAMBDA_FUNCTION_FILE_PATH_ZIP, 'rb') 
        readFile_1 = archive_1.read() 
        hash_1 = hashlib.sha256(readFile_1) 
        binary_hash_1 = hash_1.digest() 
        base64_hash_1 = base64.b64encode(binary_hash_1).decode('utf-8')

        replacements = REPLACEMENT_VALUES.copy()
        replacements["AWS_REGION"] = get_region()
        replacements["FUNCTION_NAME"] = lambda_function_name
        replacements["VERSION_NAME"] = resource_name
        replacements["HASH"] = base64_hash_1
        replacements["REVISION_ID"] = ""

        # Load Lambda CR
        resource_data = load_lambda_resource(
            "version_with_check",
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

        version_number = cr['status']['version']

        # Check version exists
        assert lambda_validator.version_exists(lambda_function_name, version_number)

        # Delete k8s resource
        _, deleted = k8s.delete_custom_resource(ref)
        assert deleted is True

        time.sleep(DELETE_WAIT_AFTER_SECONDS)

        # Check function version doesn't exist
        assert not lambda_validator.version_exists(lambda_function_name, version_number)