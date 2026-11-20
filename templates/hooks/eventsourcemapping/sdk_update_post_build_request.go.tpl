
    // We need to carefully craft the update request if a user
    // wants to delete their filterCriterias. Mainly because the
    // aws-sdk-go doesn't try to update nil fields.
    if filterCriteriasDeleted(latest, desired, delta) {
        input.FilterCriteria = &svcsdktypes.FilterCriteria{
            Filters: []svcsdktypes.Filter{},
        }
    }
    // Lambda rejects an OnFailure/OnSuccess entry without a Destination.
    // Older controller versions persisted `onFailure: {}` into the spec, so
    // strip empty entries from the request.
    input.DestinationConfig = cleanEmptyDestinationConfigInput(input.DestinationConfig)
