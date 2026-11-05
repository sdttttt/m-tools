package imgfmt

// Info describes one image file as the services report it to the page.
// HasAlpha flags sources that would need flattening when the target is
// JPEG; Format is the canonical lowercase extension-style id (png, jpg,
// gif, webp, bmp, tiff). Supported=false rows carry a SkipReason and the
// page shows them as skipped.
type Info struct {
	Path       string `json:"path"`
	Format     string `json:"format"`
	Width      int    `json:"width"`
	Height     int    `json:"height"`
	SizeBytes  int64  `json:"sizeBytes"`
	HasAlpha   bool   `json:"hasAlpha"`
	Supported  bool   `json:"supported"` // decodable by this build
	SkipReason string `json:"skipReason"`
}

// QualityLevels maps the page's quality choice to JPEG/WebP quality 0-100.
var QualityLevels = map[string]int{
	"high":     92,
	"balanced": 82,
	"compact":  62,
}

// PreviewMime maps format ids to the mime subtype used in data URLs
// (jpg's subtype is jpeg).
func PreviewMime(name string) string {
	if name == "jpg" {
		return "jpeg"
	}
	return name
}
