if ko.Spec.FunctionEventInvokeConfig != nil {
   err = rm.syncEventInvokeConfig(ctx,desired)
   if err != nil{
      return nil, err
   }
}