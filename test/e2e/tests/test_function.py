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

from acktest import tags
from acktest.k8s import resource as k8s

from e2e import service_marker
from e2e.bootstrap_resources import get_bootstrap_resources
from e2e.conftest import Wait
from e2e.service_bootstrap import LAMBDA_FUNCTION_FILE_ZIP
from e2e.tests.helper import LambdaValidator

@service_marker
@pytest.mark.canary
class TestFunction:
    @pytest.mark.function_overrides({'package_type': 'Zip', 'role_type': 'basic'})
    def test_smoke(self, lambda_client, function):
        (ref, cr) = function
        resource_name = cr["spec"]["name"]
        resources = get_bootstrap_resources()

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
        time.sleep(Wait.Function.Update)

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

        time.sleep(Wait.Function.Delete)

        # Check Lambda function doesn't exist
        assert not lambda_validator.function_exists(resource_name)

    @pytest.mark.function_overrides({
        'package_type': 'Zip',
        'role_type': 'basic',
        'reserved_concurrent_executions': 2,
    })
    def test_reserved_concurrent_executions(self, lambda_client, function):
        (ref, cr) = function
        resource_name = cr["spec"]["name"]

        lambda_validator = LambdaValidator(lambda_client)
        # Check Lambda function exists
        assert lambda_validator.function_exists(resource_name)

        reservedConcurrentExecutions = lambda_validator.get_function_concurrency(resource_name)
        assert reservedConcurrentExecutions == 2

        # Update cr
        cr["spec"]["reservedConcurrentExecutions"] = 0

        # Patch k8s resource
        k8s.patch_custom_resource(ref, cr)
        time.sleep(Wait.Function.Update)

        # Check function updated fields
        reservedConcurrentExecutions = lambda_validator.get_function_concurrency(resource_name)
        assert reservedConcurrentExecutions == 0

        # Delete k8s resource
        _, deleted = k8s.delete_custom_resource(ref)
        assert deleted is True

        time.sleep(Wait.Function.Delete)

        # Check Lambda function doesn't exist
        assert not lambda_validator.function_exists(resource_name)

    @pytest.mark.function_overrides({
        'package_type': 'Zip',
        'role_type': 'basic',
        'create_code_signing_config': True,
    })
    def test_function_code_signing_config(self, lambda_client, function):
        (ref, cr) = function
        function_name = cr["spec"]["name"]
        code_signing_config_arn = cr["spec"]["codeSigningConfigARN"]

        lambda_validator = LambdaValidator(lambda_client)
        # Check Lambda function exists
        assert lambda_validator.function_exists(function_name)

        # Check function code signing config is correct
        function_csc_arn = lambda_validator.get_function_code_signing_config(function_name)
        assert function_csc_arn == code_signing_config_arn

        # Delete function code signing config
        cr["spec"]["codeSigningConfigARN"] = ""
        k8s.patch_custom_resource(ref, cr)

        time.sleep(Wait.Function.Update)

        function_csc_arn = lambda_validator.get_function_code_signing_config(function_name)
        assert function_csc_arn is None

        # Delete k8s resource
        _, deleted = k8s.delete_custom_resource(ref)
        assert deleted is True

        time.sleep(Wait.Function.Delete)

        # Check Lambda function doesn't exist
        assert not lambda_validator.function_exists(function_name)

    @pytest.mark.function_overrides({
        'package_type': 'Image',
        'role_type': 'basic',
    })
    def test_function_package_type_image(self, lambda_client, function):
        (ref, cr) = function
        function_name = cr["spec"]["name"]

        lambda_validator = LambdaValidator(lambda_client)
        # Check Lambda function exists
        assert lambda_validator.function_exists(function_name)

        cr["spec"]["timeout"] = 10
        cr["spec"]["ephemeralStorage"] = { "size" : 512 }

        # Patch k8s resource
        k8s.patch_custom_resource(ref, cr)
        time.sleep(Wait.Function.Update)

        # Check function updated fields
        function = lambda_validator.get_function(function_name)
        assert function["Configuration"]["Timeout"] == 10
        assert function["Configuration"]["EphemeralStorage"]["Size"] == 512

        # Delete k8s resource
        _, deleted = k8s.delete_custom_resource(ref)
        assert deleted is True

        time.sleep(Wait.Function.Delete)

        # Check Lambda function doesn't exist
        assert not lambda_validator.function_exists(function_name)

    @pytest.mark.function_overrides({
        'package_type': 'Image',
        'role_type': 'basic',
        'code_signing_config': "random-csc",
    })
    def test_function_package_type_image_with_signing_config(self, lambda_client, function):
        (ref, cr) = function
        function_name = cr["spec"]["name"]

        lambda_validator = LambdaValidator(lambda_client)
        # Check Lambda function exists
        assert lambda_validator.function_exists(function_name)

        # Add invalid signing configuration
        cr["spec"]["codeSigningConfigARN"] = "random-csc"
        k8s.patch_custom_resource(ref, cr)

        time.sleep(Wait.Function.Update)

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

        time.sleep(Wait.Function.Update)
        
        # Delete k8s resource
        _, deleted = k8s.delete_custom_resource(ref)
        assert deleted is True

        time.sleep(Wait.Function.Delete)

        # Check Lambda function doesn't exist
        assert not lambda_validator.function_exists(function_name)
