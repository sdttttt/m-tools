package imgconv

// ConvertRequest asks for one image format conversion. The output keeps
// the source pixel size — resizing lives in the ImageResize tool.
type ConvertRequest struct {
	Path         string `json:"path"`
	TargetFormat string `json:"targetFormat"` // jpg | png | gif | webp | bmp | tiff
	Quality      string `json:"quality"`      // high | balanced | compact; JPEG/WebP only
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
