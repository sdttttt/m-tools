package subconv

import (
	"encoding/json"
	"fmt"
	"strings"
)

// parseSingBox extracts the outbounds array from a sing-box (>=1.8) config.
// Full configs with inbounds/route/dns are accepted — only outbounds are
// read. Built-in outbound types (selector, urltest, direct, block, dns) and
// unknown protocols land in skipped.
func parseSingBox(body []byte) ([]*Node, []Skipped, error) {
	var doc struct {
		Outbounds []map[string]any `json:"outbounds"`
	}
	if err := json.Unmarshal(body, &doc); err != nil {
		return nil, nil, fmt.Errorf("不是有效的 sing-box JSON：%w", err)
	}
	if doc.Outbounds == nil {
		return nil, nil, fmt.Errorf("未找到 outbounds 数组")
	}
	var nodes []*Node
	var skipped []Skipped
	for _, ob := range doc.Outbounds {
		n := singBoxOutboundToNode(ob)
		if n == nil {
			name := firstNonEmpty(stringOf(ob["tag"]), stringOf(ob["server"]))
			if name == "" {
				name = "未知节点"
			}
			skipped = append(skipped, Skipped{
				Name:   name,
				Reason: "不支持的出站类型 " + stringOf(ob["type"]),
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

// singBoxOutboundToNode maps one sing-box outbound. Built-in and unknown
// outbound types return nil.
func singBoxOutboundToNode(ob map[string]any) *Node {
	n := &Node{
		Name:   stringOf(ob["tag"]),
		Server: stringOf(ob["server"]),
		Port:   parsePort(ob["server_port"]),
	}

	tls, _ := ob["tls"].(map[string]any)
	readTLS := func() {
		if tls == nil {
			return
		}
		if !boolOf(tls["enabled"]) {
			return
		}
		n.Security = "tls"
		n.SNI = stringOf(tls["server_name"])
		n.Insecure = boolOf(tls["insecure"])
		n.ALPN = stringSliceOf(tls["alpn"])
		// uTLS parroting lives in the utls block since sing-box 1.8;
		// tolerate the legacy flat fingerprint string too.
		n.Fingerprint = stringOf(tls["fingerprint"])
		if utls, ok := tls["utls"].(map[string]any); ok {
			n.Fingerprint = firstNonEmpty(stringOf(utls["fingerprint"]), n.Fingerprint)
		}
		n.DisableSNI = boolOf(tls["disable_sni"])
		if reality, ok := tls["reality"].(map[string]any); ok && boolOf(reality["enabled"]) {
			n.Security = "reality"
			n.PublicKey = stringOf(reality["public_key"])
			n.ShortID = stringOf(reality["short_id"])
		}
	}

	switch stringOf(ob["type"]) {
	case "vless":
		n.Protocol = ProtoVLESS
		n.UUID = stringOf(ob["uuid"])
		n.Flow = stringOf(ob["flow"])
		readTLS()
		if n.Security == "" {
			n.Security = "none"
		}
		singBoxTransport(ob, n)
	case "vmess":
		n.Protocol = ProtoVMess
		n.UUID = stringOf(ob["uuid"])
		n.AlterID = parsePort(ob["alter_id"])
		n.Cipher = firstNonEmpty(stringOf(ob["security"]), "auto")
		readTLS()
		if n.Security == "" {
			n.Security = "none"
		}
		singBoxTransport(ob, n)
	case "trojan":
		n.Protocol = ProtoTrojan
		n.Password = stringOf(ob["password"])
		readTLS()
		if n.Security == "" {
			n.Security = "tls"
		}
		singBoxTransport(ob, n)
	case "shadowsocks":
		n.Protocol = ProtoSS
		n.Cipher = stringOf(ob["method"])
		n.Password = stringOf(ob["password"])
		n.Plugin = stringOf(ob["plugin"])
		n.PluginOpts = stringOf(ob["plugin_opts"])
	case "hysteria2":
		n.Protocol = ProtoHysteria2
		n.Password = stringOf(ob["password"])
		readTLS()
		if obfs, ok := ob["obfs"].(map[string]any); ok {
			n.Obfs = stringOf(obfs["type"])
			n.ObfsPassword = stringOf(obfs["password"])
		}
	case "tuic":
		n.Protocol = ProtoTUIC
		n.UUID = stringOf(ob["uuid"])
		n.Password = stringOf(ob["password"])
		n.CongestionControl = firstNonEmpty(stringOf(ob["congestion_control"]), "cubic")
		n.UDPRelayMode = stringOf(ob["udp_relay_mode"])
		n.ZeroRTT = boolOf(ob["zero_rtt_handshake"])
		readTLS()
	case "anytls":
		n.Protocol = ProtoAnyTLS
		n.Password = stringOf(ob["password"])
		readTLS()
	default:
		return nil
	}
	return n
}

// singBoxTransport fills the ws/grpc/http fields from a sing-box
// transport block.
func singBoxTransport(ob map[string]any, n *Node) {
	tr, ok := ob["transport"].(map[string]any)
	if !ok {
		n.Network = "tcp"
		return
	}
	t := strings.ToLower(stringOf(tr["type"]))
	n.Network = t
	switch t {
	case "ws", "httpupgrade":
		n.Path = stringOf(tr["path"])
		if headers, ok := tr["headers"].(map[string]any); ok {
			n.Host = stringOf(headers["Host"])
		}
		if n.Host == "" {
			n.Host = firstNonEmpty(stringOf(tr["host"]), n.Host)
		}
	case "http":
		n.Path = stringOf(tr["path"])
		// The HTTP transport carries its authority in a host list.
		if hosts := stringSliceOf(tr["host"]); len(hosts) > 0 {
			n.Host = hosts[0]
		}
		if n.Host == "" {
			if headers, ok := tr["headers"].(map[string]any); ok {
				n.Host = stringOf(headers["Host"])
			}
		}
	case "grpc":
		n.ServiceName = stringOf(tr["service_name"])
	default:
		// tcp and anything unrecognized keep their network value so the
		// emitters can decide whether they can represent it.
	}
}

// stringSliceOf reads a JSON string array.
func stringSliceOf(v any) []string {
	items, ok := v.([]any)
	if !ok {
		return nil
	}
	var out []string
	for _, item := range items {
		if s := stringOf(item); s != "" {
			out = append(out, s)
		}
	}
	return out
}
