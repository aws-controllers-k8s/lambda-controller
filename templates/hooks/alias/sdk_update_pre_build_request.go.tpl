if delta.DifferentAt("Spec.FunctionEventInvokeConfig"){
    _, err = rm.syncEventInvokeConfig(ctx,desired)
    if err != nil {
        return nil, err
    }
}
if !delta.DifferentExcept("Spec.FunctionEventInvokeConfig"){
    return desired, nil
}