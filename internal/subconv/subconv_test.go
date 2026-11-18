package subconv

import (
	"encoding/base64"
	"sort"
	"strings"
	"testing"
)

// sampleNodes returns one node per supported protocol, covering the field
// variety each protocol can carry.
func sampleNodes() []*Node {
	return []*Node{
		{
			Name: "reality-节点", Protocol: ProtoVLESS,
			Server: "vless.example.com", Port: 443,
			UUID:    "11111111-1111-1111-1111-111111111111",
			Network: "tcp", Security: "reality",
			SNI: "www.example.com", ALPN: []string{"h2", "http/1.1"},
			Fingerprint: "chrome", Flow: "xtls-rprx-vision",
			PublicKey: "pbk123", ShortID: "abcd", SpiderX: "/",
			Insecure: false,
		},
		{
			Name: "ws 节点", Protocol: ProtoVLESS,
			Server: "vless-ws.example.com", Port: 8443,
			UUID:    "22222222-2222-2222-2222-222222222222",
			Network: "ws", Security: "tls",
			SNI: "cdn.example.com", Path: "/ws", Host: "cdn.example.com",
		},
		{
			Name: "vmess aead", Protocol: ProtoVMess,
			Server: "vmess.example.com", Port: 443,
			UUID:   "33333333-3333-3333-3333-333333333333",
			Cipher: "auto", Network: "ws", Security: "tls",
			SNI: "vmess.example.com", Path: "/v2", Host: "vmess.example.com",
		},
		{
			Name: "trojan grpc", Protocol: ProtoTrojan,
			Server: "trojan.example.com", Port: 443,
			Password: "trojan-pass",
			Network:  "grpc", Security: "tls",
			SNI: "trojan.example.com", ServiceName: "grpc-service",
		},
		{
			Name: "ss obfs", Protocol: ProtoSS,
			Server: "ss.example.com", Port: 8388,
			Cipher: "aes-128-gcm", Password: "ss-pass",
			Plugin: "obfs-local", PluginOpts: "obfs=http;obfs-host=www.example.com",
		},
		{
			Name: "hy2 salamander", Protocol: ProtoHysteria2,
			Server: "hy2.example.com", Port: 443,
			Password: "hy2-pass", SNI: "hy2.example.com",
			Obfs: "salamander", ObfsPassword: "obfs-pass",
			Insecure: true,
		},
		{
			Name: "tuic 节点", Protocol: ProtoTUIC,
			Server: "tuic.example.com", Port: 443,
			UUID:              "44444444-4444-4444-4444-444444444444",
			Password:          "tuic-pass",
			CongestionControl: "bbr", UDPRelayMode: "native",
			SNI: "tuic.example.com", ALPN: []string{"h3"},
		},
		{
			Name: "anytls 节点", Protocol: ProtoAnyTLS,
			Server: "anytls.example.com", Port: 443,
			Password: "anytls-pass", SNI: "anytls.example.com",
			Fingerprint: "firefox",
		},
	}
}

// normPluginOpts compares plugin options as unordered k=v sets: the clash
// direction round-trips them through a YAML map whose key order varies.
func normPluginOpts(s string) string {
	if s == "" {
		return ""
	}
	parts := strings.Split(s, ";")
	sort.Strings(parts)
	return strings.Join(parts, ";")
}

