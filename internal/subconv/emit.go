package subconv

import "fmt"

// Emit renders nodes into the requested output format. Nodes without a
// mapping are reported in skipped.
func Emit(nodes []*Node, format Format) (string, []Skipped, error) {
	switch format {
	case FormatV2Ray:
		out, skipped := EmitV2Ray(nodes)
		return out, skipped, nil
	case FormatClash:
		out, skipped := EmitClash(nodes)
		return out, skipped, nil
	case FormatSingBox:
		out, skipped := EmitSingBox(nodes)
		return out, skipped, nil
	}
	return "", nil, fmt.Errorf("unknown output format %q (want v2ray|clash|singbox)", format)
}
