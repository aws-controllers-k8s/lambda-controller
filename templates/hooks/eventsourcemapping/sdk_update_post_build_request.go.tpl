
    // We need to carefully craft the update request if a user
    // wants to delete their filterCriterias. Mainly because the
    // aws-sdk-go doesn't try to update nil fields.
    if filterCriteriasDeleted(latest, desired, delta) {
        input.FilterCriteria = &svcsdk.FilterCriteria{
            Filters: []*svcsdk.Filter{},
        }
    }