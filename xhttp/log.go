package xhttp

import "encoding/json"

// RequestResponseLog 请求响应日志
type RequestResponseLog struct {
	// 基础日志信息（自动获取）
	URL      string            `json:"url"`
	Method   string            `json:"method"`
	Headers  map[string]string `json:"headers"`
	Request  []byte            `json:"request"`
	Response []byte            `json:"response"`
	Status   int               `json:"status"`
	TimeCost int64             `json:"time_cost"`
	CTime    int64             `json:"ctime"`

	// 扩展日志信息（需要调用方设置）
	Extend *LogExtend `json:"extend"`
}

// ToJSON 将日志转换为JSON字符串
func (l *RequestResponseLog) ToJSON() ([]byte, error) {
	type jsonLog struct {
		URL      string     `json:"url"`
		Method   string     `json:"method"`
		Headers  string     `json:"headers"`
		Request  string     `json:"request"`
		Response string     `json:"response"`
		Status   int        `json:"status"`
		TimeCost int64      `json:"time_cost"`
		CTime    int64      `json:"ctime"`
		Extend   *LogExtend `json:"extend"`
	}

	// Convert headers map to JSON string
	headersJSON, _ := json.Marshal(l.Headers)
	log := jsonLog{
		URL:      l.URL,
		Method:   l.Method,
		Headers:  string(headersJSON),
		Request:  string(l.Request),
		Response: string(l.Response),
		Status:   l.Status,
		TimeCost: l.TimeCost,
		CTime:    l.CTime,
		Extend:   l.Extend,
	}

	return json.Marshal(log)
}

// LogExtend 扩展日志信息
type LogExtend struct {
	Supplier   int    `json:"supplier"`
	TraceID    string `json:"trace_id"`
	AppID      string `json:"app_id"`
	SubAppID   string `json:"sub_app_id"`
	RelatedTID string `json:"related_tid"`
	RelatedUID string `json:"related_uid"`
	Expand     string `json:"expand"`
}
