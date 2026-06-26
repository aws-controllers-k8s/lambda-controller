package code_signing_config

import (
	"context"

	acktags "github.com/aws-controllers-k8s/lambda-controller/pkg/resource/tags"
)

func (rm *resourceManager) getTags(
	ctx context.Context,
	resourceARN string,
) (map[string]*string, error) {
	return acktags.GetTags(ctx, rm.sdkapi, rm.metrics, resourceARN)
}

func (rm *resourceManager) syncTags(
	ctx context.Context,
	desired *resource,
	latest *resource,
) error {
	return acktags.SyncTags(
		ctx, rm.sdkapi, rm.metrics,
		string(*latest.ko.Status.ACKResourceMetadata.ARN),
		desired.ko.Spec.Tags, latest.ko.Spec.Tags,
	)
}
