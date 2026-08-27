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

import pytest
import time
import logging
import hashlib
import base64
import io
from zipfile import ZipFile

from acktest import tags
from acktest.resources import random_suffix_name
from acktest.aws.identity import get_region, get_account_id
from acktest.k8s import resource as k8s

from e2e import service_marker, CRD_GROUP, CRD_VERSION, load_lambda_resource
from e2e.replacement_values import REPLACEMENT_VALUES
from e2e.bootstrap_resources import get_bootstrap_resources
from e2e.service_bootstrap import LAMBDA_FUNCTION_FILE_ZIP, LAMBDA_FUNCTION_FILE_PATH_ZIP
from e2e.service_bootstrap import LAMBDA_FUNCTION_UPDATED_FILE_ZIP, LAMBDA_FUNCTION_UPDATED_FILE_PATH_ZIP
from e2e.tests.helper import LambdaValidator

RESOURCE_PLURAL = "functions"

CREATE_WAIT_AFTER_SECONDS = 30
UPDATE_WAIT_AFTER_SECONDS = 30
DELETE_WAIT_AFTER_SECONDS = 30

CONTROLLER_WAIT_PERIODS = 10
CONTROLLER_PERIOD_LENGTH = 10
DELETE_WAIT_PERIODS = 3
DELETE_PERIOD_LENGTH = 10

def get_testing_image_url():
    aws_region = get_region()
    account_id = get_account_id()
    return f"{account_id}.dkr.ecr.{aws_region}.amazonaws.com/ack-e2e-testing-lambda-controller:v1"

@pytest.fixture(scope="module")
def code_signing_config():
        resource_name = random_suffix_name("lambda-csc", 24)

        resources = get_bootstrap_resources()
        logging.debug(resources)

        replacements = REPLACEMENT_VALUES.copy()
        replacements["AWS_REGION"] = get_region()
        replacements["CODE_SIGNING_CONFIG_NAME"] = resource_name
        replacements["SIGNING_PROFILE_VERSION_ARN"] = resources.SigningProfile.signing_profile_arn

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
        cr = k8s.wait_resource_consumed_by_controller(ref, wait_periods=CONTROLLER_WAIT_PERIODS, period_length=CONTROLLER_PERIOD_LENGTH)

        assert cr is not None
        assert k8s.get_resource_exists(ref)

        time.sleep(CREATE_WAIT_AFTER_SECONDS)

        yield (ref, cr)

        _, deleted = k8s.delete_custom_resource(ref, wait_periods=DELETE_WAIT_PERIODS, period_length=DELETE_PERIOD_LENGTH)
        assert deleted

