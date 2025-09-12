package protocols

/*
*

	{
	  "communityId": "hash(metaname)",
	  "name": "Community name",
	  "description": "Community description",
	  "icon": "https://example.com/icon.png",
	  "cover": "https://example.com/cover.png",
	  "metaName": "metaname",
	  "metaNameNft": "meta/codehash/genesis/tokenIndex",
	  "admins": ["metaId1", "metaId2"],
	  "reserved": "ECDH signature"
	}

*
*/
type SimpleCommunity struct {
	CommunityId string   `json:"communityId"` //{Community ID: hash(metaname) }
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Icon        string   `json:"icon"`
	Cover       string   `json:"cover"`
	MetaName    string   `json:"metaName"`
	MetaNameNft string   `json:"metaNameNft"` //meta/codehash/genesis/tokenIndex
	Admins      []string `json:"admins"`
	Reserved    string   `json:"reserved"` //Show signature, use 00 private key of metaId user who generates node to sign metaName - ECDH
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
	CommunityId string      `json:"communityId"` //{Community ID: hash(metaname) }
	State       interface{} `json:"state"`       //Join state: 1-join, -1-leave
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
	GroupId  string      `json:"groupId"`  //{Group ID: hash(metaname) }
	State    interface{} `json:"state"`    //Join state: 1-join, -1-leave
	Referrer string      `json:"referrer"` //Referrer
}

/*
*

	{
	  "groupId": "group123",
	  "communityId": "hash(metaname)",
	  "groupName": "Group name",
	  "groupNote": "Group announcement",
	  "timestamp": 1234567890,
	  "groupType": "1",
	  "status": "1",
	  "type": "1",
	  "tickId": "",
	  "collectionId": "",
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
	GroupIcon       string      `json:"groupIcon"`
	Timestamp       interface{} `json:"timestamp"`
	GroupType       interface{} `json:"groupType"`    //Room creation type "1" not encrypted "2" encrypted encryption using AES encryption algorithm
	Status          interface{} `json:"status"`       //"1" when not encrypted is "1", when encrypted is encrypted information, reserved field
	JoinType        interface{} `json:"type"`         //Join method, 1 is password, 2 is nft, 3-FT limit
	TickId          string      `json:"tickId"`       //FT-limit requires temporarily mrc20
	CollectionId    string      `json:"collectionId"` //NFT-limit requires temporarily mrc721
	LimitAmount     interface{} `json:"limitAmount"`
	ChatSettingType interface{} `json:"chatSettingType"` //Used to set speech restrictions, 0-everyone, 1-administrators
	DeleteStatus    interface{} `json:"deleteStatus"`    //Delete status, 0-normal, 1-deleted
}

/*
*

	{
	  "groupId": "group123",
	  "nickName": "User nickname",
	  "content": "Chat content",
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
	ReplyPin    string      `json:"replyPin"`
}

/*
*

	{
	  "encrypt": "0",
	  "attachment": "metafile://pinId.jpg",
	  "groupId": "group123",
	//   "channelId": "channel123",
	  "fileType": "png/jpg/gif",
	  "nickName": "User nickname",
	  "timestamp": 1234567890,
	  "replyTx": "txId"
	}

*
*/
type SimpleFileGroupChat struct {
	Encrypt    string `json:"encrypt"`    //Whether to encrypt and encryption method, 0 for no encryption; 1 for AES encryption. Default no encryption
	Attachment string `json:"attachment"` //metafile://pinId.jpg
	GroupId    string `json:"groupId"`
	// ChannelId  string      `json:"channelId"`
	FileType  string      `json:"fileType"` //png/jpg/gif
	NickName  string      `json:"nickName"`
	Timestamp interface{} `json:"timestamp"`
	ReplyPin  string      `json:"replyPin"`
}

/*
*

	{
	  "subId": "red123",
	  "groupId": "group123",
	  "code": "redcode",
	  "createTime": 1234567890,
	  "content": "Congratulations and prosperity",
	  "domain": "https://www.example.com/chat-api/",
	  "luckyBagAddress": "luckyBagAddress",
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
	  "requireType": 0,
	  "requireTickId": "tickId",
	  "requireCollectionId": "collectionId",
	  "limitAmount": 100
	}

*
*/
type SimpleGroupLuckyBag struct {
	SubId           string      `json:"subId"`
	GroupId         string      `json:"groupId"`
	Code            string      `json:"code"`
	Domain          string      `json:"domain"`
	LuckyBagAddress string      `json:"luckyBagAddress"`
	CreateTime      interface{} `json:"createTime"`
	Content         string      `json:"content"`
	Img             string      `json:"img"`
	ImgType         string      `json:"imgType"`
	Amount          interface{} `json:"amount"`  //Amount
	FeeRate         interface{} `json:"feeRate"` //Fee rate， default 1.1
	// LuckyTotalAmount    interface{}       `json:"luckyTotalAmount"` //
	// LuckyTotalFee       interface{}       `json:"luckyTotalFee"`    //
	Count               interface{}       `json:"count"`
	PayList             []*ProInfoPayList `json:"payList"`
	Type                string            `json:"type"`
	RequireType         interface{}       `json:"requireType"`         //0-no limit, 1-FT, 2-NFT
	RequireTickId       string            `json:"requireTickId"`       //FT-limit requires temporarily mrc20
	RequireCollectionId string            `json:"requireCollectionId"` //NFT-limit requires temporarily mrc721
	LimitAmount         interface{}       `json:"limitAmount"`
}