// assertRoundTrip parses emitted output back and checks the protocol and a
// fingerprint field survived the loop.
func assertRoundTrip(t *testing.T, to Format, output string, want []*Node) {
	t.Helper()
	nodes, skipped, err := Parse([]byte(output), to)
	if err != nil {
		t.Fatalf("re-parse %s output: %v", to, err)
	}
	if len(skipped) > 0 {
		t.Fatalf("re-parse %s skipped nodes: %+v", to, skipped)
	}
	if len(nodes) != len(want) {
		t.Fatalf("re-parse %s: got %d nodes, want %d", to, len(nodes), len(want))
	}
	for i, w := range want {
		g := nodes[i]
		if g.Protocol != w.Protocol || g.Server != w.Server || g.Port != w.Port {
			t.Errorf("node %d (%s): got %s %s:%d", i, w.Name, g.Protocol, g.Server, g.Port)
		}
		if g.Name != w.Name {
			t.Errorf("node %d name: got %q want %q", i, g.Name, w.Name)
		}
		if g.UUID != w.UUID || g.Password != w.Password || g.Cipher != w.Cipher {
			t.Errorf("node %d creds: got uuid=%q pass=%q cipher=%q", i, g.UUID, g.Password, g.Cipher)
		}
		normNet := func(s string) string {
			if s == "tcp" {
				return ""
			}
			return s
		}
		normSec := func(n *Node) string {
			// TLS-only protocols normalize empty to tls on the way back.
			if n.Security == "" {
				return n.tlsSecurity()
			}
			return n.Security
		}
		if g.SNI != w.SNI || normNet(g.Network) != normNet(w.Network) || g.Path != w.Path || g.Host != w.Host {
			t.Errorf("node %d transport: got sni=%q net=%q path=%q host=%q",
				i, g.SNI, g.Network, g.Path, g.Host)
		}
		if normSec(g) != normSec(w) || g.PublicKey != w.PublicKey || g.ShortID != w.ShortID || g.Flow != w.Flow {
			t.Errorf("node %d security: got sec=%q pbk=%q sid=%q flow=%q",
				i, g.Security, g.PublicKey, g.ShortID, g.Flow)
		}
		if g.Obfs != w.Obfs || g.ObfsPassword != w.ObfsPassword ||
			g.CongestionControl != w.CongestionControl || g.UDPRelayMode != w.UDPRelayMode {
			t.Errorf("node %d proto fields: got obfs=%q/%q cc=%q udp=%q",
				i, g.Obfs, g.ObfsPassword, g.CongestionControl, g.UDPRelayMode)
		}
		if g.ServiceName != w.ServiceName {
			t.Errorf("node %d grpc: got %q want %q", i, g.ServiceName, w.ServiceName)
		}
		if g.Plugin != w.Plugin || normPluginOpts(g.PluginOpts) != normPluginOpts(w.PluginOpts) {
			t.Errorf("node %d plugin: got %q / %q want %q / %q",
				i, g.Plugin, g.PluginOpts, w.Plugin, w.PluginOpts)
		}
	}
}

// TestURIRoundTrip emits the sample set as a v2ray subscription and parses
// it back.
func TestURIRoundTrip(t *testing.T) {
	want := sampleNodes()
	out, skipped := EmitV2Ray(want)
	if len(skipped) > 0 {
		t.Fatalf("emit v2ray skipped: %+v", skipped)
	}
	assertRoundTrip(t, FormatV2Ray, out, want)
}

// TestClashRoundTrip emits the sample set as Clash YAML and parses it back.
func TestClashRoundTrip(t *testing.T) {
	want := sampleNodes()
	out, skipped := EmitClash(want)
	if len(skipped) > 0 {
		t.Fatalf("emit clash skipped: %+v", skipped)
	}
	t.Log("\n" + out)
	assertRoundTrip(t, FormatClash, out, want)
}

// TestSingBoxRoundTrip emits the sample set as sing-box JSON and parses it
// back.
func TestSingBoxRoundTrip(t *testing.T) {
	want := sampleNodes()
	out, skipped := EmitSingBox(want)
	if len(skipped) > 0 {
		t.Fatalf("emit singbox skipped: %+v", skipped)
	}
	t.Log("\n" + out)
	assertRoundTrip(t, FormatSingBox, out, want)
}

// TestSingBoxSkipsUnsupportedSSPlugin reports ss plugin nodes sing-box has
// no SIP003 implementation for; obfs-local and v2ray-plugin pass through.
func TestSingBoxSkipsUnsupportedSSPlugin(t *testing.T) {
	nodes := []*Node{{
		Name: "ss gost", Protocol: ProtoSS,
		Server: "ss.example.com", Port: 8388,
		Cipher: "aes-128-gcm", Password: "p",
		Plugin: "gost-plugin", PluginOpts: "mode=websocket",
	}}
	out, skipped := EmitSingBox(nodes)
	if !strings.Contains(out, "outbounds") || len(skipped) != 1 {
		t.Fatalf("want one skip, got %+v", skipped)
	}
	if !strings.Contains(skipped[0].Reason, "gost-plugin") {
		t.Fatalf("skip reason should name the plugin, got %q", skipped[0].Reason)
	}

	// obfs-local rides along as plugin/plugin_opts.
	nodes[0].Plugin, nodes[0].PluginOpts = "obfs-local", "obfs=http;obfs-host=h.com"
	out, skipped = EmitSingBox(nodes)
	if len(skipped) != 0 {
		t.Fatalf("obfs-local should pass through, got %+v", skipped)
	}
	if !strings.Contains(out, `"plugin": "obfs-local"`) || !strings.Contains(out, `"plugin_opts": "obfs=http;obfs-host=h.com"`) {
		t.Fatalf("plugin fields missing:\n%s", out)
	}
}

