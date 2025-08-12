package models

type TalkCommunityModel struct {
	CommunityId string   `json:"communityId"` //社区Id: hash(metaname) 唯一
	TxId        string   `json:"txId"`        //
	PinId       string   `json:"pinId"`       //
	MetaId      string   `json:"metaId"`
	Address     string   `json:"address"`
	PublicKey   string   `json:"publicKey"`
	Name        string   `json:"name"`        //社区名称
	Description string   `json:"description"` //社区描述
	Cover       string   `json:"cover"`       //封面
	Icon        string   `json:"icon"`        //图标
	MetaName    string   `json:"metaName"`    //meta域名
	MetaNameNft string   `json:"metaNameNft"` //codehash/genesis/tokenIndex
	Admins      []string `json:"admins"`      //社区管理者
	Reserved    string   `json:"reserved"`    //用生成节点的metaId用户的00私钥对metaName进行签名
	// ValidState   ValidState `json:"validState"`   //验证是否有效
	Chain        string `json:"chain"` //链类型
	BlockHeight  int64  `json:"blockHeight"`
	ConfirmState int64  `json:"confirmState"`
	Timestamp    int64  `json:"timestamp"` //
}

type TalkCommunityJoinModel struct {
	TxId           string    `json:"txId"` //
	PinId          string    `json:"pinId"`
	MetaId         string    `json:"metaId"`
	ZeroAddress    string    `json:"zeroAddress"`
	Address        string    `json:"address"`
	PublicKey      string    `json:"publicKey"`
	CommunityId    string    `json:"communityId"` //社区Id: hash(metaname)
	CommunityState RoomState `json:"communityState"`
	// IsValid        bool      `json:"isValid"`
	// IsNew          bool      `json:"isNew"`
	Chain        string `json:"chain"` //链类型
	BlockHeight  int64  `json:"blockHeight"`
	ConfirmState int64  `json:"confirmState"`
	Timestamp    int64  `json:"timestamp"` //
}

type TalkCommunityInfo struct {
	CommunityId         string `json:"communityId"` //房间ID 唯一
	PersonTotal         uint64 `json:"personTotal"`
	ChatTotal           uint64 `json:"chatTotal"`
	ChatTotalUpdateTime int64  `json:"chatTotalUpdateTime"`
	Timestamp           int64  `json:"timestamp"` //时间戳
}

type TalkCommunityPerson struct {
	CommunityIdMetaIdHash string    `json:"communityIdMetaIdHash"` //房间ID与成员 唯一
	CommunityId           string    `json:"communityId"`           //房间ID 唯一
	MetaId                string    `json:"metaId"`
	AvatarTxId            string    `json:"avatarTxId"`
	UserName              string    `json:"userName"`
	UserNickName          string    `json:"userNickName"`
	CommunityState        RoomState `json:"communityState"`
	Timestamp             int64     `json:"timestamp"` //加入或离开房间的时间戳
}

type TalkGroupJoinModel struct {
	TxId        string `json:"txId"`  //
	PinId       string `json:"pinId"` //
	MetaId      string `json:"metaId"`
	ZeroAddress string `json:"zeroAddress"`
	Address     string `json:"address"`
	// PublicKey    string    `json:"publicKey"`
	GroupId    string    `json:"groupId"` //群组Id: hash(metaname)
	GroupState RoomState `json:"groupState"`
	Referrer   string    `json:"referrer"` //推荐人
	Chain      string    `json:"chain"`    //链类型
	// IsValid      bool      `json:"isValid"`
	// IsNew        bool      `json:"isNew"`
	BlockHeight  int64 `json:"blockHeight"`
	ConfirmState int64 `json:"confirmState"`
	Timestamp    int64 `json:"timestamp"` //
}

