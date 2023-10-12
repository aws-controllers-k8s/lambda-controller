for _, version := range bigList{
    if *version.Version == *resp.Version{
        ErrCannotCreateResource := errors.New("No changes were made to $LATEST since publishing last version, so no version was published.")
        return nil, ackerr.NewTerminalError(ErrCannotCreateResource)
    }
}