var (
	OpenLuckyTxSize int64 = 210 //Open lucky bag tx size
)

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
	Amount  interface{} `json:"amount"` //Amount = LuckyAmount + LuckyFee
	Address string      `json:"address"`
	Index   interface{} `json:"index"`
	// LuckyAmount interface{} `json:"luckyAmount"` //luckyAmount >= 800 satoshi
	// LuckyFee    interface{} `json:"luckyFee"`
}

/*
*

	{
	  "luckyBagTxId": "tx123",
	  "luckyBagPinId": "pin123",
	  "luckyBagMetaId": "metaId123",
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
	  "isWithdraw": false
	}

*
*/
type SimpleGroupOpenLuckyBag struct {
	LuckyBagTxId   string      `json:"luckyBagTxId"`
	LuckyBagPinId  string      `json:"luckyBagPinId"`
	LuckyBagMetaId string      `json:"luckyBagMetaId"`
	SubId          string      `json:"subId"`
	GroupId        string      `json:"groupId"`
	Code           string      `json:"code"`
	CreateTime     interface{} `json:"createTime"`
	Used           *ProUsed    `json:"used"`
	Type           string      `json:"type"`
	IsWithdraw     interface{} `json:"isWithdraw"`
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
	  "luckyBagTxId": "tx123",
	  "luckyBagPinId": "pin123",
	  "luckyBagMetaId": "metaId123",
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
type SimpleGroupResidueLuckyBag struct {
	LuckyBagTxId   string      `json:"luckyBagTxId"`
	LuckyBagPinId  string      `json:"luckyBagPinId"`
	LuckyBagMetaId string      `json:"luckyBagMetaId"`
	SubId          string      `json:"subId"`
	GroupId        string      `json:"groupId"`
	Code           string      `json:"code"`
	CreateTime     interface{} `json:"createTime"`
	Used           []*ProUsed  `json:"used"`
	Type           string      `json:"type"`
}

/*
*

	{
	"removeMetaid": "{the metaid of user that will be removed}",
	"groupId": "{the groupId}",
	"reason":"{why ban user}",
	"timestamp":0
	}

*
*/
type SimpleGroupRemoveUser struct {
	RemoveMetaid string      `json:"removeMetaid"`
	GroupId      string      `json:"groupId"`
	Reason       string      `json:"reason"`
	Timestamp    interface{} `json:"timestamp"`
}

//private chat
/*
SimpleMsg
{
	"to": "{metaid}"
	"encrypt": "",
	"content": "",
	"contentType": "",
	"timestamp": 0,
	"replyPin": "{pinId}"
}
*/
type SimpleMsg struct {
	To          string      `json:"to"`
	Encrypt     string      `json:"encrypt"`
	Content     string      `json:"content"`
	ContentType string      `json:"contentType"`
	Timestamp   interface{} `json:"timestamp"`
	ReplyPin    string      `json:"replyPin"`
}

/*
*
SimpleFileMsg

	{
		"to": "{metaid}"
		"encrypt": "",
		"attachment": "metafile://pinId.jpg",
		"fileType": "png/jpg/doc/pdf/excel",
		"timestamp": 0,
		"replyPin": "{pinId}"
	}

*
*/
type SimpleFileMsg struct {
	To         string      `json:"to"`
	Encrypt    string      `json:"encrypt"`
	Attachment string      `json:"attachment"`
	FileType   string      `json:"fileType"`
	Timestamp  interface{} `json:"timestamp"`
	ReplyPin   string      `json:"replyPin"`
}

/*
*
SimplePrivateBlock

	{
		"to": "{metaid}"
		"blockState": 1 //1:拉黑，-1:取消拉黑
	}
*/
type SimplePrivateBlock struct {
	To         string      `json:"to"`
	BlockState interface{} `json:"blockState"`
}

// Protocol constant definitions
const (
	MonitorSimpleCommunity            = "SimpleCommunity"
	MonitorSimpleCommunityJoin        = "SimpleCommunityJoin"
	MonitorSimpleGroupCreate          = "SimpleGroupCreate"
	MonitorSimpleGroupJoin            = "SimpleGroupJoin"
	MonitorSimpleGroupChat            = "SimpleGroupChat"
	MonitorSimpleFileGroupChat        = "SimpleFileGroupChat"
	MonitorSimpleGroupLuckyBag        = "SimpleGroupLuckyBag"
	MonitorSimpleGroupOpenLuckyBag    = "SimpleGroupOpenLuckyBag"
	MonitorSimpleGroupResidueLuckyBag = "SimpleGroupResidueLuckyBag"
	MonitorSimpleGroupRemoveUser      = "SimpleGroupRemoveUser"

	MonitorSimpleMsg          = "SimpleMsg"
	MonitorSimpleFileMsg      = "SimpleFileMsg"
	MonitorSimplePrivateBlock = "SimpleBlock"
)

// info
const (
	MonitorInfoChatpubkey = "chatpubkey"
)
