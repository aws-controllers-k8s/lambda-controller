	if desired.ko.Spec.CodeSigningConfigARN != nil && *desired.ko.Spec.CodeSigningConfigARN == "" {
		input.CodeSigningConfigArn = nil
	}

	if desired.ko.Spec.Environment != nil {
		envInput := &svcsdktypes.Environment{}
		combinedVariables, err := combineEnvironmentVariableSources(ctx, desired, rm)
		if err != nil {
			return nil, err
		}

		envInput.Variables = combinedVariables
		input.Environment = envInput
	}