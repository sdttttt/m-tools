module github.com/sdttttt/m-tools

go 1.27.3

require (
	github.com/egoist/mygo v0.10.0
	github.com/sdttttt/m-tools/ffmpeg v0.0.0-00010101000000-000000000000
	github.com/sdttttt/m-tools/webp v0.0.0-00010101000000-000000000000
	golang.org/x/image v0.48.0
	gopkg.in/yaml.v3 v3.0.1
)

require (
	github.com/chai2010/webp v1.4.0 // indirect
	github.com/ebitengine/purego v0.11.3 // indirect
	golang.org/x/net v0.63.0 // indirect
	golang.org/x/text v0.44.0 // indirect
)

replace github.com/sdttttt/m-tools/ffmpeg => ./ffmpeg

replace github.com/sdttttt/m-tools/webp => ./webp
