package system

// windowColors mirror the page's --background tokens in webview/src/index.css.
const (
	windowColorLight = "#ffffff"
	windowColorDark  = "#09090b"
)

// WindowBackground returns the background fill for a window created before
// the page has reported its theme: the theme module applies the same pair
// once it knows the user's mode.
func WindowBackground() string {
	return "light-dark(" + windowColorLight + ", " + windowColorDark + ")"
}
