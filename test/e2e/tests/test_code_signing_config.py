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

"""Integration tests for the Lambda code signing config API.
"""

import boto3
import pytest
import time
import logging
from typing import Dict, Tuple

from acktest.resources import random_suffix_name
from acktest.aws.identity import get_region
from acktest.k8s import resource as k8s
from e2e import service_marker, CRD_GROUP, CRD_VERSION, load_lambda_resource
from e2e.replacement_values import REPLACEMENT_VALUES
from e2e.bootstrap_resources import get_bootstrap_resources

RESOURCE_PLURAL = "codesigningconfigs"

CREATE_WAIT_AFTER_SECONDS = 10
UPDATE_WAIT_AFTER_SECONDS = 10
DELETE_WAIT_AFTER_SECONDS = 10

@pytest.fixture(scope="module")
def lambda_client():
    return boto3.client("lambda")

@service_marker
@pytest.mark.canary
class TestCodeSigningConfig:
    def get_code_signing_config(self, lambda_client, code_signing_config_arn: str) -> dict:
        try:
            resp = lambda_client.get_code_signing_config(
                CodeSigningConfigArn=code_signing_config_arn,
            )
            return resp["CodeSigningConfig"]

        except Exception as e:
            logging.debug(e)
            return None

    def code_signing_config_exists(self, lambda_client, code_signing_config_arn: str) -> bool:
        return self.get_code_signing_config(lambda_client, code_signing_config_arn) is not None

    def test_smoke(self, lambda_client):
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
            CRD_GROUP, CRD_VERSION, RESOURCE_PLURAL,
            resource_name, namespace="default",
        )
        k8s.create_custom_resource(ref, resource_data)
        cr = k8s.wait_resource_consumed_by_controller(ref)

        assert cr is not None
        assert k8s.get_resource_exists(ref)

        codeSigningConfigARN = cr['status']['ackResourceMetadata']['arn']

        time.sleep(CREATE_WAIT_AFTER_SECONDS)

        # Check Lambda code signing config exists
        exists = self.code_signing_config_exists(lambda_client, codeSigningConfigARN)
        assert exists

        # Update cr
        cr["spec"]["description"] = "new description"

        # Patch k8s resource
        k8s.patch_custom_resource(ref, cr)
        time.sleep(UPDATE_WAIT_AFTER_SECONDS)

        # Check code signing config  description
        csc = self.get_code_signing_config(lambda_client, codeSigningConfigARN)
        assert csc is not None
        assert csc["Description"] == "new description"

        # Delete k8s resource
        _, deleted = k8s.delete_custom_resource(ref)
        assert deleted

        time.sleep(DELETE_WAIT_AFTER_SECONDS)

        # Check Lambda code signing config doesn't exist
        exists = self.code_signing_config_exists(lambda_client, codeSigningConfigARN)
        assert not exists

