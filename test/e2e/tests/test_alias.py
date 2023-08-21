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

RESOURCE_PLURAL = "aliases"

CREATE_WAIT_AFTER_SECONDS = 30
UPDATE_WAIT_AFTER_SECONDS = 30
DELETE_WAIT_AFTER_SECONDS = 30

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
class TestAlias:
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

        lambda_validator = LambdaValidator(lambda_client)
        # Check alias exists
        assert lambda_validator.alias_exists(resource_name, lambda_function_name)

        cr = k8s.wait_resource_consumed_by_controller(ref)
        
        # Update cr
        cr["spec"]["description"] = ""

        # Patch k8s resource
        k8s.patch_custom_resource(ref, cr)
        time.sleep(UPDATE_WAIT_AFTER_SECONDS)

        # Check alias description
        alias = lambda_validator.get_alias(resource_name, lambda_function_name)
        assert alias is not None
        assert alias["Description"] == ""

        # Delete k8s resource
        _, deleted = k8s.delete_custom_resource(ref)
        assert deleted

        time.sleep(DELETE_WAIT_AFTER_SECONDS)

        # Check alias doesn't exist
        assert not lambda_validator.alias_exists(resource_name, lambda_function_name)

    def test_smoke_ref(self, lambda_client, lambda_function):
        (_, function_resource) = lambda_function
        function_resource_name = function_resource["metadata"]["name"]

        resource_name = random_suffix_name("lambda-alias", 24)

        replacements = REPLACEMENT_VALUES.copy()
        replacements["AWS_REGION"] = get_region()
        replacements["ALIAS_NAME"] = resource_name
        replacements["FUNCTION_REF_NAME"] = function_resource_name
        replacements["FUNCTION_VERSION"] = "$LATEST"

        # Load alias CR
        resource_data = load_lambda_resource(
            "alias-ref",
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

        lambda_validator = LambdaValidator(lambda_client)
        # Check alias exists
        assert lambda_validator.alias_exists(resource_name, function_resource_name)

        cr = k8s.wait_resource_consumed_by_controller(ref)
        
        # Update cr
        cr["spec"]["description"] = ""

        # Patch k8s resource
        k8s.patch_custom_resource(ref, cr)
        time.sleep(UPDATE_WAIT_AFTER_SECONDS)

        # Check alias description
        alias = lambda_validator.get_alias(resource_name, function_resource_name)
        assert alias is not None
        assert alias["Description"] == ""

        # Delete k8s resource
        _, deleted = k8s.delete_custom_resource(ref)
        assert deleted

        time.sleep(DELETE_WAIT_AFTER_SECONDS)

        # Check alias doesn't exist
        assert not lambda_validator.alias_exists(resource_name, function_resource_name)

    def test_provisioned_concurrency_config(self, lambda_client, lambda_function):
        (_, function_resource) = lambda_function
        lambda_function_name = function_resource["spec"]["name"]

        resource_name = random_suffix_name("lambda-alias", 24)

        resources = get_bootstrap_resources()
        logging.debug(resources)

        resp = lambda_client.publish_version(
                FunctionName = lambda_function_name
        )
        version = resp['Version']
    
        replacements = REPLACEMENT_VALUES.copy()
        replacements["AWS_REGION"] = get_region()
        replacements["ALIAS_NAME"] = resource_name
        replacements["FUNCTION_NAME"] = lambda_function_name
        replacements["FUNCTION_VERSION"] = f"\'{version}\'"
        replacements["PROVISIONED_CONCURRENT_EXECUTIONS"] = "1"

        # Load alias CR
        resource_data = load_lambda_resource(
            "alias_provisioned_concurrency",
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

         # Check alias exists
        assert lambda_validator.alias_exists(resource_name, lambda_function_name)
        
        # Update provisioned_concurrency
        cr["spec"]["provisionedConcurrencyConfig"]["provisionedConcurrentExecutions"] = 2

        # Patch k8s resource
        k8s.patch_custom_resource(ref, cr)
        time.sleep(UPDATE_WAIT_AFTER_SECONDS)

        #Check provisioned_concurrency_config update fields
        provisioned_concurrency_config = lambda_validator.get_provisioned_concurrency_config(lambda_function_name,resource_name)
        assert provisioned_concurrency_config["RequestedProvisionedConcurrentExecutions"] == 2

        # Delete provisioned_concurrency from alias
        cr = k8s.wait_resource_consumed_by_controller(ref)
        cr["spec"]["provisionedConcurrencyConfig"] = None

        # Patch k8s resource     
        k8s.patch_custom_resource(ref, cr)
        time.sleep(UPDATE_WAIT_AFTER_SECONDS)

        #Check provisioned_concurrency_config is deleted
        assert not lambda_validator.get_provisioned_concurrency_config(lambda_function_name, resource_name)

        # Delete k8s resource
        _, deleted = k8s.delete_custom_resource(ref)
        assert deleted

        time.sleep(DELETE_WAIT_AFTER_SECONDS)

        # Check alias doesn't exist
        assert not lambda_validator.alias_exists(resource_name, lambda_function_name)
    
    def test_function_event_invoke_config(self, lambda_client, lambda_function):
        (_, function_resource) = lambda_function
        lambda_function_name = function_resource["spec"]["name"]

        resource_name = random_suffix_name("lambda-alias", 24)

        resources = get_bootstrap_resources()
        logging.debug(resources)

        replacements = REPLACEMENT_VALUES.copy()
        replacements["AWS_REGION"] = get_region()
        replacements["ALIAS_NAME"] = resource_name
        replacements["FUNCTION_VERSION"] = "$LATEST"
        replacements["FUNCTION_NAME"] = lambda_function_name
        replacements["MAXIMUM_EVENT_AGE_IN_SECONDS"] = "100"
        replacements["MAXIMUM_RETRY_ATTEMPTS"] = "1"
        replacements["ON_SUCCESS_DESTINATION"] = resources.EICQueueOnSuccess.arn
        replacements["ON_FAILURE_DESTINATION"] = resources.EICQueueOnFailure.arn

        # Load alias CR
        resource_data = load_lambda_resource(
            "alias_event_invoke_config",
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

         # Check alias exists
        assert lambda_validator.alias_exists(resource_name, lambda_function_name)
        
        # Update cr
        cr["spec"]["functionEventInvokeConfig"]["maximumEventAgeInSeconds"] = 200
        cr["spec"]["functionEventInvokeConfig"]["maximumRetryAttempts"] = 2

        # Patch k8s resource
        k8s.patch_custom_resource(ref, cr)
        time.sleep(UPDATE_WAIT_AFTER_SECONDS)

        #Check function_event_invoke_config update fields
        function_event_invoke_config = lambda_validator.get_function_event_invoke_config_alias(lambda_function_name,resource_name)
        assert function_event_invoke_config["MaximumEventAgeInSeconds"] == 200
        assert function_event_invoke_config["MaximumRetryAttempts"] == 2

        # Delete k8s resource
        _, deleted = k8s.delete_custom_resource(ref)
        assert deleted

        time.sleep(DELETE_WAIT_AFTER_SECONDS)

        # Check alias doesn't exist
        assert not lambda_validator.alias_exists(resource_name, lambda_function_name)