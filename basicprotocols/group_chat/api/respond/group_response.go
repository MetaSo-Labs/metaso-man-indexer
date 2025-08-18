package respond

import "manindexer/basicprotocols/group_chat/models"

type UserInfo struct {
	Metaid      string `json:"metaid"`
	Name        string `json:"name"`
	Avatar      string `json:"avatar"`
	AvatarImage string `json:"avatarImage"`
}

type GroupResponse struct {
	Total int64        `json:"total"`
	List  []*GroupItem `json:"list"`
}

type GroupItem struct {
	CommunityId         string    `json:"communityId"`         //Community ID, unique
	GroupId             string    `json:"groupId"`             //Room ID, unique
	TxId                string    `json:"txId"`                //Room's TxId
	PinId               string    `json:"pinId"`               //Room's PinId
	RoomName            string    `json:"roomName"`            //Room creation name
	RoomNote            string    `json:"roomNote"`            //Room creation announcement
	RoomType            string    `json:"roomType"`            //Room creation type "1" unencrypted "2" encrypted, encryption uses AES algorithm
	RoomStatus          string    `json:"roomStatus"`          //"1" When unencrypted it's "1", when encrypted it's encrypted info, reserved field
	RoomJoinType        string    `json:"roomJoinType"`        //Join method, 1 for password, 2 for nft
	RoomAvatarUrl       string    `json:"roomAvatarUrl"`       //Room avatar url
	RoomNinePersonHash  string    `json:"roomNinePersonHash"`  //Hash value of first 9 members' metaId in room
	RoomNewestTxId      string    `json:"roomNewestTxId"`      //Room's latest chat content txId
	RoomNewestPinId     string    `json:"roomNewestPinId"`     //Room's latest chat content pinId
	RoomNewestMetaId    string    `json:"roomNewestMetaId"`    //Room's latest chat content MetaId
	RoomNewestUserName  string    `json:"roomNewestUserName"`  //Room's latest chat content MetaId
	RoomNewestProtocol  string    `json:"roomNewestProtocol"`  //Room's latest chat content protocol type
	RoomNewestContent   string    `json:"roomNewestContent"`   //Room's latest chat content
	RoomNewestTimestamp int64     `json:"roomNewestTimestamp"` //Room's latest chat timestamp
	CreateUserMetaId    string    `json:"createUserMetaId"`    //Creator's metaId
	CreateUserAddress   string    `json:"createUserAddress"`   //Creator's address
	CreateUserInfo      *UserInfo `json:"createUserInfo"`      //Creator's info
	UserCount           int64     `json:"userCount"`           //Room member count
	ChatSettingType     int64     `json:"chatSettingType"`     //Used for setting speech restrictions, 0-everyone, 1-admin
	DeleteStatus        int64     `json:"deleteStatus"`        //Delete status, 0-normal, 1-deleted
	Timestamp           int64     `json:"timestamp"`           //Room creation timestamp
	Chain               string    `json:"chain"`               //Chain type
	BlockHeight         int64     `json:"blockHeight"`         //Block height
}

type GroupChatResponse struct {
	Total         int64            `json:"total"`
	NextTimestamp int64            `json:"nextTimestamp"`
	List          []*GroupChatItem `json:"list"`
}

type GroupChatItem struct {
	GroupId     string          `json:"groupId"`   //Room ID, unique
	MetanetId   string          `json:"metanetId"` //
	TxId        string          `json:"txId"`
	PinId       string          `json:"pinId"`
	MetaId      string          `json:"metaId"`
	Address     string          `json:"address"`
	UserInfo    *UserInfo       `json:"userInfo"`
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
	Timestamp   int64           `json:"timestamp"`   //Chat record timestamp
	Params      string          `json:"params"`      //General field for future parameter additions
	Chain       string          `json:"chain"`       //Chain type
	BlockHeight int64           `json:"blockHeight"` //Block height
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
	Timestamp   int64           `json:"timestamp"`   //Chat record timestamp
	Chain       string          `json:"chain"`       //Chain type
	BlockHeight int64           `json:"blockHeight"` //Block height
}

type GroupMemberResponse struct {
	Total int64              `json:"total"`
	List  []*GroupMemberItem `json:"list"`
}

type GroupMemberItem struct {
	MetaId    string    `json:"metaId"`
	Address   string    `json:"address"`
	UserInfo  *UserInfo `json:"userInfo"`
	TimeStr   string    `json:"timeStr"`
	Timestamp int64     `json:"timestamp"`
}

// GroupPersonResponse Group member info response
type GroupPersonResponse struct {
	IsInGroup bool             `json:"isInGroup"` // Whether in the group
	Person    *GroupPersonItem `json:"person"`    // Member info, null if not in group
}

// GroupPersonItem Group member info item
type GroupPersonItem struct {
	GroupIdMetaIdHash string    `json:"groupIdMetaIdHash"` // Group ID and member unique identifier
	GroupId           string    `json:"groupId"`           // Group ID
	MetaId            string    `json:"metaId"`            // User MetaId
	Address           string    `json:"address"`           // User address
	UserInfo          *UserInfo `json:"userInfo"`          // User info
	AvatarTxId        string    `json:"avatarTxId"`        // Avatar TxId
	UserName          string    `json:"userName"`          // Username
	UserNickName      string    `json:"userNickName"`      // User nickname
	GroupState        int64     `json:"groupState"`        // Group status: 1-in group, -1-left
	Timestamp         int64     `json:"timestamp"`         // Join or leave group timestamp
	BlockHeight       int64     `json:"blockHeight"`       // Block height
	PinId             string    `json:"pinId"`             // Join or leave group PinId
}

