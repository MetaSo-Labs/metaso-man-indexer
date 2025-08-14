package common_service

type wsPostResponse struct {
	Code  int64       `json:"code"`
	Error string      `json:"err"`
	Data  interface{} `json:"data"`
}