// TestClashSkipsUnsupportedTransport reports kcp/quic transports as skipped
// for clash output.
func TestClashSkipsUnsupportedTransport(t *testing.T) {
	nodes := []*Node{{
		Name: "kcp", Protocol: ProtoVMess,
		Server: "kcp.example.com", Port: 443,
		UUID:    "33333333-3333-3333-3333-333333333333",
		Network: "kcp",
	}}
	_, skipped := EmitClash(nodes)
	if len(skipped) != 1 || !strings.Contains(skipped[0].Reason, "kcp") {
		t.Fatalf("want kcp skip, got %+v", skipped)
	}
}

// TestDetect covers the four sniff branches.
func TestDetect(t *testing.T) {
	b64 := base64.StdEncoding.EncodeToString([]byte("vless://x@y:443"))
	cases := []struct {
		name string
		body string
		want Format
	}{
		{"clash yaml", "port: 7890\nproxies:\n  - name: a\n", FormatClash},
		{"proxies first", "proxies:\n  - name: a\n", FormatClash},
		{"singbox json", `{"outbounds":[{"type":"direct"}]}`, FormatSingBox},
		{"vless scheme", "vless://a@b:443#n\n", FormatV2Ray},
		{"hy2 scheme", "hy2://p@b:443#n\n", FormatV2Ray},
		{"base64 blob", b64, FormatV2Ray},
		{"plain uri list", "trojan://p@b:443#n", FormatV2Ray},
	}
	for _, c := range cases {
		if got := Detect([]byte(c.body)); got != c.want {
			t.Errorf("%s: got %q want %q", c.name, got, c.want)
		}
	}
}

// TestParseV2RayBlob decodes a base64-wrapped subscription.
func TestParseV2RayBlob(t *testing.T) {
	inner := "trojan://pass@trojan.example.com:443?sni=trojan.example.com#节点A\n"
	blob := base64.URLEncoding.EncodeToString([]byte(inner))
	nodes, skipped, err := parseV2Ray([]byte(blob))
	if err != nil {
		t.Fatal(err)
	}
	if len(skipped) != 0 || len(nodes) != 1 {
		t.Fatalf("got %d nodes %d skipped", len(nodes), len(skipped))
	}
	if nodes[0].Name != "节点A" || nodes[0].SNI != "trojan.example.com" {
		t.Fatalf("bad node: %+v", nodes[0])
	}
}

// TestParseV2RayCountsBadLines reports malformed lines as skipped instead
// of aborting.
func TestParseV2RayCountsBadLines(t *testing.T) {
	body := "not-a-link\nvless://11111111-1111-1111-1111-111111111111@a.com:443#ok\n"
	nodes, skipped, err := parseV2Ray([]byte(body))
	if err != nil {
		t.Fatal(err)
	}
	if len(nodes) != 1 || len(skipped) != 1 {
		t.Fatalf("got %d nodes %d skipped", len(nodes), len(skipped))
	}
}

// TestParseClashFullConfig extracts only proxies from a full clash config.
func TestParseClashFullConfig(t *testing.T) {
	body := []byte(`port: 7890
mode: rule
proxies:
  - name: hy2 node
    type: hysteria2
    server: hy2.example.com
    port: 443
    password: hy2-pass
    sni: hy2.example.com
    obfs: salamander
    obfs-password: obfs-pass
  - name: weird
    type: socks5
    server: 10.0.0.1
    port: 1080
proxy-groups:
  - name: g
    type: select
    proxies: [hy2 node]
rules: [MATCH,g]
`)
	nodes, skipped, err := parseClash(body)
	if err != nil {
		t.Fatal(err)
	}
	if len(nodes) != 1 || len(skipped) != 1 {
		t.Fatalf("got %d nodes %d skipped", len(nodes), len(skipped))
	}
	n := nodes[0]
	if n.Protocol != ProtoHysteria2 || n.Obfs != "salamander" || n.ObfsPassword != "obfs-pass" {
		t.Fatalf("bad hy2 node: %+v", n)
	}
	if !strings.Contains(skipped[0].Reason, "socks5") {
		t.Fatalf("skip reason should name the type: %+v", skipped[0])
	}
}

