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

"""Integration tests for the Lambda FunctionURLConfig API.
"""

import boto3
import pytest
import time
import logging
from typing import Dict, Tuple

from acktest.resources import random_suffix_name
from acktest.aws.identity import get_region, get_account_id
from acktest.k8s import resource as k8s
from e2e import service_marker, CRD_GROUP, CRD_VERSION, load_lambda_resource
from e2e.replacement_values import REPLACEMENT_VALUES
from e2e.bootstrap_resources import get_bootstrap_resources
from e2e.tests.helper import LambdaValidator

RESOURCE_PLURAL = "functionurlconfigs"

CREATE_WAIT_AFTER_SECONDS = 30
UPDATE_WAIT_AFTER_SECONDS = 10
DELETE_WAIT_AFTER_SECONDS = 10

def get_testing_image_url():
    aws_region = get_region()
    account_id = get_account_id()
    return f"{account_id}.dkr.ecr.{aws_region}.amazonaws.com/ack-e2e-testing-lambda-controller:v1"

@pytest.fixture(scope="module")
def lambda_client():
    return boto3.client("lambda")

@pytest.fixture(scope="module")
def lambda_function():
        resource_name = random_suffix_name("lambda-function", 24)

        resources = get_bootstrap_resources()

        replacements = REPLACEMENT_VALUES.copy()
        replacements["FUNCTION_NAME"] = resource_name
        replacements["AWS_REGION"] = get_region()
        replacements["IMAGE_URL"] = get_testing_image_url()
        replacements["LAMBDA_ROLE"] = resources.BasicRole.arn

        # Load Lambda CR
        resource_data = load_lambda_resource(
            "function_package_type_image",
            additional_replacements=replacements,
        )
        logging.debug(resource_data)

        # Create k8s resource
        ref = k8s.CustomResourceReference(
            CRD_GROUP, CRD_VERSION, "functions",
            resource_name, namespace="default",
        )
        k8s.create_custom_resource(ref, resource_data)
        cr = k8s.wait_resource_consumed_by_controller(ref)

        assert cr is not None
        assert k8s.get_resource_exists(ref)

        time.sleep(CREATE_WAIT_AFTER_SECONDS)

        cr = k8s.wait_resource_consumed_by_controller(ref)
        logging.debug(cr)

        yield (ref, cr)

        _, deleted = k8s.delete_custom_resource(ref)
        assert deleted


@service_marker
@pytest.mark.canary
class TestFunctionURLConfig:
    def test_smoke(self, lambda_client, lambda_function):
        (_, function_resource) = lambda_function
        lambda_function_name = function_resource["spec"]["name"]

        resource_name = random_suffix_name("functionurlconfig", 24)

        replacements = REPLACEMENT_VALUES.copy()
        replacements["AWS_REGION"] = get_region()
        replacements["FUNCTION_URL_CONFIG_NAME"] = resource_name
        replacements["FUNCTION_NAME"] = lambda_function_name
        replacements["AUTH_TYPE"] = "NONE"

        # Load FunctionURLConfig CR
        resource_data = load_lambda_resource(
            "function_url_config",
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

        # Check FunctionURLConfig exists
        lambda_validator = LambdaValidator(lambda_client)

        # Check function url config exists
        function_url_config = lambda_validator.get_function_url_config(lambda_function_name)
        assert function_url_config is not None
        assert function_url_config["AuthType"] == "NONE"

        cr = k8s.wait_resource_consumed_by_controller(ref)

        # Update cr
        cr["spec"]["cors"] = {
            "maxAge": 10,
            "allowOrigins": ["https://*"],
        }

        # Patch k8s resource
        k8s.patch_custom_resource(ref, cr)
        time.sleep(UPDATE_WAIT_AFTER_SECONDS)

        # Check FunctionURLConfig MaxAge and AllowOrigins array
        function_url_config = lambda_validator.get_function_url_config(lambda_function_name)
        assert function_url_config is not None
        assert function_url_config["Cors"] is not None
        assert function_url_config["Cors"]["MaxAge"] == 10
        assert function_url_config["Cors"]["AllowOrigins"] == ["https://*"]

        # Delete k8s resource
        _, deleted = k8s.delete_custom_resource(ref)
        assert deleted

        time.sleep(DELETE_WAIT_AFTER_SECONDS)

        # Check FunctionURLConfig doesn't exist
        assert not lambda_validator.function_url_config_exists(lambda_function_name)
    
    def test_smoke_ref(self, lambda_client, lambda_function):
        (_, function_resource) = lambda_function
        function_resource_name = function_resource["metadata"]["name"]

        resource_name = random_suffix_name("functionurlconfig", 24)

        replacements = REPLACEMENT_VALUES.copy()
        replacements["AWS_REGION"] = get_region()
        replacements["FUNCTION_URL_CONFIG_NAME"] = resource_name
        replacements["FUNCTION_REF_NAME"] = function_resource_name
        replacements["AUTH_TYPE"] = "NONE"

        # Load FunctionURLConfig CR
        resource_data = load_lambda_resource(
            "function_url_config_ref",
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

        # Check FunctionURLConfig exists
        lambda_validator = LambdaValidator(lambda_client)

        # Check function url config exists
        function_url_config = lambda_validator.get_function_url_config(function_resource_name)
        assert function_url_config is not None
        assert function_url_config["AuthType"] == "NONE"

        cr = k8s.wait_resource_consumed_by_controller(ref)

        # Update cr
        cr["spec"]["cors"] = {
            "maxAge": 10,
            "allowOrigins": ["https://*"],
        }

        # Patch k8s resource
        k8s.patch_custom_resource(ref, cr)
        time.sleep(UPDATE_WAIT_AFTER_SECONDS)

        # Check FunctionURLConfig MaxAge and AllowOrigins array
        function_url_config = lambda_validator.get_function_url_config(function_resource_name)
        assert function_url_config is not None
        assert function_url_config["Cors"] is not None
        assert function_url_config["Cors"]["MaxAge"] == 10
        assert function_url_config["Cors"]["AllowOrigins"] == ["https://*"]

        # Delete k8s resource
        _, deleted = k8s.delete_custom_resource(ref)
        assert deleted

        time.sleep(DELETE_WAIT_AFTER_SECONDS)

        # Check FunctionURLConfig doesn't exist
        assert not lambda_validator.function_url_config_exists(function_resource_name)