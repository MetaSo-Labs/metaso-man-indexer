package respond

import "time"

const (
	HttpsCodeSuccess int = iota
	HttpsCodeError
	HttpsCodeErrorAuth
)

const (
	RespMessageSuccess string = "success"
	RespMessageError   string = "error"
)

type Message struct {
	Code                int         `json:"code"`
	Data                interface{} `json:"data"`
	Message             string      `json:"message"`
	ProcessingTime      int64       `json:"processingTime"`
	ProcessingTimeInMid int64       `json:"processingTimeInMid,omitempty"`
}

func RespSuccess(data interface{}, timestamp int64) Message {
	return Message{
		Code:           HttpsCodeSuccess,
		Message:        RespMessageSuccess,
		ProcessingTime: time.Now().UnixMilli() - timestamp,
		Data:           data,
	}
}

func RespErr(err error, timestamp int64, code int) Message {
	if code == 0 {
		code = HttpsCodeError
	}
	return Message{
		Code:           code,
		Message:        err.Error(),
		ProcessingTime: time.Now().UnixMilli() - timestamp,
		Data:           nil,
	}
}

// MaxIndexResponse represents the maximum index response for chat collections
type MaxIndexResponse struct {
	GroupId    string `json:"groupId,omitempty"`    // Group ID (for group chat)
	FromMetaId string `json:"fromMetaId,omitempty"` // From MetaId (for private chat)
	ToMetaId   string `json:"toMetaId,omitempty"`   // To MetaId (for private chat)
	MaxIndex   int64  `json:"maxIndex"`             // Current maximum index
}
