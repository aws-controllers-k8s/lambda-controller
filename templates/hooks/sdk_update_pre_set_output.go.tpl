	// Copy status from latest observed state
    latestKOStatus := latest.ko.DeepCopy().Status
    ko.Status = latestKOStatus