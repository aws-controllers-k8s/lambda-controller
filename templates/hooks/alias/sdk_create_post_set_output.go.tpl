   if ko.Spec.FunctionEventInvokeConfig != nil {
      _, err = rm.syncEventInvokeConfig(ctx,desired)
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

   if len(ko.Spec.Permissions) > 0 {
		aliasCopy := ko.DeepCopy()
		aliasCopy.Spec.Permissions = nil
		err = rm.syncPermissions(ctx, desired, &resource{aliasCopy})
		if err != nil {
			return nil, err
		}
   }