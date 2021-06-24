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

"""Integration tests for the AmazonMQ API Broker resource
"""

import pytest
import boto3
import datetime
import logging
import time
from typing import Dict

from acktest.resources import random_suffix_name
from acktest.k8s import resource as k8s

from e2e import service_marker, CRD_GROUP, CRD_VERSION, load_mq_resource
from e2e.replacement_values import REPLACEMENT_VALUES

RESOURCE_PLURAL = 'brokers'

DELETE_WAIT_INTERVAL_SLEEP_SECONDS = 15
DELETE_WAIT_AFTER_SECONDS = 120
# It often takes >2 minutes from calling DeleteBroker to the broker record no
# longer appearing in the AMQ API...
DELETE_TIMEOUT_SECONDS = 300

CREATE_INTERVAL_SLEEP_SECONDS = 30
# Time to wait before we get to an expected RUNNING state.
# In my experience, it regularly takes more than 6 minutes to create a
# single-instance RabbitMQ broker...
CREATE_TIMEOUT_SECONDS = 900


@pytest.fixture(scope="module")
def amq_client():
    return boto3.client('mq')


#TODO(a-hilaly): Move to test-infra
def wait_for_cr_status(
    reference: k8s.CustomResourceReference,
    status_field: str,
    desired_status: str,
    wait_periods: int,
    period_length: int,
):
    """
    Waits for the specified condition in CR status to reach the desired value.
    """
    actual_status = None
    for i in range(wait_periods):
        time.sleep(period_length)
        resource = k8s.get_resource(reference)
        actual_status = resource["status"][status_field]
        if actual_status == desired_status:
            break

    else:
        logging.error(
            f"Wait for status: {desired_status} timed out. Actual status: {actual_status}"
        )

    assert actual_status == desired_status


@pytest.fixture(scope="module")
def admin_user_pass_secret():
    ns = "default"
    name = "dbsecrets"
    key = "admin_user_password"
    secret_val = "adminpassneeds12chars"
    k8s.create_opaque_secret(ns, name, key, secret_val)
    yield ns, name, key
    k8s.delete_secret(ns, name)


@service_marker
@pytest.mark.canary
class TestRabbitMQBroker:
    def test_create_delete_non_public(
            self,
            amq_client,
            admin_user_pass_secret,
    ):
        resource_name = random_suffix_name("my-rabbit-broker-non-public", 32)
        aup_sec_ns, aup_sec_name, aup_sec_key = admin_user_pass_secret

        replacements = REPLACEMENT_VALUES.copy()
        replacements["BROKER_NAME"] = resource_name
        replacements["ADMIN_USER_PASS_SECRET_NAMESPACE"] = aup_sec_ns
        replacements["ADMIN_USER_PASS_SECRET_NAME"] = aup_sec_name
        replacements["ADMIN_USER_PASS_SECRET_KEY"] = aup_sec_key

        resource_data = load_mq_resource(
            "broker_rabbitmq_non_public",
            additional_replacements=replacements,
        )
        logging.error(resource_data)

        # Create the k8s resource
        ref = k8s.CustomResourceReference(
            CRD_GROUP, CRD_VERSION, RESOURCE_PLURAL,
            resource_name, namespace="default",
        )
        k8s.create_custom_resource(ref, resource_data)
        cr = k8s.wait_resource_consumed_by_controller(ref)

        assert cr is not None

        broker_id = cr['status']['brokerID']

        # Let's check that the Broker appears in AmazonMQ
        aws_res = amq_client.describe_broker(BrokerId=broker_id)
        assert aws_res is not None

        wait_for_cr_status(
            ref,
            "brokerState",
            "RUNNING",
            CREATE_INTERVAL_SLEEP_SECONDS,
            45,
        )

        # At this point, there should be at least one BrokerInstance record in
        # the Broker.Status.BrokerInstances collection which we can grab an
        # endpoint from.
        latest_res = k8s.get_resource(ref)
        assert latest_res['status']['brokerInstances'] is not None
        assert len(latest_res['status']['brokerInstances']) == 1
        assert len(latest_res['status']['brokerInstances'][0]['endpoints']) > 0

        # Delete the k8s resource on teardown of the module
        k8s.delete_custom_resource(ref)

        time.sleep(DELETE_WAIT_AFTER_SECONDS)

        now = datetime.datetime.now()
        timeout = now + datetime.timedelta(seconds=DELETE_TIMEOUT_SECONDS)

        # Broker should no longer appear in AmazonMQ
        while True:
            if datetime.datetime.now() >= timeout:
                pytest.fail("Timed out waiting for ES Domain to being deleted in AES API")
            time.sleep(DELETE_WAIT_INTERVAL_SLEEP_SECONDS)

            try:
                aws_res = amq_client.describe_broker(BrokerId=broker_id)
                if aws_res['BrokerState'] != "DELETION_IN_PROGRESS":
                    pytest.fail("BrokerState is not DELETION_IN_PROGRESS for broker that was deleted. BrokerState is "+aws_res['BrokerState'])
            except amq_client.exceptions.NotFoundException:
                break
