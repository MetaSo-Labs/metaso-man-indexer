package protocols

// extra protocol

type SimpleGroupLuckyBagExtra struct {
	SubId      string      `json:"subId"`
	GroupId    string      `json:"groupId"`
	ChannelId  string      `json:"channelId"`
	Code       string      `json:"code"`
	CreateTime interface{} `json:"createTime"`

	// PinId           string `json:"pinId"`
	Domain          string `json:"domain"`
	LuckyBagAddress string `json:"luckyBagAddress"`
	Type            string `json:"type"`
}
