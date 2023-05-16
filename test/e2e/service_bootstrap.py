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
"""Bootstraps the resources required to run the Lambda integration tests.
"""

import os
import boto3
import logging
from zipfile import ZipFile

from e2e import bootstrap_directory
from e2e.bootstrap_resources import BootstrapResources
from botocore.exceptions import ClientError

from acktest.bootstrapping import Resources, BootstrapFailureException
from acktest.bootstrapping.s3 import Bucket
from acktest.bootstrapping.dynamodb import Table
from acktest.bootstrapping.signer import SigningProfile
from acktest.bootstrapping.sqs import Queue
from acktest.bootstrapping.iam import Role

LAMBDA_IAM_ROLE_POLICY = '{"Version": "2012-10-17","Statement": [{ "Effect": "Allow", "Principal": {"Service": '\
                                '"lambda.amazonaws.com"}, "Action": "sts:AssumeRole"}]} '
LAMBDA_BASIC_EXECUTION_ARN = 'arn:aws:iam::aws:policy/service-role/AWSLambdaBasicExecutionRole'
LAMBDA_DYNAMODB_EXECUTION_ROLE = 'arn:aws:iam::aws:policy/service-role/AWSLambdaDynamoDBExecutionRole'
LAMBDA_SQS_QUEUE_EXECUTION_ROLE = 'arn:aws:iam::aws:policy/AmazonSQSFullAccess'

BASIC_ROLE_POLICIES = [ LAMBDA_BASIC_EXECUTION_ARN ]
ESM_ROLE_POLICIES = [ LAMBDA_BASIC_EXECUTION_ARN, LAMBDA_DYNAMODB_EXECUTION_ROLE, LAMBDA_SQS_QUEUE_EXECUTION_ROLE ]

LAMBDA_FUNCTION_FILE = "main.py"
LAMBDA_FUNCTION_FILE_ZIP = "main.zip"
LAMBDA_FUNCTION_FILE_PATH = f"./resources/lambda_function/{LAMBDA_FUNCTION_FILE}"
LAMBDA_FUNCTION_FILE_PATH_ZIP = f"./resources/lambda_function/{LAMBDA_FUNCTION_FILE_ZIP}"

LAMBDA_FUNCTION_UPDATED_FILE = "updated_main.py"
LAMBDA_FUNCTION_UPDATED_FILE_ZIP = "updated_main.zip"
LAMBDA_FUNCTION_UPDATED_FILE_PATH = f"./resources/lambda_function/{LAMBDA_FUNCTION_UPDATED_FILE}"
LAMBDA_FUNCTION_UPDATED_FILE_PATH_ZIP = f"./resources/lambda_function/{LAMBDA_FUNCTION_UPDATED_FILE_ZIP}" 

AWS_SIGNING_PLATFORM_ID = "AWSLambda-SHA384-ECDSA"

def zip_function_file(src: str, dst: str):
    with ZipFile(dst, 'w') as zipf:
        zipf.write(src, arcname=src)

def upload_function_to_bucket(file_path: str, bucket_name: str):
    object_name = os.path.basename(file_path)

    s3_client = boto3.client('s3')
    try:
        s3_client.upload_file(
            file_path,
            bucket_name,
            object_name,
        )
    except ClientError as e:
        logging.error(e)

    logging.info(f"Uploaded {file_path} to bucket {bucket_name}")

def service_bootstrap() -> Resources:
    logging.getLogger().setLevel(logging.INFO)
    resources = BootstrapResources(
        FunctionsBucket=Bucket(
            "ack-lambda-controller-tests",
        ),
        SigningProfile=SigningProfile(
            "ack_testing_signer",
            signing_platform_id=AWS_SIGNING_PLATFORM_ID,
        ),
        BasicRole=Role(
            "ack-lambda-controller-basic-role",
            principal_service="lambda.amazonaws.com",
            managed_policies=BASIC_ROLE_POLICIES,
        ),
        ESMRole=Role(
            "ack-lambda-controller-esm-role",
            principal_service="lambda.amazonaws.com",
            managed_policies=ESM_ROLE_POLICIES,
        ),
        ESMTable=Table(
            "ack-lambda-controller-table",
            attribute_definitions=[
                {
                    'AttributeName': 'id',
                    'AttributeType': 'N'
                },
                {
                    'AttributeName': 'createdAt',
                    'AttributeType': 'S'
                },
            ],
            key_schema=[
                {
                    'AttributeName': 'id',
                    'KeyType': 'HASH'
                },
                {
                    'AttributeName': 'createdAt',
                    'KeyType': 'RANGE'
                }
            ],
            stream_specification={
                'StreamEnabled': True,
                'StreamViewType': 'NEW_IMAGE'
            },
            provisioned_throughput={
                'ReadCapacityUnits': 5,
                'WriteCapacityUnits': 5
            },
        ),
        ESMQueue=Queue(
            "ack-lambda-controller-queue"
        ),
    )

    try:
        resources.bootstrap()
        zip_function_file(LAMBDA_FUNCTION_FILE_PATH, LAMBDA_FUNCTION_FILE_PATH_ZIP)
        upload_function_to_bucket(
            LAMBDA_FUNCTION_FILE_PATH_ZIP,
            resources.FunctionsBucket.name,
        )

        zip_function_file(LAMBDA_FUNCTION_UPDATED_FILE_PATH, LAMBDA_FUNCTION_UPDATED_FILE_PATH_ZIP)
        upload_function_to_bucket(
            LAMBDA_FUNCTION_UPDATED_FILE_PATH_ZIP,
            resources.FunctionsBucket.name,
        )
    except BootstrapFailureException as ex:
        exit(254)
    return resources

if __name__ == "__main__":
    config = service_bootstrap()
    # Write config to current directory by default
    config.serialize(bootstrap_directory)