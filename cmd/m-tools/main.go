// m-tools: a personal toolbox desktop app for macOS and Windows,
// built with MyGo (webview) + SolidJS.
package main

import (
	"log"

	"github.com/egoist/mygo"
	mtools "github.com/sdttttt/m-tools"
	"github.com/sdttttt/m-tools/internal/mp4gif"
	"github.com/sdttttt/m-tools/internal/system"
)

// The mtools import runs the root package's init hooks (bundled ffmpeg on a
// with_ffmpeg build) and provides shared app metadata.

func main() {
	mygo.BindAs("System", system.Service{})
	mygo.BindAs("Mp4Gif", &mp4gif.Service{})

	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:     mtools.AppName,
			Width:     960,
			Height:    640,
			MinWidth:  480,
			MinHeight: 360,
			// The page's --background tokens (webview/src/index.css); the
			// same pair SetWindowTheme applies when the user picks a mode.
			BackgroundColor: system.WindowBackground(),
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
