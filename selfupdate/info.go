package selfupdate

// versionInfo contains the name and the sha of a specific version.
type versionInfo struct {
	Version string
	Sha256  []byte
}
