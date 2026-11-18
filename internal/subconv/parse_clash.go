package subconv

import (
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
)

// parseClash extracts the proxies: list from a Clash (mihomo) config and
// maps each entry into a Node. proxy-groups/rules are ignored: the tool
// converts bare node tables.
func parseClash(body []byte) ([]*Node, []Skipped, error) {
	var doc struct {
		Proxies []map[string]any `yaml:"proxies"`
	}
	if err := yaml.Unmarshal(body, &doc); err != nil {
		return nil, nil, fmt.Errorf("不是有效的 Clash YAML：%w", err)
	}
	if doc.Proxies == nil {
		return nil, nil, fmt.Errorf("未找到 proxies: 列表")
	}
	var nodes []*Node
	var skipped []Skipped
	for _, p := range doc.Proxies {
		n := clashProxyToNode(p)
		if n == nil {
			skipped = append(skipped, Skipped{
				Name:   firstNonEmpty(stringOf(p["name"]), stringOf(p["type"]), "未知节点"),
				Reason: "不支持的节点类型 " + stringOf(p["type"]),
			})
			continue
		}
		if !n.Valid() {
			skipped = append(skipped, Skipped{Name: n.DisplayName(), Reason: "字段不完整（缺少服务器、端口或认证信息）"})
			continue
		}
		nodes = append(nodes, n)
	}
	return nodes, skipped, nil
}

// clashProxyToNode maps one mihomo proxies entry. Unknown types return nil.
func clashProxyToNode(p map[string]any) *Node {
	n := &Node{
		Name:   stringOf(p["name"]),
		Server: stringOf(p["server"]),
		Port:   parsePort(p["port"]),
	}
	tls := boolOf(p["tls"])
	if tls {
		n.Security = "tls"
	}

	switch stringOf(p["type"]) {
	case "vless":
		n.Protocol = ProtoVLESS
		n.UUID = stringOf(p["uuid"])
		n.Flow = stringOf(p["flow"])
		n.Fingerprint = firstNonEmpty(stringOf(p["client-fingerprint"]), stringOf(p["fingerprint"]))
		network := strings.ToLower(stringOf(p["network"]))
		n.Network = network
		n.Security = clashSecurity(p, tls)
		n.SNI = clashSNI(p)
		n.Insecure = boolOf(p["skip-cert-verify"])
		n.ALPN = clashALPN(p["alpn"])
		if ro, ok := p["reality-opts"].(map[string]any); ok {
			n.PublicKey = stringOf(ro["public-key"])
			n.ShortID = stringOf(ro["short-id"])
		}
		clashTransport(p, n, network)
	case "vmess":
		n.Protocol = ProtoVMess
		n.UUID = stringOf(p["uuid"])
		n.AlterID = parsePort(firstNonEmpty2(p["alterId"], p["alter-id"]))
		n.Cipher = firstNonEmpty(stringOf(p["cipher"]), "auto")
		network := strings.ToLower(stringOf(p["network"]))
		n.Network = network
		n.Security = clashSecurity(p, tls)
		n.SNI = clashSNI(p)
		n.Insecure = boolOf(p["skip-cert-verify"])
		n.ALPN = clashALPN(p["alpn"])
		clashTransport(p, n, network)
	case "trojan":
		n.Protocol = ProtoTrojan
		n.Password = stringOf(p["password"])
		network := strings.ToLower(stringOf(p["network"]))
		n.Network = network
		n.SNI = clashSNI(p)
		n.Insecure = boolOf(p["skip-cert-verify"])
		n.ALPN = clashALPN(p["alpn"])
		clashTransport(p, n, network)
	case "ss", "shadowsocks":
		n.Protocol = ProtoSS
		n.Cipher = stringOf(p["cipher"])
		n.Password = stringOf(p["password"])
		n.Plugin = stringOf(p["plugin"])
		n.PluginOpts = clashPluginOpts(p["plugin-opts"])
	case "hysteria2":
		n.Protocol = ProtoHysteria2
		n.Password = stringOf(p["password"])
		n.SNI = clashSNI(p)
		n.Insecure = boolOf(p["skip-cert-verify"])
		n.ALPN = clashALPN(p["alpn"])
		n.Obfs = stringOf(p["obfs"])
		n.ObfsPassword = stringOf(p["obfs-password"])
	case "tuic":
		n.Protocol = ProtoTUIC
		n.UUID = stringOf(p["uuid"])
		n.Password = stringOf(p["password"])
		n.CongestionControl = firstNonEmpty(stringOf(p["congestion-controller"]), "cubic")
		n.UDPRelayMode = stringOf(p["udp-relay-mode"])
		n.SNI = clashSNI(p)
		n.Insecure = boolOf(p["skip-cert-verify"])
		n.ALPN = clashALPN(p["alpn"])
		n.DisableSNI = boolOf(p["disable-sni"])
	case "anytls":
		n.Protocol = ProtoAnyTLS
		n.Password = stringOf(p["password"])
		n.SNI = clashSNI(p)
		n.Insecure = boolOf(p["skip-cert-verify"])
		n.Fingerprint = firstNonEmpty(stringOf(p["client-fingerprint"]), stringOf(p["fingerprint"]))
		n.ALPN = clashALPN(p["alpn"])
	default:
		return nil
	}
	return n
}

// clashSecurity reads the vless/vmess security mode; reality beats tls.
func clashSecurity(p map[string]any, tls bool) string {
	if _, ok := p["reality-opts"].(map[string]any); ok {
		return "reality"
	}
	if tls {
		return "tls"
	}
	return "none"
}

// clashSNI reads sni / servername (both spellings exist in the wild).
func clashSNI(p map[string]any) string {
	return firstNonEmpty(stringOf(p["sni"]), stringOf(p["servername"]))
}

// clashALPN accepts a []any or a comma-joined string.
func clashALPN(v any) []string {
	switch t := v.(type) {
	case string:
		return splitList(t)
	case []any:
		var out []string
		for _, item := range t {
			if s := stringOf(item); s != "" {
				out = append(out, s)
			}
		}
		return out
	}
	return nil
}

// clashTransport fills ws/grpc fields from ws-opts / grpc-opts.
func clashTransport(p map[string]any, n *Node, network string) {
	switch network {
	case "ws", "http", "h2":
		if ws, ok := p["ws-opts"].(map[string]any); ok {
			n.Path = stringOf(ws["path"])
			if headers, ok := ws["headers"].(map[string]any); ok {
				n.Host = stringOf(headers["Host"])
			}
		}
		// h2-opts carries its own host list.
		if h2, ok := p["h2-opts"].(map[string]any); ok {
			n.Path = firstNonEmpty(n.Path, stringOf(h2["path"]))
			if hosts, ok := h2["host"].([]any); ok && len(hosts) > 0 {
				n.Host = stringOf(hosts[0])
			}
		}
	case "grpc":
		if grpc, ok := p["grpc-opts"].(map[string]any); ok {
			n.ServiceName = stringOf(grpc["grpc-service-name"])
		}
	}
}

// clashPluginOpts renders plugin-opts back into the "k=v;k=v" form used by
// share links.
func clashPluginOpts(v any) string {
	opts, ok := v.(map[string]any)
	if !ok {
		return ""
	}
	var parts []string
	for k, val := range opts {
		parts = append(parts, k+"="+stringOf(val))
	}
	return strings.Join(parts, ";")
}

// firstNonEmpty2 picks the first present key's string value from a yaml map.
func firstNonEmpty2(v1, v2 any) string {
	if s := stringOf(v1); s != "" {
		return s
	}
	return stringOf(v2)
}
