package imgconv

// ImageInfo describes one picked image. HasAlpha flags sources that would
// need flattening when the target is JPEG; Format is the canonical
// lowercase extension-style id (png, jpg, gif, webp, bmp, tiff).
type ImageInfo struct {
	Path       string `json:"path"`
	Format     string `json:"format"`
	Width      int    `json:"width"`
	Height     int    `json:"height"`
	SizeBytes  int64  `json:"sizeBytes"`
	HasAlpha   bool   `json:"hasAlpha"`
	Supported  bool   `json:"supported"` // decodable by this build
	SkipReason string `json:"skipReason"`
}

// ConvertRequest asks for one image format conversion.
type ConvertRequest struct {
	Path         string `json:"path"`
	TargetFormat string `json:"targetFormat"` // jpg | png | gif | webp | bmp | tiff
	Quality      string `json:"quality"`      // high | balanced | compact; JPEG/WebP only
	MaxWidth     int    `json:"maxWidth"`     // 0 keeps the source size; only shrinks
}

// ConvertResult reports one finished conversion. OutputPath sits next to
// the source file (-1 suffix on name clashes) — the file is already saved;
// SaveAs exists for "save a copy elsewhere".
type ConvertResult struct {
	OutputPath     string `json:"outputPath"`
	Format         string `json:"format"`
	SizeBytes      int64  `json:"sizeBytes"`
	Width          int    `json:"width"`
	Height         int    `json:"height"`
	DurationMs     int64  `json:"durationMs"`
	PreviewDataURL string `json:"previewDataURL"`
}
