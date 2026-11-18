package subconv

import (
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
)

// EmitClash renders nodes as a bare mihomo (Clash.Meta) proxies list: a
// YAML document with only the proxies: key, no proxy-groups or rules. Nodes
// whose transport or protocol has no mihomo mapping land in skipped.
func EmitClash(nodes []*Node) (string, []Skipped) {
	var items []*yaml.Node
	var skipped []Skipped
	for _, n := range nodes {
		item, err := clashProxyNode(n)
		if err != nil {
			skipped = append(skipped, Skipped{Name: n.DisplayName(), Reason: err.Error()})
			continue
		}
		items = append(items, item)
	}

	root := yamlMap("proxies", &yaml.Node{Kind: yaml.SequenceNode, Content: items})
	var b strings.Builder
	enc := yaml.NewEncoder(&b)
	enc.SetIndent(2)
	if err := enc.Encode(root); err != nil {
		// The tree is built from scalars and maps only; Encode cannot fail.
		return "", []Skipped{{Name: "-", Reason: err.Error()}}
	}
	enc.Close()
	return b.String(), skipped
}

// clashProxyNode builds one ordered proxies entry.
func clashProxyNode(n *Node) (*yaml.Node, error) {
	if err := clashTransportOK(n); err != nil {
		return nil, err
	}

	var p []any
	addCommon := func(typeName string) {
		p = append(p, "name", n.Name, "type", typeName, "server", n.Server, "port", n.Port)
	}
	addTLS := func(sniKey string) {
		if n.Security == "tls" || n.Security == "reality" {
			p = append(p, "tls", true)
			p = append(p, sniKey, n.SNI)
			p = append(p, "skip-cert-verify", n.Insecure)
			if len(n.ALPN) > 0 {
				p = append(p, "alpn", n.ALPN)
			}
			if n.Fingerprint != "" {
				p = append(p, "client-fingerprint", n.Fingerprint)
			}
		}
		if n.Security == "reality" {
			p = append(p, "reality-opts", yamlMap(
				"public-key", n.PublicKey,
				"short-id", n.ShortID,
			))
		}
	}
	addNetwork := func() {
		switch n.Network {
		case "ws":
			p = append(p, "network", "ws")
			var ws []any
			ws = append(ws, "path", n.Path)
			if n.Host != "" {
				ws = append(ws, "headers", yamlMap("Host", n.Host))
			}
			p = append(p, "ws-opts", yamlMap(ws...))
		case "grpc":
			p = append(p, "network", "grpc")
			p = append(p, "grpc-opts", yamlMap("grpc-service-name", n.ServiceName))
		case "http", "h2":
			p = append(p, "network", "h2")
			var h2 []any
			h2 = append(h2, "path", n.Path)
			if n.Host != "" {
				h2 = append(h2, "host", []string{n.Host})
			}
			p = append(p, "h2-opts", yamlMap(h2...))
		}
	}

	switch n.Protocol {
	case ProtoVLESS:
		addCommon("vless")
		p = append(p, "uuid", n.UUID, "udp", true)
		addTLS("servername")
		if n.Flow != "" {
			p = append(p, "flow", n.Flow)
		}
		addNetwork()
	case ProtoVMess:
		addCommon("vmess")
		p = append(p, "uuid", n.UUID, "alterId", n.AlterID, "cipher", clashCipher(n.Cipher), "udp", true)
		addTLS("servername")
		addNetwork()
	case ProtoTrojan:
		addCommon("trojan")
		p = append(p, "password", n.Password, "udp", true)
		// mihomo spells the trojan SNI key "sni".
		addTLS("sni")
		addNetwork()
	case ProtoSS:
		addCommon("ss")
		p = append(p, "cipher", n.Cipher, "password", n.Password, "udp", true)
		if n.Plugin != "" {
			p = append(p, "plugin", n.Plugin, "plugin-opts", clashPluginOptsNode(n.PluginOpts))
		}
	case ProtoHysteria2:
		addCommon("hysteria2")
		p = append(p, "password", n.Password)
		if n.SNI != "" {
			p = append(p, "sni", n.SNI)
		}
		if n.Insecure {
			p = append(p, "skip-cert-verify", true)
		}
		if len(n.ALPN) > 0 {
			p = append(p, "alpn", n.ALPN)
		}
		if n.Obfs != "" {
			p = append(p, "obfs", n.Obfs, "obfs-password", n.ObfsPassword)
		}
	case ProtoTUIC:
		addCommon("tuic")
		p = append(p, "uuid", n.UUID, "password", n.Password)
		p = append(p, "congestion-controller", firstNonEmpty(n.CongestionControl, "cubic"))
		if n.UDPRelayMode != "" {
			p = append(p, "udp-relay-mode", n.UDPRelayMode)
		}
		if n.SNI != "" {
			p = append(p, "sni", n.SNI)
		}
		if n.Insecure {
			p = append(p, "skip-cert-verify", true)
		}
		if len(n.ALPN) > 0 {
			p = append(p, "alpn", n.ALPN)
		}
		if n.DisableSNI {
			p = append(p, "disable-sni", true)
		}
	case ProtoAnyTLS:
		addCommon("anytls")
		p = append(p, "password", n.Password, "udp", true)
		if n.SNI != "" {
			p = append(p, "sni", n.SNI)
		}
		if n.Insecure {
			p = append(p, "skip-cert-verify", true)
		}
		if n.Fingerprint != "" {
			p = append(p, "client-fingerprint", n.Fingerprint)
		}
	default:
		return nil, fmt.Errorf("未知协议 %q", n.Protocol)
	}
	return yamlMap(p...), nil
}

