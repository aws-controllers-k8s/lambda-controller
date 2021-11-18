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
from time import sleep
from zipfile import ZipFile
import random
import string

from acktest import resources
from acktest.aws.identity import get_region, get_account_id
from e2e import bootstrap_directory
from e2e.bootstrap_resources import TestBootstrapResources
from botocore.exceptions import ClientError

RAND_TEST_SUFFIX = (''.join(random.choice(string.ascii_lowercase) for _ in range(6)))

LAMBDA_BASIC_IAM_ROLE_NAME = 'ack-lambda-function-role-basic-' + RAND_TEST_SUFFIX
LAMBDA_ESM_IAM_ROLE_NAME = 'ack-lambda-function-role-esm-' + RAND_TEST_SUFFIX

LAMBDA_IAM_ROLE_POLICY = '{"Version": "2012-10-17","Statement": [{ "Effect": "Allow", "Principal": {"Service": '\
                                '"lambda.amazonaws.com"}, "Action": "sts:AssumeRole"}]} '
LAMBDA_BASIC_EXECUTION_ARN = 'arn:aws:iam::aws:policy/service-role/AWSLambdaBasicExecutionRole'
LAMBDA_DYNAMODB_EXECUTION_ROLE = 'arn:aws:iam::aws:policy/service-role/AWSLambdaDynamoDBExecutionRole'
LAMBDA_SQS_QUEUE_EXECUTION_ROLE = 'arn:aws:iam::aws:policy/AmazonSQSFullAccess'

BASIC_ROLES = [ LAMBDA_BASIC_EXECUTION_ARN ]
ESM_ROLES = [ LAMBDA_BASIC_EXECUTION_ARN, LAMBDA_DYNAMODB_EXECUTION_ROLE, LAMBDA_SQS_QUEUE_EXECUTION_ROLE ]

FUNCTIONS_BUCKET_NAME = "ack-lambda-function-s3-bucket-" + RAND_TEST_SUFFIX

LAMBDA_FUNCTION_FILE = "main.py"
LAMBDA_FUNCTION_FILE_ZIP = "main.zip"
LAMBDA_FUNCTION_FILE_PATH = f"./resources/lambda_function/{LAMBDA_FUNCTION_FILE}"
LAMBDA_FUNCTION_FILE_PATH_ZIP = f"./resources/lambda_function/{LAMBDA_FUNCTION_FILE_ZIP}"

AWS_SIGNING_PROFILE_NAME = "ack_testing_lambda_signing_profile"
AWS_SIGNING_PLATFORM_ID = "AWSLambda-SHA384-ECDSA"

SQS_QUEUE_NAME = "ack-lambda-sqs-queue-" + RAND_TEST_SUFFIX

DYNAMODB_TABLE_NAME = "ack-lambda-ddb-table-" + RAND_TEST_SUFFIX

def service_bootstrap() -> dict:
    logging.getLogger().setLevel(logging.INFO)
    lambda_esm_role_arn = create_lambda_function_esm_role(LAMBDA_ESM_IAM_ROLE_NAME)
    lambda_basic_role_arn = create_lambda_function_basic_role(LAMBDA_BASIC_IAM_ROLE_NAME)
    create_bucket(FUNCTIONS_BUCKET_NAME)
    zip_function_file(LAMBDA_FUNCTION_FILE_PATH, LAMBDA_FUNCTION_FILE_PATH_ZIP)
    upload_function_to_bucket(LAMBDA_FUNCTION_FILE_PATH_ZIP, FUNCTIONS_BUCKET_NAME)
    signing_profile_version_arn = ensure_signing_profile(AWS_SIGNING_PROFILE_NAME, AWS_SIGNING_PLATFORM_ID)
    sqs_queue_arn, sqs_queue_url = create_sqs_queue(SQS_QUEUE_NAME)
    dynamodb_table_stream_arn = create_dynamodb_table(DYNAMODB_TABLE_NAME)

    return TestBootstrapResources(
        FUNCTIONS_BUCKET_NAME,
        LAMBDA_FUNCTION_FILE_PATH,
        LAMBDA_FUNCTION_FILE_ZIP,
        LAMBDA_BASIC_IAM_ROLE_NAME,
        lambda_basic_role_arn,
        LAMBDA_ESM_IAM_ROLE_NAME,
        lambda_esm_role_arn,
        signing_profile_version_arn,
        sqs_queue_arn,
        sqs_queue_url,
        DYNAMODB_TABLE_NAME,
        dynamodb_table_stream_arn,
        BASIC_ROLES,
        ESM_ROLES,
    ).__dict__



