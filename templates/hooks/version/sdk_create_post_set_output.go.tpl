if ko.Spec.FunctionEventInvokeConfig != nil {
   err = rm.syncEventInvokeConfig(ctx,desired)
   if err != nil{
      return nil, err
   }
}
if ko.Spec.ProvisionedConcurrencyConfig != nil {
   err = rm.updateProvisionedConcurrency(ctx,desired)
   if err != nil{
      return nil, err
   }
}