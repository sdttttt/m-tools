package imgconv

import (
	"bytes"
	"image"
	"image/gif"
	"image/jpeg"
	"image/png"
	"io"

	bmp "golang.org/x/image/bmp"
	tiff "golang.org/x/image/tiff"

	"github.com/sdttttt/m-tools/webp"
)

// format is one convertible image format: how to read and write it. decode
// returns the first frame for animations; encode receives the decoded,
// already scaled/flattened image. useQuality says whether the quality
// choice means anything for this target.
type format struct {
	name       string
	ext        string
	magic      string // leading signature for source detection
	decode     func(io.Reader) (image.Image, error)
	encode     func(w *bytes.Buffer, m image.Image, quality int) error
	useQuality bool
}

// encodeGIF adapts the stdlib gif encoder's signature; nil options use the
// default 256-color quantization for photographic sources.
func encodeGIF(w *bytes.Buffer, m image.Image, _ int) error {
	return gif.Encode(w, m, nil)
}

// encodePNG adapts the stdlib png encoder's signature.
func encodePNG(w *bytes.Buffer, m image.Image, _ int) error {
	return png.Encode(w, m)
}

// encodeJPEG adapts the stdlib jpeg encoder's signature.
func encodeJPEG(w *bytes.Buffer, m image.Image, quality int) error {
	return jpeg.Encode(w, m, &jpeg.Options{Quality: quality})
}

// encodeBMP and encodeTIFF adapt the x/image encoders' signatures.
func encodeBMP(w *bytes.Buffer, m image.Image, _ int) error { return bmp.Encode(w, m) }

func encodeTIFF(w *bytes.Buffer, m image.Image, _ int) error { return tiff.Encode(w, m, nil) }

// formats is the conversion target table. gif/png/bmp/tiff are lossless
// targets (quality ignored); webp arrives through the webp module, which is
// only functional on with_webp builds — see webpFormat below.
var formats = map[string]*format{
	"jpg": {
		name: "jpg", ext: ".jpg", magic: "\xff\xd8\xff",
		decode: jpeg.Decode, encode: encodeJPEG, useQuality: true,
	},
	"png": {
		name: "png", ext: ".png", magic: "\x89PNG\r\n\x1a\n",
		decode: png.Decode, encode: encodePNG,
	},
	"gif": {
		name: "gif", ext: ".gif", magic: "GIF8",
		decode: gif.Decode, encode: encodeGIF,
	},
	"bmp": {
		name: "bmp", ext: ".bmp", magic: "BM",
		decode: bmp.Decode, encode: encodeBMP,
	},
	"tiff": {
		name: "tiff", ext: ".tiff", magic: "II*\x00",
		decode: tiff.Decode, encode: encodeTIFF,
	},
}

// webpFormat returns the webp table entry when this build can decode webp
// (with_webp builds registered it with image.Decode at init). The magic
// constant covers both VP8 (lossy) and VP8L/VP8X (lossless/extended)
// because the check only looks at the RIFF container header.
func webpFormat() (*format, bool) {
	if !webpSupported() {
		return nil, false
	}
	return &format{
		name: "webp", ext: ".webp", magic: "RIFF????WEBP",
		decode: webp.Decode,
		encode: func(w *bytes.Buffer, m image.Image, quality int) error {
			return webp.Encode(w, m, quality)
		},
		useQuality: true,
	}, true
}
