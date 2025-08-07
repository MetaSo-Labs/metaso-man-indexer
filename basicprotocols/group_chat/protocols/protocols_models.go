package protocols

/*
*

	{
	  "communityId": "hash(metaname)",
	  "name": "社区名称",
	  "description": "社区描述",
	  "icon": "https://example.com/icon.png",
	  "cover": "https://example.com/cover.png",
	  "metaName": "metaname",
	  "metaNameNft": "meta/codehash/genesis/tokenIndex",
	  "admins": ["metaId1", "metaId2"],
	  "reserved": "ECDH签名"
	}

*
*/
type SimpleCommunity struct {
	CommunityId string   `json:"communityId"` //{社区Id: hash(metaname) }
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Icon        string   `json:"icon"`
	Cover       string   `json:"cover"`
	MetaName    string   `json:"metaName"`
	MetaNameNft string   `json:"metaNameNft"` //meta/codehash/genesis/tokenIndex
	Admins      []string `json:"admins"`
	Reserved    string   `json:"reserved"` //show签名，用生成节点的metaId用户的00私钥对metaName进行签名 - ECDH
}

/*
*

	{
	  "communityId": "hash(metaname)",
	  "state": 1
	}

*
*/
type SimpleCommunityJoin struct {
	CommunityId string      `json:"communityId"` //{社区Id: hash(metaname) }
	State       interface{} `json:"state"`       //加入状态：1-加入，-1-离开
}

/*
*

	{
	  "groupId": "group123",
	  "state": 1
	}

*
*/
type SimpleGroupJoin struct {
	GroupId  string      `json:"groupId"`  //{群组Id: hash(metaname) }
	State    interface{} `json:"state"`    //加入状态：1-加入，-1-离开
	Referrer string      `json:"referrer"` //推荐人
}

/*
*

	{
	  "groupId": "group123",
	  "communityId": "hash(metaname)",
	  "groupName": "群组名称",
	  "groupNote": "群组公告",
	  "timestamp": 1234567890,
	  "groupType": "1",
	  "status": "1",
	  "type": "1",
	  "codehash": "codehash",
	  "genesis": "genesis",
	  "limitAmount": 100,
	  "chatSettingType": 0,
	  "deleteStatus": 0
	}

*
*/
type SimpleGroupCreate struct {
	GroupId         string      `json:"groupId"`
	CommunityId     string      `json:"communityId"`
	GroupName       string      `json:"groupName"`
	GroupNote       string      `json:"groupNote"`
	Timestamp       interface{} `json:"timestamp"`
	GroupType       interface{} `json:"groupType"`
	Status          interface{} `json:"status"`
	JoinType        interface{} `json:"type"`
	CodeHash        string      `json:"codehash"`
	Genesis         string      `json:"genesis"`
	LimitAmount     interface{} `json:"limitAmount"`
	ChatSettingType interface{} `json:"chatSettingType"` //用于设置发言限制， 0-所有人，1-管理员
	DeleteStatus    interface{} `json:"deleteStatus"`    //删除状态，0-正常，1-删除
}

/*
*

	{
	  "groupId": "group123",
	  "nickName": "用户昵称",
	  "content": "聊天内容",
	  "contentType": "text",
	  "encryption": "none",
	  "timestamp": 1234567890,
	  "replyTx": "txId"
	}

*
*/
type SimpleGroupChat struct {
	GroupId     string      `json:"groupId"`
	NickName    string      `json:"nickName"`
	Content     string      `json:"content"`
	ContentType string      `json:"contentType"`
	Encryption  string      `json:"encryption"`
	Timestamp   interface{} `json:"timestamp"`
	ReplyTx     string      `json:"replyTx"`
}

/*
*

	{
	  "encrypt": "0",
	  "attachment": "metafile://txId.jpg",
	  "groupId": "group123",
	  "channelId": "channel123",
	  "fileType": "png/jpg/gif",
	  "nickName": "用户昵称",
	  "timestamp": 1234567890,
	  "replyTx": "txId"
	}

*
*/
type SimpleFileGroupChat struct {
	Encrypt    string      `json:"encrypt"`    //是否加密和加密方式,0 为不加密；1为采用AES加密。默认不加密
	Attachment string      `json:"attachment"` //metafile://txId.jpg
	GroupId    string      `json:"groupId"`
	ChannelId  string      `json:"channelId"`
	FileType   string      `json:"fileType"` //png/jpg/gif
	NickName   string      `json:"nickName"`
	Timestamp  interface{} `json:"timestamp"`
	ReplyTx    string      `json:"replyTx"`
}

