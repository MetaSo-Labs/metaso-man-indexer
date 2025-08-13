package respond

import "manindexer/basicprotocols/group_chat/models"

type UserInfo struct {
	Name        string `json:"name"`
	Avatar      string `json:"avatar"`
	AvatarImage string `json:"avatarImage"`
}

type GroupResponse struct {
	Total int64        `json:"total"`
	List  []*GroupItem `json:"list"`
}

type GroupItem struct {
	CommunityId  string `json:"communityId"`  //社区Id 唯一
	GroupId      string `json:"groupId"`      //房间ID 唯一
	TxId         string `json:"txId"`         //房间的TxId
	PinId        string `json:"pinId"`        //房间的PinId
	RoomName     string `json:"roomName"`     //创建房间的名称
	RoomNote     string `json:"roomNote"`     //创建房间的公告
	RoomType     string `json:"roomType"`     //创建房间的类型 ”1“不加密 “2”加密 加密采用AES加密算法
	RoomStatus   string `json:"roomStatus"`   //"1" 未加密时为“1” 加密时为加密后的信息, 保留字段
	RoomJoinType string `json:"roomJoinType"` //加入方式，1为密码，2为nft
	// RoomCodeHash          string `json:"roomCodeHash"`          //roomJoinType为2时有值，codeHash
	// RoomGenesis           string `json:"roomGenesis"`           //roomJoinType为2时有值，genesis
	// RoomLimitAmount       int64  `json:"roomLimitAmount"`       //roomJoinType为2时有值，token的限制
	// RoomGenesisSeriesName string `json:"roomGenesisSeriesName"` //
	RoomAvatarUrl       string    `json:"roomAvatarUrl"`       //房间头像url
	RoomNinePersonHash  string    `json:"roomNinePersonHash"`  //房间前9位人员的metaId总hash值
	RoomNewestTxId      string    `json:"roomNewestTxId"`      //房间最新聊天内容的txId
	RoomNewestPinId     string    `json:"roomNewestPinId"`     //房间最新聊天内容的pinId
	RoomNewestMetaId    string    `json:"roomNewestMetaId"`    //房间最新聊天内容的MetaId
	RoomNewestUserName  string    `json:"roomNewestUserName"`  //房间最新聊天内容的MetaId
	RoomNewestProtocol  string    `json:"roomNewestProtocol"`  //房间最新聊天内容的协议类型
	RoomNewestContent   string    `json:"roomNewestContent"`   //房间最新聊天内容
	RoomNewestTimestamp int64     `json:"roomNewestTimestamp"` //房间最新聊天的时间戳
	CreateUserMetaId    string    `json:"createUserMetaId"`    //创建人的metaId
	CreateUserInfo      *UserInfo `json:"createUserInfo"`      //创建人的信息
	UserCount           int64     `json:"userCount"`           //房间人数
	ChatSettingType     int64     `json:"chatSettingType"`     //用于设置发言限制， 0-所有人，1-管理员
	DeleteStatus        int64     `json:"deleteStatus"`        //删除状态，0-正常，1-删除
	Timestamp           int64     `json:"timestamp"`           //创建你房间的时间戳
	Chain               string    `json:"chain"`               //链类型
	BlockHeight         int64     `json:"blockHeight"`         //区块高度
}

type GroupChatResponse struct {
	Total         int64            `json:"total"`
	NextTimestamp int64            `json:"nextTimestamp"`
	List          []*GroupChatItem `json:"list"`
}

type GroupChatItem struct {
	GroupId   string    `json:"groupId"`   //房间ID 唯一
	MetanetId string    `json:"metanetId"` //
	TxId      string    `json:"txId"`
	MetaId    string    `json:"metaId"`
	Address   string    `json:"address"`
	UserInfo  *UserInfo `json:"userInfo"`
	// AvatarTxId  string              `json:"avatarTxId"`
	// AvatarImage string              `json:"avatarImage"`
	// AvatarType  model.AvatarType    `json:"avatarType"`
	NickName    string          `json:"nickName"`
	Protocol    string          `json:"protocol"`
	Content     string          `json:"content"`
	ContentType string          `json:"contentType"`
	Encryption  string          `json:"encryption"`
	ChatType    models.ChatType `json:"chatType"` //0-msg, 1-red, 2-img
	Data        interface{}     `json:"data"`
	ReplyPin    string          `json:"replyPin"`
	ReplyInfo   *ReplyInfo      `json:"replyInfo"`
	RedMetaId   string          `json:"redMetaId"`
	Timestamp   int64           `json:"timestamp"`   //聊天记录时间戳
	Params      string          `json:"params"`      //通用字段，便于后续新增参数
	Chain       string          `json:"chain"`       //链类型
	BlockHeight int64           `json:"blockHeight"` //区块高度
}

