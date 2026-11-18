package subconv

// ConvertRequest asks for one subscription conversion.
type ConvertRequest struct {
	Text string `json:"text"` // subscription body or share-link list
	From string `json:"from"` // auto | v2ray | clash | singbox
	To   string `json:"to"`   // v2ray | clash | singbox
}

// ConvertResult reports one conversion outcome.
type ConvertResult struct {
	Output     string         `json:"output"`     // converted subscription text
	From       string         `json:"from"`       // detected (or given) input format
	To         string         `json:"to"`         // output format
	Total      int            `json:"total"`      // nodes parsed from the input
	Converted  int            `json:"converted"`  // nodes present in the output
	ByProtocol map[string]int `json:"byProtocol"` // parsed nodes per protocol
	Skipped    []Skipped      `json:"skipped"`    // nodes dropped, with reasons
}

// FetchResult is the raw body of a fetched subscription.
type FetchResult struct {
	Body        string `json:"body"`
	ContentType string `json:"contentType"`
}
