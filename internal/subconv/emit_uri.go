package subconv

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"
)

// EmitV2Ray renders nodes as a v2rayN-style subscription: one share link
// per line, the whole text base64-wrapped. Nodes whose transport cannot be
// expressed as a share link land in skipped.
func EmitV2Ray(nodes []*Node) (string, []Skipped) {
	var lines []string
	var skipped []Skipped
	for _, n := range nodes {
		line, err := nodeToURI(n)
		if err != nil {
			skipped = append(skipped, Skipped{Name: n.DisplayName(), Reason: err.Error()})
			continue
		}
		lines = append(lines, line)
	}
	blob := base64.StdEncoding.EncodeToString([]byte(strings.Join(lines, "\n")))
	return blob, skipped
}

// nodeToURI renders one node back into its share link form.
func nodeToURI(n *Node) (string, error) {
	switch n.Protocol {
	case ProtoVLESS:
		return vlessURI(n), nil
	case ProtoVMess:
		return vmessURI(n), nil
	case ProtoTrojan:
		return trojanURI(n), nil
	case ProtoSS:
		return ssURI(n), nil
	case ProtoHysteria2:
		return hy2URI(n), nil
	case ProtoTUIC:
		return tuicURI(n), nil
	case ProtoAnyTLS:
		return anytlsURI(n), nil
	}
	return "", fmt.Errorf("未知协议 %q", n.Protocol)
}

// vlessURI renders vless://uuid@host:port?security=...#name.
func vlessURI(n *Node) string {
	u := &url.URL{Scheme: "vless", User: url.User(n.UUID), Host: hostport(n)}
	v := u.Query()
	// vless links omit security=none; only tls/reality are meaningful.
	if n.Security != "" && n.Security != "none" {
		v.Set("security", n.Security)
	}
	setIf(v, "sni", n.SNI)
	setIf(v, "alpn", strings.Join(n.ALPN, ","))
	setIf(v, "fp", n.Fingerprint)
	setIf(v, "flow", n.Flow)
	setIf(v, "pbk", n.PublicKey)
	setIf(v, "sid", n.ShortID)
	setIf(v, "spx", n.SpiderX)
	setIf(v, "type", n.Network)
	setIf(v, "path", n.Path)
	setIf(v, "host", n.Host)
	setIf(v, "serviceName", n.ServiceName)
	setInsecure(v, n.Insecure)
	u.RawQuery = v.Encode()
	u.Fragment = n.Name
	return u.String()
}

// vmessURI renders vmess://<base64 JSON>.
func vmessURI(n *Node) string {
	network := n.Network
	if network == "" || network == "httpupgrade" {
		network = "tcp"
	}
	tls := ""
	if n.Security == "tls" {
		tls = "tls"
	}
	vj := map[string]any{
		"v":    2,
		"ps":   n.Name,
		"add":  n.Server,
		"port": strconv.Itoa(n.Port),
		"id":   n.UUID,
		"aid":  strconv.Itoa(n.AlterID),
		"net":  network,
		"type": "none",
		"host": vmessHost(n),
		"path": n.Path,
		"tls":  tls,
		"sni":  n.SNI,
		"alpn": strings.Join(n.ALPN, ","),
		"fp":   n.Fingerprint,
		"scy":  n.Cipher,
	}
	raw, _ := json.Marshal(vj)
	return "vmess://" + base64.RawURLEncoding.EncodeToString(raw)
}

// vmessHost folds the host header / grpc authority into the single host
// field of the vmess JSON form.
func vmessHost(n *Node) string {
	if n.Network == "grpc" {
		return n.ServiceName
	}
	return n.Host
}

