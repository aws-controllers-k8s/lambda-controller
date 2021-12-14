	if desired.ko.Spec.CodeSigningConfigARN != nil && *desired.ko.Spec.CodeSigningConfigARN == "" {
		input.CodeSigningConfigArn = nil
	}