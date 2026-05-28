	ko.Spec.Tags, err = rm.getTags(ctx, string(*ko.Status.ACKResourceMetadata.ARN))
	if err != nil {
		return nil, err
	}
