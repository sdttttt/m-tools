module github.com/sdttttt/m-tools

go 1.27.3

require (
	github.com/egoist/mygo v0.7.4
	github.com/sdttttt/m-tools/ffmpeg v0.0.0-00010101000000-000000000000
	github.com/sdttttt/m-tools/webp v0.0.0-00010101000000-000000000000
	golang.org/x/image v0.46.0
)

require (
	github.com/chai2010/webp v1.4.0 // indirect
	github.com/ebitengine/purego v0.11.3 // indirect
	golang.org/x/net v0.62.0 // indirect
	golang.org/x/text v0.43.0 // indirect
)

replace github.com/sdttttt/m-tools/ffmpeg => ./ffmpeg

replace github.com/sdttttt/m-tools/webp => ./webp
