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

from acktest.k8s import resource as k8s

from e2e import service_marker
from e2e.conftest import Wait
from e2e.tests.helper import LambdaValidator

@service_marker
@pytest.mark.canary
class TestAlias:
    @pytest.mark.function_overrides({
        'package_type': 'Zip',
        'role_type': 'basic',
    })
    def test_smoke(self, lambda_client, alias, function):
        (ref, cr) = alias
        alias_name = cr["spec"]["name"]
        function_name = cr["spec"]["functionName"]

        lambda_validator = LambdaValidator(lambda_client)
        # Check alias exists
        assert lambda_validator.alias_exists(alias_name, function_name)

        cr = k8s.wait_resource_consumed_by_controller(ref)
        
        # Update cr
        cr["spec"]["description"] = ""

        # Patch k8s resource
        k8s.patch_custom_resource(ref, cr)
        time.sleep(Wait.Alias.Update)

        # Check alias description
        alias = lambda_validator.get_alias(alias_name, function_name)
        assert alias is not None
        assert alias["Description"] == ""

        # Delete k8s resource
        _, deleted = k8s.delete_custom_resource(ref)
        assert deleted

        time.sleep(Wait.Alias.Delete)

        # Check alias doesn't exist
        assert not lambda_validator.alias_exists(alias_name, function_name)
