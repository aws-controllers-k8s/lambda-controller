	if brokerDeleteInProgress(r) {
		return r, requeueWaitWhileDeleting
	}