// ChatInfoResponse Latest chat info response (group chat + private chat)
type ChatInfoResponse struct {
	Total int64           `json:"total"`
	List  []*ChatInfoItem `json:"list"`
}

// ChatInfoItem Chat info item (group chat or private chat)
type ChatInfoItem struct {
	Type             string `json:"type"`             // Type: 1-group chat, 2-private chat
	GroupId          string `json:"groupId"`          // Group ID (for group chat)
	MetaId           string `json:"metaId"`           // Other party MetaId (for private chat)
	Address          string `json:"address"`          // Other party address (for private chat)
	Timestamp        int64  `json:"timestamp"`        // Latest message timestamp
	ChatType         int64  `json:"chatType"`         // Message type 0-msg, 1-red, 2-img
	Content          string `json:"content"`          // Message content summary
	CreateMetaId     string `json:"createMetaId"`     // Message creator's MetaId
	CreateAddress    string `json:"createAddress"`    // Message creator's address
	LastMessagePinId string `json:"lastMessagePinId"` // Latest message's PinId
	BlockHeight      int64  `json:"blockHeight"`      // Block height
	Chain            string `json:"chain"`            // Chain type

	// Private chat specific fields
	UserInfo *UserInfo `json:"userInfo,omitempty"`

	// Group chat specific fields
	CommunityId       string    `json:"communityId,omitempty"`     // Community ID (for group chat)
	RoomName          string    `json:"roomName,omitempty"`        // Room name (for group chat)
	RoomNote          string    `json:"roomNote,omitempty"`        // Room announcement (for group chat)
	RoomType          string    `json:"roomType,omitempty"`        // Room type (for group chat)
	RoomStatus        string    `json:"roomStatus,omitempty"`      // Room status (for group chat)
	RoomJoinType      string    `json:"roomJoinType,omitempty"`    // Join method (for group chat)
	RoomAvatarUrl     string    `json:"roomAvatarUrl,omitempty"`   // Room avatar (for group chat)
	CreateUserMetaId  string    `json:"createUserMetaId"`          // Creator MetaId (for group chat)
	CreateUserAddress string    `json:"createUserAddress"`         // Creator address (for group chat)
	CreateUserInfo    *UserInfo `json:"createUserInfo"`            // Creator info (for group chat)
	UserCount         int64     `json:"userCount,omitempty"`       // User count (for group chat)
	ChatSettingType   int64     `json:"chatSettingType,omitempty"` // Chat setting type (for group chat)
	DeleteStatus      int64     `json:"deleteStatus,omitempty"`    // Delete status (for group chat)
}

// PrivateChatResponse Private chat records response
type PrivateChatResponse struct {
	Total         int64              `json:"total"`
	NextTimestamp int64              `json:"nextTimestamp"`
	List          []*PrivateChatItem `json:"list"`
}

// PrivateChatItem Private chat record item
type PrivateChatItem struct {
	From        string      `json:"from"` // Sender MetaId
	To          string      `json:"to"`   // Receiver MetaId
	TxId        string      `json:"txId"`
	PinId       string      `json:"pinId"`
	MetaId      string      `json:"metaId"`   // Message creator MetaId
	Address     string      `json:"address"`  // Message creator address
	UserInfo    *UserInfo   `json:"userInfo"` // User info
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
	Timestamp   int64       `json:"timestamp"`   // Chat record timestamp
	Params      string      `json:"params"`      // General field for future parameter additions
	Chain       string      `json:"chain"`       // Chain type
	BlockHeight int64       `json:"blockHeight"` // Block height
}

type LuckyBagInfoResponse struct {
	TxId                string         `json:"txId"`
	PinId               string         `json:"pinId"`
	MetaId              string         `json:"metaId"`
	Address             string         `json:"address"`
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
	TxId         string           `json:"txId"`
	Index        int64            `json:"index"`
	Amount       string           `json:"amount"`
	Address      string           `json:"address"`
	Used         bool             `json:"used"`
	GradPinId    string           `json:"gradPinId"`
	GradMetaId   string           `json:"gradMetaId"`
	GradAddress  string           `json:"gradAddress"`
	GradTxId     string           `json:"gradTxId"`
	GradState    models.GrabState `json:"gradState"`
	GradMsg      string           `json:"gradMsg"`
	UserInfo     *UserInfo        `json:"userInfo"`
	Timestamp    int64            `json:"timestamp"`
	ScriptPubKey string           `json:"scriptPubKey"`
	IsBest       bool             `json:"isBest"`
	IsWithdraw   bool             `json:"isWithdraw"`
}

type LuckyBagUnusedResponse struct {
	PinId               string        `json:"pinId"`
	MetaId              string        `json:"metaId"`
	Address             string        `json:"address"`
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
