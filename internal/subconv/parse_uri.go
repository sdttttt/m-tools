package subconv

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
)

// ParseURI parses a single share link into a Node. Supported schemes:
// vless, vmess, trojan, ss (SIP002), hysteria2/hy2, tuic, anytls.
func ParseURI(raw string) (*Node, error) {
	switch {
	case strings.HasPrefix(raw, "vless://"):
		return parseVLESS(strings.TrimPrefix(raw, "vless://"))
	case strings.HasPrefix(raw, "vmess://"):
		return parseVMess(strings.TrimPrefix(raw, "vmess://"))
	case strings.HasPrefix(raw, "trojan://"):
		return parseTrojan(strings.TrimPrefix(raw, "trojan://"))
	case strings.HasPrefix(raw, "ss://"):
		return parseSS(strings.TrimPrefix(raw, "ss://"))
	case strings.HasPrefix(raw, "hysteria2://"):
		return parseHY2(strings.TrimPrefix(raw, "hysteria2://"))
	case strings.HasPrefix(raw, "hy2://"):
		return parseHY2(strings.TrimPrefix(raw, "hy2://"))
	case strings.HasPrefix(raw, "tuic://"):
		return parseTUIC(strings.TrimPrefix(raw, "tuic://"))
	case strings.HasPrefix(raw, "anytls://"):
		return parseAnyTLS(strings.TrimPrefix(raw, "anytls://"))
	}
	return nil, fmt.Errorf("无法识别的链接类型")
}

// parseVLESS handles vless://uuid@host:port?security=reality&pbk=...#name.
func parseVLESS(rest string) (*Node, error) {
	u, err := url.Parse("vless://" + rest)
	if err != nil {
		return nil, err
	}
	n := &Node{
		Protocol:    ProtoVLESS,
		Name:        nameOf(u),
		Server:      u.Hostname(),
		Port:        portOf(u),
		UUID:        u.User.Username(),
		Security:    strings.ToLower(q(u, "security")),
		SNI:         firstNonEmpty(q(u, "sni"), q(u, "peer")),
		Fingerprint: q(u, "fp"),
		Flow:        q(u, "flow"),
		Insecure:    insecureOf(u),
	}
	n.ALPN = splitList(q(u, "alpn"))
	n.PublicKey = q(u, "pbk")
	n.ShortID = q(u, "sid")
	n.SpiderX = q(u, "spx")
	n.Network = strings.ToLower(q(u, "type"))
	n.Path = q(u, "path")
	n.Host = firstNonEmpty(q(u, "host"), q(u, "authority"))
	n.ServiceName = q(u, "serviceName")
	if n.Security == "" {
		n.Security = "none"
	}
	return n, nil
}

// vmessJSON mirrors the vmess:// base64 JSON payload used by v2rayN.
type vmessJSON struct {
	V    any    `json:"v"`
	Ps   string `json:"ps"`
	Add  string `json:"add"`
	Port any    `json:"port"`
	ID   string `json:"id"`
	Aid  any    `json:"aid"`
	Net  string `json:"net"`
	Type string `json:"type"`
	Host string `json:"host"`
	Path string `json:"path"`
	TLS  string `json:"tls"`
	SNI  string `json:"sni"`
	ALPN string `json:"alpn"`
	FP   string `json:"fp"`
	Scy  string `json:"scy"`
}

// parseVMess handles vmess://<base64 JSON>.
func parseVMess(rest string) (*Node, error) {
	payload := strings.TrimSpace(rest)
	payload = strings.SplitN(payload, "#", 2)[0] // tolerate a stray fragment
	raw, err := decodeBase64(payload)
	if err != nil {
		return nil, fmt.Errorf("vmess 负载不是有效的 base64：%w", err)
	}
	var vj vmessJSON
	if err := json.Unmarshal(raw, &vj); err != nil {
		return nil, fmt.Errorf("vmess 负载不是有效的 JSON：%w", err)
	}
	n := &Node{
		Protocol:    ProtoVMess,
		Name:        vj.Ps,
		Server:      vj.Add,
		Port:        parsePort(vj.Port),
		UUID:        vj.ID,
		AlterID:     parsePort(vj.Aid),
		Network:     strings.ToLower(vj.Net),
		Security:    strings.ToLower(vj.TLS),
		SNI:         firstNonEmpty(vj.SNI, vj.Host),
		Fingerprint: vj.FP,
		Cipher:      vj.Scy,
		Path:        vj.Path,
	}
	if n.Cipher == "" {
		n.Cipher = "auto"
	}
	if n.Network == "" {
		n.Network = "tcp"
	}
	if n.Security == "" {
		n.Security = "none"
	}
	n.ALPN = splitList(vj.ALPN)
	// ws/h2 host header; for grpc the host field doubles as authority.
	if n.Network == "ws" || n.Network == "h2" || n.Network == "http" {
		n.Host = vj.Host
	} else if n.Network == "grpc" {
		n.ServiceName = firstNonEmpty(vj.Path, vj.Host)
	}
	return n, nil
}

