// Package mtools holds app-level assets and build-time hooks that must live
// at the repository root: go:embed cannot reference paths above the package
// directory, so the bundled-ffmpeg embedding (with_ffmpeg build) registers
// itself with the ffmpeg module from here. cmd/m-tools imports this package
// for that side effect and for the shared app metadata.
package mtools

// AppName is the human-facing app name, matching mygo.config.ts.
const AppName = "M-Tools"
