if ko.Spec.FunctionEventInvokeConfig != nil {
   _, err = rm.syncEventInvokeConfig(ctx,desired)
   if err != nil{
      return nil, err
   }
}