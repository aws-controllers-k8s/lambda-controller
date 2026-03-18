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
        cr = k8s.wait_resource_consumed_by_controller(ref)

        assert cr is not None
        assert k8s.get_resource_exists(ref)

        time.sleep(CREATE_WAIT_AFTER_SECONDS)

        cr = k8s.wait_resource_consumed_by_controller(ref)

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
        _, deleted = k8s.delete_custom_resource(ref)
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
        cr = k8s.wait_resource_consumed_by_controller(ref)

        assert cr is not None
        assert k8s.get_resource_exists(ref)

        time.sleep(CREATE_WAIT_AFTER_SECONDS)

        cr = k8s.wait_resource_consumed_by_controller(ref)

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
        _, deleted = k8s.delete_custom_resource(ref)
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
        cr = k8s.wait_resource_consumed_by_controller(ref)

        assert cr is not None
        assert k8s.get_resource_exists(ref)

        time.sleep(CREATE_WAIT_AFTER_SECONDS)

        cr = k8s.wait_resource_consumed_by_controller(ref)

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
        _, deleted = k8s.delete_custom_resource(ref)
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
        cr = k8s.wait_resource_consumed_by_controller(ref)

        assert cr is not None
        assert k8s.get_resource_exists(ref)

        time.sleep(CREATE_WAIT_AFTER_SECONDS)

        cr = k8s.wait_resource_consumed_by_controller(ref)

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
        _, deleted = k8s.delete_custom_resource(ref)
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
        cr = k8s.wait_resource_consumed_by_controller(ref)

        assert cr is not None
        assert k8s.get_resource_exists(ref)

        time.sleep(CREATE_WAIT_AFTER_SECONDS)

        cr = k8s.wait_resource_consumed_by_controller(ref)

        lambda_validator = LambdaValidator(lambda_client)
        # Check Lambda function exists
        assert lambda_validator.function_exists(resource_name)

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
        cr = k8s.wait_resource_consumed_by_controller(ref)

        assert cr is not None
        assert k8s.get_resource_exists(ref)

        time.sleep(CREATE_WAIT_AFTER_SECONDS*3)

        cr = k8s.wait_resource_consumed_by_controller(ref)

        lambda_validator = LambdaValidator(lambda_client)
        # Check Lambda function exists
        assert lambda_validator.function_exists(resource_name)

        assert cr["status"]["state"] == "Active"

        function = lambda_validator.get_function(resource_name)
        assert function is not None
        assert function["Configuration"]["State"] == "Active"

        # Delete k8s resource
        _, deleted = k8s.delete_custom_resource(ref)
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
        cr = k8s.wait_resource_consumed_by_controller(ref)

        assert cr is not None
        assert k8s.get_resource_exists(ref)

        time.sleep(CREATE_WAIT_AFTER_SECONDS)

        cr = k8s.wait_resource_consumed_by_controller(ref)

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
        _, deleted = k8s.delete_custom_resource(ref)
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
        cr = k8s.wait_resource_consumed_by_controller(ref)

        assert cr is not None
        assert k8s.get_resource_exists(ref)

        time.sleep(CREATE_WAIT_AFTER_SECONDS)

        cr = k8s.wait_resource_consumed_by_controller(ref)

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
        _, deleted = k8s.delete_custom_resource(ref)
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
        cr = k8s.wait_resource_consumed_by_controller(ref)

        assert cr is not None
        assert k8s.get_resource_exists(ref)

        time.sleep(CREATE_WAIT_AFTER_SECONDS)

        cr = k8s.wait_resource_consumed_by_controller(ref)

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
        _, deleted = k8s.delete_custom_resource(ref)
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
        cr = k8s.wait_resource_consumed_by_controller(ref)

        assert cr is not None
        assert k8s.get_resource_exists(ref)

        time.sleep(CREATE_WAIT_AFTER_SECONDS)

        cr = k8s.wait_resource_consumed_by_controller(ref)

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
        _, deleted = k8s.delete_custom_resource(ref)
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
        cr = k8s.wait_resource_consumed_by_controller(ref)

        assert cr is not None
        assert k8s.get_resource_exists(ref)

        time.sleep(CREATE_WAIT_AFTER_SECONDS)

        cr = k8s.wait_resource_consumed_by_controller(ref)

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
        _, deleted = k8s.delete_custom_resource(ref)
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
        cr = k8s.wait_resource_consumed_by_controller(ref)

        assert cr is not None
        assert k8s.get_resource_exists(ref)

        time.sleep(CREATE_WAIT_AFTER_SECONDS)

        cr = k8s.wait_resource_consumed_by_controller(ref)

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
        cr = k8s.wait_resource_consumed_by_controller(ref)
        cr["spec"]["functionEventInvokeConfig"] =  None

        # Patch k8s resource
        k8s.patch_custom_resource(ref, cr)
        time.sleep(UPDATE_WAIT_AFTER_SECONDS)

        # Check if FunctionEventInvokeConfig is deleted
        assert not lambda_validator.get_function_event_invoke_config(resource_name)

        # Delete k8s resource
        _, deleted = k8s.delete_custom_resource(ref)
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
        cr = k8s.wait_resource_consumed_by_controller(ref)

        assert cr is not None
        assert k8s.get_resource_exists(ref)

        time.sleep(CREATE_WAIT_AFTER_SECONDS)

        cr = k8s.wait_resource_consumed_by_controller(ref)

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
        _, deleted = k8s.delete_custom_resource(ref)
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
        cr = k8s.wait_resource_consumed_by_controller(ref)

        assert cr is not None
        assert k8s.get_resource_exists(ref)

        time.sleep(CREATE_WAIT_AFTER_SECONDS)

        cr = k8s.wait_resource_consumed_by_controller(ref)

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
        _, deleted = k8s.delete_custom_resource(ref)
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
        cr = k8s.wait_resource_consumed_by_controller(ref)

        assert cr is not None
        assert k8s.get_resource_exists(ref)

        time.sleep(CREATE_WAIT_AFTER_SECONDS)

        cr = k8s.wait_resource_consumed_by_controller(ref)

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
        _, deleted = k8s.delete_custom_resource(ref)
        assert deleted is True

        time.sleep(DELETE_WAIT_AFTER_SECONDS)

        # Check Lambda function doesn't exist
        assert not lambda_validator.function_exists(resource_name)