type ReplyInfo struct {
	PinId       string          `json:"pinId"`
	MetaId      string          `json:"metaId"`
	Address     string          `json:"address"`
	UserInfo    *UserInfo       `json:"userInfo"`
	NickName    string          `json:"nickName"`
	Protocol    string          `json:"protocol"`
	Content     string          `json:"content"`
	ContentType string          `json:"contentType"`
	Encryption  string          `json:"encryption"`
	ChatType    models.ChatType `json:"chatType"`    //0-msg, 1-red, 2-img
	Timestamp   int64           `json:"timestamp"`   //聊天记录时间戳
	Chain       string          `json:"chain"`       //链类型
	BlockHeight int64           `json:"blockHeight"` //区块高度
}

type GroupMemberResponse struct {
	Total int64              `json:"total"`
	List  []*GroupMemberItem `json:"list"`
}

type GroupMemberItem struct {
	MetaId string `json:"metaId"`
	// Name      string    `json:"name"`
	Address   string    `json:"address"`
	UserInfo  *UserInfo `json:"userInfo"`
	TimeStr   string    `json:"timeStr"`
	Timestamp int64     `json:"timestamp"`
}

// GroupPersonResponse 群组成员信息响应
type GroupPersonResponse struct {
	IsInGroup bool             `json:"isInGroup"` // 是否在群组中
	Person    *GroupPersonItem `json:"person"`    // 成员信息，如果不在群组中则为null
}

// GroupPersonItem 群组成员信息项
type GroupPersonItem struct {
	GroupIdMetaIdHash string `json:"groupIdMetaIdHash"` // 群组ID与成员唯一标识
	GroupId           string `json:"groupId"`           // 群组ID
	MetaId            string `json:"metaId"`            // 用户MetaId
	Address           string `json:"address"`           // 用户地址
	AvatarTxId        string `json:"avatarTxId"`        // 头像TxId
	UserName          string `json:"userName"`          // 用户名
	UserNickName      string `json:"userNickName"`      // 用户昵称
	GroupState        int64  `json:"groupState"`        // 群组状态：1-在群中，-1-已离开
	Timestamp         int64  `json:"timestamp"`         // 加入或离开群组的时间戳
	BlockHeight       int64  `json:"blockHeight"`       // 区块高度
	PinId             string `json:"pinId"`             // 加入或离开群组的PinId
}

// ChatInfoResponse 最新聊天信息响应（群聊+私聊）
type ChatInfoResponse struct {
	Total int64           `json:"total"`
	List  []*ChatInfoItem `json:"list"`
}

// ChatInfoItem 聊天信息项（群聊或私聊）
type ChatInfoItem struct {
	Type             string `json:"type"`             // 类型：1-群聊，2-私聊
	GroupId          string `json:"groupId"`          // 群组ID（群聊时）
	MetaId           string `json:"metaId"`           // 对方MetaId（私聊时）
	Address          string `json:"address"`          // 对方地址（私聊时）
	Timestamp        int64  `json:"timestamp"`        // 最新消息时间戳
	ChatType         int64  `json:"chatType"`         // 消息类型 0-msg, 1-red, 2-img
	Content          string `json:"content"`          // 消息内容摘要
	CreateMetaId     string `json:"createMetaId"`     // 消息创建者的MetaId
	CreateAddress    string `json:"createAddress"`    // 消息创建者地址
	LastMessagePinId string `json:"lastMessagePinId"` // 最新消息的PinId
	BlockHeight      int64  `json:"blockHeight"`      // 区块高度
	Chain            string `json:"chain"`            // 链类型

	// 私聊特有字段
	UserInfo *UserInfo `json:"userInfo,omitempty"`

	// 群聊特有字段
	CommunityId      string `json:"communityId,omitempty"`     // 社区Id（群聊时）
	RoomName         string `json:"roomName,omitempty"`        // 房间名称（群聊时）
	RoomNote         string `json:"roomNote,omitempty"`        // 房间公告（群聊时）
	RoomType         string `json:"roomType,omitempty"`        // 房间类型（群聊时）
	RoomStatus       string `json:"roomStatus,omitempty"`      // 房间状态（群聊时）
	RoomJoinType     string `json:"roomJoinType,omitempty"`    // 加入方式（群聊时）
	RoomAvatarUrl    string `json:"roomAvatarUrl,omitempty"`   // 房间头像（群聊时）
	CreateUserMetaId string `json:"createUserMetaId"`          // 创建人MetaId（群聊时）
	UserCount        int64  `json:"userCount,omitempty"`       // 用户数量（群聊时）
	ChatSettingType  int64  `json:"chatSettingType,omitempty"` // 聊天设置类型（群聊时）
	DeleteStatus     int64  `json:"deleteStatus,omitempty"`    // 删除状态（群聊时）
}

