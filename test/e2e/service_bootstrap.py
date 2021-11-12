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

LAMBDA_IAM_ROLE_NAME = 'ack-lambda-function-role-' + RAND_TEST_SUFFIX
LAMBDA_IAM_ROLE_POLICY = '{"Version": "2012-10-17","Statement": [{ "Effect": "Allow", "Principal": {"Service": '\
                                '"lambda.amazonaws.com"}, "Action": "sts:AssumeRole"}]} '
LAMBDA_BASIC_POLICY_ARN = 'arn:aws:iam::aws:policy/service-role/AWSLambdaBasicExecutionRole'

FUNCTIONS_BUCKET_NAME = "ack-lambda-function-s3-bucket-" + RAND_TEST_SUFFIX

LAMBDA_FUNCTION_FILE = "main.py"
LAMBDA_FUNCTION_FILE_ZIP = "main.zip"
LAMBDA_FUNCTION_FILE_PATH = f"./resources/lambda_function/{LAMBDA_FUNCTION_FILE}"
LAMBDA_FUNCTION_FILE_PATH_ZIP = f"./resources/lambda_function/{LAMBDA_FUNCTION_FILE_ZIP}"

def service_bootstrap() -> dict:
    logging.getLogger().setLevel(logging.INFO)
    lambda_role_arn = create_lambda_role(LAMBDA_IAM_ROLE_NAME)
    create_bucket(FUNCTIONS_BUCKET_NAME)
    zip_function_file(LAMBDA_FUNCTION_FILE_PATH, LAMBDA_FUNCTION_FILE_PATH_ZIP)
    upload_function_to_bucket(LAMBDA_FUNCTION_FILE_PATH_ZIP, FUNCTIONS_BUCKET_NAME)

    return TestBootstrapResources(
        LAMBDA_IAM_ROLE_NAME,
        LAMBDA_IAM_ROLE_POLICY,
        LAMBDA_BASIC_POLICY_ARN,
        FUNCTIONS_BUCKET_NAME,
        LAMBDA_FUNCTION_FILE_PATH,
        lambda_role_arn,
        LAMBDA_FUNCTION_FILE_ZIP,
    ).__dict__


def create_lambda_role(lambda_iam_role_name: str) -> str:
    region = get_region()
    iam_client = boto3.client("iam", region_name=region)

    logging.debug(f"Creating function iam role {lambda_iam_role_name}")
    try:
        iam_client.get_role(RoleName=lambda_iam_role_name)
        raise RuntimeError(f"Expected {lambda_iam_role_name} role to not exist."
                           f" Did previous test cleanup successfully?")
    except iam_client.exceptions.NoSuchEntityException:
        pass

    resp = iam_client.create_role(
        RoleName=lambda_iam_role_name,
        AssumeRolePolicyDocument=LAMBDA_IAM_ROLE_POLICY
    )
    iam_client.attach_role_policy(RoleName=lambda_iam_role_name, PolicyArn=LAMBDA_BASIC_POLICY_ARN)
    logging.info(f"Created role {lambda_iam_role_name}")

    return resp['Role']['Arn']

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


if __name__ == "__main__":
    config = service_bootstrap()
    # Write config to current directory by default
    resources.write_bootstrap_config(config, bootstrap_directory)