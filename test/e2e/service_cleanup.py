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
"""Cleans up the resources created by Lambda bootstrapping process.
"""

import boto3
import logging

from acktest import resources
from acktest.aws.identity import get_region

from e2e import bootstrap_directory
from e2e.bootstrap_resources import TestBootstrapResources

def service_cleanup(config: dict):
    logging.getLogger().setLevel(logging.INFO)
    resources = TestBootstrapResources(
        **config
    )

    try:
        detach_policies_and_delete_role(resources.LambdaBasicRoleName, resources.BasicExecutionRoleARNs)
    except:
        logging.exception(f"Unable to delete basic role {resources.LambdaBasicRoleARN}")

    try:
        detach_policies_and_delete_role(resources.LambdaESMRoleName, resources.ESMExecutionRoleARNs)
    except:
        logging.exception(f"Unable to delete esm role {resources.LambdaESMRoleARN}")

    try:
        clean_up_and_delete_bucket(resources.FunctionsBucketName)
    except:
        logging.exception(f"Unable to delete bucket {resources.FunctionsBucketName}")

    try:
        delete_sqs_queue(resources.SQSQueueURL)
    except:
        logging.exception(f"Unable to delete queue {resources.SQSQueueURL}")

    try:
        delete_dynamodb_table(resources.DynamoDBTableName)
    except:
        logging.exception(f"Unable to delete table {resources.DynamoDBTableName}")

def detach_policies_and_delete_role(iam_role_name: str, iam_policy_arns: list):
    region = get_region()
    iam_client = boto3.client("iam", region_name=region)
    for iam_policy_arn in iam_policy_arns:
        iam_client.detach_role_policy(RoleName=iam_role_name, PolicyArn=iam_policy_arn)
    iam_client.delete_role(RoleName=iam_role_name)
    logging.info(f"Deleted role {iam_role_name}")

def clean_up_and_delete_bucket(bucket_name: str):
    region = get_region()
    s3_client = boto3.client("s3", region_name=region)

    resp = s3_client.list_objects(Bucket=bucket_name)
    for object in resp['Contents']:
        s3_client.delete_object(Bucket=bucket_name, Key=object['Key'])

    s3_client.delete_bucket(
        Bucket=bucket_name,
    )
    logging.info(f"Deleted bucket {bucket_name}")

def delete_sqs_queue(queue_url: str) -> str:
    region = get_region()
    sqs_client = boto3.client('sqs', region_name=region)
    sqs_client.delete_queue(
        QueueUrl=queue_url,
    )
    logging.info(f"Deleted SQS queue {queue_url}")

def delete_dynamodb_table(table_name: str) -> str:
    region = get_region()
    ddb_client = boto3.client('dynamodb', region_name=region)
    ddb_client.delete_table(
        TableName=table_name,
    )
    logging.info(f"Deleted DynamoDB table {table_name}")

if __name__ == "__main__":   
    bootstrap_config = resources.read_bootstrap_config(bootstrap_directory)
    service_cleanup(bootstrap_config)