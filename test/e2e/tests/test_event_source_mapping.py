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

from acktest.k8s import resource as k8s

from e2e import service_marker
from e2e.conftest import Wait
from e2e.bootstrap_resources import get_bootstrap_resources
from e2e.tests.helper import LambdaValidator

@service_marker
@pytest.mark.canary
class TestEventSourceMapping:
    @pytest.mark.function_overrides({
        'package_type': 'Zip',
        'role_type': 'esm',
    })
    @pytest.mark.esm_overrides({'source': 'sqsqueue'})
    def test_smoke_sqs_queue_stream(self, lambda_client, event_source_mapping):
        (ref, cr) = event_source_mapping
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
        time.sleep(Wait.EventSourceMapping.Update)

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
        time.sleep(Wait.EventSourceMapping.Update)

        # Check filters have been deleted
        esm = lambda_validator.get_event_source_mapping(esm_uuid)
        assert esm is not None
        assert "FilterCriteria" not in esm

        # Delete k8s resource
        _, deleted = k8s.delete_custom_resource(ref)
        assert deleted

        time.sleep(Wait.EventSourceMapping.Delete)

        # Check ESM doesn't exist
        assert not lambda_validator.event_source_mapping_exists(esm_uuid)

    @pytest.mark.function_overrides({
        'package_type': 'Zip',
        'role_type': 'esm',
    })
    @pytest.mark.esm_overrides({'source': 'ddbtable'})
    def test_smoke_dynamodb_table_stream(self, lambda_client, event_source_mapping):
        (ref, cr) = event_source_mapping
        esm_uuid = cr['status']['uuid']
        resources = get_bootstrap_resources()

        lambda_validator = LambdaValidator(lambda_client)
        # Check ESM exists
        assert lambda_validator.event_source_mapping_exists(esm_uuid)

        # Update cr
        cr["spec"]["maximumRetryAttempts"] = 3
        cr["spec"]["destinationConfig"] = {
            'onFailure': {
                'destination': resources.ESMQueue.arn,
            }
        }

        # Patch k8s resource
        k8s.patch_custom_resource(ref, cr)
        time.sleep(Wait.EventSourceMapping.Update)

        # Check ESM maximum retry attempts
        esm = lambda_validator.get_event_source_mapping(esm_uuid)
        assert esm is not None
        assert esm["MaximumRetryAttempts"] == 3

        # Delete k8s resource
        _, deleted = k8s.delete_custom_resource(ref)
        assert deleted

        time.sleep(Wait.EventSourceMapping.Delete)

        # Check ESM doesn't exist
        assert not lambda_validator.event_source_mapping_exists(esm_uuid)