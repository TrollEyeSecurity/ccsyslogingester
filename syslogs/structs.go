package syslogs

type Log struct {
	SrcAddr string `json:"srcAddr"`
	Hdr     string `json:"hdr"`
	Msg     string `json:"msg"`
}

type Task struct {
	Id     string `json:"id"`
	LogId  string `json:"log_id"`
	Type   string `json:"type"`
	Source string `json:"source"`
}

type CefMessage struct {
	CefVersion     int    `json:"cef_version"`
	ProductVendor  string `json:"product_vendor"`
	Product        string `json:"product"`
	ProductVersion string `json:"product_version"`
	EventClass     string `json:"event_class"`
	EventName      string `json:"event_name"`
	EventSeverity  string `json:"event_severity"`
	SyslogMsg      string `json:"syslog_msg"`
	JsonMsg        string `json:"json_msg"`
}
