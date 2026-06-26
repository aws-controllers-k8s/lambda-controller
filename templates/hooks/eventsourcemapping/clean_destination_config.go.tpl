	if ko.Spec.DestinationConfig != nil {
		if ko.Spec.DestinationConfig.OnFailure != nil && ko.Spec.DestinationConfig.OnFailure.Destination == nil {
			ko.Spec.DestinationConfig.OnFailure = nil
		}
		if ko.Spec.DestinationConfig.OnSuccess != nil && ko.Spec.DestinationConfig.OnSuccess.Destination == nil {
			ko.Spec.DestinationConfig.OnSuccess = nil
		}
		if ko.Spec.DestinationConfig.OnFailure == nil && ko.Spec.DestinationConfig.OnSuccess == nil {
			ko.Spec.DestinationConfig = nil
		}
	}