@service_marker
@pytest.mark.canary
class TestFunction:

    def test_smoke(self, lambda_client):
        resource_name = random_suffix_name("lambda-function", 24)

        resources = get_bootstrap_resources()
        logging.debug(resources)

        replacements = REPLACEMENT_VALUES.copy()
        replacements["FUNCTION_NAME"] = resource_name
        replacements["BUCKET_NAME"] = resources.FunctionsBucket.name
        replacements["LAMBDA_ROLE"] = resources.BasicRole.arn
        replacements["LAMBDA_FILE_NAME"] = LAMBDA_FUNCTION_FILE_ZIP
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
        cr = k8s.wait_resource_consumed_by_controller(ref, wait_periods=CONTROLLER_WAIT_PERIODS, period_length=CONTROLLER_PERIOD_LENGTH)

        assert cr is not None
        assert k8s.get_resource_exists(ref)

        time.sleep(CREATE_WAIT_AFTER_SECONDS)

        cr = k8s.wait_resource_consumed_by_controller(ref, wait_periods=CONTROLLER_WAIT_PERIODS, period_length=CONTROLLER_PERIOD_LENGTH)

        lambda_validator = LambdaValidator(lambda_client)

        # Assert that the original code.s3Bucket and code.s3Key is still part of
        # the function's CR
        assert cr["spec"]["code"]["s3Bucket"] == resources.FunctionsBucket.name
        assert cr["spec"]["code"]["s3Key"] == LAMBDA_FUNCTION_FILE_ZIP

        # Check Lambda function exists
        assert lambda_validator.function_exists(resource_name)

        # Update cr
        update_tags = {
            "v1": "k1",
            "v2": "k2",
            "v3": "k3",
        }
        cr["spec"]["description"] = "Updated description"
        cr["spec"]["timeout"] = 10
        cr["spec"]["tags"] = update_tags

        # Patch k8s resource
        k8s.patch_custom_resource(ref, cr)
        time.sleep(UPDATE_WAIT_AFTER_SECONDS)

        # Check function updated fields
        function = lambda_validator.get_function(resource_name)
        assert function is not None
        assert function["Configuration"]["Description"] == "Updated description"
        assert function["Configuration"]["Timeout"] == 10

        function_tags = function["Tags"]
        tags.assert_ack_system_tags(
            tags=function_tags,
        )
        tags.assert_equal_without_ack_tags(
            expected=update_tags,
            actual=function_tags,
        )

        # Delete k8s resource
        _, deleted = k8s.delete_custom_resource(ref, wait_periods=DELETE_WAIT_PERIODS, period_length=DELETE_PERIOD_LENGTH)
        assert deleted is True

        time.sleep(DELETE_WAIT_AFTER_SECONDS)

        # Check Lambda function doesn't exist
        assert not lambda_validator.function_exists(resource_name)

    def test_reserved_concurrent_executions(self, lambda_client):
        resource_name = random_suffix_name("lambda-function", 24)

        resources = get_bootstrap_resources()
        logging.debug(resources)

        replacements = REPLACEMENT_VALUES.copy()
        replacements["FUNCTION_NAME"] = resource_name
        replacements["BUCKET_NAME"] = resources.FunctionsBucket.name
        replacements["LAMBDA_ROLE"] = resources.BasicRole.arn
        replacements["LAMBDA_FILE_NAME"] = LAMBDA_FUNCTION_FILE_ZIP
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
        cr = k8s.wait_resource_consumed_by_controller(ref, wait_periods=CONTROLLER_WAIT_PERIODS, period_length=CONTROLLER_PERIOD_LENGTH)

        assert cr is not None
        assert k8s.get_resource_exists(ref)

        time.sleep(CREATE_WAIT_AFTER_SECONDS)

        cr = k8s.wait_resource_consumed_by_controller(ref, wait_periods=CONTROLLER_WAIT_PERIODS, period_length=CONTROLLER_PERIOD_LENGTH)

        lambda_validator = LambdaValidator(lambda_client)
        # Check Lambda function exists
        assert lambda_validator.function_exists(resource_name)

        reservedConcurrentExecutions = lambda_validator.get_function_concurrency(resource_name)
        assert reservedConcurrentExecutions == 2

        # Update cr
        cr["spec"]["reservedConcurrentExecutions"] = 0

        # Patch k8s resource
        k8s.patch_custom_resource(ref, cr)
        time.sleep(UPDATE_WAIT_AFTER_SECONDS)

        # Check function updated fields
        reservedConcurrentExecutions = lambda_validator.get_function_concurrency(resource_name)
        assert reservedConcurrentExecutions == 0

        # Delete k8s resource
        _, deleted = k8s.delete_custom_resource(ref, wait_periods=DELETE_WAIT_PERIODS, period_length=DELETE_PERIOD_LENGTH)
        assert deleted is True

        time.sleep(DELETE_WAIT_AFTER_SECONDS)

        # Check Lambda function doesn't exist
        assert not lambda_validator.function_exists(resource_name)

    def test_function_code_signing_config(self, lambda_client, code_signing_config):
        (_, csc_resource) = code_signing_config
        code_signing_config_arn = csc_resource["status"]["ackResourceMetadata"]["arn"]
        resource_name = random_suffix_name("lambda-function", 24)

        resources = get_bootstrap_resources()

        replacements = REPLACEMENT_VALUES.copy()
        replacements["FUNCTION_NAME"] = resource_name
        replacements["BUCKET_NAME"] = resources.FunctionsBucket.name
        replacements["LAMBDA_ROLE"] = resources.BasicRole.arn
        replacements["LAMBDA_FILE_NAME"] = LAMBDA_FUNCTION_FILE_ZIP
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
        cr = k8s.wait_resource_consumed_by_controller(ref, wait_periods=CONTROLLER_WAIT_PERIODS, period_length=CONTROLLER_PERIOD_LENGTH)

        assert cr is not None
        assert k8s.get_resource_exists(ref)

        time.sleep(CREATE_WAIT_AFTER_SECONDS)

        cr = k8s.wait_resource_consumed_by_controller(ref, wait_periods=CONTROLLER_WAIT_PERIODS, period_length=CONTROLLER_PERIOD_LENGTH)

        lambda_validator = LambdaValidator(lambda_client)
        # Check Lambda function exists
        assert lambda_validator.function_exists(resource_name)

        # Check function code signing config is correct
        function_csc_arn = lambda_validator.get_function_code_signing_config(resource_name)
        assert function_csc_arn == code_signing_config_arn

        # Delete function code signing config
        cr["spec"]["codeSigningConfigARN"] = ""
        k8s.patch_custom_resource(ref, cr)

        time.sleep(UPDATE_WAIT_AFTER_SECONDS)

        function_csc_arn = lambda_validator.get_function_code_signing_config(resource_name)
        assert function_csc_arn is None

        # Delete k8s resource
        _, deleted = k8s.delete_custom_resource(ref, wait_periods=DELETE_WAIT_PERIODS, period_length=DELETE_PERIOD_LENGTH)
        assert deleted is True

        time.sleep(DELETE_WAIT_AFTER_SECONDS)

        # Check Lambda function doesn't exist
        assert not lambda_validator.function_exists(resource_name)

    def test_function_package_type_image(self, lambda_client):
        resource_name = random_suffix_name("lambda-function", 24)

        resources = get_bootstrap_resources()

        replacements = REPLACEMENT_VALUES.copy()
        replacements["FUNCTION_NAME"] = resource_name
        replacements["LAMBDA_ROLE"] = resources.BasicRole.arn
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
        cr = k8s.wait_resource_consumed_by_controller(ref, wait_periods=CONTROLLER_WAIT_PERIODS, period_length=CONTROLLER_PERIOD_LENGTH)

        assert cr is not None
        assert k8s.get_resource_exists(ref)

        time.sleep(CREATE_WAIT_AFTER_SECONDS)

        cr = k8s.wait_resource_consumed_by_controller(ref, wait_periods=CONTROLLER_WAIT_PERIODS, period_length=CONTROLLER_PERIOD_LENGTH)

        lambda_validator = LambdaValidator(lambda_client)
        # Check Lambda function exists
        assert lambda_validator.function_exists(resource_name)

        cr["spec"]["timeout"] = 10
        cr["spec"]["ephemeralStorage"] = { "size" : 1024 }

        # Patch k8s resource
        k8s.patch_custom_resource(ref, cr)
        time.sleep(UPDATE_WAIT_AFTER_SECONDS)

        # Check function updated fields
        function = lambda_validator.get_function(resource_name)
        assert function["Configuration"]["Timeout"] == 10
        assert function["Configuration"]["EphemeralStorage"]["Size"] == 1024

        # Delete k8s resource
        _, deleted = k8s.delete_custom_resource(ref, wait_periods=DELETE_WAIT_PERIODS, period_length=DELETE_PERIOD_LENGTH)
        assert deleted is True

        time.sleep(DELETE_WAIT_AFTER_SECONDS)

        # Check Lambda function doesn't exist
        assert not lambda_validator.function_exists(resource_name)

    def test_function_package_type_image_with_signing_config(self, lambda_client):
        resource_name = random_suffix_name("lambda-function", 24)

        resources = get_bootstrap_resources()

        replacements = REPLACEMENT_VALUES.copy()
        replacements["FUNCTION_NAME"] = resource_name
        replacements["LAMBDA_ROLE"] = resources.BasicRole.arn
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
        cr = k8s.wait_resource_consumed_by_controller(ref, wait_periods=CONTROLLER_WAIT_PERIODS, period_length=CONTROLLER_PERIOD_LENGTH)

        assert cr is not None
        assert k8s.get_resource_exists(ref)

        time.sleep(CREATE_WAIT_AFTER_SECONDS)

        cr = k8s.wait_resource_consumed_by_controller(ref, wait_periods=CONTROLLER_WAIT_PERIODS, period_length=CONTROLLER_PERIOD_LENGTH)

        lambda_validator = LambdaValidator(lambda_client)
        # Check Lambda function exists
        assert lambda_validator.function_exists(resource_name)

        # Add signing configuration
        cr["spec"]["codeSigningConfigARN"] = "random-csc"
        k8s.patch_custom_resource(ref, cr)

        time.sleep(UPDATE_WAIT_AFTER_SECONDS)

        cr = k8s.wait_resource_consumed_by_controller(ref, wait_periods=CONTROLLER_WAIT_PERIODS, period_length=CONTROLLER_PERIOD_LENGTH)
        # assert condition
        assert k8s.assert_condition_state_message(
            ref,
            "ACK.Terminal",
            "True",
            "cannot set function code signing config when package type is Image",
        )

        cr = k8s.wait_resource_consumed_by_controller(ref, wait_periods=CONTROLLER_WAIT_PERIODS, period_length=CONTROLLER_PERIOD_LENGTH)

        # Remove signing configuration
        cr["spec"]["codeSigningConfigARN"] = ""
        k8s.patch_custom_resource(ref, cr)

        time.sleep(UPDATE_WAIT_AFTER_SECONDS)
        
        # Delete k8s resource
        _, deleted = k8s.delete_custom_resource(ref, wait_periods=DELETE_WAIT_PERIODS, period_length=DELETE_PERIOD_LENGTH)
        assert deleted is True

        time.sleep(DELETE_WAIT_AFTER_SECONDS)

        # Check Lambda function doesn't exist
        assert not lambda_validator.function_exists(resource_name)

    def test_function_is_synced(self, lambda_client):
        resource_name = random_suffix_name("lambda-function", 24)

        resources = get_bootstrap_resources()
        logging.debug(resources)

        replacements = REPLACEMENT_VALUES.copy()
        replacements["FUNCTION_NAME"] = resource_name
        replacements["BUCKET_NAME"] = resources.FunctionsBucket.name
        replacements["LAMBDA_ROLE"] = resources.BasicRole.arn
        replacements["LAMBDA_FILE_NAME"] = LAMBDA_FUNCTION_FILE_ZIP
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
        cr = k8s.wait_resource_consumed_by_controller(ref, wait_periods=CONTROLLER_WAIT_PERIODS, period_length=CONTROLLER_PERIOD_LENGTH)

        assert cr is not None
        assert k8s.get_resource_exists(ref)

        time.sleep(CREATE_WAIT_AFTER_SECONDS*3)

        cr = k8s.wait_resource_consumed_by_controller(ref, wait_periods=CONTROLLER_WAIT_PERIODS, period_length=CONTROLLER_PERIOD_LENGTH)

        lambda_validator = LambdaValidator(lambda_client)
        # Check Lambda function exists
        assert lambda_validator.function_exists(resource_name)

        assert cr["status"]["state"] == "Active"

        function = lambda_validator.get_function(resource_name)
        assert function is not None
        assert function["Configuration"]["State"] == "Active"

        # Delete k8s resource
        _, deleted = k8s.delete_custom_resource(ref, wait_periods=DELETE_WAIT_PERIODS, period_length=DELETE_PERIOD_LENGTH)
        assert deleted is True

        time.sleep(DELETE_WAIT_AFTER_SECONDS)

        # Check Lambda function doesn't exist
        assert not lambda_validator.function_exists(resource_name)
    
    def test_function_snapstart(self, lambda_client):
        resource_name = random_suffix_name("functionsnapstart", 24)

        resources = get_bootstrap_resources()
        logging.debug(resources)

        replacements = REPLACEMENT_VALUES.copy()
        replacements["FUNCTION_NAME"] = resource_name
        replacements["BUCKET_NAME"] = resources.FunctionsBucket.name
        replacements["LAMBDA_ROLE"] = resources.BasicRole.arn
        replacements["LAMBDA_FILE_NAME"] = LAMBDA_FUNCTION_FILE_ZIP
        replacements["RESERVED_CONCURRENT_EXECUTIONS"] = "0"
        replacements["CODE_SIGNING_CONFIG_ARN"] = ""
        replacements["AWS_REGION"] = get_region()

        # Load Lambda CR
        resource_data = load_lambda_resource(
            "function_snapstart",
            additional_replacements=replacements,
        )
        logging.debug(resource_data)

        # Create k8s resource
        ref = k8s.CustomResourceReference(
            CRD_GROUP, CRD_VERSION, RESOURCE_PLURAL,
            resource_name, namespace="default",
        )
        k8s.create_custom_resource(ref, resource_data)
        cr = k8s.wait_resource_consumed_by_controller(ref, wait_periods=CONTROLLER_WAIT_PERIODS, period_length=CONTROLLER_PERIOD_LENGTH)

        assert cr is not None
        assert k8s.get_resource_exists(ref)

        time.sleep(CREATE_WAIT_AFTER_SECONDS)

        cr = k8s.wait_resource_consumed_by_controller(ref, wait_periods=CONTROLLER_WAIT_PERIODS, period_length=CONTROLLER_PERIOD_LENGTH)

        lambda_validator = LambdaValidator(lambda_client)

        # Check Lambda function exists
        assert lambda_validator.function_exists(resource_name)

        # Update cr
        cr["spec"]["snapStart"] = { "applyOn" : "PublishedVersions" }

        #Patch k8s resource
        k8s.patch_custom_resource(ref, cr)
        time.sleep(UPDATE_WAIT_AFTER_SECONDS)

        #Check function_snapstart update fields
        function = lambda_validator.get_function(resource_name)
        assert function["Configuration"]["SnapStart"]["ApplyOn"] == "PublishedVersions"

        # Delete k8s resource
        _, deleted = k8s.delete_custom_resource(ref, wait_periods=DELETE_WAIT_PERIODS, period_length=DELETE_PERIOD_LENGTH)
        assert deleted is True

        time.sleep(DELETE_WAIT_AFTER_SECONDS)

        # Check Lambda function doesn't exist
        assert not lambda_validator.function_exists(resource_name)

    def test_function_architecture(self, lambda_client):
        resource_name = random_suffix_name("functionsarchitecture", 24)

        resources = get_bootstrap_resources()
        logging.debug(resources)

        replacements = REPLACEMENT_VALUES.copy()
        replacements["FUNCTION_NAME"] = resource_name
        replacements["BUCKET_NAME"] = resources.FunctionsBucket.name
        replacements["LAMBDA_ROLE"] = resources.BasicRole.arn
        replacements["LAMBDA_FILE_NAME"] = LAMBDA_FUNCTION_FILE_ZIP
        replacements["RESERVED_CONCURRENT_EXECUTIONS"] = "0"
        replacements["CODE_SIGNING_CONFIG_ARN"] = ""
        replacements["AWS_REGION"] = get_region()
        replacements["ARCHITECTURES"] = 'x86_64'

        # Load Lambda CR
        resource_data = load_lambda_resource(
            "function_architectures",
            additional_replacements=replacements,
        )
        logging.debug(resource_data)

        # Create k8s resource
        ref = k8s.CustomResourceReference(
            CRD_GROUP, CRD_VERSION, RESOURCE_PLURAL,
            resource_name, namespace="default",
        )
        k8s.create_custom_resource(ref, resource_data)
        cr = k8s.wait_resource_consumed_by_controller(ref, wait_periods=CONTROLLER_WAIT_PERIODS, period_length=CONTROLLER_PERIOD_LENGTH)

        assert cr is not None
        assert k8s.get_resource_exists(ref)

        time.sleep(CREATE_WAIT_AFTER_SECONDS)

        cr = k8s.wait_resource_consumed_by_controller(ref, wait_periods=CONTROLLER_WAIT_PERIODS, period_length=CONTROLLER_PERIOD_LENGTH)

        lambda_validator = LambdaValidator(lambda_client)

        # Check Lambda function exists
        assert lambda_validator.function_exists(resource_name)

        # Update cr
        cr["spec"]["architectures"] = ['arm64']
        cr["spec"]["code"]["s3Bucket"] = resources.FunctionsBucket.name
        cr["spec"]["code"]["s3Key"] = LAMBDA_FUNCTION_FILE_ZIP

        #Patch k8s resource
        k8s.patch_custom_resource(ref, cr)
        time.sleep(UPDATE_WAIT_AFTER_SECONDS)

        #Check function_snapstart update fields
        function = lambda_validator.get_function(resource_name)
        assert function["Configuration"]["Architectures"] == ['arm64']

        # Delete k8s resource
        _, deleted = k8s.delete_custom_resource(ref, wait_periods=DELETE_WAIT_PERIODS, period_length=DELETE_PERIOD_LENGTH)
        assert deleted is True

        time.sleep(DELETE_WAIT_AFTER_SECONDS)

        # Check Lambda function doesn't exist
        assert not lambda_validator.function_exists(resource_name)

    def test_function_features(self, lambda_client):
        resource_name = random_suffix_name("functionfeatures", 24)

        resources = get_bootstrap_resources()
        logging.debug(resources)

        replacements = REPLACEMENT_VALUES.copy()
        replacements["FUNCTION_NAME"] = resource_name
        replacements["BUCKET_NAME"] = resources.FunctionsBucket.name
        replacements["LAMBDA_ROLE"] = resources.EICRole.arn
        replacements["LAMBDA_FILE_NAME"] = LAMBDA_FUNCTION_FILE_ZIP
        replacements["AWS_REGION"] = get_region()
        replacements["DEAD_LETTER_CONFIG_TARGET_ARN"] = resources.EICQueueOnSuccess.arn

        # Load Lambda CR
        resource_data = load_lambda_resource(
            "function_features",
            additional_replacements=replacements,
        )
        logging.debug(resource_data)

        # Create k8s resource
        ref = k8s.CustomResourceReference(
            CRD_GROUP, CRD_VERSION, RESOURCE_PLURAL,
            resource_name, namespace="default",
        )
        k8s.create_custom_resource(ref, resource_data)
        cr = k8s.wait_resource_consumed_by_controller(ref, wait_periods=CONTROLLER_WAIT_PERIODS, period_length=CONTROLLER_PERIOD_LENGTH)

        assert cr is not None
        assert k8s.get_resource_exists(ref)

        time.sleep(CREATE_WAIT_AFTER_SECONDS)

        cr = k8s.wait_resource_consumed_by_controller(ref, wait_periods=CONTROLLER_WAIT_PERIODS, period_length=CONTROLLER_PERIOD_LENGTH)

        lambda_validator = LambdaValidator(lambda_client)

        # Check Lambda function exists
        assert lambda_validator.function_exists(resource_name)

        # Update cr
        cr["spec"]["deadLetterConfig"]["targetARN"] = resources.EICQueueOnFailure.arn

        #Patch k8s resource
        k8s.patch_custom_resource(ref, cr)
        time.sleep(UPDATE_WAIT_AFTER_SECONDS)

        #Check function_snapstart update fields
        function = lambda_validator.get_function(resource_name)
        assert function["Configuration"]["DeadLetterConfig"]["TargetArn"] == resources.EICQueueOnFailure.arn

        # Delete k8s resource
        _, deleted = k8s.delete_custom_resource(ref, wait_periods=DELETE_WAIT_PERIODS, period_length=DELETE_PERIOD_LENGTH)
        assert deleted is True

        time.sleep(DELETE_WAIT_AFTER_SECONDS)

        # Check Lambda function doesn't exist
        assert not lambda_validator.function_exists(resource_name)
    
    def test_function_runtime(self, lambda_client):
        resource_name = random_suffix_name("function", 24)

        resources = get_bootstrap_resources()
        logging.debug(resources)

        replacements = REPLACEMENT_VALUES.copy()
        replacements["FUNCTION_NAME"] = resource_name
        replacements["BUCKET_NAME"] = resources.FunctionsBucket.name
        replacements["LAMBDA_ROLE"] = resources.BasicRole.arn
        replacements["LAMBDA_FILE_NAME"] = LAMBDA_FUNCTION_FILE_ZIP
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
        cr = k8s.wait_resource_consumed_by_controller(ref, wait_periods=CONTROLLER_WAIT_PERIODS, period_length=CONTROLLER_PERIOD_LENGTH)

        assert cr is not None
        assert k8s.get_resource_exists(ref)

        time.sleep(CREATE_WAIT_AFTER_SECONDS)

        cr = k8s.wait_resource_consumed_by_controller(ref, wait_periods=CONTROLLER_WAIT_PERIODS, period_length=CONTROLLER_PERIOD_LENGTH)

        lambda_validator = LambdaValidator(lambda_client)

        # Check Lambda function exists
        assert lambda_validator.function_exists(resource_name)

        # Update cr
        cr["spec"]["runtime"] = "java21"

        #Patch k8s resource
        k8s.patch_custom_resource(ref, cr)
        time.sleep(UPDATE_WAIT_AFTER_SECONDS)

        #Check function_snapstart update fields
        function = lambda_validator.get_function(resource_name)
        assert function["Configuration"]["Runtime"] == "java21"

        # Delete k8s resource
        _, deleted = k8s.delete_custom_resource(ref, wait_periods=DELETE_WAIT_PERIODS, period_length=DELETE_PERIOD_LENGTH)
        assert deleted is True

        time.sleep(DELETE_WAIT_AFTER_SECONDS)

        # Check Lambda function doesn't exist
        assert not lambda_validator.function_exists(resource_name)
    
    def test_function_layers(self, lambda_client):
        resource_name = random_suffix_name("functionlayers", 24)

        resources = get_bootstrap_resources()
        logging.debug(resources)

        replacements = REPLACEMENT_VALUES.copy()
        replacements["FUNCTION_NAME"] = resource_name
        replacements["BUCKET_NAME"] = resources.FunctionsBucket.name
        replacements["LAMBDA_ROLE"] = resources.EICRole.arn
        replacements["LAMBDA_FILE_NAME"] = LAMBDA_FUNCTION_FILE_ZIP
        replacements["AWS_REGION"] = get_region()
        replacements["LAYERS"] = "arn:aws:lambda:us-west-2:336392948345:layer:AWSSDKPandas-Python310:14"

        # Load Lambda CR
        resource_data = load_lambda_resource(
            "function_layers",
            additional_replacements=replacements,
        )
        logging.debug(resource_data)

        # Create k8s resource
        ref = k8s.CustomResourceReference(
            CRD_GROUP, CRD_VERSION, RESOURCE_PLURAL,
            resource_name, namespace="default",
        )
        k8s.create_custom_resource(ref, resource_data)
        cr = k8s.wait_resource_consumed_by_controller(ref, wait_periods=CONTROLLER_WAIT_PERIODS, period_length=CONTROLLER_PERIOD_LENGTH)

        assert cr is not None
        assert k8s.get_resource_exists(ref)

        time.sleep(CREATE_WAIT_AFTER_SECONDS)

        cr = k8s.wait_resource_consumed_by_controller(ref, wait_periods=CONTROLLER_WAIT_PERIODS, period_length=CONTROLLER_PERIOD_LENGTH)

        lambda_validator = LambdaValidator(lambda_client)

        # Check Lambda function exists
        assert lambda_validator.function_exists(resource_name)

        # Update cr
        layers_list = ["arn:aws:lambda:us-west-2:017000801446:layer:AWSLambdaPowertoolsPythonV2:68", "arn:aws:lambda:us-west-2:580247275435:layer:LambdaInsightsExtension:52"]
        cr["spec"]["layers"] = layers_list

        #Patch k8s resource
        k8s.patch_custom_resource(ref, cr)
        time.sleep(UPDATE_WAIT_AFTER_SECONDS)

        #Check function_snapstart update fields
        function = lambda_validator.get_function(resource_name)
        for i in range(len(function["Configuration"]["Layers"])) :
            assert function["Configuration"]["Layers"][i]["Arn"] == layers_list[i]

        # Delete k8s resource
        _, deleted = k8s.delete_custom_resource(ref, wait_periods=DELETE_WAIT_PERIODS, period_length=DELETE_PERIOD_LENGTH)
        assert deleted is True

        time.sleep(DELETE_WAIT_AFTER_SECONDS)

        # Check Lambda function doesn't exist
        assert not lambda_validator.function_exists(resource_name)

    def test_function_add_layers_to_function_without_layers(self, lambda_client):
        resource_name = random_suffix_name("functionnolayers", 24)

        resources = get_bootstrap_resources()
        logging.debug(resources)

        replacements = REPLACEMENT_VALUES.copy()
        replacements["FUNCTION_NAME"] = resource_name
        replacements["BUCKET_NAME"] = resources.FunctionsBucket.name
        replacements["LAMBDA_ROLE"] = resources.EICRole.arn
        replacements["LAMBDA_FILE_NAME"] = LAMBDA_FUNCTION_FILE_ZIP
        replacements["AWS_REGION"] = get_region()

        # Load Lambda CR that does not declare any layer
        resource_data = load_lambda_resource(
            "function_no_layers",
            additional_replacements=replacements,
        )
        logging.debug(resource_data)

        # Create k8s resource
        ref = k8s.CustomResourceReference(
            CRD_GROUP, CRD_VERSION, RESOURCE_PLURAL,
            resource_name, namespace="default",
        )
        k8s.create_custom_resource(ref, resource_data)
        cr = k8s.wait_resource_consumed_by_controller(ref, wait_periods=CONTROLLER_WAIT_PERIODS, period_length=CONTROLLER_PERIOD_LENGTH)

        assert cr is not None
        assert k8s.get_resource_exists(ref)

        time.sleep(CREATE_WAIT_AFTER_SECONDS)

        cr = k8s.wait_resource_consumed_by_controller(ref, wait_periods=CONTROLLER_WAIT_PERIODS, period_length=CONTROLLER_PERIOD_LENGTH)

        lambda_validator = LambdaValidator(lambda_client)

        # Check Lambda function exists and has no layer attached
        assert lambda_validator.function_exists(resource_name)
        function = lambda_validator.get_function(resource_name)
        assert "Layers" not in function["Configuration"]

        # Add layers to a function that currently has none. GetFunction returns
        # no layer for it, so the controller has to clear Spec.Layers on the
        # latest resource for the delta against the desired spec to be computed.
        layers_list = ["arn:aws:lambda:us-west-2:017000801446:layer:AWSLambdaPowertoolsPythonV2:68", "arn:aws:lambda:us-west-2:580247275435:layer:LambdaInsightsExtension:52"]
        cr["spec"]["layers"] = layers_list

        # Patch k8s resource
        k8s.patch_custom_resource(ref, cr)
        time.sleep(UPDATE_WAIT_AFTER_SECONDS)

        # Check the layers were attached to the function in AWS
        function = lambda_validator.get_function(resource_name)
        assert len(function["Configuration"]["Layers"]) == len(layers_list)
        for i in range(len(layers_list)):
            assert function["Configuration"]["Layers"][i]["Arn"] == layers_list[i]

        # Check the layers are also reported back in the CR status
        cr = k8s.get_resource(ref)
        assert cr["status"]["layerStatuses"] is not None
        assert len(cr["status"]["layerStatuses"]) == len(layers_list)

        # Delete k8s resource
        _, deleted = k8s.delete_custom_resource(ref, wait_periods=DELETE_WAIT_PERIODS, period_length=DELETE_PERIOD_LENGTH)
        assert deleted is True

        time.sleep(DELETE_WAIT_AFTER_SECONDS)

        # Check Lambda function doesn't exist
        assert not lambda_validator.function_exists(resource_name)

    def test_function_event_invoke_config(self, lambda_client):
        resource_name = random_suffix_name("lambda-function", 24)

        resources = get_bootstrap_resources()
        logging.debug(resources)

        replacements = REPLACEMENT_VALUES.copy()
        replacements["FUNCTION_NAME"] = resource_name
        replacements["BUCKET_NAME"] = resources.FunctionsBucket.name
        replacements["LAMBDA_ROLE"] = resources.EICRole.arn
        replacements["LAMBDA_FILE_NAME"] = LAMBDA_FUNCTION_FILE_ZIP
        replacements["AWS_REGION"] = get_region()
        replacements["MAXIMUM_EVENT_AGE_IN_SECONDS"] = "100"
        replacements["MAXIMUM_RETRY_ATTEMPTS"] = "1"
        replacements["ON_SUCCESS_DESTINATION"] = resources.EICQueueOnSuccess.arn
        replacements["ON_FAILURE_DESTINATION"] = resources.EICQueueOnFailure.arn

        # Load Lambda CR
        resource_data = load_lambda_resource(
            "function_event_invoke_config",
            additional_replacements=replacements,
        )
        logging.debug(resource_data)

        # Create k8s resource
        ref = k8s.CustomResourceReference(
            CRD_GROUP, CRD_VERSION, RESOURCE_PLURAL,
            resource_name, namespace="default",
        )
        k8s.create_custom_resource(ref, resource_data)
        cr = k8s.wait_resource_consumed_by_controller(ref, wait_periods=CONTROLLER_WAIT_PERIODS, period_length=CONTROLLER_PERIOD_LENGTH)

        assert cr is not None
        assert k8s.get_resource_exists(ref)

        time.sleep(CREATE_WAIT_AFTER_SECONDS)

        cr = k8s.wait_resource_consumed_by_controller(ref, wait_periods=CONTROLLER_WAIT_PERIODS, period_length=CONTROLLER_PERIOD_LENGTH)

        lambda_validator = LambdaValidator(lambda_client)

        # Check Lambda function exists
        assert lambda_validator.function_exists(resource_name)

        # Update cr
        cr["spec"]["functionEventInvokeConfig"]["maximumEventAgeInSeconds"] = 200
        cr["spec"]["functionEventInvokeConfig"]["maximumRetryAttempts"] = 2

        #Patch k8s resource
        k8s.patch_custom_resource(ref, cr)
        time.sleep(UPDATE_WAIT_AFTER_SECONDS)

        #Check function_event_invoke_config update fields
        function_event_invoke_config = lambda_validator.get_function_event_invoke_config(resource_name)
        assert function_event_invoke_config["MaximumEventAgeInSeconds"] == 200
        assert function_event_invoke_config["MaximumRetryAttempts"] == 2
        
        # Delete FunctionEventInvokeConfig
        cr = k8s.wait_resource_consumed_by_controller(ref, wait_periods=CONTROLLER_WAIT_PERIODS, period_length=CONTROLLER_PERIOD_LENGTH)
        cr["spec"]["functionEventInvokeConfig"] =  None

        # Patch k8s resource
        k8s.patch_custom_resource(ref, cr)
        time.sleep(UPDATE_WAIT_AFTER_SECONDS)

        # Check if FunctionEventInvokeConfig is deleted
        assert not lambda_validator.get_function_event_invoke_config(resource_name)

        # Delete k8s resource
        _, deleted = k8s.delete_custom_resource(ref, wait_periods=DELETE_WAIT_PERIODS, period_length=DELETE_PERIOD_LENGTH)
        assert deleted is True

        time.sleep(DELETE_WAIT_AFTER_SECONDS)

        # Check Lambda function doesn't exist
        assert not lambda_validator.function_exists(resource_name)

    def test_function_event_invoke_config_partial(self, lambda_client):
        # A FunctionEventInvokeConfig that only sets `maximumRetryAttempts`
        # (no `maximumEventAgeInSeconds`, no destinations) previously caused
        # every subsequent reconcile to panic with a nil pointer dereference
        # in setFunctionEventInvokeConfigFromResponse, because
        # GetFunctionEventInvokeConfig legitimately omits
        # MaximumEventAgeInSeconds and DestinationConfig when they were never
        # set. This test ensures the controller reads back a partial
        # configuration without panicking and that the resource converges to
        # ACK.ResourceSynced=True.
        resource_name = random_suffix_name("lambda-function", 24)

        resources = get_bootstrap_resources()
        logging.debug(resources)

        replacements = REPLACEMENT_VALUES.copy()
        replacements["FUNCTION_NAME"] = resource_name
        replacements["BUCKET_NAME"] = resources.FunctionsBucket.name
        replacements["LAMBDA_ROLE"] = resources.EICRole.arn
        replacements["LAMBDA_FILE_NAME"] = LAMBDA_FUNCTION_FILE_ZIP
        replacements["AWS_REGION"] = get_region()
        replacements["MAXIMUM_RETRY_ATTEMPTS"] = "0"

        # Load Lambda CR with only `maximumRetryAttempts` set on
        # functionEventInvokeConfig
        resource_data = load_lambda_resource(
            "function_event_invoke_config_partial",
            additional_replacements=replacements,
        )
        logging.debug(resource_data)

        # Create k8s resource
        ref = k8s.CustomResourceReference(
            CRD_GROUP, CRD_VERSION, RESOURCE_PLURAL,
            resource_name, namespace="default",
        )
        k8s.create_custom_resource(ref, resource_data)
        cr = k8s.wait_resource_consumed_by_controller(ref, wait_periods=CONTROLLER_WAIT_PERIODS, period_length=CONTROLLER_PERIOD_LENGTH)

        assert cr is not None
        assert k8s.get_resource_exists(ref)

        time.sleep(CREATE_WAIT_AFTER_SECONDS)

        lambda_validator = LambdaValidator(lambda_client)

        # Check Lambda function exists
        assert lambda_validator.function_exists(resource_name)

        # Check FunctionEventInvokeConfig was created with the partial config
        function_event_invoke_config = lambda_validator.get_function_event_invoke_config(resource_name)
        assert function_event_invoke_config is not None
        assert function_event_invoke_config["MaximumRetryAttempts"] == 0

        # Force at least one more reconcile after creation. Before the fix,
        # sdkFind panicked on this very next reconcile because the `Get`
        # response has MaximumEventAgeInSeconds and DestinationConfig unset.
        cr = k8s.wait_resource_consumed_by_controller(ref, wait_periods=CONTROLLER_WAIT_PERIODS, period_length=CONTROLLER_PERIOD_LENGTH)
        assert cr is not None

        # Resource must converge to ACK.ResourceSynced=True instead of being
        # permanently stuck at the create-time snapshot
        assert k8s.wait_on_condition(
            ref,
            "ACK.ResourceSynced",
            "True",
            wait_periods=CONTROLLER_WAIT_PERIODS,
            period_length=CONTROLLER_PERIOD_LENGTH,
        )

        # Confirm the spec's optional fields were read back as absent/None
        # rather than the reconcile crashing before ever reaching this point
        cr = k8s.get_resource(ref)
        assert cr["spec"]["functionEventInvokeConfig"]["maximumRetryAttempts"] == 0
        assert cr["spec"]["functionEventInvokeConfig"].get("maximumEventAgeInSeconds") is None
        assert cr["spec"]["functionEventInvokeConfig"].get("destinationConfig") is None

        # A subsequent spec update must also be applied successfully, proving
        # reconciliation is not stuck
        cr["spec"]["functionEventInvokeConfig"]["maximumRetryAttempts"] = 1
        k8s.patch_custom_resource(ref, cr)
        time.sleep(UPDATE_WAIT_AFTER_SECONDS)

        function_event_invoke_config = lambda_validator.get_function_event_invoke_config(resource_name)
        assert function_event_invoke_config["MaximumRetryAttempts"] == 1

        # Delete k8s resource
        _, deleted = k8s.delete_custom_resource(ref, wait_periods=DELETE_WAIT_PERIODS, period_length=DELETE_PERIOD_LENGTH)
        assert deleted is True

        time.sleep(DELETE_WAIT_AFTER_SECONDS)

        # Check Lambda function doesn't exist
        assert not lambda_validator.function_exists(resource_name)

    def test_function_code_s3(self, lambda_client):
        resource_name = random_suffix_name("functioncodes3", 24)

        resources = get_bootstrap_resources()
        logging.debug(resources)

        archive_1 = open(LAMBDA_FUNCTION_FILE_PATH_ZIP, 'rb') 
        readFile_1 = archive_1.read() 
        hash_1 = hashlib.sha256(readFile_1) 
        binary_hash_1 = hash_1.digest() 
        base64_hash_1 = base64.b64encode(binary_hash_1).decode('utf-8')

        archive_2 = open(LAMBDA_FUNCTION_UPDATED_FILE_PATH_ZIP, 'rb') 
        readFile_2 = archive_2.read() 
        hash_2 = hashlib.sha256(readFile_2) 
        binary_hash_2 = hash_2.digest() 
        base64_hash_2 = base64.b64encode(binary_hash_2).decode('utf-8')

        replacements = REPLACEMENT_VALUES.copy()
        replacements["FUNCTION_NAME"] = resource_name
        replacements["BUCKET_NAME"] = resources.FunctionsBucket.name
        replacements["LAMBDA_ROLE"] = resources.BasicRole.arn
        replacements["LAMBDA_FILE_NAME"] = LAMBDA_FUNCTION_FILE_ZIP
        replacements["RESERVED_CONCURRENT_EXECUTIONS"] = "0"
        replacements["CODE_SIGNING_CONFIG_ARN"] = ""
        replacements["AWS_REGION"] = get_region()
        replacements["ARCHITECTURES"] = 'x86_64'
        replacements["HASH"] = base64_hash_1

        # Load Lambda CR
        resource_data = load_lambda_resource(
            "function_code_s3",
            additional_replacements=replacements,
        )
        logging.debug(resource_data)

        # Create k8s resource
        ref = k8s.CustomResourceReference(
            CRD_GROUP, CRD_VERSION, RESOURCE_PLURAL,
            resource_name, namespace="default",
        )
        k8s.create_custom_resource(ref, resource_data)
        cr = k8s.wait_resource_consumed_by_controller(ref, wait_periods=CONTROLLER_WAIT_PERIODS, period_length=CONTROLLER_PERIOD_LENGTH)

        assert cr is not None
        assert k8s.get_resource_exists(ref)

        time.sleep(CREATE_WAIT_AFTER_SECONDS)

        cr = k8s.wait_resource_consumed_by_controller(ref, wait_periods=CONTROLLER_WAIT_PERIODS, period_length=CONTROLLER_PERIOD_LENGTH)

        lambda_validator = LambdaValidator(lambda_client)

        # Assert that the original code.s3Bucket and code.s3Key is still part of
        # the function's CR
        assert cr["spec"]["code"]["s3Bucket"] == resources.FunctionsBucket.name
        assert cr["spec"]["code"]["s3Key"] == LAMBDA_FUNCTION_FILE_ZIP

        # Check Lambda function exists
        assert lambda_validator.function_exists(resource_name)

        # Update cr
        cr["spec"]["code"]["sha256"] = base64_hash_2
        cr["spec"]["code"]["s3Key"] = LAMBDA_FUNCTION_UPDATED_FILE_ZIP

        # Patch k8s resource
        k8s.patch_custom_resource(ref, cr)
        time.sleep(UPDATE_WAIT_AFTER_SECONDS)

        # Check function updated fields
        function = lambda_validator.get_function(resource_name)
        assert function is not None
        assert function["Configuration"]["CodeSha256"] == base64_hash_2

        # Delete k8s resource
        _, deleted = k8s.delete_custom_resource(ref, wait_periods=DELETE_WAIT_PERIODS, period_length=DELETE_PERIOD_LENGTH)
        assert deleted is True

        time.sleep(DELETE_WAIT_AFTER_SECONDS)

        # Check Lambda function doesn't exist
        assert not lambda_validator.function_exists(resource_name)
    
    def test_function_update_code_and_architecture(self, lambda_client):
        resource_name = random_suffix_name("functionupdatecode", 24)

        resources = get_bootstrap_resources()
        logging.debug(resources)

        archive_1 = open(LAMBDA_FUNCTION_FILE_PATH_ZIP, 'rb') 
        readFile_1 = archive_1.read() 
        hash_1 = hashlib.sha256(readFile_1) 
        binary_hash_1 = hash_1.digest() 
        base64_hash_1 = base64.b64encode(binary_hash_1).decode('utf-8')

        archive_2 = open(LAMBDA_FUNCTION_UPDATED_FILE_PATH_ZIP, 'rb') 
        readFile_2 = archive_2.read() 
        hash_2 = hashlib.sha256(readFile_2) 
        binary_hash_2 = hash_2.digest() 
        base64_hash_2 = base64.b64encode(binary_hash_2).decode('utf-8')

        replacements = REPLACEMENT_VALUES.copy()
        replacements["FUNCTION_NAME"] = resource_name
        replacements["BUCKET_NAME"] = resources.FunctionsBucket.name
        replacements["LAMBDA_ROLE"] = resources.BasicRole.arn
        replacements["LAMBDA_FILE_NAME"] = LAMBDA_FUNCTION_FILE_ZIP
        replacements["RESERVED_CONCURRENT_EXECUTIONS"] = "0"
        replacements["CODE_SIGNING_CONFIG_ARN"] = ""
        replacements["AWS_REGION"] = get_region()
        replacements["ARCHITECTURES"] = 'x86_64'
        replacements["HASH"] = base64_hash_1

        # Load Lambda CR
        resource_data = load_lambda_resource(
            "function_code_s3",
            additional_replacements=replacements,
        )
        logging.debug(resource_data)

        # Create k8s resource
        ref = k8s.CustomResourceReference(
            CRD_GROUP, CRD_VERSION, RESOURCE_PLURAL,
            resource_name, namespace="default",
        )
        k8s.create_custom_resource(ref, resource_data)
        cr = k8s.wait_resource_consumed_by_controller(ref, wait_periods=CONTROLLER_WAIT_PERIODS, period_length=CONTROLLER_PERIOD_LENGTH)

        assert cr is not None
        assert k8s.get_resource_exists(ref)

        time.sleep(CREATE_WAIT_AFTER_SECONDS)

        cr = k8s.wait_resource_consumed_by_controller(ref, wait_periods=CONTROLLER_WAIT_PERIODS, period_length=CONTROLLER_PERIOD_LENGTH)

        lambda_validator = LambdaValidator(lambda_client)

        # Assert that the original code.s3Bucket and code.s3Key is still part of
        # the function's CR
        assert cr["spec"]["code"]["s3Bucket"] == resources.FunctionsBucket.name
        assert cr["spec"]["code"]["s3Key"] == LAMBDA_FUNCTION_FILE_ZIP

        # Check Lambda function exists
        assert lambda_validator.function_exists(resource_name)

        # Update cr
        cr["spec"]["code"]["sha256"] = base64_hash_2
        cr["spec"]["code"]["s3Key"] = LAMBDA_FUNCTION_UPDATED_FILE_ZIP
        cr["spec"]["architectures"] = ['arm64']

        # Patch k8s resource
        k8s.patch_custom_resource(ref, cr)
        time.sleep(UPDATE_WAIT_AFTER_SECONDS)

        # Check function updated fields
        function = lambda_validator.get_function(resource_name)
        assert function is not None
        assert function["Configuration"]["CodeSha256"] == base64_hash_2
        assert function["Configuration"]["Architectures"] == ['arm64']

        # Delete k8s resource
        _, deleted = k8s.delete_custom_resource(ref, wait_periods=DELETE_WAIT_PERIODS, period_length=DELETE_PERIOD_LENGTH)
        assert deleted is True

        time.sleep(DELETE_WAIT_AFTER_SECONDS)

        # Check Lambda function doesn't exist
        assert not lambda_validator.function_exists(resource_name)

    def test_function_tenancy(self, lambda_client):
        resource_name = random_suffix_name("functiontenancy", 24)

        resources = get_bootstrap_resources()
        logging.debug(resources)

        replacements = REPLACEMENT_VALUES.copy()
        replacements["FUNCTION_NAME"] = resource_name
        replacements["BUCKET_NAME"] = resources.FunctionsBucket.name
        replacements["LAMBDA_ROLE"] = resources.BasicRole.arn
        replacements["LAMBDA_FILE_NAME"] = LAMBDA_FUNCTION_FILE_ZIP
        replacements["RESERVED_CONCURRENT_EXECUTIONS"] = "0"
        replacements["CODE_SIGNING_CONFIG_ARN"] = ""
        replacements["AWS_REGION"] = get_region()

        # Load Lambda CR
        resource_data = load_lambda_resource(
            "function_tenancy",
            additional_replacements=replacements,
        )
        logging.debug(resource_data)

        # Create k8s resource
        ref = k8s.CustomResourceReference(
            CRD_GROUP, CRD_VERSION, RESOURCE_PLURAL,
            resource_name, namespace="default",
        )
        k8s.create_custom_resource(ref, resource_data)
        cr = k8s.wait_resource_consumed_by_controller(ref, wait_periods=CONTROLLER_WAIT_PERIODS, period_length=CONTROLLER_PERIOD_LENGTH)

        assert cr is not None
        assert k8s.get_resource_exists(ref)

        time.sleep(CREATE_WAIT_AFTER_SECONDS)

        cr = k8s.wait_resource_consumed_by_controller(ref, wait_periods=CONTROLLER_WAIT_PERIODS, period_length=CONTROLLER_PERIOD_LENGTH)

        lambda_validator = LambdaValidator(lambda_client)

        # Check Lambda function exists
        assert lambda_validator.function_exists(resource_name)

        # Verify tenancyConfig was set in the CR spec
        assert cr["spec"]["tenancyConfig"]["tenantIsolationMode"] == "PER_TENANT"

        # Check Lambda function exists and is properly configured
        function = lambda_validator.get_function(resource_name)
        assert function is not None

        # Verify tenancyConfig was applied to the Lambda function
        assert function["Configuration"]["TenancyConfig"]["TenantIsolationMode"] == "PER_TENANT"

        # Delete k8s resource
        _, deleted = k8s.delete_custom_resource(ref, wait_periods=DELETE_WAIT_PERIODS, period_length=DELETE_PERIOD_LENGTH)
        assert deleted is True

        time.sleep(DELETE_WAIT_AFTER_SECONDS)

        # Check Lambda function doesn't exist
        assert not lambda_validator.function_exists(resource_name)

    def test_function_durable_config(self, lambda_client):
        resource_name = random_suffix_name("functiondurableconfig", 24)

        resources = get_bootstrap_resources()
        logging.debug(resources)

        replacements = REPLACEMENT_VALUES.copy()
        replacements["FUNCTION_NAME"] = resource_name
        replacements["BUCKET_NAME"] = resources.FunctionsBucket.name
        replacements["LAMBDA_ROLE"] = resources.BasicRole.arn
        replacements["LAMBDA_FILE_NAME"] = LAMBDA_FUNCTION_FILE_ZIP
        replacements["RESERVED_CONCURRENT_EXECUTIONS"] = "0"
        replacements["AWS_REGION"] = get_region()
        replacements["EXECUTION_TIMEOUT"] = "3600"
        replacements["RETENTION_PERIOD_IN_DAYS"] = "14"

        # Load Lambda CR
        resource_data = load_lambda_resource(
            "function_durable_config",
            additional_replacements=replacements,
        )
        logging.debug(resource_data)

        # Create k8s resource
        ref = k8s.CustomResourceReference(
            CRD_GROUP, CRD_VERSION, RESOURCE_PLURAL,
            resource_name, namespace="default",
        )
        k8s.create_custom_resource(ref, resource_data)
        cr = k8s.wait_resource_consumed_by_controller(ref, wait_periods=CONTROLLER_WAIT_PERIODS, period_length=CONTROLLER_PERIOD_LENGTH)

        assert cr is not None
        assert k8s.get_resource_exists(ref)

        time.sleep(CREATE_WAIT_AFTER_SECONDS)

        cr = k8s.wait_resource_consumed_by_controller(ref, wait_periods=CONTROLLER_WAIT_PERIODS, period_length=CONTROLLER_PERIOD_LENGTH)

        lambda_validator = LambdaValidator(lambda_client)

        # Check Lambda function exists
        assert lambda_validator.function_exists(resource_name)

        # Verify durableConfig was set in the CR spec
        assert cr["spec"]["durableConfig"]["executionTimeout"] == 3600
        assert cr["spec"]["durableConfig"]["retentionPeriodInDays"] == 14

        # Check Lambda function exists and is properly configured
        function = lambda_validator.get_function(resource_name)
        assert function is not None

        # Verify durableConfig was applied to the Lambda function
        assert function["Configuration"]["DurableConfig"]["ExecutionTimeout"] == 3600
        assert function["Configuration"]["DurableConfig"]["RetentionPeriodInDays"] == 14

        # Update durableConfig fields
        cr["spec"]["durableConfig"]["executionTimeout"] = 7200
        cr["spec"]["durableConfig"]["retentionPeriodInDays"] = 30

        # Patch k8s resource
        k8s.patch_custom_resource(ref, cr)
        time.sleep(UPDATE_WAIT_AFTER_SECONDS)

        # Check function updated fields
        function = lambda_validator.get_function(resource_name)
        assert function is not None
        assert function["Configuration"]["DurableConfig"]["ExecutionTimeout"] == 7200
        assert function["Configuration"]["DurableConfig"]["RetentionPeriodInDays"] == 30

        # Delete k8s resource
        _, deleted = k8s.delete_custom_resource(ref, wait_periods=10, period_length=DELETE_WAIT_AFTER_SECONDS)
        assert deleted is True

        # Check Lambda function doesn't exist
        assert not lambda_validator.function_exists(resource_name)

    def test_function_code_signing_in_unsupported_region(self, lambda_client):
        """In regions where AWS Signer is unavailable (e.g. eu-central-2):
        1. A function without codeSigningConfigARN should sync successfully
        2. Adding codeSigningConfigARN should produce a terminal condition
        """
        resource_name = random_suffix_name("lambda-csc-region", 24)

        resources = get_bootstrap_resources()
        logging.debug(resources)

        # Build a minimal inline zip to avoid cross-region S3 issues
        buf = io.BytesIO()
        with ZipFile(buf, 'w') as zf:
            zf.writestr("main.py", "def handler(event, context):\n    return 'hello'\n")
        zip_file_b64 = base64.b64encode(buf.getvalue()).decode("utf-8")

        replacements = REPLACEMENT_VALUES.copy()
        replacements["FUNCTION_NAME"] = resource_name
        replacements["LAMBDA_ROLE"] = resources.BasicRole.arn
        replacements["ZIP_FILE"] = zip_file_b64
        replacements["FUNCTION_REGION"] = "eu-central-2"

        resource_data = load_lambda_resource(
            "function_no_code_signing",
            additional_replacements=replacements,
        )
        logging.debug(resource_data)

        ref = k8s.CustomResourceReference(
            CRD_GROUP, CRD_VERSION, RESOURCE_PLURAL,
            resource_name, namespace="default",
        )
        k8s.create_custom_resource(ref, resource_data)
        cr = k8s.wait_resource_consumed_by_controller(
            ref, wait_periods=CONTROLLER_WAIT_PERIODS, period_length=CONTROLLER_PERIOD_LENGTH
        )

        assert cr is not None
        assert k8s.get_resource_exists(ref)

        time.sleep(CREATE_WAIT_AFTER_SECONDS)

        # Without the fix, sdkFind returns the AccessDeniedException from
        # GetFunctionCodeSigningConfig and the resource stays stuck with
        # ACK.Recoverable=True permanently.
        assert k8s.wait_on_condition(
            ref,
            "ACK.ResourceSynced",
            "True",
            wait_periods=CONTROLLER_WAIT_PERIODS,
            period_length=CONTROLLER_PERIOD_LENGTH,
        )

        # Now patch the function to add a code signing config ARN.
        # PutFunctionCodeSigningConfig will fail with AccessDeniedException
        # because Signer is not available in this region.
        cr = k8s.get_resource(ref)
        cr["spec"]["codeSigningConfigARN"] = "arn:aws:lambda:eu-central-2:123456789012:code-signing-config:csc-does-not-exist"
        k8s.patch_custom_resource(ref, cr)

        time.sleep(UPDATE_WAIT_AFTER_SECONDS)

        cr = k8s.wait_resource_consumed_by_controller(ref, wait_periods=CONTROLLER_WAIT_PERIODS, period_length=CONTROLLER_PERIOD_LENGTH)
        
        # Should get a terminal condition indicating code signing is not available
        assert k8s.assert_condition_state_message(
            ref,
            "ACK.Terminal",
            "True",
            "code signing is not available in this region",
        )

        # Remove the code signing config to allow cleanup
        cr = k8s.get_resource(ref)
        cr["spec"]["codeSigningConfigARN"] = ""
        k8s.patch_custom_resource(ref, cr)

        time.sleep(UPDATE_WAIT_AFTER_SECONDS)

        # Cleanup
        _, deleted = k8s.delete_custom_resource(
            ref, wait_periods=DELETE_WAIT_PERIODS, period_length=DELETE_PERIOD_LENGTH
        )
        assert deleted is True

        time.sleep(DELETE_WAIT_AFTER_SECONDS)

    def test_function_update_code_and_environment_variable(self, lambda_client):
        # Regression test: a single update that changes both the code (sha256)
        # and the configuration (environment variables) must apply BOTH changes.
        # The controller can only issue UpdateFunctionCode or
        # UpdateFunctionConfiguration in one reconcile, so it applies the code
        # change first and must requeue to apply the deferred configuration
        # change instead of prematurely reporting the resource as synced.
        resource_name = random_suffix_name("functionupdatecodeenv", 24)

        resources = get_bootstrap_resources()
        logging.debug(resources)

        archive_1 = open(LAMBDA_FUNCTION_FILE_PATH_ZIP, 'rb')
        readFile_1 = archive_1.read()
        hash_1 = hashlib.sha256(readFile_1)
        binary_hash_1 = hash_1.digest()
        base64_hash_1 = base64.b64encode(binary_hash_1).decode('utf-8')

        archive_2 = open(LAMBDA_FUNCTION_UPDATED_FILE_PATH_ZIP, 'rb')
        readFile_2 = archive_2.read()
        hash_2 = hashlib.sha256(readFile_2)
        binary_hash_2 = hash_2.digest()
        base64_hash_2 = base64.b64encode(binary_hash_2).decode('utf-8')

        replacements = REPLACEMENT_VALUES.copy()
        replacements["FUNCTION_NAME"] = resource_name
        replacements["BUCKET_NAME"] = resources.FunctionsBucket.name
        replacements["LAMBDA_ROLE"] = resources.BasicRole.arn
        replacements["LAMBDA_FILE_NAME"] = LAMBDA_FUNCTION_FILE_ZIP
        replacements["RESERVED_CONCURRENT_EXECUTIONS"] = "0"
        replacements["CODE_SIGNING_CONFIG_ARN"] = ""
        replacements["AWS_REGION"] = get_region()
        replacements["ARCHITECTURES"] = 'x86_64'
        replacements["HASH"] = base64_hash_1

        # Load Lambda CR
        resource_data = load_lambda_resource(
            "function_code_s3",
            additional_replacements=replacements,
        )
        logging.debug(resource_data)

        # Create k8s resource
        ref = k8s.CustomResourceReference(
            CRD_GROUP, CRD_VERSION, RESOURCE_PLURAL,
            resource_name, namespace="default",
        )
        k8s.create_custom_resource(ref, resource_data)
        cr = k8s.wait_resource_consumed_by_controller(ref, wait_periods=CONTROLLER_WAIT_PERIODS, period_length=CONTROLLER_PERIOD_LENGTH)

        assert cr is not None
        assert k8s.get_resource_exists(ref)

        time.sleep(CREATE_WAIT_AFTER_SECONDS)

        cr = k8s.wait_resource_consumed_by_controller(ref, wait_periods=CONTROLLER_WAIT_PERIODS, period_length=CONTROLLER_PERIOD_LENGTH)

        lambda_validator = LambdaValidator(lambda_client)

        # Assert the original code and empty environment are in place
        assert cr["spec"]["code"]["s3Bucket"] == resources.FunctionsBucket.name
        assert cr["spec"]["code"]["s3Key"] == LAMBDA_FUNCTION_FILE_ZIP
        assert cr["spec"].get("environment", {}).get("variables", {}) == {}

        function = lambda_validator.get_function(resource_name)
        assert function is not None
        assert function["Configuration"]["CodeSha256"] == base64_hash_1
        assert function["Configuration"].get("Environment", {}).get("Variables", {}) == {}

        # Update both the code and the environment variables in a single patch
        cr["spec"]["code"]["sha256"] = base64_hash_2
        cr["spec"]["code"]["s3Key"] = LAMBDA_FUNCTION_UPDATED_FILE_ZIP
        cr["spec"]["environment"] = {"variables": {"TEST_ENV_VAR": "test_value"}}

        # Patch k8s resource
        k8s.patch_custom_resource(ref, cr)
        time.sleep(UPDATE_WAIT_AFTER_SECONDS)

        # The code change is applied first, then the environment change is
        # applied on a requeued reconcile. Wait for the controller to converge
        # (this must exceed the code-update requeue interval; a single
        # UPDATE_WAIT_AFTER_SECONDS sleep is not enough to catch a regression).
        assert k8s.wait_on_condition(
            ref,
            "ACK.ResourceSynced",
            "True",
            wait_periods=CONTROLLER_WAIT_PERIODS,
            period_length=CONTROLLER_PERIOD_LENGTH,
        )

        # Both the code and the environment variables must be applied
        function = lambda_validator.get_function(resource_name)
        assert function is not None
        assert function["Configuration"]["CodeSha256"] == base64_hash_2
        assert function["Configuration"].get("Environment", {}).get("Variables", {}) == {
            "TEST_ENV_VAR": "test_value"
        }

        # Delete k8s resource
        _, deleted = k8s.delete_custom_resource(ref, wait_periods=DELETE_WAIT_PERIODS, period_length=DELETE_PERIOD_LENGTH)
        assert deleted is True

        time.sleep(DELETE_WAIT_AFTER_SECONDS)

        # Check Lambda function doesn't exist
        assert not lambda_validator.function_exists(resource_name)
