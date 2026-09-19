#!/usr/bin/env bash
# Creates AWS resources needed for Lambda controller e2e tests.
# Writes resource names/ARNs to bootstrap_resources.env for use by pickle.sh.
#
# Usage: ./setup.sh [--pickle]

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ENV_FILE="${SCRIPT_DIR}/bootstrap_resources.env"

# Generate random 8-char suffix
SUFFIX="$(LC_ALL=C tr -dc 'a-z0-9' < /dev/urandom | head -c 8 || true)"
echo "Using random suffix: ${SUFFIX}"

# Resource names
BUCKET_NAME="ack-lambda-controller-tests-${SUFFIX}"
SIGNING_PROFILE_NAME="ack_testing_signer_${SUFFIX}"
BASIC_ROLE_NAME="ack-lambda-controller-basic-role-${SUFFIX}"
ESM_ROLE_NAME="ack-lambda-controller-esm-role-${SUFFIX}"
ESM_TABLE_NAME="ack-lambda-controller-table-${SUFFIX}"
ESM_QUEUE_NAME="ack-lambda-controller-queue-${SUFFIX}"
EIC_ROLE_NAME="ack-lambda-controller-eic-role-${SUFFIX}"
EIC_QUEUE_ONSUCCESS_NAME="ack-lambda-controller-function-queue-eic-onsuccess-${SUFFIX}"
EIC_QUEUE_ONFAILURE_NAME="ack-lambda-controller-function-queue-eic-onfailure-${SUFFIX}"

AWS_REGION="${AWS_DEFAULT_REGION:-$(aws configure get region 2>/dev/null || echo "us-west-2")}"
AWS_ACCOUNT_ID="$(aws sts get-caller-identity --query Account --output text)"

ASSUME_ROLE_POLICY='{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Principal": { "Service": "lambda.amazonaws.com" },
      "Action": "sts:AssumeRole"
    }
  ]
}'

LAMBDA_BASIC_EXECUTION_ARN="arn:aws:iam::aws:policy/service-role/AWSLambdaBasicExecutionRole"
LAMBDA_DYNAMODB_EXECUTION_ROLE="arn:aws:iam::aws:policy/service-role/AWSLambdaDynamoDBExecutionRole"
LAMBDA_SQS_QUEUE_EXECUTION_ROLE="arn:aws:iam::aws:policy/AmazonSQSFullAccess"

# --- S3 Bucket ---
echo "Creating S3 bucket: ${BUCKET_NAME}"
if [ "${AWS_REGION}" = "us-east-1" ]; then
    aws s3api create-bucket --bucket "${BUCKET_NAME}"
else
    aws s3api create-bucket --bucket "${BUCKET_NAME}" \
        --create-bucket-configuration LocationConstraint="${AWS_REGION}"
fi

# --- Signing Profile ---
echo "Creating signing profile: ${SIGNING_PROFILE_NAME}"
SIGNING_PROFILE_OUTPUT="$(aws signer put-signing-profile \
    --profile-name "${SIGNING_PROFILE_NAME}" \
    --platform-id AWSLambda-SHA384-ECDSA)"
SIGNING_PROFILE_ARN="$(echo "${SIGNING_PROFILE_OUTPUT}" | python3 -c "import sys,json; print(json.load(sys.stdin)['profileVersionArn'])")"
echo "  Signing profile ARN: ${SIGNING_PROFILE_ARN}"

# --- IAM Roles ---
create_role() {
    local role_name="$1"
    shift
    local policies=("$@")

    echo "Creating IAM role: ${role_name}" >&2
    local role_arn
    role_arn="$(aws iam create-role \
        --role-name "${role_name}" \
        --assume-role-policy-document "${ASSUME_ROLE_POLICY}" \
        --output text --query 'Role.Arn')"

    for policy_arn in "${policies[@]}"; do
        echo "  Attaching policy: ${policy_arn}" >&2
        aws iam attach-role-policy \
            --role-name "${role_name}" \
            --policy-arn "${policy_arn}"
    done

    echo "${role_arn}"
}

