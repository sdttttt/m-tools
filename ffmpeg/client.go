// Package ffmpeg wraps the ffmpeg/ffprobe command-line tools behind a small
// Go API. Executable lookup, CLI arguments, filter chains and output parsing
// stay inside the package: callers pass plain data in and get plain data
// out.
package ffmpeg

// Client is a ready-to-use handle on resolved ffmpeg and ffprobe
// executables. It is stateless and safe for concurrent use.
type Client struct {
	ffmpeg  string
	ffprobe string
}

// New resolves the ffmpeg and ffprobe executables and returns a client.
// Lookup order: $EnvDir, the directory returned by the resolver registered
// via UseBundledDir (see the with_ffmpeg build), a bin directory next to the
// running executable, then PATH.
func New() (*Client, error) {
	ffmpeg, ffprobe, err := findTools()
	if err != nil {
		return nil, err
	}
	return &Client{ffmpeg: ffmpeg, ffprobe: ffprobe}, nil
}
