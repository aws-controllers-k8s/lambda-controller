	if resp.Layers != nil {
		f16 := []*svcapitypes.Layer{}
		for _, f16iter := range resp.Layers {
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