BASIC_ROLE_ARN="$(create_role "${BASIC_ROLE_NAME}" \
    "${LAMBDA_BASIC_EXECUTION_ARN}")"

ESM_ROLE_ARN="$(create_role "${ESM_ROLE_NAME}" \
    "${LAMBDA_BASIC_EXECUTION_ARN}" \
    "${LAMBDA_DYNAMODB_EXECUTION_ROLE}" \
    "${LAMBDA_SQS_QUEUE_EXECUTION_ROLE}")"

EIC_ROLE_ARN="$(create_role "${EIC_ROLE_NAME}" \
    "${LAMBDA_BASIC_EXECUTION_ARN}" \
    "${LAMBDA_SQS_QUEUE_EXECUTION_ROLE}")"

echo "Waiting 30s for IAM role propagation..."
sleep 30

# --- DynamoDB Table ---
echo "Creating DynamoDB table: ${ESM_TABLE_NAME}"
aws dynamodb create-table \
    --table-name "${ESM_TABLE_NAME}" \
    --attribute-definitions \
        AttributeName=id,AttributeType=N \
        AttributeName=createdAt,AttributeType=S \
    --key-schema \
        AttributeName=id,KeyType=HASH \
        AttributeName=createdAt,KeyType=RANGE \
    --provisioned-throughput ReadCapacityUnits=5,WriteCapacityUnits=5 \
    --stream-specification StreamEnabled=true,StreamViewType=NEW_IMAGE \
    --output text --query 'TableDescription.TableArn' > /dev/null

echo "Waiting for table to become ACTIVE..."
aws dynamodb wait table-exists --table-name "${ESM_TABLE_NAME}"

ESM_TABLE_STREAM_ARN="$(aws dynamodb describe-table \
    --table-name "${ESM_TABLE_NAME}" \
    --query 'Table.LatestStreamArn' --output text)"
echo "  Table stream ARN: ${ESM_TABLE_STREAM_ARN}"

# --- SQS Queues ---
create_queue() {
    local queue_name="$1"
    echo "Creating SQS queue: ${queue_name}" >&2
    local queue_url
    queue_url="$(aws sqs create-queue --queue-name "${queue_name}" \
        --query 'QueueUrl' --output text)"
    local queue_arn
    queue_arn="$(aws sqs get-queue-attributes --queue-url "${queue_url}" \
        --attribute-names QueueArn --query 'Attributes.QueueArn' --output text)"
    echo "  Queue URL: ${queue_url}" >&2
    echo "  Queue ARN: ${queue_arn}" >&2
    echo "${queue_url} ${queue_arn}"
}

ESM_QUEUE_OUTPUT="$(create_queue "${ESM_QUEUE_NAME}")"
ESM_QUEUE_URL="$(echo "${ESM_QUEUE_OUTPUT}" | tail -1 | cut -d' ' -f1)"
ESM_QUEUE_ARN="$(echo "${ESM_QUEUE_OUTPUT}" | tail -1 | cut -d' ' -f2)"

EIC_ONSUCCESS_OUTPUT="$(create_queue "${EIC_QUEUE_ONSUCCESS_NAME}")"
EIC_QUEUE_ONSUCCESS_URL="$(echo "${EIC_ONSUCCESS_OUTPUT}" | tail -1 | cut -d' ' -f1)"
EIC_QUEUE_ONSUCCESS_ARN="$(echo "${EIC_ONSUCCESS_OUTPUT}" | tail -1 | cut -d' ' -f2)"

EIC_ONFAILURE_OUTPUT="$(create_queue "${EIC_QUEUE_ONFAILURE_NAME}")"
EIC_QUEUE_ONFAILURE_URL="$(echo "${EIC_ONFAILURE_OUTPUT}" | tail -1 | cut -d' ' -f1)"
EIC_QUEUE_ONFAILURE_ARN="$(echo "${EIC_ONFAILURE_OUTPUT}" | tail -1 | cut -d' ' -f2)"

