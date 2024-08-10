package syslogs

import "time"

type CefMessage struct {
	RemoteAddr     string    `json:"remote_addr"`
	CefVersion     int       `json:"cef_version"`
	ProductVendor  string    `json:"product_vendor"`
	Product        string    `json:"product"`
	ProductVersion string    `json:"product_version"`
	EventClass     string    `json:"event_class"`
	EventName      string    `json:"event_name"`
	EventSeverity  string    `json:"event_severity"`
	SyslogMsg      string    `json:"syslog_msg"`
	JsonMsg        []byte    `json:"json_msg"`
	Time           time.Time `json:"time"`
}
