	if resp.Code != nil {
		// We need to keep the desired .Code s3Bucket s3Key and s3ObjectVersion
		// part of the function's spec. So instead of setting Spec.Code to nil
		// we only set ImageURI
		//
		// When adopting a Function resource, Spec.Code field should be manually
		// initialised before injecting ImageURI.
		if ko.Spec.Code == nil {
			ko.Spec.Code = &svcapitypes.FunctionCode{}
		}
		if resp.Code.ImageUri != nil {
			ko.Spec.Code.ImageURI = resp.Code.ImageUri
		}
	}
	if resp.Configuration.Layers != nil {
		f16 := []*svcapitypes.Layer{}
		for _, f16iter := range resp.Configuration.Layers {
			f16elem := &svcapitypes.Layer{}
			if f16iter.Arn != nil {
				f16elem.ARN = f16iter.Arn
			}
			if f16iter.CodeSize != nil {
				f16elem.CodeSize = f16iter.CodeSize
			}
			if f16iter.SigningJobArn != nil {
				f16elem.SigningJobARN = f16iter.SigningJobArn
			}
			if f16iter.SigningProfileVersionArn != nil {
				f16elem.SigningProfileVersionARN = f16iter.SigningProfileVersionArn
			}
			f16 = append(f16, f16elem)
		}
		ko.Status.LayerStatuses = f16
	} else {
		ko.Status.LayerStatuses = nil
	}
	if resp.Tags != nil {
		expectedOutput := map[string]*string{}
		for expectedOutputKey, expectedOutputIter := range resp.Tags {
			var expectedOutputVal string
			expectedOutputVal = *expectedOutputIter
			expectedOutput[expectedOutputKey] = &expectedOutputVal
		}
		ko.Spec.Tags = expectedOutput  
	}
	if err := rm.setResourceAdditionalFields(ctx, ko); err != nil {
		return nil, err
	}