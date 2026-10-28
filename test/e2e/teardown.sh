#!/usr/bin/env bash
# Tears down AWS resources created by setup.sh.
# Reads resource names from bootstrap.pkl (via Python deserialization)
# or falls back to bootstrap_resources.env.
#
# Usage: ./teardown.sh

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ENV_FILE="${SCRIPT_DIR}/bootstrap_resources.env"
PKL_FILE="${SCRIPT_DIR}/bootstrap.pkl"

# Try to load resource info from pickle file first, fall back to env file
if [ -f "${PKL_FILE}" ]; then
    echo "Reading resources from bootstrap.pkl..."
    eval "$(python3 << PYEOF
import sys
sys.path.insert(0, "${SCRIPT_DIR}/..")

from pathlib import Path
from e2e.bootstrap_resources import BootstrapResources

resources = BootstrapResources.deserialize(Path("${SCRIPT_DIR}"))

print(f'BUCKET_NAME="{resources.FunctionsBucket.name}"')
print(f'SIGNING_PROFILE_NAME="{resources.SigningProfile.name}"')
print(f'BASIC_ROLE_NAME="{resources.BasicRole.name}"')
print(f'BASIC_ROLE_ARN="{resources.BasicRole.arn}"')
print(f'ESM_ROLE_NAME="{resources.ESMRole.name}"')
print(f'ESM_ROLE_ARN="{resources.ESMRole.arn}"')
print(f'ESM_TABLE_NAME="{resources.ESMTable.name}"')
print(f'ESM_QUEUE_NAME="{resources.ESMQueue.name}"')
print(f'ESM_QUEUE_URL="{resources.ESMQueue.url}"')
print(f'EIC_ROLE_NAME="{resources.EICRole.name}"')
print(f'EIC_ROLE_ARN="{resources.EICRole.arn}"')
print(f'EIC_QUEUE_ONSUCCESS_NAME="{resources.EICQueueOnSuccess.name}"')
print(f'EIC_QUEUE_ONSUCCESS_URL="{resources.EICQueueOnSuccess.url}"')
print(f'EIC_QUEUE_ONFAILURE_NAME="{resources.EICQueueOnFailure.name}"')
print(f'EIC_QUEUE_ONFAILURE_URL="{resources.EICQueueOnFailure.url}"')
PYEOF
)"
elif [ -f "${ENV_FILE}" ]; then
    echo "Reading resources from bootstrap_resources.env..."
    # shellcheck source=/dev/null
    source "${ENV_FILE}"
else
    echo "Error: Neither bootstrap.pkl nor bootstrap_resources.env found."
    echo "Nothing to tear down."
    exit 1
fi

echo ""
echo "=== Tearing down resources ==="

# --- Empty and delete S3 bucket ---
echo "Deleting S3 bucket: ${BUCKET_NAME}"
aws s3 rm "s3://${BUCKET_NAME}" --recursive 2>/dev/null || true
aws s3api delete-bucket --bucket "${BUCKET_NAME}" 2>/dev/null || echo "  Warning: bucket deletion failed (may already be deleted)"

# --- Cancel signing profile ---
echo "Cancelling signing profile: ${SIGNING_PROFILE_NAME}"
aws signer cancel-signing-profile --profile-name "${SIGNING_PROFILE_NAME}" 2>/dev/null || echo "  Warning: signing profile cancellation failed (may already be cancelled)"

# --- Delete SQS queues ---
for queue_var in ESM_QUEUE EIC_QUEUE_ONSUCCESS EIC_QUEUE_ONFAILURE; do
    name_var="${queue_var}_NAME"
    url_var="${queue_var}_URL"
    echo "Deleting SQS queue: ${!name_var}"
    aws sqs delete-queue --queue-url "${!url_var}" 2>/dev/null || echo "  Warning: queue deletion failed (may already be deleted)"
done

# --- Delete DynamoDB table ---
echo "Deleting DynamoDB table: ${ESM_TABLE_NAME}"
aws dynamodb delete-table --table-name "${ESM_TABLE_NAME}" 2>/dev/null || echo "  Warning: table deletion failed (may already be deleted)"

# --- Detach policies and delete IAM roles ---
LAMBDA_BASIC_EXECUTION_ARN="arn:aws:iam::aws:policy/service-role/AWSLambdaBasicExecutionRole"
LAMBDA_DYNAMODB_EXECUTION_ROLE="arn:aws:iam::aws:policy/service-role/AWSLambdaDynamoDBExecutionRole"
LAMBDA_SQS_QUEUE_EXECUTION_ROLE="arn:aws:iam::aws:policy/AmazonSQSFullAccess"

delete_role() {
    local role_name="$1"
    shift
    local policies=("$@")

    echo "Deleting IAM role: ${role_name}"
    for policy_arn in "${policies[@]}"; do
        aws iam detach-role-policy --role-name "${role_name}" --policy-arn "${policy_arn}" 2>/dev/null || true
    done
    aws iam delete-role --role-name "${role_name}" 2>/dev/null || echo "  Warning: role deletion failed (may already be deleted)"
}

delete_role "${EIC_ROLE_NAME}" \
    "${LAMBDA_BASIC_EXECUTION_ARN}" \
    "${LAMBDA_SQS_QUEUE_EXECUTION_ROLE}"

delete_role "${ESM_ROLE_NAME}" \
    "${LAMBDA_BASIC_EXECUTION_ARN}" \
    "${LAMBDA_DYNAMODB_EXECUTION_ROLE}" \
    "${LAMBDA_SQS_QUEUE_EXECUTION_ROLE}"

delete_role "${BASIC_ROLE_NAME}" \
    "${LAMBDA_BASIC_EXECUTION_ARN}"

# --- Clean up local files ---
if [ -f "${ENV_FILE}" ]; then
    echo "Removing ${ENV_FILE}"
    rm -f "${ENV_FILE}"
fi

echo ""
echo "=== Teardown complete ==="
