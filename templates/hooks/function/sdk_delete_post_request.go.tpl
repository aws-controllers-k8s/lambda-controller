	if err == nil {
		// Durable Functions need to drain durable function executions before the Function is fully deleted.
		// This can result in the Function persisting in the "deleting" state for an extended period.
		if r.ko.Spec.DurableConfig != nil {
			return r, requeueWaitWhileDeleting
		}
	}
