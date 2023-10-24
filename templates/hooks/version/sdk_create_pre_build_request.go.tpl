res := &svcsdk.ListVersionsByFunctionInput{}
res.FunctionName = desired.ko.Spec.FunctionName
var list *svcsdk.ListVersionsByFunctionOutput
list, err = rm.sdkapi.ListVersionsByFunctionWithContext(ctx, res)
if err != nil {
	return nil, err
}
versionList := list.Versions

for ok := list.NextMarker != nil; ok; ok = (list.NextMarker != nil) {
	res.Marker = list.NextMarker
	list, err = rm.sdkapi.ListVersionsByFunctionWithContext(ctx, res)
	if err != nil {
		return nil, err
	}
	versionList = append(versionList, list.Versions...)
}

