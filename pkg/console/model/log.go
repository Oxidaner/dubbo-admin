package model

// SearchLogsReq is the request struct for search logs
type SearchLogsReq struct {
	Mesh         string `json:"mesh"`
	AppName      string `json:"appName"`
	ServiceName  string `json:"serviceName"`
	InstanceName string `json:"instanceName"`
	Keywords     string `json:"keywords"`
	StartTime    string `json:"startTime"`
	EndTime      string `json:"endTime"`
	Limit        int    `json:"limit"`
	NextToken    string `json:"nextToken"`
}

type LogItem struct {
	Timestamp    string            `json:"timestamp"` // The timestamp of the log
	AppName      string            `json:"appName,omitempty"`
	ServiceName  string            `json:"serviceName,omitempty"`
	InstanceName string            `json:"instanceName,omitempty"`
	Severity     string            `json:"severity,omitempty"` // The log severity level (e.g., "INFO", "ERROR")
	Message      string            `json:"message"`
	TraceID      string            `json:"traceId,omitempty"`
	SpanID       string            `json:"spanId,omitempty"`
	Attributes   map[string]string `json:"attributes,omitempty"` // The log attributes
	Raw          string            `json:"raw,omitempty"`        // The raw log message
}

// SearchLogsResp is the response struct for search logs
type SearchLogsResp struct {
	Logs      []LogItem `json:"logs"`
	NextToken string    `json:"nextToken,omitempty"`
}

type AnalyzeErrorLogsReq struct {
	SearchLogsReq
	GroupBy string `json:"groupBy"`
}

// ErrorBucket represents a bucket of error logs with a key and count
type ErrorBucket struct {
	Key   string `json:"key"`
	Count int    `json:"count"`
}

type AnalyzeErrorLogsResp struct {
	Summary   string        `json:"summary"`
	TopErrors []ErrorBucket `json:"topErrors"`
}
