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

"""Integration tests for the Lambda FunctionURLConfig API.
"""

import boto3
import pytest
import time
import logging

from acktest.k8s import resource as k8s

from e2e import service_marker
from e2e.conftest import Wait
from e2e.tests.helper import LambdaValidator

@service_marker
@pytest.mark.canary
class TestFunctionURLConfig:
    @pytest.mark.function_overrides({
        'package_type': 'Image',
        'role_type': 'basic',
    })
    def test_smoke(self, lambda_client, function_url_config):
        (ref, cr) = function_url_config
        function_name = cr["spec"]["functionName"]

        # Check FunctionURLConfig exists
        lambda_validator = LambdaValidator(lambda_client)

        # Check function url config exists
        function_url_config = lambda_validator.get_function_url_config(function_name)
        assert function_url_config is not None
        assert function_url_config["AuthType"] == "NONE"

        cr = k8s.wait_resource_consumed_by_controller(ref)

        # Update cr
        cr["spec"]["cors"] = {
            "maxAge": 10,
            "allowOrigins": ["https://*"],
        }

        # Patch k8s resource
        k8s.patch_custom_resource(ref, cr)
        time.sleep(Wait.FunctionURLConfig.Update)

        # Check FunctionURLConfig MaxAge and AllowOrigins array
        function_url_config = lambda_validator.get_function_url_config(function_name)
        assert function_url_config is not None
        assert function_url_config["Cors"] is not None
        assert function_url_config["Cors"]["MaxAge"] == 10
        assert function_url_config["Cors"]["AllowOrigins"] == ["https://*"]

        # Delete k8s resource
        _, deleted = k8s.delete_custom_resource(ref)
        assert deleted

        time.sleep(Wait.FunctionURLConfig.Delete)

        # Check FunctionURLConfig doesn't exist
        assert not lambda_validator.function_url_config_exists(function_name)