type TalkGroupPerson struct {
	GroupIdMetaIdHash string    `json:"groupIdMetaIdHash"` //群组ID与成员 唯一
	GroupId           string    `json:"groupId"`           //群组ID 唯一
	MetaId            string    `json:"metaId"`
	Address           string    `json:"address"`
	AvatarTxId        string    `json:"avatarTxId"`
	UserName          string    `json:"userName"`
	UserNickName      string    `json:"userNickName"`
	GroupState        RoomState `json:"groupState"`
	Timestamp         int64     `json:"timestamp"`   //加入或离开群组的时间戳
	BlockHeight       int64     `json:"blockHeight"` //区块高度
	PinId             string    `json:"pinId"`       //加入或离开群组的PinId
}

type TalkGroupModel struct {
	GroupId       string `json:"groupId"`       //房间ID 唯一
	CommunityId   string `json:"communityId"`   //社区Id 唯一
	TxId          string `json:"txId"`          //房间的TxId
	PinId         string `json:"pinId"`         //房间的PinId
	RoomPublicKey string `json:"roomPublicKey"` //房间公钥
	RoomName      string `json:"roomName"`      //创建房间的名称
	RoomNote      string `json:"roomNote"`      //创建房间的公告
	RoomType      string `json:"roomType"`      //创建房间的类型 "1"不加密 "2"加密 加密采用AES加密算法
	RoomStatus    string `json:"roomStatus"`    //"1" 未加密时为"1" 加密时为加密后的信息, 保留字段
	RoomJoinType  string `json:"roomJoinType"`  //加入方式，1为密码，2为nft
	// RoomCodeHash          string `json:"roomCodeHash"`          //roomJoinType为2时有值，codeHash
	// RoomGenesis           string `json:"roomGenesis"`           //roomJoinType为2时有值，genesis
	// RoomLimitAmount       int64  `json:"roomLimitAmount"`       //roomJoinType为2时有值，token的限制
	// RoomGenesisSeriesName string `json:"roomGenesisSeriesName"` //roomJoinType为2时有值，genesis
	RoomAvatarUrl string `json:"roomAvatarUrl"` //房间头像url
	// RoomNinePersonHash string `json:"roomNinePersonHash"` //房间前9位人员的metaId总hash值
	// RoomNewestTxId        string `json:"roomNewestTxId"`        //房间最新聊天内容的txId
	// RoomNewestMetaId      string `json:"roomNewestMetaId"`      //房间最新聊天内容的MetaId
	// RoomNewestProtocol    string `json:"roomNewestProtocol"`    //房间最新聊天内容的协议类型
	// RoomNewestContent     string `json:"roomNewestContent"`     //房间最新聊天内容
	// RoomNewestTimestamp   int64  `json:"roomNewestTimestamp"`   //房间最新聊天的时间戳
	CreateUserMetaId  string `json:"createUserMetaId"`  //创建人的metaId
	CreateUserAddress string `json:"createUserAddress"` //创建人的address
	ChatSettingType   int64  `json:"chatSettingType"`   //用于设置发言限制， 0-所有人，1-管理员
	// ValidState            ValidState `json:"validState"`                       //验证是否有效
	Chain        string `json:"chain"`        //链类型
	DeleteStatus int64  `json:"deleteStatus"` //删除状态，0-正常，1-删除
	Timestamp    int64  `json:"timestamp"`    //创建你房间的时间戳
	BlockHeight  int64  `json:"blockHeight"`  //区块高度
}