def create_role_with_policies(iam_role_name: str, policies: list) -> str:
    region = get_region()
    iam_client = boto3.client("iam", region_name=region)

    logging.debug(f"Creating iam role {iam_role_name}")
    try:
        iam_client.get_role(RoleName=iam_role_name)
        raise RuntimeError(f"Expected {iam_role_name} role to not exist."
                           f" Did previous test cleanup successfully?")
    except iam_client.exceptions.NoSuchEntityException:
        pass

    resp = iam_client.create_role(
        RoleName=iam_role_name,
        AssumeRolePolicyDocument=LAMBDA_IAM_ROLE_POLICY
    )

    for policyARN in policies:
        iam_client.attach_role_policy(RoleName=iam_role_name, PolicyArn=policyARN)

    logging.info(f"Created role {iam_role_name}")
    return resp['Role']['Arn']


def create_lambda_function_basic_role(iam_role_name: str) -> str:
    return create_role_with_policies(
        iam_role_name,
        BASIC_ROLES,
    )

def create_lambda_function_esm_role(iam_role_name: str) -> str:
    return create_role_with_policies(
        iam_role_name,
        ESM_ROLES,
    )

def create_bucket(bucket_name: str):
    region = get_region()
    s3_client = boto3.resource('s3')
    logging.debug(f"Creating s3 data bucket {bucket_name}")
    try:
        s3_client.create_bucket(
            Bucket=bucket_name,
            CreateBucketConfiguration={"LocationConstraint": region}
        )
    except s3_client.exceptions.BucketAlreadyExists:
        raise RuntimeError(f"Expected {bucket_name} bucket to not exist."
                           f" Did previous test cleanup successfully?")

    logging.info(f"Created bucket {bucket_name}")

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

def ensure_signing_profile(signing_profile_name: str, platform_id: str) -> str:
    region = get_region()
    signer_client = boto3.client("signer", region_name=region)

    # Signing profiles cannot be deleted. We just reuse the same signing profile
    # for ACK lambda controller e2e tests.
    try:
        resp = signer_client.get_signing_profile(
            profileName=signing_profile_name,
        )
        return resp['profileVersionArn']
    except:
        resp = signer_client.put_signing_profile(
            profileName=signing_profile_name,
            platformId=platform_id,
        )
        logging.info(f"Created signing profile {signing_profile_name}")
        return resp['profileVersionArn']

def create_sqs_queue(queue_name: str) -> str:
    region = get_region()
    sqs_client = boto3.resource('sqs', region_name=region)
    logging.debug(f"Creating SQS queue {queue_name}")
    resp = sqs_client.create_queue(
        QueueName=queue_name,
    )
    logging.info(f"Created SQS queue {queue_name}")
    return resp.attributes['QueueArn'], resp.url

def create_dynamodb_table(table_name: str) -> str:
    region = get_region()
    dynamodb_client = boto3.resource('dynamodb', region_name=region)
    resp = dynamodb_client.create_table(
        TableName=table_name,
        KeySchema=[
            {
                'AttributeName': 'id',
                'KeyType': 'HASH'
            },
            {
                'AttributeName': 'createdAt',
                'KeyType': 'RANGE'
            }
        ],
        AttributeDefinitions=[
            {
                'AttributeName': 'id',
                'AttributeType': 'N'
            },
            {
                'AttributeName': 'createdAt',
                'AttributeType': 'S'
            },
        ],
        ProvisionedThroughput={
            'ReadCapacityUnits': 5,
            'WriteCapacityUnits': 5
        },
        StreamSpecification={
            'StreamEnabled': True,
            'StreamViewType': 'NEW_IMAGE'
        }
    )
    logging.info(f"Created Dynamodb table {table_name}")
    return resp.latest_stream_arn

if __name__ == "__main__":
    config = service_bootstrap()
    # Write config to current directory by default
    resources.write_bootstrap_config(config, bootstrap_directory)