	if brokerDeleteInProgress(r) {
		return requeueWaitWhileDeleting
	}
