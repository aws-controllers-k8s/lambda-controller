    if delta.DifferentAt("Spec.FunctionEventInvokeConfig"){
        _, err = rm.syncEventInvokeConfig(ctx,desired)
        if err != nil {
            return nil, err
        }
    }
    if delta.DifferentAt("Spec.ProvisionedConcurrencyConfig"){
        err = rm.updateProvisionedConcurrency(ctx, desired)
        if err != nil {
            return nil, err
        }
    }
	if delta.DifferentAt("Spec.Permissions") {
		err = rm.syncPermissions(ctx, desired, latest)
		if err != nil {
			return nil, err
		}
	}
    if !delta.DifferentExcept("Spec.ProvisionedConcurrencyConfig", "Spec.FunctionEventInvokeConfig", "Spec.Permissions") {
        return desired, nil
    }