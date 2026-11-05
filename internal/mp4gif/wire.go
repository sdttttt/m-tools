package mp4gif

// ConvertRequest asks for one video→GIF conversion.
type ConvertRequest struct {
	Path        string  `json:"path"`
	StartSec    float64 `json:"startSec"`
	DurationSec float64 `json:"durationSec"`
	FPS         int     `json:"fps"`
	MaxWidth    int     `json:"maxWidth"`
	Quality     string  `json:"quality"` // high | balanced | compact
}

// ConvertResult reports a finished conversion.
type ConvertResult struct {
	OutputPath     string `json:"outputPath"`
	SizeBytes      int64  `json:"sizeBytes"`
	Width          int    `json:"width"`
	Height         int    `json:"height"`
	Frames         int    `json:"frames"`
	DurationMs     int64  `json:"durationMs"`
	PreviewDataURL string `json:"previewDataURL"`
}

// previewLimit caps how big a GIF may be inlined as a data URL for preview.
const previewLimit = 8 << 20 // 8 MiB

// PreviewRequest asks for a quick low-resolution preview of a clip range.
type PreviewRequest struct {
	Path        string  `json:"path"`
	StartSec    float64 `json:"startSec"`
	DurationSec float64 `json:"durationSec"`
	FPS         int     `json:"fps"`
	MaxWidth    int     `json:"maxWidth"`
	Quality     string  `json:"quality"` // high | balanced | compact
}

// PreviewResult is a rendered preview of the trimmed range.
type PreviewResult struct {
	DataURL     string  `json:"dataURL"`
	Width       int     `json:"width"`
	Height      int     `json:"height"`
	Frames      int     `json:"frames"`
	DurationSec float64 `json:"durationSec"` // rendered length, capped for speed
	SizeBytes   int64   `json:"sizeBytes"`
	DurationMs  int64   `json:"durationMs"`
}

// Preview caps: a preview renders fast and stays small enough to inline.
const (
	previewMaxSec   = 3.0
	previewMaxWidth = 320
	previewMaxFPS   = 15
)
