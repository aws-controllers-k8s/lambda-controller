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

import pytest
import time
import logging

from acktest.k8s import resource as k8s

from e2e import service_marker
from e2e.conftest import Wait
from e2e.tests.helper import LambdaValidator

@service_marker
@pytest.mark.canary
class TestCodeSigningConfig:
    def test_smoke(self, lambda_client, code_signing_config):
        (ref, cr) = code_signing_config
        codeSigningConfigARN = cr['status']['ackResourceMetadata']['arn']

        lambda_validator = LambdaValidator(lambda_client)
        # Check Lambda code signing config exists
        assert lambda_validator.code_signing_config_exists(codeSigningConfigARN)

        # Update cr
        cr["spec"]["description"] = "new description"

        # Patch k8s resource
        k8s.patch_custom_resource(ref, cr)
        time.sleep(Wait.CodeSigningConfig.Update)

        # Check code signing config  description
        csc = lambda_validator.get_code_signing_config(codeSigningConfigARN)
        assert csc is not None
        assert csc["Description"] == "new description"

        # Delete k8s resource
        _, deleted = k8s.delete_custom_resource(ref)
        assert deleted

        time.sleep(Wait.CodeSigningConfig.Delete)
        # Check Lambda code signing config doesn't exist
        assert not lambda_validator.code_signing_config_exists(codeSigningConfigARN)
