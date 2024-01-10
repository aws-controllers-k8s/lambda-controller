var marker *string = nil
var versionList []*svcsdk.FunctionConfiguration
for {
	listVersionsInput := &svcsdk.ListVersionsByFunctionInput{
		FunctionName: desired.ko.Spec.FunctionName,
		Marker: marker,
	}
	listVersionResponse, err := rm.sdkapi.ListVersionsByFunctionWithContext(ctx, listVersionsInput)
	if err != nil {
		return nil, err
	}
	versionList = append(versionList, listVersionResponse.Versions...)
	
	if listVersionResponse.NextMarker == nil {
		break
	}
	marker = listVersionResponse.NextMarker 
}