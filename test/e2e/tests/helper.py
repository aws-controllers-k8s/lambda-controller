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

"""Helper functions for Lambda e2e tests
"""

import logging

class LambdaValidator:
    def __init__(self, lambda_client):
        self.lambda_client = lambda_client

    def get_function(self, function_name: str) -> dict:
        try:
            resp = self.lambda_client.get_function(
                FunctionName=function_name
            )
            return resp

        except Exception as e:
            logging.debug(e)
            return None

    def get_function_concurrency(self, function_name: str) -> int:
        try:
            resp = self.lambda_client.get_function_concurrency(
                FunctionName=function_name
            )
            return resp['ReservedConcurrentExecutions']

        except Exception as e:
            logging.debug(e)
            return None

    def get_function_code_signing_config(self, function_name: str) -> int:
        try:
            resp = self.lambda_client.get_function_code_signing_config(
                FunctionName=function_name
            )
            return resp['CodeSigningConfigArn']

        except Exception as e:
            logging.debug(e)
            return None

    def function_exists(self, function_name) -> bool:
        return self.get_function(function_name) is not None

    def get_event_source_mapping(self, esm_uuid: str) -> dict:
        try:
            resp = self.lambda_client.get_event_source_mapping(
                UUID=esm_uuid,
            )
            return resp

        except Exception as e:
            logging.debug(e)
            return None

    def event_source_mapping_exists(self, esm_uuid: str) -> bool:
        return self.get_event_source_mapping(esm_uuid) is not None

    def get_code_signing_config(self, code_signing_config_arn: str) -> dict:
        try:
            resp = self.lambda_client.get_code_signing_config(
                CodeSigningConfigArn=code_signing_config_arn,
            )
            return resp["CodeSigningConfig"]

        except Exception as e:
            logging.debug(e)
            return None

    def code_signing_config_exists(self, code_signing_config_arn: str) -> bool:
        return self.get_code_signing_config(code_signing_config_arn) is not None

    def get_alias(self, alias_name: str, function_name: str) -> dict:
        try:
            resp = self.lambda_client.get_alias(
                Name=alias_name,
                FunctionName=function_name
            )
            return resp

        except Exception as e:
            logging.debug(e)
            return None

    def alias_exists(self, alias_name: str, function_name: str) -> bool:
        return self.get_alias(alias_name, function_name) is not None

    def get_function_url_config(self, function_name: str) -> dict:
        try:
            resp = self.lambda_client.get_function_url_config(
                FunctionName=function_name
            )
            return resp

        except Exception as e:
            logging.debug(e)
            return None

    def function_url_config_exists(self, function_name: str) -> bool:
        return self.get_function_url_config(function_name) is not None
    
    def get_layer_version(self, layer_name: str, version_number: int) -> dict:
        try:
            resp = self.lambda_client.get_layer_version(
                LayerName=layer_name,
                VersionNumber=version_number
            )
            return resp

        except Exception as e:
            logging.debug(e)
            return None

    def layer_version_exists(self, layer_name:str, version_number: int) -> bool:
        return self.get_layer_version(layer_name, version_number) is not None

    def list_layer_versions(self, layer_name:str) -> list:
        try:
            resp = self.lambda_client.list_layer_versions(
                LayerName = layer_name
            )
            return resp
        except Exception as e:
            logging.debug(e)
            return None
    
    def get_function_event_invoke_config(self, function_name:str) -> dict:
        try:
            resp = self.lambda_client.get_function_event_invoke_config(
                FunctionName = function_name,
            )
            return resp
        except Exception as e:
            logging.debug(e)
            return None
    
    def get_function_event_invoke_config_alias(self, function_name:str, qualifier:str) -> dict:
        try:
            resp = self.lambda_client.get_function_event_invoke_config(
                FunctionName = function_name,
                Qualifier = qualifier
            )
            return resp
        except Exception as e:
            logging.debug(e)
            return None
    
    def function_event_invoke_config_exists(self, function_name: str) -> bool:
        return self.get_function_event_invoke_config(function_name) is not None
    
    def get_provisioned_concurrency_config(self, function_name: str, qualifier:str) -> dict:
        try: 
            resp = self.lambda_client.get_provisioned_concurrency_config(
                FunctionName = function_name,
                Qualifier = qualifier
            )
            return resp
        except Exception as e:
            logging.debug(e)
            return None
        
    def get_function_config(self, function_name:str, qualifier: str) -> dict:
        try:
            resp = self.lambda_client.get_function_configuration(
                FunctionName = function_name,
                Qualifier = qualifier
            )
            return resp
        except Exception as e:
            logging.debug(e)
            return None
    
    def version_exists(self, function_name: str, qualifier: str) -> bool:
        return self.get_function_config(function_name, qualifier) is not None
    
    def get_function_policy(self, function_name: str) -> dict:
        """Get the resource-based policy for a Lambda function or alias.
        
        Args:
            function_name: Lambda function name or alias ARN
        
        Returns:
            Policy document as a dict, or None if not found
        """
        try:
            resp = self.lambda_client.get_policy(
                FunctionName=function_name
            )
            # Response contains policy as a JSON string, parse it
            import json
            policy = json.loads(resp['Policy'])
            return policy
        except Exception as e:
            logging.debug(e)
            return None

    def function_has_permission(self, function_name: str, statement_id: str) -> bool:
        """Check if a Lambda function has a specific permission by statement ID.
        
        Args:
            function_name: Lambda function name or alias ARN
            statement_id: The statement ID to look for
        
        Returns:
            True if the permission exists, False otherwise
        """
        policy = self.get_function_policy(function_name)
        if not policy or 'Statement' not in policy:
            return False
            
        for statement in policy['Statement']:
            if statement.get('Sid') == statement_id:
                return True
        
        return False