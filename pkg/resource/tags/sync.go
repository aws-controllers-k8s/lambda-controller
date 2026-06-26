package tags

import (
	"context"

	ackcompare "github.com/aws-controllers-k8s/runtime/pkg/compare"
	ackrtlog "github.com/aws-controllers-k8s/runtime/pkg/runtime/log"
	acktags "github.com/aws-controllers-k8s/runtime/pkg/tags"
	svcsdk "github.com/aws/aws-sdk-go-v2/service/lambda"
)

type metricsRecorder interface {
	RecordAPICall(opType string, opID string, err error)
}

type tagsClient interface {
	ListTags(context.Context, *svcsdk.ListTagsInput, ...func(*svcsdk.Options)) (*svcsdk.ListTagsOutput, error)
	TagResource(context.Context, *svcsdk.TagResourceInput, ...func(*svcsdk.Options)) (*svcsdk.TagResourceOutput, error)
	UntagResource(context.Context, *svcsdk.UntagResourceInput, ...func(*svcsdk.Options)) (*svcsdk.UntagResourceOutput, error)
}

func GetTags(
	ctx context.Context,
	client tagsClient,
	mr metricsRecorder,
	resourceARN string,
) (map[string]*string, error) {
	resp, err := client.ListTags(ctx, &svcsdk.ListTagsInput{
		Resource: &resourceARN,
	})
	mr.RecordAPICall("GET", "ListTags", err)
	if err != nil {
		return nil, err
	}
	tags := make(map[string]*string, len(resp.Tags))
	for k, v := range resp.Tags {
		vCopy := v
		tags[k] = &vCopy
	}
	return tags, nil
}

func SyncTags(
	ctx context.Context,
	client tagsClient,
	mr metricsRecorder,
	resourceARN string,
	desiredTags map[string]*string,
	latestTags map[string]*string,
) error {
	var err error
	rlog := ackrtlog.FromContext(ctx)
	exit := rlog.Trace("tags.SyncTags")
	defer func() { exit(err) }()

	from := toACKTags(latestTags)
	to := toACKTags(desiredTags)

	added, _, removed := ackcompare.GetTagsDifference(from, to)

	if len(removed) > 0 {
		removedKeys := make([]string, 0, len(removed))
		for k := range removed {
			removedKeys = append(removedKeys, k)
		}
		_, err = client.UntagResource(ctx, &svcsdk.UntagResourceInput{
			Resource: &resourceARN,
			TagKeys:  removedKeys,
		})
		mr.RecordAPICall("UPDATE", "UntagResource", err)
		if err != nil {
			return err
		}
	}

	if len(added) > 0 {
		addTags := make(map[string]string, len(added))
		for k, v := range added {
			addTags[k] = v
		}
		_, err = client.TagResource(ctx, &svcsdk.TagResourceInput{
			Resource: &resourceARN,
			Tags:     addTags,
		})
		mr.RecordAPICall("UPDATE", "TagResource", err)
		if err != nil {
			return err
		}
	}

	return nil
}

func toACKTags(tags map[string]*string) acktags.Tags {
	result := acktags.NewTags()
	for k, v := range tags {
		if v != nil {
			result[k] = *v
		} else {
			result[k] = ""
		}
	}
	return result
}