# --- Zip and upload Lambda functions ---
echo "Zipping and uploading Lambda functions to bucket..."
MAIN_ZIP="$(mktemp -d)/main.zip"
UPDATED_ZIP="$(mktemp -d)/updated_main.zip"

(cd "${SCRIPT_DIR}" && zip -j "${MAIN_ZIP}" ./resources/lambda_function/main.py)
(cd "${SCRIPT_DIR}" && zip -j "${UPDATED_ZIP}" ./resources/lambda_function/updated_main.py)

aws s3 cp "${MAIN_ZIP}" "s3://${BUCKET_NAME}/main.zip" --checksum-algorithm SHA256
aws s3 cp "${UPDATED_ZIP}" "s3://${BUCKET_NAME}/updated_main.zip" --checksum-algorithm SHA256
rm -f "${MAIN_ZIP}" "${UPDATED_ZIP}"

# --- Write env file ---
cat > "${ENV_FILE}" << EOF
# Generated by setup.sh on $(date -u +%Y-%m-%dT%H:%M:%SZ)
BUCKET_NAME='${BUCKET_NAME}'
SIGNING_PROFILE_NAME='${SIGNING_PROFILE_NAME}'
SIGNING_PROFILE_ARN='${SIGNING_PROFILE_ARN}'
BASIC_ROLE_NAME='${BASIC_ROLE_NAME}'
BASIC_ROLE_ARN='${BASIC_ROLE_ARN}'
ESM_ROLE_NAME='${ESM_ROLE_NAME}'
ESM_ROLE_ARN='${ESM_ROLE_ARN}'
ESM_TABLE_NAME='${ESM_TABLE_NAME}'
ESM_TABLE_STREAM_ARN='${ESM_TABLE_STREAM_ARN}'
ESM_QUEUE_NAME='${ESM_QUEUE_NAME}'
ESM_QUEUE_URL='${ESM_QUEUE_URL}'
ESM_QUEUE_ARN='${ESM_QUEUE_ARN}'
EIC_ROLE_NAME='${EIC_ROLE_NAME}'
EIC_ROLE_ARN='${EIC_ROLE_ARN}'
EIC_QUEUE_ONSUCCESS_NAME='${EIC_QUEUE_ONSUCCESS_NAME}'
EIC_QUEUE_ONSUCCESS_URL='${EIC_QUEUE_ONSUCCESS_URL}'
EIC_QUEUE_ONSUCCESS_ARN='${EIC_QUEUE_ONSUCCESS_ARN}'
EIC_QUEUE_ONFAILURE_NAME='${EIC_QUEUE_ONFAILURE_NAME}'
EIC_QUEUE_ONFAILURE_URL='${EIC_QUEUE_ONFAILURE_URL}'
EIC_QUEUE_ONFAILURE_ARN='${EIC_QUEUE_ONFAILURE_ARN}'
EOF

echo ""
echo "=== Bootstrap Resources Created ==="
echo "  S3 Bucket:          ${BUCKET_NAME}"
echo "  Signing Profile:    ${SIGNING_PROFILE_NAME}"
echo "  Basic Role:         ${BASIC_ROLE_NAME}"
echo "  ESM Role:           ${ESM_ROLE_NAME}"
echo "  ESM Table:          ${ESM_TABLE_NAME}"
echo "  ESM Queue:          ${ESM_QUEUE_NAME}"
echo "  EIC Role:           ${EIC_ROLE_NAME}"
echo "  EIC Queue Success:  ${EIC_QUEUE_ONSUCCESS_NAME}"
echo "  EIC Queue Failure:  ${EIC_QUEUE_ONFAILURE_NAME}"
echo ""
echo "Resource details written to: ${ENV_FILE}"

# Optionally run pickle.sh
if [[ "${1:-}" == "--pickle" ]]; then
    echo ""
    echo "Running pickle.sh..."
    "${SCRIPT_DIR}/pickle.sh"
fi
