package subconv

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/egoist/mygo"
)

// SubConv converts proxy subscriptions between the v2ray, Clash and
// sing-box formats. Everything runs locally: Fetch downloads a subscription
// body over plain HTTP and Convert parses + re-emits it in memory.
type Service struct {
	mu     sync.Mutex // guards cancel
	cancel context.CancelFunc
}

// fetchTimeout bounds the subscription download.
const fetchTimeout = 15 * time.Second

// fetchMaxBytes caps the downloaded body (subscriptions are text, a few MB
// at most; anything larger is suspicious).
const fetchMaxBytes = 16 << 20

// begin registers cancel as the running job. Only one job runs at a time.
func (s *Service) begin(cancel context.CancelFunc) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.cancel != nil {
		return errors.New("已有任务在运行，请先取消")
	}
	s.cancel = cancel
	return nil
}

// end clears the running job.
func (s *Service) end() {
	s.mu.Lock()
	s.cancel = nil
	s.mu.Unlock()
}

// CancelActive stops the running fetch or conversion, if any.
func (s *Service) CancelActive() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.cancel != nil {
		s.cancel()
	}
}

// Fetch downloads a subscription body. Only http(s) URLs are accepted; the
// download never leaves the user's machine through any third party.
func (s *Service) Fetch(rawURL string) (FetchResult, error) {
	u, err := url.Parse(strings.TrimSpace(rawURL))
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return FetchResult{}, fmt.Errorf("订阅地址必须是 http(s) URL")
	}

	ctx, cancel := context.WithTimeout(context.Background(), fetchTimeout)
	defer cancel()
	if err := s.begin(cancel); err != nil {
		return FetchResult{}, err
	}
	defer s.end()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return FetchResult{}, err
	}
	req.Header.Set("User-Agent", "m-tools/1.0")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return FetchResult{}, fmt.Errorf("拉取订阅失败：%w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return FetchResult{}, fmt.Errorf("订阅服务器返回 %s", resp.Status)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, fetchMaxBytes+1))
	if err != nil {
		return FetchResult{}, fmt.Errorf("读取订阅内容失败：%w", err)
	}
	if int64(len(body)) > fetchMaxBytes {
		return FetchResult{}, fmt.Errorf("订阅内容超过 %d MB，已中止", fetchMaxBytes>>20)
	}
	return FetchResult{Body: string(body), ContentType: resp.Header.Get("Content-Type")}, nil
}

// Convert parses the input text in the `from` format and re-emits it in
// the `to` format. Per-node problems are reported in the result instead of
// failing the whole conversion.
func (s *Service) Convert(req ConvertRequest) (ConvertResult, error) {
	from := Format(strings.TrimSpace(strings.ToLower(req.From)))
	to := Format(strings.TrimSpace(strings.ToLower(req.To)))
	if from == "" {
		from = FormatV2Ray
	}
	if req.Text == "" {
		return ConvertResult{}, fmt.Errorf("订阅内容为空")
	}
	if to != FormatV2Ray && to != FormatClash && to != FormatSingBox {
		return ConvertResult{}, fmt.Errorf("不支持的目标格式 %q", req.To)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if err := s.begin(cancel); err != nil {
		return ConvertResult{}, err
	}
	defer s.end()

	var nodes []*Node
	var skipped []Skipped
	var err error
	if from == "auto" {
		from, nodes, skipped, err = ParseAuto([]byte(req.Text))
	} else {
		nodes, skipped, err = Parse([]byte(req.Text), from)
	}
	if err != nil {
		return ConvertResult{}, err
	}

	select {
	case <-ctx.Done():
		return ConvertResult{}, ctx.Err()
	default:
	}

	output, emitSkipped, err := Emit(nodes, to)
	if err != nil {
		return ConvertResult{}, err
	}
	parseSkipped := len(skipped)
	skipped = append(skipped, emitSkipped...)

	// Names reflect exactly what made it into the output: re-parse the
	// emitted text rather than guessing which nodes the emitters kept.
	emitted, _, _ := Parse([]byte(output), to)
	names := make([]string, len(emitted))
	for i, n := range emitted {
		names[i] = n.DisplayName()
	}

	return ConvertResult{
		Output:     output,
		From:       string(from),
		To:         string(to),
		Total:      len(nodes) + parseSkipped + len(emitSkipped),
		Converted:  len(nodes) - len(emitSkipped),
		ByProtocol: countByProtocol(nodes),
		Skipped:    skipped,
		Names:      names,
	}, nil
}

// countByProtocol tallies parsed nodes per protocol.
func countByProtocol(nodes []*Node) map[string]int {
	out := map[string]int{}
	for _, n := range nodes {
		out[n.Protocol]++
	}
	return out
}

// SaveAs shows a native save dialog and writes the converted text there.
// Returns the destination path, or "" when canceled.
func (s *Service) SaveAs(content, defaultName string) (string, error) {
	dest, err := mygo.Dialog.Save(mygo.SaveDialogOptions{
		Title:       "保存订阅",
		DefaultPath: defaultName,
		Filters: []mygo.FileFilter{
			{Name: "订阅文件", Extensions: []string{"txt", "yaml", "yml", "json"}},
			{Name: "所有文件", Extensions: []string{"*"}},
		},
	})
	if err != nil {
		return "", err
	}
	if dest == "" {
		return "", nil
	}
	if err := os.WriteFile(dest, []byte(content), 0o644); err != nil {
		return "", err
	}
	return dest, nil
}
