package selfupdate

// SpecificFileUpdatableResolver resolves the thing to be updated with a path
// to a specific file on disk.
type SpecificFileUpdatableResolver struct {
	path string
}

// NewSpecificFileUpdatableResolver returns a resolver that resolves to the
// specific file path passed in.
func NewSpecificFileUpdatableResolver(path string) SpecificFileUpdatableResolver {
	return SpecificFileUpdatableResolver{path}
}

// Resolve attempts to find the current executable running this program
func (c SpecificFileUpdatableResolver) Resolve() (string, error) {
	return c.path, nil
}
