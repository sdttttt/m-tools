package imgconv

import (
	"bytes"
	"image"
	"image/color"
	"testing"

	"github.com/sdttttt/m-tools/webp"
)

// testWebPBytes produces webp bytes for probe tests: a real encode on
// with_webp builds, a bare RIFF/WEBP container header otherwise (the stub
// path reports the missing codec before any decode is attempted).
func testWebPBytes(t *testing.T) []byte {
	t.Helper()
	if webp.Supported() {
		var buf bytes.Buffer
		m := image.NewNRGBA(image.Rect(0, 0, 8, 8))
		for y := 0; y < 8; y++ {
			for x := 0; x < 8; x++ {
				m.Set(x, y, color.RGBA{B: 255, A: 255})
			}
		}
		if err := webp.Encode(&buf, m, 90); err != nil {
			t.Fatalf("webp.Encode: %v", err)
		}
		return buf.Bytes()
	}
	hdr := make([]byte, 32)
	copy(hdr, "RIFF")
	putLE32(hdr[4:], 24)
	copy(hdr[8:], "WEBP")
	copy(hdr[12:], "VP8 ")
	putLE32(hdr[16:], 16)
	return hdr
}

func putLE32(b []byte, v uint32) {
	b[0] = byte(v)
	b[1] = byte(v >> 8)
	b[2] = byte(v >> 16)
	b[3] = byte(v >> 24)
}

// TestStubDecodeRejectsWebp pins the stub contract: default builds never
// decode webp through image.Decode, because Register is a no-op.
func TestStubDecodeRejectsWebp(t *testing.T) {
	if webp.Supported() {
		t.Skip("webp codec compiled in; stub behavior not reachable")
	}
	data := testWebPBytes(t)
	if _, _, err := image.Decode(bytes.NewReader(data)); err == nil {
		t.Fatal("image.Decode accepted webp bytes on a stub build")
	}
}