// TestParseSingBoxFullConfig extracts only outbounds from a full sing-box
// config and skips built-in outbound types.
func TestParseSingBoxFullConfig(t *testing.T) {
	body := []byte(`{
  "log": {"level": "warn"},
  "inbounds": [{"type": "mixed", "tag": "in", "listen": "127.0.0.1", "listen_port": 2080}],
  "outbounds": [
    {"type": "vless", "tag": "reality node", "server": "v.example.com", "server_port": 443,
     "uuid": "11111111-1111-1111-1111-111111111111", "flow": "xtls-rprx-vision",
     "tls": {"enabled": true, "server_name": "www.example.com", "utls": {"enabled": true, "fingerprint": "chrome"},
             "reality": {"enabled": true, "public_key": "pbk", "short_id": "sid"}}},
    {"type": "selector", "tag": "pick", "outbounds": ["reality node"]},
    {"type": "urltest", "tag": "auto", "outbounds": ["reality node"]},
    {"type": "tuic", "tag": "tuic node", "server": "t.example.com", "server_port": 443,
     "uuid": "44444444-4444-4444-4444-444444444444", "password": "tp",
     "congestion_control": "bbr", "udp_relay_mode": "quic",
     "tls": {"enabled": true, "alpn": ["h3"]}}
  ],
  "route": {"final": "pick"}
}`)
	nodes, skipped, err := parseSingBox(body)
	if err != nil {
		t.Fatal(err)
	}
	if len(nodes) != 2 {
		t.Fatalf("got %d nodes: %+v", len(nodes), nodes)
	}
	if len(skipped) != 2 {
		t.Fatalf("selector/urltest should be skipped: %+v", skipped)
	}
	r := nodes[0]
	if r.Security != "reality" || r.PublicKey != "pbk" || r.ShortID != "sid" || r.Flow != "xtls-rprx-vision" {
		t.Fatalf("bad reality node: %+v", r)
	}
	tu := nodes[1]
	if tu.CongestionControl != "bbr" || tu.UDPRelayMode != "quic" || len(tu.ALPN) != 1 {
		t.Fatalf("bad tuic node: %+v", tu)
	}
}

// TestNameWithSpecialChars survives URL-fragment encoding round trips.
func TestNameWithSpecialChars(t *testing.T) {
	nodes := []*Node{{
		Name: "节点 #1 & more/plus", Protocol: ProtoTrojan,
		Server: "t.example.com", Port: 443, Password: "p@ss/word",
		SNI: "t.example.com", Security: "tls",
	}}
	out, skipped := EmitV2Ray(nodes)
	if len(skipped) > 0 {
		t.Fatal(skipped)
	}
	back, skipped, err := Parse([]byte(out), FormatV2Ray)
	if err != nil || len(skipped) > 0 {
		t.Fatalf("re-parse: %v %+v", err, skipped)
	}
	if back[0].Name != nodes[0].Name || back[0].Password != nodes[0].Password {
		t.Fatalf("name/password mangled: got %q / %q", back[0].Name, back[0].Password)
	}
}

// TestConvertAuto runs the full auto-detect pipeline through the Service
// surface with a mixed-protocol clash input.
func TestConvertAuto(t *testing.T) {
	clash, _ := EmitClash(sampleNodes())
	var svc Service
	res, err := svc.Convert(ConvertRequest{Text: clash, From: "auto", To: "singbox"})
	if err != nil {
		t.Fatal(err)
	}
	if res.From != "clash" || res.To != "singbox" {
		t.Fatalf("bad formats: %+v", res)
	}
	if res.Converted != 8 {
		t.Fatalf("converted=%d skipped=%+v", res.Converted, res.Skipped)
	}
	if len(res.Skipped) != 0 {
		t.Fatalf("sample set should fully convert to singbox: %+v", res.Skipped)
	}
	back, err := svc.Convert(ConvertRequest{Text: res.Output, From: "auto", To: "clash"})
	if err != nil {
		t.Fatal(err)
	}
	if back.From != "singbox" || back.Converted != 8 {
		t.Fatalf("round trip back: %+v", back)
	}
}

// TestConvertRejectsGarbage surfaces a clear error for non-subscription
// input.
func TestConvertRejectsGarbage(t *testing.T) {
	var svc Service
	if _, err := svc.Convert(ConvertRequest{Text: "hello world", From: "auto", To: "clash"}); err == nil {
		t.Fatal("want error for garbage input")
	}
	if _, err := svc.Convert(ConvertRequest{Text: "vless://a@b:443#n", From: "auto", To: "bamboo"}); err == nil {
		t.Fatal("want error for bad target format")
	}
	if _, err := svc.Convert(ConvertRequest{Text: "", From: "auto", To: "clash"}); err == nil {
		t.Fatal("want error for empty input")
	}
}
