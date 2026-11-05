//go:build !with_webp

package webp

import (
	"bytes"
	"image"
	"io"
)

// webpBuilt records the build variant for tests.
const webpBuilt = false

// Supported reports whether WebP decode/encode is compiled in.
func Supported() bool { return false }

// Register is a no-op: without the codec, image.Decode must not claim webp.
func Register() {}

// Decode always returns ErrUnsupported on default builds.
func Decode(r io.Reader) (image.Image, error) { return nil, ErrUnsupported }

// DecodeConfig always returns ErrUnsupported on default builds.
func DecodeConfig(r io.Reader) (image.Config, error) { return image.Config{}, ErrUnsupported }

// Encode always returns ErrUnsupported on default builds.
func Encode(w *bytes.Buffer, m image.Image, quality int) error { return ErrUnsupported }

// EncodeLossless always returns ErrUnsupported on default builds.
func EncodeLossless(m image.Image) ([]byte, error) { return nil, ErrUnsupported }
