package selfupdate

//go:generate mockgen -destination=./mocks/updatable_resolver.go -package=mocks -source=updatable_resolver.go

// UpdatableResolver finds the thing that needs to be updated.
type UpdatableResolver interface {
	Resolve() (string, error)
}
