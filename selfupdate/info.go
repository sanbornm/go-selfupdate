package selfupdate

// VersionInfo contains the name and the sha of a specific version.
type VersionInfo struct {
	Version string
	Sha256  []byte
}