/*
*

	{
	  "subId": "red123",
	  "groupId": "group123",
	  "code": "redcode",
	  "createTime": 1234567890,
	  "content": "恭喜发财",
	  "img": "https://example.com/red.png",
	  "imgType": "png",
	  "amount": "100",
	  "count": "10",
	  "payList": [
	    {
	      "amount": "10",
	      "address": "address1",
	      "index": 0
	    }
	  ],
	  "type": "btc",
	  "ftGenesis": "genesis",
	  "ftSensibleId": "sensibleId",
	  "ftCodehash": "codehash",
	  "ftDecimalNum": 8,
	  "ftIcon": "icon",
	  "ftSymbol": "BTC",
	  "ftName": "Bitcoin",
	  "requireType": 0,
	  "requireCodehash": "codehash",
	  "requireGenesis": "genesis",
	  "limitAmount": 100
	}

*
*/
type SimpleGroupRedEnvelope struct {
	SubId           string            `json:"subId"`
	GroupId         string            `json:"groupId"`
	Code            string            `json:"code"`
	CreateTime      interface{}       `json:"createTime"`
	Content         string            `json:"content"`
	Img             string            `json:"img"`
	ImgType         string            `json:"imgType"`
	Amount          interface{}       `json:"amount"`
	Count           interface{}       `json:"count"`
	PayList         []*ProInfoPayList `json:"payList"`
	Type            string            `json:"type"`
	FtGenesis       string            `json:"ftGenesis"`
	FtSensibleId    string            `json:"ftSensibleId"`
	FtCodehash      string            `json:"ftCodehash"`
	FtDecimalNum    interface{}       `json:"ftDecimalNum"`
	FtIcon          string            `json:"ftIcon"`
	FtSymbol        string            `json:"ftSymbol"`
	FtName          string            `json:"ftName"`
	RequireType     interface{}       `json:"requireType"`
	RequireCodehash string            `json:"requireCodehash"`
	RequireGenesis  string            `json:"requireGenesis"`
	LimitAmount     interface{}       `json:"limitAmount"`
}

/*
*

	{
	  "amount": "10",
	  "address": "address1",
	  "index": 0
	}

*
*/
type ProInfoPayList struct {
	Amount  interface{} `json:"amount"`
	Address string      `json:"address"`
	Index   interface{} `json:"index"`
}

/*
*

	{
	  "redEnvelopeTxId": "tx123",
	  "redEnvelopeMetaId": "metaId123",
	  "subId": "red123",
	  "groupId": "group123",
	  "code": "redcode",
	  "createTime": 1234567890,
	  "used": {
	    "amount": "10",
	    "address": "address1",
	    "index": 0
	  },
	  "type": "btc",
	  "ftGenesis": "genesis",
	  "ftSensibleId": "sensibleId",
	  "ftCodehash": "codehash",
	  "ftDecimalNum": 8,
	  "ftIcon": "icon",
	  "ftSymbol": "BTC",
	  "ftName": "Bitcoin",
	  "isWithdraw": false
	}

*
*/
type SimpleGroupOpenRedEnvelope struct {
	RedEnvelopeTxId   string      `json:"redEnvelopeTxId"`
	RedEnvelopeMetaId string      `json:"redEnvelopeMetaId"`
	SubId             string      `json:"subId"`
	GroupId           string      `json:"groupId"`
	Code              string      `json:"code"`
	CreateTime        interface{} `json:"createTime"`
	Used              *ProUsed    `json:"used"`
	Type              string      `json:"type"`
	FtGenesis         string      `json:"ftGenesis"`
	FtSensibleId      string      `json:"ftSensibleId"`
	FtCodehash        string      `json:"ftCodehash"`
	FtDecimalNum      interface{} `json:"ftDecimalNum"`
	FtIcon            string      `json:"ftIcon"`
	FtSymbol          string      `json:"ftSymbol"`
	FtName            string      `json:"ftName"`
	IsWithdraw        interface{} `json:"isWithdraw"`
}

/*
*

	{
	  "amount": "10",
	  "address": "address1",
	  "index": 0
	}

*
*/
type ProUsed struct {
	Amount  interface{} `json:"amount"`
	Address string      `json:"address"`
	Index   interface{} `json:"index"`
}

/*
*

	{
	  "redEnvelopeTxId": "tx123",
	  "redEnvelopeMetaId": "metaId123",
	  "subId": "red123",
	  "groupId": "group123",
	  "code": "redcode",
	  "createTime": 1234567890,
	  "used": [
	    {
	      "amount": "10",
	      "address": "address1",
	      "index": 0
	    }
	  ],
	  "type": "btc",
	  "ftGenesis": "genesis",
	  "ftSensibleId": "sensibleId",
	  "ftCodehash": "codehash",
	  "ftDecimalNum": 8,
	  "ftIcon": "icon",
	  "ftSymbol": "BTC",
	  "ftName": "Bitcoin"
	}

*
*/
type SimpleGroupResidueRedEnvelope struct {
	RedEnvelopeTxId   string      `json:"redEnvelopeTxId"`
	RedEnvelopeMetaId string      `json:"redEnvelopeMetaId"`
	SubId             string      `json:"subId"`
	GroupId           string      `json:"groupId"`
	Code              string      `json:"code"`
	CreateTime        interface{} `json:"createTime"`
	Used              []*ProUsed  `json:"used"`
	Type              string      `json:"type"`
	FtGenesis         string      `json:"ftGenesis"`
	FtSensibleId      string      `json:"ftSensibleId"`
	FtCodehash        string      `json:"ftCodehash"`
	FtDecimalNum      interface{} `json:"ftDecimalNum"`
	FtIcon            string      `json:"ftIcon"`
	FtSymbol          string      `json:"ftSymbol"`
	FtName            string      `json:"ftName"`
}

// 协议常量定义
const (
	MonitorSimpleCommunity               = "SimpleCommunity"
	MonitorSimpleCommunityJoin           = "SimpleCommunityJoin"
	MonitorSimpleGroupCreate             = "SimpleGroupCreate"
	MonitorSimpleGroupJoin               = "SimpleGroupJoin"
	MonitorSimpleGroupChat               = "SimpleGroupChat"
	MonitorSimpleFileGroupChat           = "SimpleFileGroupChat"
	MonitorSimpleGroupRedEnvelope        = "SimpleGroupRedEnvelope"
	MonitorSimpleGroupOpenRedEnvelope    = "SimpleGroupOpenRedEnvelope"
	MonitorSimpleGroupResidueRedEnvelope = "SimpleGroupResidueRedEnvelope"
)
