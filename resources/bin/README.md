# Bundled ffmpeg binaries (build tag `with_ffmpeg`)

Place `ffmpeg[.exe]` and `ffprobe[.exe]` in this directory before building
with `go build -tags with_ffmpeg`. Layout is flat; the build embeds
everything except this README and extracts the binaries to the user cache
directory at runtime.

Without the tag, the app looks up ffmpeg via `$MTOOLS_FFMPEG_DIR`, a `bin`
directory next to the executable, then PATH.

This directory is gitignored except for this file.
