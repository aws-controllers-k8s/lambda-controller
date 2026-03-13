
## Testing

You will need an AWS account, a kubernetes cluster (running locally is fine)
and python

### Container image build

This assumes a target docker architecture of linux/arm64 (typically for OS X)

1. `CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -o bin/controller ./cmd/controller/`
2. `docker build -f Dockerfile.local -t lambda-controller:local .`

### Kubernetes setup

Create a temporary credentials file in `~/.aws.temporary.credss` that you don't mind using with ack-controller

```
[temp-profile]
aws_access_key_id=<access key>
aws_secret_access_key=<secret key>
aws_session_token=<session token>
```

1. `kubectl create namespace ack-system`
2. `kubectl create secret generic aws-credentials --from-file=credentials=$HOME/.aws.temporary.creds -n ack-system
3. ```
   helm install ack-lambda-controller ./helm \
     --namespace ack-system \
     --set image.repository=lambda-controller \
     --set image.tag=local \
     --set aws.region=ap-southeast-2 \
     --set aws.credentials.secretName=aws-credentials \
     --set aws.credentials.secretKey=credentials \
     --set aws.credentials.profile=temp-profile \
     --set installScope=cluster \
     --set leaderElection.enabled=false
   ```

### Python set-up

In `test/e2e`:

1. Create a virtual environment `python -m venv venv`
2. Activate the virtual environment `source venv/bin/activate`
3. Install testing requirements `pip install -r requirements.txt`

### AWS setup

In `test/e2e`:

1. Run `AWS_PROFILE=my-profile ./setup.sh --pickle` (or run `setup.sh` and then
   `pickle.sh`)

### Setup the docker test image used by the lambda

In `test/e2e/resources/lambda_function`:

1. `AWS_PROFILE=my-profile aws ecr create-repository --repository-name ack-e2e-testing-lambda-controller`
2. Run `AWS_PROFILE=my-profile make`
3. ```
   repo=$(AWS_PROFILE=my-profile aws ecr describe-repositories --repository-names ack-e2e-testing-lambda-controller --query 'repositories[].repositoryUri' --output text)
   AWS_PROFILE=my-profile aws ecr get-login-password | docker login --password-stdin -u "${repo%/*}"
   docker push ${repo}:v1
   ```
4. Run `zip main.zip main.py` and `zip updated_main.zip updated_main.py`

### Run the test suite

In `test/e2e`:

1. Run `AWS_PROFILE=my-profile pytest`

### Clean up

In test/e2e:

1. Run `AWS_PROFILE=my-profile ./teardown.sh`