type TalkGroupTxV3 struct {
	TxId          string `json:"txId"`          //房间的TxId
	MetaId        string `json:"metaId"`        //metaId
	GroupId       string `json:"groupId"`       //房间ID 唯一
	CommunityId   string `json:"communityId"`   //社区Id 唯一
	RoomPublicKey string `json:"roomPublicKey"` //房间公钥
	RoomName      string `json:"roomName"`      //创建房间的名称
	RoomNote      string `json:"roomNote"`      //创建房间的公告
	RoomType      string `json:"roomType"`      //创建房间的类型 "1"不加密 "2"加密 加密采用AES加密算法
	RoomStatus    string `json:"roomStatus"`    //"1" 未加密时为"1" 加密时为加密后的信息, 保留字段
	RoomJoinType  string `json:"roomJoinType"`  //加入方式，1为密码，2为nft
	// RoomCodeHash          string `json:"roomCodeHash"`          //roomJoinType为2时有值，codeHash
	// RoomGenesis           string `json:"roomGenesis"`           //roomJoinType为2时有值，genesis
	// RoomLimitAmount       int64  `json:"roomLimitAmount"`       //roomJoinType为2时有值，token的限制
	// RoomGenesisSeriesName string `json:"roomGenesisSeriesName"` //roomJoinType为2时有值，genesis
	ChatSettingType int64 `json:"chatSettingType"` //用于设置发言限制， 0-所有人，1-管理员
	// ValidState            ValidState `json:"validState"`                       //验证是否有效
	DeleteStatus int64 `json:"deleteStatus"` //删除状态，0-正常，1-删除
	// IsValid      bool  `json:"isValid"`
	// IsNew        bool  `json:"isNew"`
	BlockHeight  int64  `json:"blockHeight"`
	ConfirmState int64  `json:"confirmState"`
	Timestamp    int64  `json:"timestamp"` //创建你房间的时间戳
	Chain        string `json:"chain"`     //链类型
}

type ChatInsideIndex int64

const (
	ChatInsideIndexIn  ChatInsideIndex = 0 //0-in
	ChatInsideIndexOut ChatInsideIndex = 1 //1-out
)

type TalkGroupChatV3 struct {
	CommunityId string          `json:"communityId"` //社区Id 唯一
	GroupId     string          `json:"groupId"`     //房间ID 唯一
	TxId        string          `json:"txId"`
	PinId       string          `json:"pinId"` //
	MetaId      string          `json:"metaId"`
	Address     string          `json:"address"`
	AvatarTxId  string          `json:"avatarTxId"`
	NickName    string          `json:"nickName"`
	Protocol    string          `json:"protocol"`
	Content     string          `json:"content"`
	ContentType string          `json:"contentType"`
	Encryption  string          `json:"encryption"`
	ChatType    ChatType        `json:"chatType"`    //0-msg, 1-red, 2-img
	InsideIndex ChatInsideIndex `json:"insideIndex"` //0-in, 1-out
	ReplyPin    string          `json:"replyPin"`
	ReplyInfo   *ReplyInfo      `json:"replyInfo"`
	Timestamp   int64           `json:"timestamp"`   //聊天记录时间戳
	Chain       string          `json:"chain"`       //链类型
	BlockHeight int64           `json:"blockHeight"` //区块高度
}

type ReplyInfo struct {
	TxId        string          `json:"txId"`
	MetaId      string          `json:"metaId"`
	NickName    string          `json:"nickName"`
	Protocol    string          `json:"protocol"`
	Content     string          `json:"content"`
	ContentType string          `json:"contentType"`
	Encryption  string          `json:"encryption"`
	ChatType    ChatType        `json:"chatType"`    //0-msg, 1-red, 2-img
	InsideIndex ChatInsideIndex `json:"insideIndex"` //0-in, 1-out
	Timestamp   int64           `json:"timestamp"`   //聊天记录时间戳
}

