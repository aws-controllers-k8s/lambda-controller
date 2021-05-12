	if brokerCreateFailed(latest) {
		msg := "Broker state is CREATION_FAILED"
		setTerminalCondition(desired, corev1.ConditionTrue, &msg, nil)
		setSyncedCondition(desired, corev1.ConditionTrue, nil, nil)
		return desired, nil
	}
	if brokerCreateInProgress(latest) {
		msg := "Broker state is CREATION_IN_PROGRESS"
		setSyncedCondition(desired, corev1.ConditionFalse, &msg, nil)
		return desired, requeueWaitWhileCreating
	}
	if brokerDeleteInProgress(latest) {
		msg := "Broker state is DELETION_IN_PROGRESS"
		setSyncedCondition(desired, corev1.ConditionFalse, &msg, nil)
		return desired, requeueWaitWhileDeleting
	}
