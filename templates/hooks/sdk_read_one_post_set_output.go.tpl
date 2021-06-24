	if brokerCreateInProgress(&resource{ko}) {
		return &resource{ko}, requeueWaitWhileCreating
	}