// clashTransportOK rejects transports mihomo has no mapping for.
func clashTransportOK(n *Node) error {
	switch n.Protocol {
	case ProtoVLESS, ProtoVMess, ProtoTrojan:
		switch n.Network {
		case "", "tcp":
			return nil
		case "ws", "grpc":
			return nil
		case "http", "h2":
			if n.Protocol == ProtoVMess {
				return nil
			}
			return fmt.Errorf("传输层 h2 仅 VMess 受 Clash 支持")
		}
		return fmt.Errorf("传输层 %s 不受 Clash 支持", n.Network)
	}
	return nil
}

// clashCipher normalizes the vmess security spelling; share links use
// "auto" while mihomo's default cipher key is "auto" too, so this mostly
// guards empty values.
func clashCipher(cipher string) string {
	if cipher == "" {
		return "auto"
	}
	return cipher
}

// clashPluginOptsNode splits "k=v;k=v" back into a plugin-opts map.
func clashPluginOptsNode(opts string) *yaml.Node {
	var p []any
	for _, kv := range strings.Split(opts, ";") {
		k, v, ok := strings.Cut(kv, "=")
		if !ok || k == "" {
			continue
		}
		p = append(p, k, v)
	}
	if len(p) == 0 {
		return yamlMap()
	}
	return yamlMap(p...)
}

// yamlMap builds a mapping node from alternating key/value pairs, keeping
// the insertion order that yaml.v3 would otherwise sort away.
func yamlMap(pairs ...any) *yaml.Node {
	node := &yaml.Node{Kind: yaml.MappingNode}
	for i := 0; i+1 < len(pairs); i += 2 {
		key, _ := pairs[i].(string)
		node.Content = append(node.Content,
			&yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: key},
			yamlValue(pairs[i+1]),
		)
	}
	return node
}

// yamlValue wraps plain Go values into yaml nodes.
func yamlValue(v any) *yaml.Node {
	switch t := v.(type) {
	case *yaml.Node:
		return t
	case string:
		return &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: t}
	case int:
		return &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!int", Value: fmt.Sprintf("%d", t)}
	case bool:
		return &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!bool", Value: fmt.Sprintf("%t", t)}
	case []string:
		seq := &yaml.Node{Kind: yaml.SequenceNode}
		for _, s := range t {
			seq.Content = append(seq.Content, yamlValue(s))
		}
		return seq
	}
	return &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!null", Value: ""}
}
