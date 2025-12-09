package logutil

type Config struct {
	IntervalSec    int64  `json:"IntervalSec"`
	Limit          int    `json:"Limit"`
	DisableStmtLog bool   `json:"DisableStmtLog"`
	NotifyChannel  string `json:"NotifyChannel"`
	NotifyWebhook  string `json:"NotifyWebhook"`
	NotifySecret   string `json:"NotifySecret"`
}
