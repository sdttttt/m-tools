// Package subconv converts proxy subscriptions between the v2ray (base64
// share-link list), Clash (mihomo YAML) and sing-box (>=1.8 JSON) formats.
// All three inputs parse into the unified Node representation and all three
// outputs emit from it, so format pairs never convert directly. Everything
// runs locally: subscription URLs are fetched over plain HTTP GET and the
// conversion never leaves the machine.
package subconv

// Protocol identifiers, one per supported proxy protocol.
const (
	ProtoVLESS     = "vless"
	ProtoVMess     = "vmess"
	ProtoTrojan    = "trojan"
	ProtoSS        = "shadowsocks"
	ProtoHysteria2 = "hysteria2"
	ProtoTUIC      = "tuic"
	ProtoAnyTLS    = "anytls"
)

// Node is the normalized representation of a proxy endpoint. Fields are a
// superset of what each protocol needs; unused fields stay zero. Names of
// fields follow the share-link vocabulary (SNI, ALPN, Reality public key…)
// rather than any single client's config keys.
type Node struct {
	Name     string `json:"name"`
	Protocol string `json:"protocol"`
	Server   string `json:"server"`
	Port     int    `json:"port"`

	// Credentials: which fields matter depends on the protocol —
	// VLESS/VMess/TUIC use UUID, Trojan/Hysteria2/AnyTLS use Password,
	// Shadowsocks uses Cipher+Password.
	UUID     string `json:"uuid,omitempty"`
	Password string `json:"password,omitempty"`
	Cipher   string `json:"cipher,omitempty"`
	AlterID  int    `json:"alterId,omitempty"`

	// Transport + security, shared by the TLS-family protocols
	// (VLESS/VMess/Trojan) and reused where the client supports it.
	Network     string   `json:"network,omitempty"`  // tcp|ws|grpc|http|httpupgrade
	Security    string   `json:"security,omitempty"` // none|tls|reality
	SNI         string   `json:"sni,omitempty"`
	ALPN        []string `json:"alpn,omitempty"`
	Fingerprint string   `json:"fingerprint,omitempty"` // uTLS client fingerprint
	Insecure    bool     `json:"insecure,omitempty"`

	// VLESS Reality.
	PublicKey string `json:"publicKey,omitempty"`
	ShortID   string `json:"shortID,omitempty"`
	SpiderX   string `json:"spiderX,omitempty"`
	Flow      string `json:"flow,omitempty"` // xtls-rprx-vision

	// WebSocket / HTTP transport.
	Path string `json:"path,omitempty"`
	Host string `json:"host,omitempty"` // Host header for ws/http

	// gRPC transport.
	ServiceName string `json:"serviceName,omitempty"`

	// Shadowsocks plugin (obfs-local / v2ray-plugin), "name;opt=val;...".
	Plugin     string `json:"plugin,omitempty"`
	PluginOpts string `json:"pluginOpts,omitempty"`

	// Hysteria2 obfs.
	Obfs         string `json:"obfs,omitempty"` // salamander
	ObfsPassword string `json:"obfsPassword,omitempty"`

	// TUIC.
	CongestionControl string `json:"congestionControl,omitempty"` // cubic|new_reno|bbr
	UDPRelayMode      string `json:"udpRelayMode,omitempty"`      // native|quic
	ZeroRTT           bool   `json:"zeroRTT,omitempty"`
	DisableSNI        bool   `json:"disableSNI,omitempty"`
}

// Skipped reports one node that did not survive parsing or emitting,
// with the reason shown to the user.
type Skipped struct {
	Name   string `json:"name"`
	Reason string `json:"reason"`
}

// Valid reports whether the node carries the minimum fields its protocol
// needs to be worth emitting.
func (n *Node) Valid() bool {
	if n.Server == "" || n.Port <= 0 || n.Port > 65535 {
		return false
	}
	switch n.Protocol {
	case ProtoVLESS, ProtoVMess:
		return n.UUID != ""
	case ProtoTrojan, ProtoHysteria2, ProtoAnyTLS:
		return n.Password != ""
	case ProtoTUIC:
		return n.UUID != "" && n.Password != ""
	case ProtoSS:
		return n.Cipher != "" && n.Password != ""
	}
	return false
}

// DisplayName falls back to a protocol+endpoint label when the share link
// or config entry carries no name.
func (n *Node) DisplayName() string {
	if n.Name != "" {
		return n.Name
	}
	return n.Protocol + "-" + n.Server
}
