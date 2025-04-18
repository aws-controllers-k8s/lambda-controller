	if r.ko.Spec.VPCConfig != nil {
		if ko.Spec.VPCConfig == nil {
			ko.Spec.VPCConfig = &svcapitypes.VPCConfig{}
		}
		ko.Spec.VPCConfig.SecurityGroupRefs = r.ko.Spec.VPCConfig.SecurityGroupRefs
		ko.Spec.VPCConfig.SubnetRefs = r.ko.Spec.VPCConfig.SubnetRefs
	}
	if resp.Code != nil {
		if ko.Spec.Code == nil {
			ko.Spec.Code = &svcapitypes.FunctionCode{}
		}
		if resp.Code.ImageUri != nil {
			ko.Spec.Code.ImageURI = resp.Code.ImageUri
		}
	}
	if r.ko.Spec.Code != nil && r.ko.Spec.Code.S3BucketRef != nil {
		ko.Spec.Code.S3BucketRef = r.ko.Spec.Code.S3BucketRef
	}
	if resp.Configuration.Layers != nil {
		f16 := []*svcapitypes.Layer{}
		layer := []*string{}
		for _, f16iter := range resp.Configuration.Layers {
			f16elem := &svcapitypes.Layer{}
			if f16iter.Arn != nil {
				f16elem.ARN = f16iter.Arn
			}
			if f16iter.CodeSize != 0 {
				f16elem.CodeSize = aws.Int64(f16iter.CodeSize)
			}
			if f16iter.SigningJobArn != nil {
				f16elem.SigningJobARN = f16iter.SigningJobArn
			}
			if f16iter.SigningProfileVersionArn != nil {
				f16elem.SigningProfileVersionARN = f16iter.SigningProfileVersionArn
			}
			f16 = append(f16, f16elem)
			layer = append(layer, f16iter.Arn)
		}
		ko.Spec.Layers = layer
		ko.Status.LayerStatuses = f16
	} else {
		ko.Status.LayerStatuses = nil
	}
	if resp.Tags != nil {
		expectedOutput := map[string]*string{}
		for expectedOutputKey, expectedOutputIter := range resp.Tags {
			var expectedOutputVal string
			expectedOutputVal = expectedOutputIter
			expectedOutput[expectedOutputKey] = &expectedOutputVal
		}
		ko.Spec.Tags = expectedOutput  
	}
	if err := rm.setResourceAdditionalFields(ctx, ko); err != nil {
		return nil, err
	}