// parseTrojan handles trojan://password@host:port?sni=...#name.
func parseTrojan(rest string) (*Node, error) {
	u, err := url.Parse("trojan://" + rest)
	if err != nil {
		return nil, err
	}
	n := &Node{
		Protocol:    ProtoTrojan,
		Name:        nameOf(u),
		Server:      u.Hostname(),
		Port:        portOf(u),
		Password:    u.User.Username(),
		Security:    "tls",
		SNI:         firstNonEmpty(q(u, "sni"), q(u, "peer")),
		Fingerprint: q(u, "fp"),
		Insecure:    insecureOf(u),
	}
	n.ALPN = splitList(q(u, "alpn"))
	n.Network = strings.ToLower(q(u, "type"))
	n.Path = q(u, "path")
	n.Host = firstNonEmpty(q(u, "host"), q(u, "authority"))
	n.ServiceName = q(u, "serviceName")
	if n.Network == "" {
		n.Network = "tcp"
	}
	return n, nil
}

// parseSS handles ss:// in the SIP002 form
// ss://base64(method:password)@host:port/?plugin=...#name plus the legacy
// ss://base64(method:password@host:port)#name form.
func parseSS(rest string) (*Node, error) {
	// Split off the name fragment first so url.Parse never chokes on
	// unencoded '#' in names (names come percent-encoded from sane
	// generators, but tolerate both).
	name := ""
	if i := strings.LastIndex(rest, "#"); i >= 0 {
		name, _ = url.QueryUnescape(rest[i+1:])
		rest = rest[:i]
	}
	u, err := url.Parse("ss://" + rest)
	if err != nil {
		return nil, err
	}

	n := &Node{Protocol: ProtoSS, Name: name}
	if u.User != nil && u.Host != "" {
		// SIP002: userinfo carries base64(method:password).
		decoded, err := decodeBase64(u.User.Username())
		if err != nil {
			return nil, fmt.Errorf("ss 用户信息不是有效的 base64(method:password)：%w", err)
		}
		method, password, ok := strings.Cut(string(decoded), ":")
		if !ok {
			return nil, fmt.Errorf("ss 用户信息缺少 method:password 结构")
		}
		n.Cipher, n.Password = method, password
		n.Server, n.Port = u.Hostname(), portOf(u)
		n.Plugin, n.PluginOpts = pluginOf(q(u, "plugin"))
		return n, nil
	}

	// Legacy: the whole authority after ss:// is base64(method:password@host:port).
	if rest == "" {
		return nil, fmt.Errorf("ss 链接为空")
	}
	authority := rest
	if i := strings.Index(authority, "/"); i >= 0 {
		authority = authority[:i]
	}
	decoded, err := decodeBase64(authority)
	if err != nil {
		return nil, fmt.Errorf("ss 链接不是有效的 base64：%w", err)
	}
	inner := string(decoded)
	plugin := ""
	if i := strings.Index(inner, "/?"); i >= 0 {
		plugin = inner[i+2:]
		inner = inner[:i]
	}
	userinfo, hostport, ok := strings.Cut(inner, "@")
	if !ok {
		return nil, fmt.Errorf("ss 链接缺少 @ 分隔")
	}
	method, password, ok := strings.Cut(userinfo, ":")
	if !ok {
		return nil, fmt.Errorf("ss 链接缺少 method:password")
	}
	n.Cipher, n.Password = method, password
	if h, p, ok := strings.Cut(hostport, ":"); ok {
		n.Server = h
		fmt.Sscanf(p, "%d", &n.Port)
	}
	if plugin != "" {
		if strings.HasPrefix(plugin, "plugin=") {
			plugin = strings.TrimPrefix(plugin, "plugin=")
		}
		n.Plugin, n.PluginOpts = pluginOf(plugin)
	}
	return n, nil
}

