package imgresize

// ResizeRequest asks for one image size adjustment. Mode percent sizes by
// a percentage of the source (aspect always kept); mode exact uses the
// given width/height, where an empty axis is derived from the source ratio
// when Lock is set. TargetFormat doubles as the output format (may equal
// the source format for a pure resize).
type ResizeRequest struct {
	Path         string `json:"path"`
	TargetFormat string `json:"targetFormat"` // jpg | png | gif | webp | bmp | tiff
	Quality      string `json:"quality"`      // high | balanced | compact; JPEG/WebP only
	Mode         string `json:"mode"`         // percent | exact
	Percent      int    `json:"percent"`      // 1–400, percent mode
	Width        int    `json:"width"`        // exact mode; 0 = derive/keep
	Height       int    `json:"height"`       // exact mode; 0 = derive/keep
	Lock         bool   `json:"lock"`         // derive the missing axis from the source ratio
}

// ResizeResult reports one finished resize. OutputPath sits next to the
// source file (-1 suffix on name clashes).
type ResizeResult struct {
	OutputPath     string `json:"outputPath"`
	Format         string `json:"format"`
	SizeBytes      int64  `json:"sizeBytes"`
	Width          int    `json:"width"`
	Height         int    `json:"height"`
	DurationMs     int64  `json:"durationMs"`
	PreviewDataURL string `json:"previewDataURL"`
}
