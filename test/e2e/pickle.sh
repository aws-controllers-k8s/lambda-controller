#!/usr/bin/env bash
# Generates bootstrap.pkl from bootstrap_resources.env.
# The pickle file contains serialized acktest BootstrapResources objects
# that the e2e tests load at startup.
#
# Usage: ./pickle.sh

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ENV_FILE="${SCRIPT_DIR}/bootstrap_resources.env"
PKL_FILE="${SCRIPT_DIR}/bootstrap.pkl"

if [ ! -f "${ENV_FILE}" ]; then
    echo "Error: ${ENV_FILE} not found. Run setup.sh first."
    exit 1
fi

# shellcheck source=/dev/null
source "${ENV_FILE}"

# Back up existing pickle file
if [ -f "${PKL_FILE}" ]; then
    BACKUP="${PKL_FILE}.$(date +%s)"
    echo "Backing up existing bootstrap.pkl to ${BACKUP}"
    cp "${PKL_FILE}" "${BACKUP}"
fi

echo "Generating bootstrap.pkl..."

python3 << PYEOF
import sys
sys.path.insert(0, "${SCRIPT_DIR}/..")

from pathlib import Path
from acktest.bootstrapping.s3 import Bucket
from acktest.bootstrapping.dynamodb import Table
from acktest.bootstrapping.signer import SigningProfile
from acktest.bootstrapping.sqs import Queue
from acktest.bootstrapping.iam import Role
from e2e.bootstrap_resources import BootstrapResources

# Construct resource objects with the correct output attributes set

bucket = Bucket("ack-lambda-controller-tests")
bucket.name = "${BUCKET_NAME}"

signing_profile = SigningProfile("ack_testing_signer", signing_platform_id="AWSLambda-SHA384-ECDSA")
signing_profile.name = "${SIGNING_PROFILE_NAME}"
signing_profile.signing_profile_arn = "${SIGNING_PROFILE_ARN}"

basic_role = Role("ack-lambda-controller-basic-role", principal_service="lambda.amazonaws.com",
                  managed_policies=["arn:aws:iam::aws:policy/service-role/AWSLambdaBasicExecutionRole"])
basic_role.name = "${BASIC_ROLE_NAME}"
basic_role.arn = "${BASIC_ROLE_ARN}"

esm_role = Role("ack-lambda-controller-esm-role", principal_service="lambda.amazonaws.com",
                managed_policies=[
                    "arn:aws:iam::aws:policy/service-role/AWSLambdaBasicExecutionRole",
                    "arn:aws:iam::aws:policy/service-role/AWSLambdaDynamoDBExecutionRole",
                    "arn:aws:iam::aws:policy/AmazonSQSFullAccess",
                ])
esm_role.name = "${ESM_ROLE_NAME}"
esm_role.arn = "${ESM_ROLE_ARN}"

esm_table = Table("ack-lambda-controller-table",
                  attribute_definitions=[
                      {"AttributeName": "id", "AttributeType": "N"},
                      {"AttributeName": "createdAt", "AttributeType": "S"},
                  ],
                  key_schema=[
                      {"AttributeName": "id", "KeyType": "HASH"},
                      {"AttributeName": "createdAt", "KeyType": "RANGE"},
                  ],
                  stream_specification={"StreamEnabled": True, "StreamViewType": "NEW_IMAGE"},
                  provisioned_throughput={"ReadCapacityUnits": 5, "WriteCapacityUnits": 5})
esm_table.name = "${ESM_TABLE_NAME}"
esm_table.latest_stream_arn = "${ESM_TABLE_STREAM_ARN}"

esm_queue = Queue("ack-lambda-controller-queue")
esm_queue.name = "${ESM_QUEUE_NAME}"
esm_queue.arn = "${ESM_QUEUE_ARN}"
esm_queue.url = "${ESM_QUEUE_URL}"

eic_role = Role("ack-lambda-controller-eic-role", principal_service="lambda.amazonaws.com",
                managed_policies=[
                    "arn:aws:iam::aws:policy/service-role/AWSLambdaBasicExecutionRole",
                    "arn:aws:iam::aws:policy/AmazonSQSFullAccess",
                ])
eic_role.name = "${EIC_ROLE_NAME}"
eic_role.arn = "${EIC_ROLE_ARN}"

eic_queue_onsuccess = Queue("ack-lambda-controller-function-queue-eic-onsuccess")
eic_queue_onsuccess.name = "${EIC_QUEUE_ONSUCCESS_NAME}"
eic_queue_onsuccess.arn = "${EIC_QUEUE_ONSUCCESS_ARN}"
eic_queue_onsuccess.url = "${EIC_QUEUE_ONSUCCESS_URL}"

eic_queue_onfailure = Queue("ack-lambda-controller-function-queue-eic-onfailure")
eic_queue_onfailure.name = "${EIC_QUEUE_ONFAILURE_NAME}"
eic_queue_onfailure.arn = "${EIC_QUEUE_ONFAILURE_ARN}"
eic_queue_onfailure.url = "${EIC_QUEUE_ONFAILURE_URL}"

resources = BootstrapResources(
    FunctionsBucket=bucket,
    SigningProfile=signing_profile,
    BasicRole=basic_role,
    ESMRole=esm_role,
    ESMTable=esm_table,
    ESMQueue=esm_queue,
    EICRole=eic_role,
    EICQueueOnSuccess=eic_queue_onsuccess,
    EICQueueOnFailure=eic_queue_onfailure,
)

output_dir = Path("${SCRIPT_DIR}")
resources.serialize(output_dir)
print(f"bootstrap.pkl written to {output_dir / 'bootstrap.pkl'}")
PYEOF
