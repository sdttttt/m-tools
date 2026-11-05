// Package imgfmt is the shared image-format substrate for the image tools:
// the format registry (magic sniffing, first-frame decode, per-target
// encode), webp support reporting and the decode→resize→flatten→encode
// pipeline used by conversion and resize. Format knowledge lives here so
// imgconv (format conversion) and imgresize (size adjustment) cannot drift
// apart; service-level concerns (dialogs, jobs, wire types) live in the
// services.
package imgfmt

import (
	"errors"

	"github.com/sdttttt/m-tools/webp"
)

// ErrUnsupported reports a webp source on a default build; the services
// surface it verbatim as a user-facing hint.
var ErrUnsupported = errors.New("webp 支持未编译：需使用 -tags with_webp 构建的版本")

// Supported reports whether WebP decode/encode is compiled in (with_webp
// build). The page calls it once on mount to hide webp options otherwise.
func Supported() bool { return webp.Supported() }
