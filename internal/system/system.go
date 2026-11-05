// Package system binds the app's host metadata service and keeps the native
// window chrome in sync with the page theme. It is bound as "System" (see
// cmd/m-tools/main.go), so its methods stay the page's System.* API.
package system

import (
	"context"
	"runtime"

	"github.com/egoist/mygo"
	mtools "github.com/sdttttt/m-tools"
)

// SystemInfo describes the machine the app runs on.
type SystemInfo struct {
	OS        string `json:"os"`
	Arch      string `json:"arch"`
	GoVersion string `json:"goVersion"`
	AppVer    string `json:"appVer"`
}

// System is callable from the frontend: `mygo generate` turns its methods
// into typed TypeScript functions in src/mygo.ts.
type Service struct{}

// Info returns information about the host system and this app build.
func (Service) Info() SystemInfo {
	return SystemInfo{
		OS:        runtime.GOOS,
		Arch:      runtime.GOARCH,
		GoVersion: runtime.Version(),
		AppVer:    mtools.Version,
	}
}

// SetWindowTheme syncs the native window chrome with the page theme mode
// (light | dark | system): the background fill shown until the page paints,
// and the title-bar buttons theme on macOS.
func (Service) SetWindowTheme(ctx context.Context, mode string) error {
	win := mygo.CallerWindow(ctx)
	if win == nil {
		return nil
	}
	switch mode {
	case "light":
		_ = win.SetBackgroundColor(windowColorLight)
		win.SetWindowButtonsTheme(mygo.ThemeLight)
	case "dark":
		_ = win.SetBackgroundColor(windowColorDark)
		win.SetWindowButtonsTheme(mygo.ThemeDark)
	default:
		_ = win.SetBackgroundColor(WindowBackground())
		win.SetWindowButtonsTheme(mygo.ThemeSystem)
	}
	return nil
}
