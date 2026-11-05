package mp4gif

import (
	"github.com/egoist/mygo"
	"github.com/sdttttt/m-tools/ffmpeg"
)

// mp4GifProgress streams conversion progress to the page. The payload is
// the ffmpeg module's GIFProgress (json: stage/percent).
var mp4GifProgress = mygo.NewEvent[ffmpeg.GIFProgress]("mp4gifProgress")
