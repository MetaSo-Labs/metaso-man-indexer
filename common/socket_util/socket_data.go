package socket_util

import (
	"encoding/json"
	"strings"

	"github.com/tidwall/gjson"
)

// SocketData WebSocket generic data structure
type SocketData struct {
	M string      `json:"M"`           // method
	C interface{} `json:"C"`           // code
	D interface{} `json:"D,omitempty"` // data
}

// SocketDataFromStringMsg Create SocketData from string message
func SocketDataFromStringMsg(msg string) *SocketData {
	ws := &SocketData{}
	msg = strings.Trim(msg, "\"")
	if !gjson.Valid(msg) {
		msg = strings.ReplaceAll(msg, "\\", "")
		if !gjson.Valid(msg) {
			return nil
		}
	}
	ws.M = gjson.Get(msg, "M").String()
	ws.C = gjson.Get(msg, "C").Int()
	ws.D = gjson.Get(msg, "D").String()
	return ws
}

// ToString Convert SocketData to string
func (w *SocketData) ToString() (string, error) {
	data, err := json.Marshal(w)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// Generic WebSocket method constants
const (
	// Heartbeat
	HEART_BEAT                    = "HEART_BEAT"
	WS_SERVER_NOTIFY_PRIVATE_CHAT = "WS_SERVER_NOTIFY_PRIVATE_CHAT"
	WS_SERVER_NOTIFY_GROUP_CHAT   = "WS_SERVER_NOTIFY_GROUP_CHAT"
	WS_SERVER_NOTIFY_GROUP_ROLE   = "WS_SERVER_NOTIFY_GROUP_ROLE"

	// Generic response
	WS_RESPONSE_SUCCESS = "WS_RESPONSE_SUCCESS"
	WS_RESPONSE_ERROR   = "WS_RESPONSE_ERROR"
)

// Generic WebSocket code constants
const (
	WS_CODE_HEART_BEAT      = 10
	WS_CODE_HEART_BEAT_BACK = 10
	WS_CODE_SERVER          = 0
	WS_CODE_SEND_SUCCESS    = 200
	WS_CODE_SEND_ERROR      = 400
)
