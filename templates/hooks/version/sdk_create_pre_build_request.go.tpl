var marker *string = nil
var versionList []svcsdktypes.FunctionConfiguration
for {
	listVersionsInput := &svcsdk.ListVersionsByFunctionInput{
		FunctionName: desired.ko.Spec.FunctionName,
		Marker: marker,
	}
	listVersionResponse, err := rm.sdkapi.ListVersionsByFunction(ctx, listVersionsInput)
	if err != nil {
		return nil, err
	}
	versionList = append(versionList, listVersionResponse.Versions...)
	
	if listVersionResponse.NextMarker == nil {
		break
	}
	marker = listVersionResponse.NextMarker 
}