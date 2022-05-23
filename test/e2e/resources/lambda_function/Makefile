AWS_REGION ?= "us-west-2"
ECR_REPOSITORY ?= ack-e2e-testing-lambda-controller
IMAGE_TAG ?= v1

AWS_ACCOUNT_ID ?= $(shell aws sts get-caller-identity --query "Account" --output text)
IMAGE_URL ?= $(AWS_ACCOUNT_ID).dkr.ecr.us-west-2.amazonaws.com/$(ECR_REPOSITORY):$(IMAGE_TAG)

build-image:
	docker build -t $(IMAGE_URL) .

publish-image:
	docker push $(IMAGE_URL)

create-ecr-repository:
	aws ecr create-repository --region $(AWS_REGION) --repository-name $(ECR_REPOSITORY) >/dev/null

all: build-image publish-image