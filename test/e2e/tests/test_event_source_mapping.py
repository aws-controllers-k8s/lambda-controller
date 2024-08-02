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

"""Integration tests for the Lambda event source mapping API.
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

RESOURCE_PLURAL = "eventsourcemappings"

CREATE_WAIT_AFTER_SECONDS = 20
UPDATE_WAIT_AFTER_SECONDS = 20
DELETE_WAIT_AFTER_SECONDS = 20
TESTING_NAMESPACE = random_suffix_name("testing-esm-namespace", 28)

log = logging.getLogger()

@pytest.fixture(scope="function")
def lambda_function(request):
    resource_name = random_suffix_name("lambda-function", 24) 
    resources = get_bootstrap_resources()

    marker = request.node.get_closest_marker("resource_data")
    filename = "function"
    namespace = "default"

    replacements = REPLACEMENT_VALUES.copy()

    if marker is not None:
        data = marker.args[0]
        if 'withNamespace' in data and data['withNamespace']:
            filename = "function_namespace"
            namespace = TESTING_NAMESPACE
            replacements['FUNCTION_NAMESPACE'] = namespace
            k8s.create_k8s_namespace(
                namespace
            )
            time.sleep(CREATE_WAIT_AFTER_SECONDS)
            time.sleep(CREATE_WAIT_AFTER_SECONDS)
            time.sleep(CREATE_WAIT_AFTER_SECONDS)
    
    replacements["FUNCTION_NAME"] = resource_name
    replacements["BUCKET_NAME"] = resources.FunctionsBucket.name
    replacements["LAMBDA_ROLE"] = resources.ESMRole.arn
    replacements["LAMBDA_FILE_NAME"] = LAMBDA_FUNCTION_FILE_ZIP
    replacements["RESERVED_CONCURRENT_EXECUTIONS"] = "10"
    replacements["CODE_SIGNING_CONFIG_ARN"] = ""
    replacements["AWS_REGION"] = get_region()

    # Load function CR
    resource_data = load_lambda_resource(
        filename,
        additional_replacements=replacements,
    )
    logging.debug(resource_data)

    # Create k8s resource
    function_reference = k8s.CustomResourceReference(
        CRD_GROUP, CRD_VERSION, "functions",
        resource_name, namespace=namespace,
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
class TestEventSourceMapping:
    # @pytest.mark.resource_data({'withNamespace': False})
    # def test_smoke_sqs_queue_stream(self, lambda_client, lambda_function):
    #     (_, function_resource) = lambda_function
    #     lambda_function_name = function_resource["spec"]["name"]

    #     resource_name = random_suffix_name("lambda-esm", 24)
    #     resources = get_bootstrap_resources()

    #     replacements = REPLACEMENT_VALUES.copy()
    #     replacements["AWS_REGION"] = get_region()
    #     replacements["EVENT_SOURCE_MAPPING_NAME"] = resource_name
    #     replacements["BATCH_SIZE"] = "10"
    #     replacements["FUNCTION_NAME"] = lambda_function_name
    #     replacements["EVENT_SOURCE_ARN"] = resources.ESMQueue.arn
    #     replacements["MAXIMUM_BATCHING_WINDOW_IN_SECONDS"] = "1"

    #     # Load ESM CR
    #     resource_data = load_lambda_resource(
    #         "event_source_mapping_sqs",
    #         additional_replacements=replacements,
    #     )
    #     logging.debug(resource_data)

    #     # Create k8s resource
    #     ref = k8s.CustomResourceReference(
    #         CRD_GROUP, CRD_VERSION, RESOURCE_PLURAL,
    #         resource_name, namespace="default",
    #     )
    #     k8s.create_custom_resource(ref, resource_data)
    #     cr = k8s.wait_resource_consumed_by_controller(ref)

    #     assert cr is not None
    #     assert k8s.get_resource_exists(ref)

    #     time.sleep(CREATE_WAIT_AFTER_SECONDS)

    #     esm_uuid = cr['status']['uuid']

    #     lambda_validator = LambdaValidator(lambda_client)
    #     # Check ESM exists
    #     assert lambda_validator.event_source_mapping_exists(esm_uuid)

    #     # Update cr
    #     cr["spec"]["batchSize"] = 20
    #     cr["spec"]["filterCriteria"] = {
    #         "filters": [
    #             {
    #                 "pattern": "{\"controller-version\":[\"v1\"]}"
    #             },
    #         ]
    #     }
    #     cr["spec"]["scalingConfig"] = {"maximumConcurrency": 4}

    #     # Patch k8s resource
    #     k8s.patch_custom_resource(ref, cr)
    #     time.sleep(UPDATE_WAIT_AFTER_SECONDS)

    #     # Check ESM batch size & filters
    #     esm = lambda_validator.get_event_source_mapping(esm_uuid)
    #     assert esm is not None
    #     assert esm["BatchSize"] == 20
    #     assert esm["FilterCriteria"]["Filters"] == [
    #         {
    #             "Pattern": "{\"controller-version\":[\"v1\"]}"
    #         },
    #     ]
    #     assert esm["ScalingConfig"]["MaximumConcurrency"] == 4


    #     # Delete the filterCriteria field
    #     cr = k8s.wait_resource_consumed_by_controller(ref)
    #     cr["spec"]["filterCriteria"] = None

    #     # Patch k8s resource
    #     k8s.patch_custom_resource(ref, cr)
    #     time.sleep(UPDATE_WAIT_AFTER_SECONDS)

    #     # Check filters have been deleted
    #     esm = lambda_validator.get_event_source_mapping(esm_uuid)
    #     assert esm is not None
    #     assert "FilterCriteria" not in esm

    #     # Delete k8s resource
    #     _, deleted = k8s.delete_custom_resource(ref)
    #     assert deleted

    #     time.sleep(DELETE_WAIT_AFTER_SECONDS)

    #     # Check ESM doesn't exist
    #     assert not lambda_validator.event_source_mapping_exists(esm_uuid)
    
    # @pytest.mark.resource_data({'withNamespace': False})
    # def test_smoke_sqs_queue_stream_ref(self, lambda_client, lambda_function):
    #     (_, function_resource) = lambda_function
    #     function_resource_name = function_resource["metadata"]["name"]

    #     resource_name = random_suffix_name("lambda-esm", 24)
    #     resources = get_bootstrap_resources()

    #     replacements = REPLACEMENT_VALUES.copy()
    #     replacements["AWS_REGION"] = get_region()
    #     replacements["EVENT_SOURCE_MAPPING_NAME"] = resource_name
    #     replacements["BATCH_SIZE"] = "10"
    #     replacements["FUNCTION_REF_NAME"] = function_resource_name
    #     replacements["EVENT_SOURCE_ARN"] = resources.ESMQueue.arn
    #     replacements["MAXIMUM_BATCHING_WINDOW_IN_SECONDS"] = "1"

    #     # Load ESM CR
    #     resource_data = load_lambda_resource(
    #         "event_source_mapping_sqs_ref",
    #         additional_replacements=replacements,
    #     )
    #     logging.debug(resource_data)

    #     # Create k8s resource
    #     ref = k8s.CustomResourceReference(
    #         CRD_GROUP, CRD_VERSION, RESOURCE_PLURAL,
    #         resource_name, namespace="default",
    #     )
    #     k8s.create_custom_resource(ref, resource_data)
    #     cr = k8s.wait_resource_consumed_by_controller(ref)

    #     assert cr is not None
    #     assert k8s.get_resource_exists(ref)

    #     time.sleep(CREATE_WAIT_AFTER_SECONDS)

    #     esm_uuid = cr['status']['uuid']

    #     lambda_validator = LambdaValidator(lambda_client)
    #     # Check ESM exists
    #     assert lambda_validator.event_source_mapping_exists(esm_uuid)

    #     # Update cr
    #     cr["spec"]["batchSize"] = 20
    #     cr["spec"]["filterCriteria"] = {
    #         "filters": [
    #             {
    #                 "pattern": "{\"controller-version\":[\"v1\"]}"
    #             },
    #         ]
    #     }

    #     # Patch k8s resource
    #     k8s.patch_custom_resource(ref, cr)
    #     time.sleep(UPDATE_WAIT_AFTER_SECONDS)

    #     # Check ESM batch size & filters
    #     esm = lambda_validator.get_event_source_mapping(esm_uuid)
    #     assert esm is not None
    #     assert esm["BatchSize"] == 20
    #     assert esm["FilterCriteria"]["Filters"] == [
    #         {
    #             "Pattern": "{\"controller-version\":[\"v1\"]}"
    #         },
    #     ]

    #     # Delete the filterCriteria field
    #     cr = k8s.wait_resource_consumed_by_controller(ref)
    #     cr["spec"]["filterCriteria"] = None

    #     # Patch k8s resource
    #     k8s.patch_custom_resource(ref, cr)
    #     time.sleep(UPDATE_WAIT_AFTER_SECONDS)

    #     # Check filters have been deleted
    #     esm = lambda_validator.get_event_source_mapping(esm_uuid)
    #     assert esm is not None
    #     assert "FilterCriteria" not in esm

    #     # Delete k8s resource
    #     _, deleted = k8s.delete_custom_resource(ref)
    #     assert deleted

    #     time.sleep(DELETE_WAIT_AFTER_SECONDS)

    #     # Check ESM doesn't exist
    #     assert not lambda_validator.event_source_mapping_exists(esm_uuid)

    @pytest.mark.resource_data({'withNamespace': True})
    def test_smoke_sqs_queue_stream_namespace_ref(self, lambda_client, lambda_function):
        (_, function_resource) = lambda_function
        function_resource_name = function_resource["metadata"]["name"]

        resource_name = random_suffix_name("lambda-esm", 24)
        resources = get_bootstrap_resources()

        replacements = REPLACEMENT_VALUES.copy()
        replacements["AWS_REGION"] = get_region()
        replacements["EVENT_SOURCE_MAPPING_NAME"] = resource_name
        replacements["BATCH_SIZE"] = "10"
        replacements["FUNCTION_REF_NAME"] = function_resource_name
        replacements["EVENT_SOURCE_ARN"] = resources.ESMQueue.arn
        replacements["MAXIMUM_BATCHING_WINDOW_IN_SECONDS"] = "1"
        replacements["FUNCTION_REF_NAMESPACE"] = TESTING_NAMESPACE

        # Load ESM CR
        resource_data = load_lambda_resource(
            "event_source_mapping_sqs_ref_namespace",
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

        logging.error("THISISSOMETHING "+ cr['status'])

        esm_uuid = cr['status']['uuid']

        lambda_validator = LambdaValidator(lambda_client)
        # Check ESM exists
        assert lambda_validator.event_source_mapping_exists(esm_uuid)

        # Update cr
        cr["spec"]["batchSize"] = 20
        cr["spec"]["filterCriteria"] = {
            "filters": [
                {
                    "pattern": "{\"controller-version\":[\"v1\"]}"
                },
            ]
        }

        # Patch k8s resource
        k8s.patch_custom_resource(ref, cr)
        time.sleep(UPDATE_WAIT_AFTER_SECONDS)

        # Check ESM batch size & filters
        esm = lambda_validator.get_event_source_mapping(esm_uuid)
        assert esm is not None
        assert esm["BatchSize"] == 20
        assert esm["FilterCriteria"]["Filters"] == [
            {
                "Pattern": "{\"controller-version\":[\"v1\"]}"
            },
        ]

        # Delete the filterCriteria field
        cr = k8s.wait_resource_consumed_by_controller(ref)
        cr["spec"]["filterCriteria"] = None

        # Patch k8s resource
        k8s.patch_custom_resource(ref, cr)
        time.sleep(UPDATE_WAIT_AFTER_SECONDS)

        # Check filters have been deleted
        esm = lambda_validator.get_event_source_mapping(esm_uuid)
        assert esm is not None
        assert "FilterCriteria" not in esm

        # Delete k8s resource
        _, deleted = k8s.delete_custom_resource(ref)
        assert deleted

        time.sleep(DELETE_WAIT_AFTER_SECONDS)

        # Check ESM doesn't exist
        assert not lambda_validator.event_source_mapping_exists(esm_uuid)

    # @pytest.mark.resource_data({'withNamespace': False})
    # def test_smoke_dynamodb_table_stream(self, lambda_client, lambda_function):
    #     (_, function_resource) = lambda_function
    #     lambda_function_name = function_resource["spec"]["name"]

    #     resource_name = random_suffix_name("lambda-esm", 24)
    #     resources = get_bootstrap_resources()

    #     replacements = REPLACEMENT_VALUES.copy()
    #     replacements["AWS_REGION"] = get_region()
    #     replacements["EVENT_SOURCE_MAPPING_NAME"] = resource_name
    #     replacements["BATCH_SIZE"] = "10"
    #     replacements["FUNCTION_NAME"] = lambda_function_name
    #     replacements["EVENT_SOURCE_ARN"] = resources.ESMTable.latest_stream_arn
    #     replacements["STARTING_POSITION"] = "LATEST"
    #     replacements["MAXIMUM_RETRY_ATTEMPTS"] = "-1"

    #     # Load ESM CR
    #     resource_data = load_lambda_resource(
    #         "event_source_mapping_dynamodb",
    #         additional_replacements=replacements,
    #     )
    #     logging.debug(resource_data)

    #     # Create k8s resource
    #     ref = k8s.CustomResourceReference(
    #         CRD_GROUP, CRD_VERSION, RESOURCE_PLURAL,
    #         resource_name, namespace="default",
    #     )
    #     k8s.create_custom_resource(ref, resource_data)
    #     cr = k8s.wait_resource_consumed_by_controller(ref)

    #     assert cr is not None
    #     assert k8s.get_resource_exists(ref)

    #     time.sleep(CREATE_WAIT_AFTER_SECONDS)

    #     esm_uuid = cr['status']['uuid']

    #     lambda_validator = LambdaValidator(lambda_client)
    #     # Check ESM exists
    #     assert lambda_validator.event_source_mapping_exists(esm_uuid)

    #     # Update cr
    #     cr["spec"]["maximumRetryAttempts"] = 3
    #     cr["spec"]["destinationConfig"] = {
    #         'onFailure': {
    #             'destination': resources.ESMQueue.arn,
    #         }
    #     }

    #     # Patch k8s resource
    #     k8s.patch_custom_resource(ref, cr)
    #     time.sleep(UPDATE_WAIT_AFTER_SECONDS)

    #     # Check ESM maximum retry attempts
    #     esm = lambda_validator.get_event_source_mapping(esm_uuid)
    #     assert esm is not None
    #     logging.info(esm)
    #     assert esm["MaximumRetryAttempts"] == 3

    #     # Delete k8s resource
    #     _, deleted = k8s.delete_custom_resource(ref)
    #     assert deleted

    #     time.sleep(DELETE_WAIT_AFTER_SECONDS)

    #     # Check ESM doesn't exist
    #     assert not lambda_validator.event_source_mapping_exists(esm_uuid)