package agentbase

type credentialError struct{ cause error }

func (e *credentialError) Error() string { return "authentication failed" }
func (e *credentialError) Unwrap() error { return e.cause }

func authenticationError(err error) error { return &credentialError{cause: err} }
