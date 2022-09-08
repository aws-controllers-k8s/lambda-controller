# Copyright Amazon.com Inc. or its affiliates. All Rights Reserved.
#
# Licensed under the Apache License, Version 2.0 (the "License"). You may
# not use this file except in compliance with the License. A copy of the
# License is located at
#
#	 http://aws.amazon.com/apache2.0/
#
# or in the "license" file accompanying this file. This file is distributed
# on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either
# express or implied. See the License for the specific language governing
# permissions and limitations under the License.

import pytest
import boto3
import logging
import time
from dataclasses import dataclass

from acktest.resources import random_suffix_name
from acktest.aws.identity import get_region, get_account_id
from acktest import k8s as k8sclient
from acktest.k8s import resource as k8s

from e2e import CRD_GROUP, CRD_VERSION, load_lambda_resource
from e2e.service_bootstrap import LAMBDA_FUNCTION_FILE_ZIP
from e2e.replacement_values import REPLACEMENT_VALUES
from e2e.bootstrap_resources import get_bootstrap_resources    

FunctionPackageTypeImage = "Image"
FunctionPackageTypeZip = "Zip"
FunctionTestingRoleBasic = "basic"
FunctionTestingRoleESM = "esm"

@dataclass
class ResourceWaitTimes:
    Create: int
    Update: int
    Delete: int

@dataclass
class ServiceWaitTimes:
    Function: ResourceWaitTimes
    Alias: ResourceWaitTimes
    CodeSigningConfig: ResourceWaitTimes
    EventSourceMapping: ResourceWaitTimes
    FunctionURLConfig: ResourceWaitTimes

Wait = ServiceWaitTimes(
    Function = ResourceWaitTimes(30, 30, 30),
    Alias = ResourceWaitTimes(10, 10, 10),
    CodeSigningConfig = ResourceWaitTimes(10, 10, 10),
    EventSourceMapping = ResourceWaitTimes(20, 20, 20),
    FunctionURLConfig = ResourceWaitTimes(30, 10, 10),
)

def get_testing_image_url():
    aws_region = get_region()
    account_id = get_account_id()
    return f"{account_id}.dkr.ecr.{aws_region}.amazonaws.com/ack-e2e-testing-lambda-controller:v1"

def pytest_addoption(parser):
    parser.addoption("--runslow", action="store_true", default=False, help="run slow tests")

def pytest_configure(config):
    config.addinivalue_line(
        "markers", "canary: mark test to also run in canary tests"
    )
    config.addinivalue_line(
        "markers", "service(arg): mark test associated with a given service"
    )
    config.addinivalue_line(
        "markers", "slow: mark test as slow to run"
    )
    config.addinivalue_line(
        "markers", "function_overrides: function parameters to override when creating fixture"
    )
    config.addinivalue_line(
        "markers", "esm_overrides: event source mapping parameters to override when creating fixture"
    )

def pytest_collection_modifyitems(config, items):
    if config.getoption("--runslow"):
        return
    skip_slow = pytest.mark.skip(reason="need --runslow option to run")
    for item in items:
        if "slow" in item.keywords:
            item.add_marker(skip_slow)

# Provide a k8s client to interact with the integration test cluster
@pytest.fixture(scope='class')
def k8s_client():
    return k8sclient._get_k8s_api_client()

@pytest.fixture(scope='module')
def lambda_client():
    return boto3.client('lambda', region_name=get_region())

@pytest.fixture
def function(request, code_signing_config):
        resource_name = random_suffix_name("function", 24)

        resources = get_bootstrap_resources()
        replacements = REPLACEMENT_VALUES.copy()
        replacements["FUNCTION_NAME"] = resource_name
        replacements["IMAGE_URL"] = get_testing_image_url()
        replacements["CODE_SIGNING_CONFIG_ARN"] = ""
        replacements["RESERVED_CONCURRENT_EXECUTIONS"] = ""

        resource_file = ""

        marker = request.node.get_closest_marker("function_overrides")
        if marker is not None:
            data = marker.args[0]
            if 'package_type' in data:
                if data['package_type'] == FunctionPackageTypeZip:
                    resource_file = "function"
                    resource_name = random_suffix_name("function-zip", 32)
                    replacements["FUNCTION_NAME"] = resource_name
                    replacements["BUCKET_NAME"] = resources.FunctionsBucket.name
                    replacements["LAMBDA_FILE_NAME"] = LAMBDA_FUNCTION_FILE_ZIP
                    replacements["RESERVED_CONCURRENT_EXECUTIONS"] = "0"
                    replacements["CODE_SIGNING_CONFIG_ARN"] = ""
                
                elif data['package_type'] == FunctionPackageTypeImage:
                    resource_file = "function_package_type_image"
                    resource_name = random_suffix_name("function-image", 32)
                    replacements["FUNCTION_NAME"] = resource_name
                    replacements["IMAGE_URL"] = get_testing_image_url()
                    replacements["LAMBDA_ROLE"] = resources.BasicRole.arn

            if 'role_type' in data:
                if data['role_type'] == FunctionTestingRoleBasic:
                    replacements["LAMBDA_ROLE"] = resources.BasicRole.arn
                elif data['role_type'] == FunctionTestingRoleESM:
                    replacements["LAMBDA_ROLE"] = resources.ESMRole.arn
            
            if 'reserved_concurrent_executions' in data:
                replacements["RESERVED_CONCURRENT_EXECUTIONS"] = str(data["reserved_concurrent_executions"])

            if 'code_signing_config_arn' in data:
                replacements["CODE_SIGNING_CONFIG_ARN"] = data['code_signing_config_arn']
            elif 'create_code_signing_config' in data and data["create_code_signing_config"]:
                (_, cr) = code_signing_config
                replacements["CODE_SIGNING_CONFIG_ARN"] = cr["status"]["ackResourceMetadata"]["arn"]

        # Load Function CR
        resource_data = load_lambda_resource(
            resource_file,
            additional_replacements=replacements,
        )
        logging.debug(resource_data)

        # Create k8s resource
        ref = k8s.CustomResourceReference(
            CRD_GROUP, CRD_VERSION, "functions",
            resource_name, namespace="default",
        )
        k8s.create_custom_resource(ref, resource_data)
        time.sleep(Wait.Function.Create)

        cr = k8s.wait_resource_consumed_by_controller(ref)
        assert cr is not None
        assert k8s.get_resource_exists(ref)

        yield (ref, cr)

        # Try to delete, if doesn't already exist
        try:
            _, deleted = k8s.delete_custom_resource(ref, 3, 10)
            assert deleted
        except:
            pass

