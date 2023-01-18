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
	if resp.Configuration.CodeSha256 != nil {
		ko.Status.CodeSHA256 = resp.Configuration.CodeSha256
	} else {
		ko.Status.CodeSHA256 = nil
	}
	if resp.Configuration.CodeSize != nil {
		ko.Status.CodeSize = resp.Configuration.CodeSize
	} else {
		ko.Status.CodeSize = nil
	}
	if resp.Configuration.DeadLetterConfig != nil {
		f2 := &svcapitypes.DeadLetterConfig{}
		if resp.Configuration.DeadLetterConfig.TargetArn != nil {
			f2.TargetARN = resp.Configuration.DeadLetterConfig.TargetArn
		}
		ko.Spec.DeadLetterConfig = f2
	} else {
		ko.Spec.DeadLetterConfig = nil
	}
	if resp.Configuration.Description != nil {
		ko.Spec.Description = resp.Configuration.Description
	} else {
		ko.Spec.Description = nil
	}
	if resp.Configuration.Environment != nil {
		f4 := &svcapitypes.Environment{}
		if resp.Configuration.Environment.Variables != nil {
			f4f1 := map[string]*string{}
			for f4f1key, f4f1valiter := range resp.Configuration.Environment.Variables {
				var f4f1val string
				f4f1val = *f4f1valiter
				f4f1[f4f1key] = &f4f1val
			}
			f4.Variables = f4f1
		}
		ko.Spec.Environment = f4
	} else {
		ko.Spec.Environment = nil
	}
	if resp.Configuration.FileSystemConfigs != nil {
		f5 := []*svcapitypes.FileSystemConfig{}
		for _, f5iter := range resp.Configuration.FileSystemConfigs {
			f5elem := &svcapitypes.FileSystemConfig{}
			if f5iter.Arn != nil {
				f5elem.ARN = f5iter.Arn
			}
			if f5iter.LocalMountPath != nil {
				f5elem.LocalMountPath = f5iter.LocalMountPath
			}
			f5 = append(f5, f5elem)
		}
		ko.Spec.FileSystemConfigs = f5
	} else {
		ko.Spec.FileSystemConfigs = nil
	}

	if ko.Status.ACKResourceMetadata == nil {
		ko.Status.ACKResourceMetadata = &ackv1alpha1.ResourceMetadata{}
	}
	if resp.Configuration.FunctionArn != nil {
		arn := ackv1alpha1.AWSResourceName(*resp.Configuration.FunctionArn)
		ko.Status.ACKResourceMetadata.ARN = &arn
	}
	if resp.Configuration.FunctionName != nil {
		ko.Spec.Name = resp.Configuration.FunctionName
	} else {
		ko.Spec.Name = nil
	}
	if resp.Configuration.Handler != nil {
		ko.Spec.Handler = resp.Configuration.Handler
	} else {
		ko.Spec.Handler = nil
	}
	if resp.Configuration.ImageConfigResponse != nil {
		f9 := &svcapitypes.ImageConfigResponse{}
		if resp.Configuration.ImageConfigResponse.Error != nil {
			f9f0 := &svcapitypes.ImageConfigError{}
			if resp.Configuration.ImageConfigResponse.Error.ErrorCode != nil {
				f9f0.ErrorCode = resp.Configuration.ImageConfigResponse.Error.ErrorCode
			}
			if resp.Configuration.ImageConfigResponse.Error.Message != nil {
				f9f0.Message = resp.Configuration.ImageConfigResponse.Error.Message
			}
			f9.Error = f9f0
		}
		if resp.Configuration.ImageConfigResponse.ImageConfig != nil {
			f9f1 := &svcapitypes.ImageConfig{}
			if resp.Configuration.ImageConfigResponse.ImageConfig.Command != nil {
				f9f1f0 := []*string{}
				for _, f9f1f0iter := range resp.Configuration.ImageConfigResponse.ImageConfig.Command {
					var f9f1f0elem string
					f9f1f0elem = *f9f1f0iter
					f9f1f0 = append(f9f1f0, &f9f1f0elem)
				}
				f9f1.Command = f9f1f0
			}
			if resp.Configuration.ImageConfigResponse.ImageConfig.EntryPoint != nil {
				f9f1f1 := []*string{}
				for _, f9f1f1iter := range resp.Configuration.ImageConfigResponse.ImageConfig.EntryPoint {
					var f9f1f1elem string
					f9f1f1elem = *f9f1f1iter
					f9f1f1 = append(f9f1f1, &f9f1f1elem)
				}
				f9f1.EntryPoint = f9f1f1
			}
			if resp.Configuration.ImageConfigResponse.ImageConfig.WorkingDirectory != nil {
				f9f1.WorkingDirectory = resp.Configuration.ImageConfigResponse.ImageConfig.WorkingDirectory
			}
			f9.ImageConfig = f9f1
		}
		ko.Status.ImageConfigResponse = f9
	} else {
		ko.Status.ImageConfigResponse = nil
	}
	if resp.Configuration.KMSKeyArn != nil {
		ko.Spec.KMSKeyARN = resp.Configuration.KMSKeyArn
	} else {
		ko.Spec.KMSKeyARN = nil
	}
	if resp.Configuration.LastModified != nil {
		ko.Status.LastModified = resp.Configuration.LastModified
	} else {
		ko.Status.LastModified = nil
	}
	if resp.Configuration.LastUpdateStatus != nil {
		ko.Status.LastUpdateStatus = resp.Configuration.LastUpdateStatus
	} else {
		ko.Status.LastUpdateStatus = nil
	}
	if resp.Configuration.LastUpdateStatusReason != nil {
		ko.Status.LastUpdateStatusReason = resp.Configuration.LastUpdateStatusReason
	} else {
		ko.Status.LastUpdateStatusReason = nil
	}
	if resp.Configuration.LastUpdateStatusReasonCode != nil {
		ko.Status.LastUpdateStatusReasonCode = resp.Configuration.LastUpdateStatusReasonCode
	} else {
		ko.Status.LastUpdateStatusReasonCode = nil
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
	if resp.Configuration.MasterArn != nil {
		ko.Status.MasterARN = resp.Configuration.MasterArn
	} else {
		ko.Status.MasterARN = nil
	}
	if resp.Configuration.MemorySize != nil {
		ko.Spec.MemorySize = resp.Configuration.MemorySize
	} else {
		ko.Spec.MemorySize = nil
	}
	if resp.Configuration.PackageType != nil {
		ko.Spec.PackageType = resp.Configuration.PackageType
	} else {
		ko.Spec.PackageType = nil
	}
	if resp.Configuration.RevisionId != nil {
		ko.Status.RevisionID = resp.Configuration.RevisionId
	} else {
		ko.Status.RevisionID = nil
	}
	if resp.Configuration.Role != nil {
		ko.Spec.Role = resp.Configuration.Role
	} else {
		ko.Spec.Role = nil
	}
	if resp.Configuration.Runtime != nil {
		ko.Spec.Runtime = resp.Configuration.Runtime
	} else {
		ko.Spec.Runtime = nil
	}
	if resp.Configuration.SigningJobArn != nil {
		ko.Status.SigningJobARN = resp.Configuration.SigningJobArn
	} else {
		ko.Status.SigningJobARN = nil
	}
	if resp.Configuration.SigningProfileVersionArn != nil {
		ko.Status.SigningProfileVersionARN = resp.Configuration.SigningProfileVersionArn
	} else {
		ko.Status.SigningProfileVersionARN = nil
	}
	if resp.Configuration.State != nil {
		ko.Status.State = resp.Configuration.State
	} else {
		ko.Status.State = nil
	}
	if resp.Configuration.StateReason != nil {
		ko.Status.StateReason = resp.Configuration.StateReason
	} else {
		ko.Status.StateReason = nil
	}
	if resp.Configuration.StateReasonCode != nil {
		ko.Status.StateReasonCode = resp.Configuration.StateReasonCode
	} else {
		ko.Status.StateReasonCode = nil
	}
	if resp.Configuration.Timeout != nil {
		ko.Spec.Timeout = resp.Configuration.Timeout
	} else {
		ko.Spec.Timeout = nil
	}
	if resp.Configuration.TracingConfig != nil {
		f28 := &svcapitypes.TracingConfig{}
		if resp.Configuration.TracingConfig.Mode != nil {
			f28.Mode = resp.Configuration.TracingConfig.Mode
		}
		ko.Spec.TracingConfig = f28
	} else {
		ko.Spec.TracingConfig = nil
	}
	if resp.Configuration.Version != nil {
		ko.Status.Version = resp.Configuration.Version
	} else {
		ko.Status.Version = nil
	}
	if resp.Configuration.VpcConfig != nil {
		f30 := &svcapitypes.VPCConfig{}
		if resp.Configuration.VpcConfig.SecurityGroupIds != nil {
			f30f0 := []*string{}
			for _, f30f0iter := range resp.Configuration.VpcConfig.SecurityGroupIds {
				var f30f0elem string
				f30f0elem = *f30f0iter
				f30f0 = append(f30f0, &f30f0elem)
			}
			f30.SecurityGroupIDs = f30f0
		}
		if resp.Configuration.VpcConfig.SubnetIds != nil {
			f30f1 := []*string{}
			for _, f30f1iter := range resp.Configuration.VpcConfig.SubnetIds {
				var f30f1elem string
				f30f1elem = *f30f1iter
				f30f1 = append(f30f1, &f30f1elem)
			}
			f30.SubnetIDs = f30f1
		}
		ko.Spec.VPCConfig = f30
	} else {
		ko.Spec.VPCConfig = nil
	}
	if err := rm.setResourceAdditionalFields(ctx, ko); err != nil {
		return nil, err
	}
	if resp.Configuration.EphemeralStorage != nil {
		f31 := &svcapitypes.EphemeralStorage{}
		if resp.Configuration.EphemeralStorage.Size != nil {
			f31.Size = resp.Configuration.EphemeralStorage.Size
		}
		ko.Spec.EphemeralStorage = f31
	} else {
		ko.Spec.EphemeralStorage = nil
	}
	if resp.Configuration.SnapStart != nil {
		f32 := &svcapitypes.SnapStart{}
		if resp.Configuration.SnapStart.ApplyOn != nil {
			f32.ApplyOn = resp.Configuration.SnapStart.ApplyOn
		}
		ko.Spec.SnapStart = f32
	} else {
		ko.Spec.SnapStart = nil
	}