// PrivateChatResponse 私聊记录响应
type PrivateChatResponse struct {
	Total         int64              `json:"total"`
	NextTimestamp int64              `json:"nextTimestamp"`
	List          []*PrivateChatItem `json:"list"`
}

// PrivateChatItem 私聊记录项
type PrivateChatItem struct {
	From        string      `json:"from"` // 发送者MetaId
	To          string      `json:"to"`   // 接收者MetaId
	TxId        string      `json:"txId"`
	PinId       string      `json:"pinId"`
	MetaId      string      `json:"metaId"`   // 消息创建者MetaId
	UserInfo    *UserInfo   `json:"userInfo"` // 用户信息
	NickName    string      `json:"nickName"`
	Protocol    string      `json:"protocol"`
	Content     string      `json:"content"`
	ContentType string      `json:"contentType"`
	Encryption  string      `json:"encryption"`
	ChatType    int64       `json:"chatType"` // 0-msg, 1-red, 2-img
	Data        interface{} `json:"data"`
	ReplyPin    string      `json:"replyPin"`
	ReplyInfo   *ReplyInfo  `json:"replyInfo"`
	RedMetaId   string      `json:"redMetaId"`
	Timestamp   int64       `json:"timestamp"`   // 聊天记录时间戳
	Params      string      `json:"params"`      // 通用字段，便于后续新增参数
	Chain       string      `json:"chain"`       // 链类型
	BlockHeight int64       `json:"blockHeight"` // 区块高度
}

type LuckyBagInfoResponse struct {
	TxId                string         `json:"txId"`
	MetaId              string         `json:"metaId"`
	UserInfo            *UserInfo      `json:"userInfo"`
	SubId               string         `json:"subId"`
	Code                string         `json:"code"`
	CreateTime          string         `json:"createTime"`
	Content             string         `json:"content"`
	Img                 string         `json:"img"`
	ImgType             string         `json:"imgType"`
	Amount              string         `json:"amount"`
	Count               string         `json:"count"`
	UsedCount           string         `json:"usedCount"`
	PayList             []*InfoPayList `json:"payList"`
	Type                string         `json:"type"`
	TokenCount          uint64         `json:"tokenCount"`
	RequireType         string         `json:"requireType"`
	RequireTickId       string         `json:"requireTickId"`
	RequireCollectionId string         `json:"requireCollectionId"`
	LimitAmount         uint64         `json:"limitAmount"`
}
type InfoPayList struct {
	TxId         string    `json:"txId"`
	Index        int64     `json:"index"`
	Amount       string    `json:"amount"`
	Address      string    `json:"address"`
	Used         bool      `json:"used"`
	GradTxId     string    `json:"gradTxId"`
	GradPinId    string    `json:"gradPinId"`
	GradMetaId   string    `json:"gradMetaId"`
	GradAddress  string    `json:"gradAddress"`
	UserInfo     *UserInfo `json:"userInfo"`
	Timestamp    int64     `json:"timestamp"`
	ScriptPubKey string    `json:"scriptPubKey"`
	IsBest       bool      `json:"isBest"`
	IsWithdraw   bool      `json:"isWithdraw"`
}

type LuckyBagUnusedResponse struct {
	MetaId              string        `json:"metaId"`
	UserInfo            *UserInfo     `json:"userInfo"`
	SubId               string        `json:"subId"`
	Code                string        `json:"code"`
	CreateTime          string        `json:"createTime"`
	Amount              string        `json:"amount"`
	Count               string        `json:"count"`
	Content             string        `json:"content"`
	Img                 string        `json:"img"`
	ImgType             string        `json:"imgType"`
	Unused              []*UnusedList `json:"unused"`
	Type                string        `json:"type"`
	TokenCount          uint64        `json:"tokenCount"`
	RequireType         string        `json:"requireType"`
	RequireTickId       string        `json:"requireTickId"`
	RequireCollectionId string        `json:"requireCollectionId"`
	LimitAmount         uint64        `json:"limitAmount"`
}
type UnusedList struct {
	Index        int64  `json:"index"`
	Amount       string `json:"amount"`
	Address      string `json:"address"`
	ScriptPubKey string `json:"scriptPubKey"`
}
