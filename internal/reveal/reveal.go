// Package reveal opens a directory in the OS file manager. Platform
// differences live in build-tagged files; every toolbox service can share
// it.
package reveal

// Open opens dir in the OS file manager: Finder on macOS, Explorer on
// Windows, xdg-open elsewhere. It does not wait for the file manager to
// exit.
func Open(dir string) error {
	return open(dir)
}
