package ffmpeg

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
)

// Thumbnails extracts up to count JPEG stills spread evenly across
// durationSec of input, each scaled to width pixels wide. The frames are
// what a trim timeline shows as its filmstrip.
func (c *Client) Thumbnails(ctx context.Context, input string, count, width int, durationSec float64) ([][]byte, error) {
	if input == "" {
		return nil, errors.New("Thumbnails.Input 为空")
	}
	if durationSec <= 0 {
		return nil, errors.New("Thumbnails: 视频时长无效")
	}
	count = clampInt(count, 1, 30)
	width = clampInt(width, 32, 640)

	dir, err := os.MkdirTemp("", "m-tools-thumbs-*")
	if err != nil {
		return nil, fmt.Errorf("创建临时缩略图目录: %w", err)
	}
	defer os.RemoveAll(dir)

	// A frame rate of count/durationSec samples the whole clip evenly; the
	// frame cap stops rounding from adding one more at the end.
	rate := strconv.FormatFloat(float64(count)/durationSec, 'f', 6, 64)
	pattern := filepath.Join(dir, "t-%03d.jpg")
	args := []string{
		"-y", "-v", "error",
		"-i", input,
		"-vf", fmt.Sprintf("fps=%s,scale=%d:-2:flags=lanczos", rate, width),
		"-frames:v", strconv.Itoa(count),
		"-q:v", "4",
		pattern,
	}
	cmd := exec.CommandContext(ctx, c.ffmpeg, args...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		return nil, fmt.Errorf("ffmpeg: %v\n%s", err, tail(stderr.String()))
	}

	paths, err := filepath.Glob(filepath.Join(dir, "t-*.jpg"))
	if err != nil {
		return nil, fmt.Errorf("读取缩略图: %w", err)
	}
	if len(paths) == 0 {
		return nil, errors.New("生成缩略图失败：没有输出帧")
	}
	sort.Strings(paths)

	thumbs := make([][]byte, 0, len(paths))
	for _, path := range paths {
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("读取缩略图: %w", err)
		}
		thumbs = append(thumbs, data)
	}
	return thumbs, nil
}
