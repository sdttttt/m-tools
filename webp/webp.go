// Package webp bridges WebP decode/encode behind a small Go API, mirroring
// the ffmpeg module's boundary rule: callers pass plain data in and get
// plain data out; the cgo dependency stays inside this module.
//
// The codec itself is only compiled into `-tags with_webp` builds: the cgo
// wrapper github.com/chai2010/webp (which vendors the libwebp C sources)
// keeps default builds pure Go. Without the tag the same functions return
// ErrUnsupported, and image.RegisterFormat is skipped so image.Decode does
// not claim webp bytes it cannot read.
//
// Supported callsites must register the format with image/image at init on
// real builds (Register() does this); services should gate webp features on
// Supported() and surface ErrUnsupported text to the user otherwise.
package webp

import "errors"

// ErrUnsupported is returned by every function when the with_webp build tag
// is absent. Errors.Is-compatible; the service layer surfaces it as a
// user-facing notice.
var ErrUnsupported = errors.New("webp 支持未编译：请使用 -tags with_webp 构建")
