	if err != nil && strings.Contains(err.Error(), "The role defined for the function cannot be assumed by Lambda") {
		return nil, requeueWaitWhileRoleCannotBeAssumed
	}
