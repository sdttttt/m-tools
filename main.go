// m-tools: a personal toolbox desktop app for macOS and Windows,
// built with MyGo (webview) + SolidJS.
package main

import (
	"log"
	"runtime"

	"github.com/egoist/mygo"
)

// Version is the app version, kept in sync with mygo.config.ts.
const Version = "0.1.0"

// SystemInfo describes the machine the app runs on.
type SystemInfo struct {
	OS        string `json:"os"`
	Arch      string `json:"arch"`
	GoVersion string `json:"goVersion"`
	AppVer    string `json:"appVer"`
}

// System is callable from the frontend: `mygo generate` turns its methods
// into typed TypeScript functions in src/mygo.ts.
type System struct{}

// Info returns information about the host system and this app build.
func (System) Info() SystemInfo {
	return SystemInfo{
		OS:        runtime.GOOS,
		Arch:      runtime.GOARCH,
		GoVersion: runtime.Version(),
		AppVer:    Version,
	}
}

func main() {
	mygo.Bind(System{})

	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:           "M-Tools",
			Width:           960,
			Height:          640,
			MinWidth:        480,
			MinHeight:       360,
			BackgroundColor: "light-dark(#f6f7f9, #0f1115)", // the page's --bg in src/style.css
			// Opens where the user left it last time.
			StateKey: "main",
			// The frontend: devUrl during `mygo dev`, frontendDist
			// embedded by `mygo build` otherwise (see mygo.config.ts).
			URL: "/",
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