@pytest.fixture
def code_signing_config():
        resource_name = random_suffix_name("csc", 24)
        resources = get_bootstrap_resources()

        replacements = REPLACEMENT_VALUES.copy()
        replacements["CODE_SIGNING_CONFIG_NAME"] = resource_name
        replacements["SIGNING_PROFILE_VERSION_ARN"] = resources.SigningProfile.signing_profile_arn

        # Load CodeSigningConfig CR
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
        time.sleep(Wait.CodeSigningConfig.Create)

        cr = k8s.wait_resource_consumed_by_controller(ref)
        assert cr is not None
        assert k8s.get_resource_exists(ref)

        yield (ref, cr)

        # Try to delete, if doesn't already exist
        try:
            _, deleted = k8s.delete_custom_resource(ref, 3, 10)
            assert deleted
        except:
            pass

@pytest.fixture
def alias(function):
        (_, function_resource) = function
        lambda_function_name = function_resource["spec"]["name"]

        resource_name = random_suffix_name("alias", 24)

        replacements = REPLACEMENT_VALUES.copy()
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
            CRD_GROUP, CRD_VERSION, "aliases",
            resource_name, namespace="default",
        )
        k8s.create_custom_resource(ref, resource_data)
        time.sleep(Wait.Alias.Create)

        cr = k8s.wait_resource_consumed_by_controller(ref)
        assert cr is not None
        assert k8s.get_resource_exists(ref)

        yield (ref, cr)

        # Try to delete, if doesn't already exist
        try:
            _, deleted = k8s.delete_custom_resource(ref, 3, 10)
            assert deleted
        except:
            pass

@pytest.fixture
def event_source_mapping(request, function):
        (_, function_resource) = function
        lambda_function_name = function_resource["spec"]["name"]

        resource_name = random_suffix_name("eventsourcemapping", 24)
        resources = get_bootstrap_resources()

        replacements = REPLACEMENT_VALUES.copy()
        replacements["EVENT_SOURCE_MAPPING_NAME"] = resource_name
        replacements["FUNCTION_NAME"] = lambda_function_name
        replacements["BATCH_SIZE"] = "10"

        resource_file = ""

        marker = request.node.get_closest_marker("esm_overrides")
        if marker is not None:
            data = marker.args[0]
            if 'source' in data:
                if data['source'] == "sqsqueue":
                    resource_file = "event_source_mapping_sqs"
                    replacements["FUNCTION_NAME"] = lambda_function_name
                    replacements["EVENT_SOURCE_ARN"] = resources.ESMQueue.arn
                    replacements["MAXIMUM_BATCHING_WINDOW_IN_SECONDS"] = "1"

                elif data['source'] == "ddbtable":
                    resource_file = "event_source_mapping_dynamodb"
                    replacements["EVENT_SOURCE_ARN"] = resources.ESMTable.latest_stream_arn
                    replacements["STARTING_POSITION"] = "LATEST"
                    replacements["MAXIMUM_RETRY_ATTEMPTS"] = "-1"

        # Load EventSourceMapping CR
        resource_data = load_lambda_resource(
            resource_file,
            additional_replacements=replacements,
        )

        # Create k8s resource
        ref = k8s.CustomResourceReference(
            CRD_GROUP, CRD_VERSION, "eventsourcemappings",
            resource_name, namespace="default",
        )
        k8s.create_custom_resource(ref, resource_data)
        time.sleep(Wait.EventSourceMapping.Create)

        cr = k8s.wait_resource_consumed_by_controller(ref)
        assert cr is not None
        assert k8s.get_resource_exists(ref)

        yield (ref, cr)

        # Try to delete, if doesn't already exist
        try:
            _, deleted = k8s.delete_custom_resource(ref, 3, 10)
            assert deleted
        except:
            pass

@pytest.fixture
def function_url_config(function):
        (_, function_resource) = function
        lambda_function_name = function_resource["spec"]["name"]

        resource_name = random_suffix_name("functionurlconfig", 24)

        replacements = REPLACEMENT_VALUES.copy()
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
            CRD_GROUP, CRD_VERSION, "functionurlconfigs",
            resource_name, namespace="default",
        )
        k8s.create_custom_resource(ref, resource_data)
        time.sleep(Wait.FunctionURLConfig.Create)

        cr = k8s.wait_resource_consumed_by_controller(ref)
        assert cr is not None
        assert k8s.get_resource_exists(ref)

        yield (ref, cr)

        # Try to delete, if doesn't already exist
        try:
            _, deleted = k8s.delete_custom_resource(ref, 3, 10)
            assert deleted
        except:
            pass