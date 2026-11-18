package subconv

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
)

// EmitSingBox renders nodes as a bare sing-box (>=1.8) outbound table: a
// JSON object with only the outbounds array, ready to be merged into a full
// config. Nodes with no sing-box mapping (unsupported transport, ss plugins
// beyond the built-in obfs-local/v2ray-plugin) land in skipped.
func EmitSingBox(nodes []*Node) (string, []Skipped) {
	used := map[string]int{}
	var items []any
	var skipped []Skipped
	for _, n := range nodes {
		ob, err := singBoxNode(n)
		if err != nil {
			skipped = append(skipped, Skipped{Name: n.DisplayName(), Reason: err.Error()})
			continue
		}
		ob = append(ob, jpair{k: "tag", v: uniqueTag(used, n.DisplayName())})
		items = append(items, jobj(ob))
	}

	cfg := jobj{{k: "outbounds", v: items}}
	raw, err := json.Marshal(cfg)
	if err != nil {
		return "", []Skipped{{Name: "-", Reason: err.Error()}}
	}
	var pretty bytes.Buffer
	if err := json.Indent(&pretty, raw, "", "  "); err != nil {
		return "", []Skipped{{Name: "-", Reason: err.Error()}}
	}
	return pretty.String() + "\n", skipped
}

// singBoxNode builds the ordered field list of one outbound; the tag is
// appended by the caller after tag uniquification.
func singBoxNode(n *Node) ([]jpair, error) {
	base := []jpair{
		{k: "type", v: sbType(n.Protocol)},
		{k: "server", v: n.Server},
		{k: "server_port", v: n.Port},
	}
	if base[0].v == "" {
		return nil, fmt.Errorf("未知协议 %q", n.Protocol)
	}

	var ob []jpair
	switch n.Protocol {
	case ProtoVLESS:
		transport, err := sbTransport(n)
		if err != nil {
			return nil, err
		}
		ob = append(ob, base...)
		ob = append(ob, jpair{k: "uuid", v: n.UUID})
		if n.Flow != "" {
			ob = append(ob, jpair{k: "flow", v: n.Flow})
		}
		ob = append(ob, sbTLS(n)...)
		ob = append(ob, transport...)
	case ProtoVMess:
		transport, err := sbTransport(n)
		if err != nil {
			return nil, err
		}
		ob = append(ob, base...)
		ob = append(ob,
			jpair{k: "uuid", v: n.UUID},
			jpair{k: "security", v: firstNonEmpty(n.Cipher, "auto")},
			jpair{k: "alter_id", v: n.AlterID},
		)
		ob = append(ob, sbTLS(n)...)
		ob = append(ob, transport...)
	case ProtoTrojan:
		transport, err := sbTransport(n)
		if err != nil {
			return nil, err
		}
		ob = append(ob, base...)
		ob = append(ob, jpair{k: "password", v: n.Password})
		ob = append(ob, sbTLS(n)...)
		ob = append(ob, transport...)
	case ProtoSS:
		if n.Plugin != "" && n.Plugin != "obfs-local" && n.Plugin != "v2ray-plugin" {
			return nil, fmt.Errorf("sing-box 不支持 Shadowsocks 插件（%s）", n.Plugin)
		}
		ob = append(ob, base...)
		ob = append(ob,
			jpair{k: "method", v: n.Cipher},
			jpair{k: "password", v: n.Password},
		)
		if n.Plugin != "" {
			ob = append(ob,
				jpair{k: "plugin", v: n.Plugin},
				jpair{k: "plugin_opts", v: n.PluginOpts},
			)
		}
	case ProtoHysteria2:
		ob = append(ob, base...)
		ob = append(ob, jpair{k: "password", v: n.Password})
		ob = append(ob, sbTLS(n)...)
		if n.Obfs != "" {
			ob = append(ob, jpair{k: "obfs", v: jobj{
				{k: "type", v: n.Obfs},
				{k: "password", v: n.ObfsPassword},
			}})
		}
	case ProtoTUIC:
		ob = append(ob, base...)
		ob = append(ob,
			jpair{k: "uuid", v: n.UUID},
			jpair{k: "password", v: n.Password},
			jpair{k: "congestion_control", v: firstNonEmpty(n.CongestionControl, "cubic")},
		)
		if n.UDPRelayMode != "" {
			ob = append(ob, jpair{k: "udp_relay_mode", v: n.UDPRelayMode})
		}
		if n.ZeroRTT {
			ob = append(ob, jpair{k: "zero_rtt_handshake", v: true})
		}
		ob = append(ob, sbTLS(n)...)
	case ProtoAnyTLS:
		ob = append(ob, base...)
		ob = append(ob, jpair{k: "password", v: n.Password})
		ob = append(ob, sbTLS(n)...)
	}
	return ob, nil
}