// parseHY2 handles hysteria2:// and hy2://auth@host:port?sni=&obfs=salamander...#name.
func parseHY2(rest string) (*Node, error) {
	u, err := url.Parse("hy2://" + rest)
	if err != nil {
		return nil, err
	}
	n := &Node{
		Protocol: ProtoHysteria2,
		Name:     nameOf(u),
		Server:   u.Hostname(),
		Port:     portOf(u),
		Password: u.User.Username(),
		SNI:      firstNonEmpty(q(u, "sni"), q(u, "peer")),
		Insecure: insecureOf(u),
	}
	n.ALPN = splitList(q(u, "alpn"))
	n.Fingerprint = q(u, "fp")
	n.Obfs = q(u, "obfs")
	n.ObfsPassword = firstNonEmpty(q(u, "obfs-password"), q(u, "obfs_password"))
	return n, nil
}

// parseTUIC handles tuic://uuid:password@host:port?congestion_control=...#name.
func parseTUIC(rest string) (*Node, error) {
	u, err := url.Parse("tuic://" + rest)
	if err != nil {
		return nil, err
	}
	n := &Node{
		Protocol:   ProtoTUIC,
		Name:       nameOf(u),
		Server:     u.Hostname(),
		Port:       portOf(u),
		SNI:        firstNonEmpty(q(u, "sni"), q(u, "peer")),
		Insecure:   insecureOf(u),
		DisableSNI: boolQuery(u, "disable_sni"),
	}
	n.UUID = u.User.Username()
	if u.User != nil {
		n.Password, _ = u.User.Password()
	}
	n.ALPN = splitList(q(u, "alpn"))
	n.Fingerprint = q(u, "fp")
	n.CongestionControl = firstNonEmpty(q(u, "congestion_control"), q(u, "congestion-controller"))
	n.UDPRelayMode = firstNonEmpty(q(u, "udp_relay_mode"), q(u, "udp-relay-mode"))
	n.ZeroRTT = boolQuery(u, "zero_rtt_handshake")
	return n, nil
}

// parseAnyTLS handles anytls://password@host[:port]/?sni=&insecure=1#name.
func parseAnyTLS(rest string) (*Node, error) {
	u, err := url.Parse("anytls://" + rest)
	if err != nil {
		return nil, err
	}
	n := &Node{
		Protocol: ProtoAnyTLS,
		Name:     nameOf(u),
		Server:   u.Hostname(),
		Port:     portOf(u),
		Password: u.User.Username(),
		SNI:      q(u, "sni"),
		Insecure: insecureOf(u),
	}
	n.ALPN = splitList(q(u, "alpn"))
	n.Fingerprint = q(u, "fp")
	if n.Port == 0 {
		n.Port = 443 // per the anytls URI spec, port defaults to 443
	}
	return n, nil
}

// q reads one query value, tolerating case variations that some
// generators produce.
func q(u *url.URL, key string) string {
	if v := u.Query().Get(key); v != "" {
		return v
	}
	return u.Query().Get(strings.ReplaceAll(key, "_", "-"))
}

// nameOf decodes the URI fragment as the display name.
func nameOf(u *url.URL) string {
	if u.Fragment == "" {
		return ""
	}
	if name, err := url.QueryUnescape(u.Fragment); err == nil {
		return name
	}
	return u.Fragment
}

// portOf reads the port, defaulting to 0 when absent so Valid() can reject.
func portOf(u *url.URL) int {
	var port int
	fmt.Sscanf(u.Port(), "%d", &port)
	return port
}

// insecureOf accepts every common spelling of allow_insecure.
func insecureOf(u *url.URL) bool {
	return boolQuery(u, "insecure") ||
		boolQuery(u, "allow_insecure") ||
		boolQuery(u, "allowInsecure") ||
		boolQuery(u, "skip_cert_verify")
}

// boolQuery parses a query flag present with no value or with 1/true.
func boolQuery(u *url.URL, key string) bool {
	values, ok := u.Query()[key]
	if !ok {
		return false
	}
	if len(values) == 0 || values[0] == "" {
		return true
	}
	b, err := strconvParseBool(values[0])
	return err == nil && b
}

// pluginOf splits an ss plugin value "obfs-local;obfs=http;obfs-host=..."
// into plugin name and a comma/semicolon joined options blob.
func pluginOf(v string) (string, string) {
	if v == "" {
		return "", ""
	}
	parts := strings.Split(v, ";")
	return parts[0], strings.Join(parts[1:], ";")
}

// decodeBase64 tolerates standard and URL-safe alphabets, padded or not.
func decodeBase64(s string) ([]byte, error) {
	for _, enc := range []*base64.Encoding{
		base64.StdEncoding,
		base64.URLEncoding,
		base64.RawStdEncoding,
		base64.RawURLEncoding,
	} {
		if raw, err := enc.DecodeString(strings.TrimSpace(s)); err == nil {
			return raw, nil
		}
	}
	return nil, fmt.Errorf("base64 解码失败")
}

// firstNonEmpty returns the first non-empty string.
func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}