// trojanURI renders trojan://password@host:port?...#name. The password is
// percent-encoded into the userinfo when it carries reserved characters.
func trojanURI(n *Node) string {
	u := &url.URL{Scheme: "trojan", User: url.UserPassword("", n.Password), Host: hostport(n)}
	// url.UserPassword with empty username renders ":pass@"; trojan wants
	// just the password in userinfo, so rebuild it.
	u.User = url.User(n.Password)
	v := u.Query()
	setIf(v, "sni", n.SNI)
	setIf(v, "alpn", strings.Join(n.ALPN, ","))
	setIf(v, "fp", n.Fingerprint)
	setIf(v, "type", n.Network)
	setIf(v, "path", n.Path)
	setIf(v, "host", n.Host)
	setIf(v, "serviceName", n.ServiceName)
	setInsecure(v, n.Insecure)
	u.RawQuery = v.Encode()
	u.Fragment = n.Name
	return u.String()
}

// ssURI renders the SIP002 form ss://base64(method:password)@host:port with
// an optional plugin query and name fragment.
func ssURI(n *Node) string {
	userinfo := base64.RawURLEncoding.EncodeToString([]byte(n.Cipher + ":" + n.Password))
	u := &url.URL{Scheme: "ss", User: url.User(userinfo), Host: hostport(n)}
	if n.Plugin != "" {
		plugin := n.Plugin
		if n.PluginOpts != "" {
			plugin += ";" + n.PluginOpts
		}
		v := u.Query()
		v.Set("plugin", plugin)
		u.RawQuery = v.Encode()
	}
	u.Fragment = n.Name
	return u.String()
}

// hy2URI renders hy2://auth@host:port?...#name.
func hy2URI(n *Node) string {
	u := &url.URL{Scheme: "hy2", User: url.User(n.Password), Host: hostport(n)}
	v := u.Query()
	setIf(v, "sni", n.SNI)
	setIf(v, "alpn", strings.Join(n.ALPN, ","))
	setIf(v, "fp", n.Fingerprint)
	setIf(v, "obfs", n.Obfs)
	setIf(v, "obfs-password", n.ObfsPassword)
	setInsecure(v, n.Insecure)
	u.RawQuery = v.Encode()
	u.Fragment = n.Name
	return u.String()
}

// tuicURI renders tuic://uuid:password@host:port?...#name.
func tuicURI(n *Node) string {
	u := &url.URL{Scheme: "tuic", User: url.UserPassword(n.UUID, n.Password), Host: hostport(n)}
	v := u.Query()
	setIf(v, "congestion_control", n.CongestionControl)
	setIf(v, "udp_relay_mode", n.UDPRelayMode)
	setIf(v, "alpn", strings.Join(n.ALPN, ","))
	setIf(v, "sni", n.SNI)
	setIf(v, "fp", n.Fingerprint)
	setInsecure(v, n.Insecure)
	if n.ZeroRTT {
		v.Set("zero_rtt_handshake", "1")
	}
	if n.DisableSNI {
		v.Set("disable_sni", "1")
	}
	u.RawQuery = v.Encode()
	u.Fragment = n.Name
	return u.String()
}

// anytlsURI renders anytls://password@host:port/?...#name.
func anytlsURI(n *Node) string {
	u := &url.URL{Scheme: "anytls", User: url.User(n.Password), Host: hostport(n)}
	v := u.Query()
	setIf(v, "sni", n.SNI)
	setIf(v, "alpn", strings.Join(n.ALPN, ","))
	setIf(v, "fp", n.Fingerprint)
	setInsecure(v, n.Insecure)
	u.RawQuery = v.Encode()
	u.Fragment = n.Name
	return u.String()
}

// hostport joins server and port, bracketing IPv6 literals.
func hostport(n *Node) string {
	if strings.Contains(n.Server, ":") && !strings.HasPrefix(n.Server, "[") {
		return "[" + n.Server + "]:" + strconv.Itoa(n.Port)
	}
	return n.Server + ":" + strconv.Itoa(n.Port)
}

// setIf sets a query key only when the value is non-empty.
func setIf(v url.Values, key, val string) {
	if val != "" {
		v.Set(key, val)
	}
}

// setInsecure emits insecure=1, the spelling every client accepts.
func setInsecure(v url.Values, insecure bool) {
	if insecure {
		v.Set("insecure", "1")
	}
}
