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
from acktest.aws.identity import get_region, get_account_id
from acktest.k8s import resource as k8s
from e2e import service_marker, CRD_GROUP, CRD_VERSION, load_lambda_resource
from e2e.replacement_values import REPLACEMENT_VALUES
from e2e.bootstrap_resources import get_bootstrap_resources

RESOURCE_PLURAL = "functions"

CREATE_WAIT_AFTER_SECONDS = 25
UPDATE_WAIT_AFTER_SECONDS = 25
DELETE_WAIT_AFTER_SECONDS = 25


def get_testing_image_url():
    aws_region = get_region()
    account_id = get_account_id()
    return f"{account_id}.dkr.ecr.{aws_region}.amazonaws.com/ack-e2e-testing-lambda-controller:v1"

@pytest.fixture(scope="module")
def lambda_client():
    return boto3.client("lambda")

@pytest.fixture(scope="module")
def code_signing_config():
        resource_name = random_suffix_name("lambda-csc", 24)

        resources = get_bootstrap_resources()
        logging.debug(resources)

        replacements = REPLACEMENT_VALUES.copy()
        replacements["AWS_REGION"] = get_region()
        replacements["CODE_SIGNING_CONFIG_NAME"] = resource_name
        replacements["SIGNING_PROFILE_VERSION_ARN"] = resources.SigningProfileVersionArn

        # Load Lambda CR
        resource_data = load_lambda_resource(
            "code_signing_config",
            additional_replacements=replacements,
        )
        logging.debug(resource_data)

        # Create k8s resource
        ref = k8s.CustomResourceReference(
            CRD_GROUP, CRD_VERSION, "codesigningconfigs",
            resource_name, namespace="default",
        )
        k8s.create_custom_resource(ref, resource_data)
        cr = k8s.wait_resource_consumed_by_controller(ref)

        assert cr is not None
        assert k8s.get_resource_exists(ref)

        time.sleep(CREATE_WAIT_AFTER_SECONDS)

        yield (ref, cr)

        _, deleted = k8s.delete_custom_resource(ref)
        assert deleted


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

    def get_function_code_signing_config(self, lambda_client, function_name: str) -> int:
        try:
            resp = lambda_client.get_function_code_signing_config(
                FunctionName=function_name
            )
            return resp['CodeSigningConfigArn']

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
        replacements["CODE_SIGNING_CONFIG_ARN"] = ""
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

        cr = k8s.wait_resource_consumed_by_controller(ref)

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
        replacements["CODE_SIGNING_CONFIG_ARN"] = ""
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

        cr = k8s.wait_resource_consumed_by_controller(ref)

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

    def test_function_code_signing_config(self, lambda_client, code_signing_config):
        (_, csc_resource) = code_signing_config
        code_signing_config_arn = csc_resource["status"]["ackResourceMetadata"]["arn"]
        resource_name = random_suffix_name("lambda-function", 24)

        resources = get_bootstrap_resources()

        replacements = REPLACEMENT_VALUES.copy()
        replacements["FUNCTION_NAME"] = resource_name
        replacements["BUCKET_NAME"] = resources.FunctionsBucketName
        replacements["LAMBDA_ROLE"] = resources.LambdaBasicRoleARN
        replacements["LAMBDA_FILE_NAME"] = resources.LambdaFunctionFileZip
        replacements["RESERVED_CONCURRENT_EXECUTIONS"] = "2"
        replacements["CODE_SIGNING_CONFIG_ARN"] = code_signing_config_arn
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

        cr = k8s.wait_resource_consumed_by_controller(ref)

        # Check Lambda function exists
        exists = self.function_exists(lambda_client, resource_name)
        assert exists

        # Check function code signing config is correct
        function_csc_arn = self.get_function_code_signing_config(lambda_client, resource_name)
        assert function_csc_arn == code_signing_config_arn

        # Delete function code signing config
        cr["spec"]["codeSigningConfigARN"] = ""
        k8s.patch_custom_resource(ref, cr)

        time.sleep(UPDATE_WAIT_AFTER_SECONDS)

        function_csc_arn = self.get_function_code_signing_config(lambda_client, resource_name)
        assert function_csc_arn is None

        # Delete k8s resource
        _, deleted = k8s.delete_custom_resource(ref)
        assert deleted is True

        time.sleep(DELETE_WAIT_AFTER_SECONDS)

        # Check Lambda function doesn't exist
        exists = self.function_exists(lambda_client, resource_name)
        assert not exists

    def test_function_package_type_image(self, lambda_client, code_signing_config):
        resource_name = random_suffix_name("lambda-function", 24)

        resources = get_bootstrap_resources()

        replacements = REPLACEMENT_VALUES.copy()
        replacements["FUNCTION_NAME"] = resource_name
        replacements["LAMBDA_ROLE"] = resources.LambdaBasicRoleARN
        replacements["AWS_REGION"] = get_region()
        replacements["IMAGE_URL"] = get_testing_image_url()

        # Load Lambda CR
        resource_data = load_lambda_resource(
            "function_package_type_image",
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

        # Check Lambda function exists
        exists = self.function_exists(lambda_client, resource_name)
        assert exists

        # Delete k8s resource
        _, deleted = k8s.delete_custom_resource(ref)
        assert deleted is True

        time.sleep(DELETE_WAIT_AFTER_SECONDS)

        # Check Lambda function doesn't exist
        exists = self.function_exists(lambda_client, resource_name)
        assert not exists

    def test_function_package_type_image_with_signing_config(self, lambda_client, code_signing_config):
        resource_name = random_suffix_name("lambda-function", 24)

        resources = get_bootstrap_resources()

        replacements = REPLACEMENT_VALUES.copy()
        replacements["FUNCTION_NAME"] = resource_name
        replacements["LAMBDA_ROLE"] = resources.LambdaBasicRoleARN
        replacements["AWS_REGION"] = get_region()
        replacements["IMAGE_URL"] = get_testing_image_url()

        # Load Lambda CR
        resource_data = load_lambda_resource(
            "function_package_type_image",
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

        # Check Lambda function exists
        exists = self.function_exists(lambda_client, resource_name)
        assert exists

        # Add signing configuration
        cr["spec"]["codeSigningConfigARN"] = "random-csc"
        k8s.patch_custom_resource(ref, cr)

        time.sleep(UPDATE_WAIT_AFTER_SECONDS)

        cr = k8s.wait_resource_consumed_by_controller(ref)
        # assert condition
        assert k8s.assert_condition_state_message(
            ref,
            "ACK.Terminal",
            "True",
            "cannot set function code signing config when package type is Image",
        )

        cr = k8s.wait_resource_consumed_by_controller(ref)

        # Remove signing configuration
        cr["spec"]["codeSigningConfigARN"] = ""
        k8s.patch_custom_resource(ref, cr)

        time.sleep(UPDATE_WAIT_AFTER_SECONDS)
        
        # Delete k8s resource
        _, deleted = k8s.delete_custom_resource(ref)
        assert deleted is True

        time.sleep(DELETE_WAIT_AFTER_SECONDS)

        # Check Lambda function doesn't exist
        exists = self.function_exists(lambda_client, resource_name)
        assert not exists
