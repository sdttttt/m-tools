package subconv

import (
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
)

// Format identifies one of the three subscription shapes.
type Format string

const (
	FormatV2Ray   Format = "v2ray"   // base64 share-link list
	FormatClash   Format = "clash"   // mihomo YAML with proxies:
	FormatSingBox Format = "singbox" // sing-box >=1.8 JSON with outbounds
)

// ParseAuto sniffs the body and parses it as the detected format.
// The detected format is returned so callers can echo it back to the page.
func ParseAuto(body []byte) (Format, []*Node, []Skipped, error) {
	format := Detect(body)
	nodes, skipped, err := Parse(body, format)
	return format, nodes, skipped, err
}

// Parse parses a body as the given format. Unknown or malformed input is
// reported per node in skipped; an error is only returned when the body
// cannot be interpreted as the format at all.
func Parse(body []byte, format Format) ([]*Node, []Skipped, error) {
	switch format {
	case FormatV2Ray:
		return parseV2Ray(body)
	case FormatClash:
		return parseClash(body)
	case FormatSingBox:
		return parseSingBox(body)
	}
	return nil, nil, fmt.Errorf("unknown input format %q (want v2ray|clash|singbox)", format)
}

// Detect picks an input format from a raw body. YAML takes priority, then
// JSON, then a known URI scheme, then a base64 blob.
func Detect(body []byte) Format {
	trimmed := strings.TrimSpace(string(body))
	if trimmed == "" {
		return FormatV2Ray
	}
	// Clash configs almost always expose a top-level proxies: key.
	if strings.HasPrefix(trimmed, "proxies:") || strings.Contains(trimmed, "\nproxies:") {
		return FormatClash
	}
	if strings.HasPrefix(trimmed, "{") {
		return FormatSingBox
	}
	for _, scheme := range []string{
		"vless://", "vmess://", "trojan://", "ss://",
		"hysteria2://", "hy2://", "tuic://", "anytls://",
	} {
		if strings.HasPrefix(trimmed, scheme) {
			return FormatV2Ray
		}
	}
	if _, err := decodeBase64Lines(trimmed); err == nil {
		return FormatV2Ray
	}
	return FormatV2Ray
}

// parseV2Ray decodes a v2rayN-style subscription: a base64 blob (possibly
// wrapped) whose decoded content is one share link per line.
func parseV2Ray(body []byte) ([]*Node, []Skipped, error) {
	text := strings.TrimSpace(string(body))
	decoded, err := decodeBase64Lines(text)
	if err == nil {
		text = decoded
	}
	var nodes []*Node
	var skipped []Skipped
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		n, err := ParseURI(line)
		if err != nil {
			skipped = append(skipped, Skipped{Name: truncate(line, 48), Reason: err.Error()})
			continue
		}
		if !n.Valid() {
			skipped = append(skipped, Skipped{Name: n.DisplayName(), Reason: "字段不完整（缺少服务器、端口或认证信息）"})
			continue
		}
		nodes = append(nodes, n)
	}
	if len(nodes) == 0 {
		return nil, nil, errors.New("未找到可识别的节点链接")
	}
	return nodes, skipped, nil
}

// decodeBase64Lines decodes a whole-blob base64 body. The blob may use
// standard or URL-safe alphabet, with or without padding, and the decoded
// text may carry \r or wrapped lines.
func decodeBase64Lines(s string) (string, error) {
	compact := strings.Map(func(r rune) rune {
		if r == '\n' || r == '\r' || r == ' ' || r == '\t' {
			return -1
		}
		return r
	}, s)
	var raw []byte
	var err error
	for _, enc := range []*base64.Encoding{
		base64.StdEncoding,
		base64.URLEncoding,
		base64.RawStdEncoding,
		base64.RawURLEncoding,
	} {
		raw, err = enc.DecodeString(compact)
		if err == nil {
			break
		}
	}
	if err != nil {
		return "", err
	}
	text := string(raw)
	if !strings.Contains(text, "://") {
		return "", errors.New("解码结果不含节点链接")
	}
	return text, nil
}

// parsePort accepts sing-box/JSON port values that arrive as numbers or
// numeric strings (VMess JSON historically uses a string).
func parsePort(v any) int {
	switch t := v.(type) {
	case float64:
		return int(t)
	case int:
		return t
	case string:
		var p int
		if _, err := fmt.Sscanf(strings.TrimSpace(t), "%d", &p); err == nil {
			return p
		}
	}
	return 0
}

// splitList splits comma/space separated values such as ALPN lists.
func splitList(s string) []string {
	fields := strings.FieldsFunc(s, func(r rune) bool {
		return r == ',' || r == ' ' || r == '\t'
	})
	var out []string
	for _, f := range fields {
		if f = strings.TrimSpace(f); f != "" {
			out = append(out, f)
		}
	}
	return out
}

func stringOf(v any) string {
	switch t := v.(type) {
	case string:
		return t
	case float64:
		// JSON numbers that happen to hold integer-ish values.
		if t == float64(int64(t)) {
			return fmt.Sprintf("%d", int64(t))
		}
		return fmt.Sprintf("%v", t)
	case bool:
		return fmt.Sprintf("%t", t)
	}
	return ""
}

func boolOf(v any) bool {
	switch t := v.(type) {
	case bool:
		return t
	case string:
		b, _ := strconvParseBool(t)
		return b
	case float64:
		return t != 0
	}
	return false
}

func strconvParseBool(s string) (bool, error) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "1", "true", "yes", "on":
		return true, nil
	case "0", "false", "no", "off":
		return false, nil
	}
	return false, fmt.Errorf("invalid boolean %q", s)
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "…"
}
