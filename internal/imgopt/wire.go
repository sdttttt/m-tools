package imgopt

// OptimizeRequest asks for one lossless image optimization. It has no
// options: the tool always keeps pixels identical and the output format
// identical to the source.
type OptimizeRequest struct {
	Path string `json:"path"`
}

// OptimizeResult reports one optimization attempt. OutputPath is the -min
// file written next to the source (empty when NotNeeded or Skipped — no
// file was written in either case). PreviewDataURL inlines the optimized
// image as a data URL for small files.
type OptimizeResult struct {
	OutputPath     string `json:"outputPath"`
	Format         string `json:"format"`
	OriginalBytes  int64  `json:"originalBytes"`
	OptimizedBytes int64  `json:"optimizedBytes"`
	DurationMs     int64  `json:"durationMs"`
	PreviewDataURL string `json:"previewDataURL"`
	NotNeeded      bool   `json:"notNeeded"` // nothing to shave; output would be >= source
	SkipReason     string `json:"skipReason"`
}
