	if isFunctionDeleting(r) {
		return r, requeueWaitWhileDeleting
	}