type TalkGroupRedEnvelopeV3 struct {
	CommunityId     string            `json:"communityId"` //房间ID 唯一
	GroupId         string            `json:"groupId"`     //频道ID 唯一
	TxId            string            `json:"txId"`
	PinId           string            `json:"pinId"` //
	MetaId          string            `json:"metaId"`
	Protocol        string            `json:"protocol"`
	SubId           string            `json:"subId"`
	Code            string            `json:"code"`
	CreateTimeStr   string            `json:"createTimeStr"`
	Content         string            `json:"content"`
	Img             string            `json:"img"`
	ImgType         string            `json:"imgType"`
	Amount          string            `json:"amount"`
	Count           string            `json:"count"`
	PayList         []*ProInfoPayList `json:"payList"`
	RedVouts        []*RedOutput      `json:"redVouts"`
	Type            string            `json:"type"`
	FtGenesis       string            `json:"ftGenesis"`
	FtSensibleId    string            `json:"ftSensibleId"`
	FtCodehash      string            `json:"ftCodehash"`
	FtDecimalNum    string            `json:"ftDecimalNum"`
	FtIcon          string            `json:"ftIcon"`
	FtSymbol        string            `json:"ftSymbol"`
	FtName          string            `json:"ftName"`
	RequireType     string            `json:"requireType"`
	RequireCodehash string            `json:"requireCodehash"`
	RequireGenesis  string            `json:"requireGenesis"`
	LimitAmount     uint64            `json:"limitAmount"`
	Timestamp       int64             `json:"timestamp"` //聊天记录时间戳
}
type ProInfoPayList struct {
	Amount  string `json:"amount"`
	Address string `json:"address"`
	Index   int64  `json:"index"`
}
type RedOutput struct {
	ScriptPubKey string `json:"scriptPubKey"`
	Amount       uint64 `json:"amount"`
	Address      string `json:"address"`
	Index        int64  `json:"index"`
}

type TalkGroupOpenRedEnvelopeV3 struct {
	CommunityId       string  `json:"communityId"` //房间ID 唯一
	GroupId           string  `json:"groupId"`     //频道ID 唯一
	TxId              string  `json:"txId"`
	PinId             string  `json:"pinId"` //
	MetaId            string  `json:"metaId"`
	Protocol          string  `json:"protocol"`
	SubId             string  `json:"subId"`
	Code              string  `json:"code"`
	CreateTimeStr     string  `json:"createTimeStr"`
	Address           string  `json:"address"`
	Index             int64   `json:"index"`
	Amount            string  `json:"amount"`
	Vins              []*TxIn `json:"vins"`
	Type              string  `json:"type"`
	FtGenesis         string  `json:"ftGenesis"`
	FtSensibleId      string  `json:"ftSensibleId"`
	FtCodehash        string  `json:"ftCodehash"`
	FtDecimalNum      string  `json:"ftDecimalNum"`
	FtIcon            string  `json:"ftIcon"`
	FtSymbol          string  `json:"ftSymbol"`
	FtName            string  `json:"ftName"`
	RedEnvelopeTxId   string  `json:"redEnvelopeTxId"`
	RedEnvelopeMetaId string  `json:"redEnvelopeMetaId"`
	OpenNftCodehash   string  `json:"openNftCodehash"`
	OpenNftGenesis    string  `json:"openNftGenesis"`
	OpenNftTokenIndex string  `json:"openNftTokenIndex"`
	IsWithdraw        bool    `json:"isWithdraw"`
	Timestamp         int64   `json:"timestamp"` //聊天记录时间戳
}
type TxIn struct {
	OutTxID string `json:"outTxId"` //所在out-txId
	Index   uint64 `json:"index"`
	//Value       uint64 `json:"value" bson:"value"`
	//Address     string `json:"address" bson:"address"`
	//PublicKey   string `json:"publicKey" bson:"publicKey"`
	//OutTxScript string `json:"outTxScript" bson:"outTxScript"`
}

