package selfupdate

//go:generate mockgen -destination=./mocks/platform_resolver.go -package=mocks -source=platform_resolver.go

// PlatformResolver determines the platform that that the file exists for that
// needs to be updated.
type PlatformResolver interface {
	Resolve() (string, error)
}
