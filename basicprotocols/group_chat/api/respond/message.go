package respond

const (
	Version_1_0_0 string = "1.0.0"
)

type Message struct {
	Code                int         `json:"code"`
	Data                interface{} `json:"data"`
	Message             string      `json:"message"`
	ProcessingTime      int64       `json:"processingTime"`
	ProcessingTimeInMid int64       `json:"processingTimeInMid,omitempty"`
}