// sbType maps the internal protocol id to the sing-box outbound type.
func sbType(protocol string) string {
	switch protocol {
	case ProtoSS:
		return "shadowsocks"
	case ProtoHysteria2:
		return "hysteria2"
	default:
		return protocol // vless/vmess/trojan/tuic/anytls match directly
	}
}

// tlsSecurity resolves the effective security mode: Trojan/Hysteria2/TUIC/
// AnyTLS are TLS-only protocols, so an empty Security means tls there.
func (n *Node) tlsSecurity() string {
	if n.Security != "" {
		return n.Security
	}
	switch n.Protocol {
	case ProtoTrojan, ProtoHysteria2, ProtoTUIC, ProtoAnyTLS:
		return "tls"
	}
	return "none"
}

// sbTLS builds the tls block; empty for plaintext nodes. Reality rides
// inside the tls block in sing-box >=1.8.
func sbTLS(n *Node) []jpair {
	security := n.tlsSecurity()
	if security != "tls" && security != "reality" {
		return nil
	}
	var tls []jpair
	tls = append(tls, jpair{k: "enabled", v: true})
	if n.SNI != "" {
		tls = append(tls, jpair{k: "server_name", v: n.SNI})
	}
	if n.Insecure {
		tls = append(tls, jpair{k: "insecure", v: true})
	}
	if len(n.ALPN) > 0 {
		tls = append(tls, jpair{k: "alpn", v: n.ALPN})
	}
	if n.Fingerprint != "" {
		// uTLS parroting lives in the utls block since sing-box 1.8.
		tls = append(tls, jpair{k: "utls", v: jobj{
			{k: "enabled", v: true},
			{k: "fingerprint", v: n.Fingerprint},
		}})
	}
	if n.DisableSNI {
		tls = append(tls, jpair{k: "disable_sni", v: true})
	}
	if security == "reality" {
		tls = append(tls, jpair{k: "reality", v: jobj{
			{k: "enabled", v: true},
			{k: "public_key", v: n.PublicKey},
			{k: "short_id", v: n.ShortID},
		}})
	}
	return []jpair{{k: "tls", v: jobj(tls)}}
}

// sbTransport builds the transport block; tcp/absent means no block.
func sbTransport(n *Node) ([]jpair, error) {
	switch n.Network {
	case "", "tcp":
		return nil, nil
	case "ws":
		var headers []jpair
		if n.Host != "" {
			headers = append(headers, jpair{k: "Host", v: n.Host})
		}
		tr := []jpair{{k: "type", v: "ws"}}
		if n.Path != "" {
			tr = append(tr, jpair{k: "path", v: n.Path})
		}
		if len(headers) > 0 {
			tr = append(tr, jpair{k: "headers", v: jobj(headers)})
		}
		return []jpair{{k: "transport", v: jobj(tr)}}, nil
	case "grpc":
		return []jpair{{k: "transport", v: jobj{
			{k: "type", v: "grpc"},
			{k: "service_name", v: n.ServiceName},
		}}}, nil
	case "http", "h2":
		// The HTTP transport carries its authority in a host list, not in
		// headers like WebSocket.
		tr := []jpair{
			{k: "type", v: "http"},
			{k: "path", v: n.Path},
		}
		if n.Host != "" {
			tr = append(tr, jpair{k: "host", v: []string{n.Host}})
		}
		return []jpair{{k: "transport", v: jobj(tr)}}, nil
	case "httpupgrade":
		tr := []jpair{{k: "type", v: "httpupgrade"}}
		if n.Path != "" {
			tr = append(tr, jpair{k: "path", v: n.Path})
		}
		if n.Host != "" {
			tr = append(tr, jpair{k: "host", v: n.Host})
		}
		return []jpair{{k: "transport", v: jobj(tr)}}, nil
	}
	return nil, fmt.Errorf("传输层 %s 不受 sing-box 支持", n.Network)
}

// uniqueTag appends a numeric suffix when a tag repeats, so the emitted
// outbounds stay referable.
func uniqueTag(used map[string]int, tag string) string {
	if used[tag] == 0 {
		used[tag] = 1
		return tag
	}
	used[tag]++
	return fmt.Sprintf("%s-%d", tag, used[tag])
}

// jpair is one ordered key/value entry of a JSON object.
type jpair struct {
	k string
	v any
}

// jobj marshals as a JSON object with insertion-ordered keys (encoding/json
// would sort map keys alphabetically).
type jobj []jpair

func (o jobj) MarshalJSON() ([]byte, error) {
	var b strings.Builder
	b.WriteByte('{')
	for i, p := range o {
		if i > 0 {
			b.WriteByte(',')
		}
		key, err := json.Marshal(p.k)
		if err != nil {
			return nil, err
		}
		val, err := json.Marshal(p.v)
		if err != nil {
			return nil, err
		}
		b.Write(key)
		b.WriteByte(':')
		b.Write(val)
	}
	b.WriteByte('}')
	return []byte(b.String()), nil
}