type TalkGroupResidueRedEnvelopeV3 struct {
	CommunityId       string            `json:"communityId"` //房间ID 唯一
	GroupId           string            `json:"groupId"`     //频道ID 唯一
	TxId              string            `json:"txId"`
	PinId             string            `json:"pinId"` //
	MetaId            string            `json:"metaId"`
	Protocol          string            `json:"protocol"`
	SubId             string            `json:"subId"`
	Code              string            `json:"code"`
	CreateTimeStr     string            `json:"createTimeStr"`
	UsedList          []*ProInfoPayList `json:"usedList"`
	Vins              []*TxIn           `json:"vins"`
	Type              string            `json:"type"`
	FtGenesis         string            `json:"ftGenesis"`
	FtSensibleId      string            `json:"ftSensibleId"`
	FtCodehash        string            `json:"ftCodehash"`
	FtDecimalNum      string            `json:"ftDecimalNum"`
	FtIcon            string            `json:"ftIcon"`
	FtSymbol          string            `json:"ftSymbol"`
	FtName            string            `json:"ftName"`
	RedEnvelopeTxId   string            `json:"redEnvelopeTxId"`
	RedEnvelopeMetaId string            `json:"redEnvelopeMetaId"`
	Timestamp         int64             `json:"timestamp"` //聊天记录时间戳
}

// 用户群列表项 + 用户私聊列表项
type MetaIdContextItem struct {
	GroupId          string   `json:"groupId"`          // 群组ID
	MetaId           string   `json:"metaId"`           // 消息创建者的MetaId
	Type             string   `json:"type"`             // 类型，1-群聊，2-私聊
	Timestamp        int64    `json:"timestamp"`        // 最新消息时间戳
	ChatType         ChatType `json:"chatType"`         // 消息类型
	Content          string   `json:"content"`          // 消息内容摘要
	CreateMetaId     string   `json:"createMetaId"`     // 消息创建者的MetaId
	CreateAddress    string   `json:"createAddress"`    // 消息创建者地址
	LastMessagePinId string   `json:"lastMessagePinId"` // 最新消息的PinId
	BlockHeight      int64    `json:"blockHeight"`      // 区块高度
}

// 用户群列表
type MetaIdContextList struct {
	MetaId string               `json:"metaId"` // 用户MetaId
	Items  []*MetaIdContextItem `json:"items"`  // 群列表项
}

// 群组最新聊天记录
type TalkGroupLatestChat struct {
	GroupId          string   `json:"groupId"`          // 群组ID
	Timestamp        int64    `json:"timestamp"`        // 最新消息时间戳
	ChatType         ChatType `json:"chatType"`         // 消息类型
	Content          string   `json:"content"`          // 消息内容摘要
	CreateAddress    string   `json:"createAddress"`    // 消息创建者地址
	LastMessagePinId string   `json:"lastMessagePinId"` // 最新消息的PinId
	MetaId           string   `json:"metaId"`           // 消息创建者的MetaId
	TxId             string   `json:"txId"`             // 消息的TxId
	PinId            string   `json:"pinId"`            // 消息的PinId
	Protocol         string   `json:"protocol"`         // 协议类型
	ContentType      string   `json:"contentType"`      // 内容类型
	Encryption       string   `json:"encryption"`       // 加密信息
	ReplyPin         string   `json:"replyPin"`         // 回复消息的PinId
	Chain            string   `json:"chain"`            // 链类型
	BlockHeight      int64    `json:"blockHeight"`      // 区块高度
}

// 私聊消息模型
type TalkPrivateChatV3 struct {
	From        string     `json:"from"`        // 发送者MetaId
	FromAddress string     `json:"fromAddress"` // 发送者地址
	To          string     `json:"to"`          // 接收者MetaId
	ToAddress   string     `json:"toAddress"`   // 接收者地址
	TxId        string     `json:"txId"`
	PinId       string     `json:"pinId"` //
	Protocol    string     `json:"protocol"`
	Content     string     `json:"content"`
	ContentType string     `json:"contentType"`
	Encryption  string     `json:"encryption"`
	ChatType    ChatType   `json:"chatType"` //0-msg, 1-red, 3-img
	ReplyPin    string     `json:"replyPin"`
	ReplyInfo   *ReplyInfo `json:"replyInfo"`
	Timestamp   int64      `json:"timestamp"`   //聊天记录时间戳
	Chain       string     `json:"chain"`       //链类型
	BlockHeight int64      `json:"blockHeight"` //区块高度
}
