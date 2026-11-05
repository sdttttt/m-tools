//go:build with_webp

package webp

import (
	"bytes"
	"image"
	"io"

	"github.com/chai2010/webp"
)

// webpBuilt records the build variant for tests.
const webpBuilt = true

// Supported reports whether WebP decode/encode is compiled in.
func Supported() bool { return true }

// Register adds webp to the image package's format registry. Called from
// init; stub builds skip it so image.Decode keeps returning ErrUnsupported.
func Register() { image.RegisterFormat("webp", "RIFF????WEBPVP8", Decode, DecodeConfig) }

// Decode reads a WebP image from r.
func Decode(r io.Reader) (image.Image, error) { return webp.Decode(r) }

// DecodeConfig reads the dimensions of the WebP image in r.
func DecodeConfig(r io.Reader) (image.Config, error) { return webp.DecodeConfig(r) }

// Encode writes m to w as a lossy WebP image (quality 0-100, like libwebp).
func Encode(w *bytes.Buffer, m image.Image, quality int) error {
	data, err := webp.EncodeRGBA(m, float32(quality))
	if err != nil {
		return err
	}
	_, err = w.Write(data)
	return err
}

// EncodeLossless writes m to w as a lossless WebP image (VP8L): the
// decoder output is bit-identical to the encoder input.
func EncodeLossless(m image.Image) ([]byte, error) {
	return webp.EncodeLosslessRGBA(m)
}
