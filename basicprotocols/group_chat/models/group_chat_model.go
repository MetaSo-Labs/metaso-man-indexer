package models

type TalkCommunityModel struct {
	CommunityId string   `json:"communityId"` // Community ID: hash(metaname) unique
	TxId        string   `json:"txId"`        //
	PinId       string   `json:"pinId"`       //
	MetaId      string   `json:"metaId"`
	Address     string   `json:"address"`
	PublicKey   string   `json:"publicKey"`
	Name        string   `json:"name"`        // Community name
	Description string   `json:"description"` // Community description
	Cover       string   `json:"cover"`       // Cover image
	Icon        string   `json:"icon"`        // Icon
	MetaName    string   `json:"metaName"`    // Meta domain name
	MetaNameNft string   `json:"metaNameNft"` // Codehash/genesis/tokenIndex
	Admins      []string `json:"admins"`      // Community administrators
	Reserved    string   `json:"reserved"`    // Use 00 private key of metaId user who generates node to sign metaName
	// ValidState   ValidState `json:"validState"`   // Verify if valid
	Chain        string `json:"chain"` // Chain type
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
	CommunityId    string    `json:"communityId"` // Community ID: hash(metaname)
	CommunityState RoomState `json:"communityState"`
	// IsValid        bool      `json:"isValid"`
	// IsNew          bool      `json:"isNew"`
	Chain        string `json:"chain"` // Chain type
	BlockHeight  int64  `json:"blockHeight"`
	ConfirmState int64  `json:"confirmState"`
	Timestamp    int64  `json:"timestamp"` //
}

type TalkCommunityInfo struct {
	CommunityId         string `json:"communityId"` // Room ID unique
	PersonTotal         uint64 `json:"personTotal"`
	ChatTotal           uint64 `json:"chatTotal"`
	ChatTotalUpdateTime int64  `json:"chatTotalUpdateTime"`
	Timestamp           int64  `json:"timestamp"` // Timestamp
}

type TalkCommunityPerson struct {
	CommunityIdMetaIdHash string    `json:"communityIdMetaIdHash"` // Room ID and member unique
	CommunityId           string    `json:"communityId"`           // Room ID unique
	MetaId                string    `json:"metaId"`
	AvatarTxId            string    `json:"avatarTxId"`
	UserName              string    `json:"userName"`
	UserNickName          string    `json:"userNickName"`
	CommunityState        RoomState `json:"communityState"`
	Timestamp             int64     `json:"timestamp"` // Timestamp when joining or leaving room
}

type TalkGroupJoinModel struct {
	TxId        string `json:"txId"`  //
	PinId       string `json:"pinId"` //
	MetaId      string `json:"metaId"`
	ZeroAddress string `json:"zeroAddress"`
	Address     string `json:"address"`
	// PublicKey    string    `json:"publicKey"`
	GroupId    string    `json:"groupId"` // Group ID: hash(metaname)
	GroupState RoomState `json:"groupState"`
	Referrer   string    `json:"referrer"` // Referrer
	Chain      string    `json:"chain"`    // Chain type
	// IsValid      bool      `json:"isValid"`
	// IsNew        bool      `json:"isNew"`
	BlockHeight  int64 `json:"blockHeight"`
	ConfirmState int64 `json:"confirmState"`
	Timestamp    int64 `json:"timestamp"` //
}

type TalkGroupPerson struct {
	GroupIdMetaIdHash string    `json:"groupIdMetaIdHash"` // Group ID and member unique
	GroupId           string    `json:"groupId"`           // Group ID unique
	MetaId            string    `json:"metaId"`
	Address           string    `json:"address"`
	AvatarTxId        string    `json:"avatarTxId"`
	UserName          string    `json:"userName"`
	UserNickName      string    `json:"userNickName"`
	GroupState        RoomState `json:"groupState"`
	Timestamp         int64     `json:"timestamp"`   // Timestamp when joining or leaving group
	BlockHeight       int64     `json:"blockHeight"` // Block height
	PinId             string    `json:"pinId"`       // PinId when joining or leaving group
}

// Group person list model for TalkGroupPersonListCollection
type TalkGroupPersonList struct {
	GroupId string             `json:"groupId"` // Group ID
	Persons []*TalkGroupPerson `json:"persons"` // List of group members
	Total   int64              `json:"total"`   // Total number of group members
}

type TalkGroupModel struct {
	GroupId       string `json:"groupId"`       // Room ID unique
	CommunityId   string `json:"communityId"`   // Community ID unique
	TxId          string `json:"txId"`          // Room's TxId
	PinId         string `json:"pinId"`         // Room's PinId
	RoomPublicKey string `json:"roomPublicKey"` // Room public key
	RoomName      string `json:"roomName"`      // Room creation name
	RoomNote      string `json:"roomNote"`      // Room creation announcement
	RoomIcon      string `json:"roomIcon"`      // Room creation icon
	RoomType      string `json:"roomType"`      // Room creation type "1" not encrypted "2" encrypted encryption using AES encryption algorithm
	RoomStatus    string `json:"roomStatus"`    // "1" when not encrypted is "1", when encrypted is encrypted information, reserved field
	RoomJoinType  string `json:"roomJoinType"`  // Join method, 1 is password, 2 is nft
	// RoomCodeHash          string `json:"roomCodeHash"`          // Has value when roomJoinType is 2, codeHash
	// RoomGenesis           string `json:"roomGenesis"`           // Has value when roomJoinType is 2, genesis
	// RoomLimitAmount       int64  `json:"roomLimitAmount"`       // Has value when roomJoinType is 2, token limit
	// RoomGenesisSeriesName string `json:"roomGenesisSeriesName"` // Has value when roomJoinType is 2, genesis
	RoomAvatarUrl string `json:"roomAvatarUrl"` // Room avatar URL
	// RoomNinePersonHash string `json:"roomNinePersonHash"` // Total hash value of metaId of first 9 people in the room
	// RoomNewestTxId        string `json:"roomNewestTxId"`        // TxId of the latest chat content in the room
	// RoomNewestMetaId      string `json:"roomNewestMetaId"`      // MetaId of the latest chat content in the room
	// RoomNewestProtocol    string `json:"roomNewestProtocol"`    // Protocol type of the latest chat content in the room
	// RoomNewestContent     string `json:"roomNewestContent"`     // Latest chat content in the room
	// RoomNewestTimestamp   int64  `json:"roomNewestTimestamp"`   // Timestamp of the latest chat in the room
	CreateUserMetaId  string `json:"createUserMetaId"`  // MetaId of the creator
	CreateUserAddress string `json:"createUserAddress"` // Address of the creator
	ChatSettingType   int64  `json:"chatSettingType"`   // Used to set speech restrictions, 0-everyone, 1-administrators
	// ValidState            ValidState `json:"validState"`                       // Verify if valid
	Chain        string `json:"chain"`        // Chain type
	DeleteStatus int64  `json:"deleteStatus"` // Delete status, 0-normal, 1-deleted
	Timestamp    int64  `json:"timestamp"`    // Timestamp when creating the room
	BlockHeight  int64  `json:"blockHeight"`  // Block height
}

type TalkGroupTxV3 struct {
	TxId          string `json:"txId"`          // Room's TxId
	MetaId        string `json:"metaId"`        // metaId
	GroupId       string `json:"groupId"`       // Room ID unique
	CommunityId   string `json:"communityId"`   // Community ID unique
	RoomPublicKey string `json:"roomPublicKey"` // Room public key
	RoomName      string `json:"roomName"`      // Room creation name
	RoomNote      string `json:"roomNote"`      // Room creation announcement
	RoomType      string `json:"roomType"`      // Room creation type "1" not encrypted "2" encrypted encryption using AES encryption algorithm
	RoomStatus    string `json:"roomStatus"`    // "1" when not encrypted is "1", when encrypted is encrypted information, reserved field
	RoomJoinType  string `json:"roomJoinType"`  // Join method, 1 is password, 2 is nft
	// RoomCodeHash          string `json:"roomCodeHash"`          // Has value when roomJoinType is 2, codeHash
	// RoomGenesis           string `json:"roomGenesis"`           // Has value when roomJoinType is 2, genesis
	// RoomLimitAmount       int64  `json:"roomLimitAmount"`       // Has value when roomJoinType is 2, token limit
	// RoomGenesisSeriesName string `json:"roomGenesisSeriesName"` // Has value when roomJoinType is 2, genesis
	ChatSettingType int64 `json:"chatSettingType"` // Used to set speech restrictions, 0-everyone, 1-administrators
	// ValidState            ValidState `json:"validState"`                       // Verify if valid
	DeleteStatus int64 `json:"deleteStatus"` // Delete status, 0-normal, 1-deleted
	// IsValid      bool  `json:"isValid"`
	// IsNew        bool  `json:"isNew"`
	BlockHeight  int64  `json:"blockHeight"`
	ConfirmState int64  `json:"confirmState"`
	Timestamp    int64  `json:"timestamp"` // Timestamp when creating the room
	Chain        string `json:"chain"`     // Chain type
}

type ChatInsideIndex int64

const (
	ChatInsideIndexIn  ChatInsideIndex = 0 //0-in
	ChatInsideIndexOut ChatInsideIndex = 1 //1-out
)

type TalkGroupChatV3 struct {
	CommunityId string          `json:"communityId"` // Community ID unique
	GroupId     string          `json:"groupId"`     // Room ID unique
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
	Timestamp   int64           `json:"timestamp"`   // Chat record timestamp
	Chain       string          `json:"chain"`       // Chain type
	BlockHeight int64           `json:"blockHeight"` // Block height
	Index       int64           `json:"index"`       // Index default -1
}

type ReplyInfo struct {
	TxId        string          `json:"txId"`
	PinId       string          `json:"pinId"`
	MetaId      string          `json:"metaId"`
	NickName    string          `json:"nickName"`
	Protocol    string          `json:"protocol"`
	Content     string          `json:"content"`
	ContentType string          `json:"contentType"`
	Encryption  string          `json:"encryption"`
	ChatType    ChatType        `json:"chatType"`    //0-msg, 1-red, 2-img
	InsideIndex ChatInsideIndex `json:"insideIndex"` //0-in, 1-out
	Timestamp   int64           `json:"timestamp"`   // Chat record timestamp
	Index       int64           `json:"index"`       // Index default -1
}

type TalkGroupLuckyBagV3 struct {
	CommunityId           string            `json:"communityId"` // Room ID unique
	GroupId               string            `json:"groupId"`     // Channel ID unique
	TxId                  string            `json:"txId"`
	PinId                 string            `json:"pinId"` //
	MetaId                string            `json:"metaId"`
	Address               string            `json:"address"`
	Protocol              string            `json:"protocol"`
	SubId                 string            `json:"subId"`
	Code                  string            `json:"code"`
	CreateTimeStr         string            `json:"createTimeStr"`
	Domain                string            `json:"domain"`
	LuckyBagAddress       string            `json:"luckyBagAddress"`
	GenType               int64             `json:"genType"`  // 0-normal, 1-internal, 2-external
	GenState              int64             `json:"genState"` // 0-normal, 1-success, 2-failed
	Content               string            `json:"content"`
	Img                   string            `json:"img"`
	ImgType               string            `json:"imgType"`
	Amount                string            `json:"amount"`
	Count                 string            `json:"count"`
	ValidCount            string            `json:"validCount"`
	ErrCount              string            `json:"errCount"`
	PayList               []*ProInfoPayList `json:"payList"`
	ErrPayList            []*ProInfoPayList `json:"errPayList"`
	LuckyBagVouts         []*LuckyBagOutput `json:"luckyBagVouts"`
	ErrLuckyBagVouts      []*LuckyBagOutput `json:"errLuckyBagVouts"`
	OriginalPayList       []*ProInfoPayList `json:"originalPayList"`
	OriginalLuckyBagVouts []*LuckyBagOutput `json:"originalLuckyBagVouts"`
	Type                  string            `json:"type"`
	RequireType           string            `json:"requireType"`         //0-no limit, 1-FT, 2-NFT
	RequireTickId         string            `json:"requireTickId"`       // FT-limit requires temporarily mrc20
	RequireCollectionId   string            `json:"requireCollectionId"` // NFT-limit requires temporarily mrc721
	LimitAmount           uint64            `json:"limitAmount"`
	Timestamp             int64             `json:"timestamp"`   // Chat record timestamp
	BlockHeight           int64             `json:"blockHeight"` // Block height
	Chain                 string            `json:"chain"`       // Chain type
	State                 int               `json:"state"`       // 1-pending, 2-completed, 3-timeout residue, 4-err, 5-err timeout residue
}
type ProInfoPayList struct {
	Amount   string `json:"amount"`
	Address  string `json:"address"`
	PkScript string `json:"pkScript"`
	Index    int64  `json:"index"`
}
type LuckyBagOutput struct {
	ScriptPubKey string `json:"scriptPubKey"`
	Amount       uint64 `json:"amount"`
	Address      string `json:"address"`
	Index        int64  `json:"index"`
}
type GrabState int

const (
	GrabStateChain          GrabState = 0
	GrabStateOpen           GrabState = 1
	GrabStateOpenAndSend    GrabState = 2
	GrabStateOpenAndSendErr GrabState = 3
	GrabStateDuplicate      GrabState = 4

	GrabStateReclaim           GrabState = 5
	GrabStateReclaimAndSend    GrabState = 6
	GrabStateReclaimAndSendErr GrabState = 7
)

type TalkGroupOpenLuckyBagV3 struct {
	CommunityId         string    `json:"communityId"` // Room ID unique
	GroupId             string    `json:"groupId"`     // Channel ID unique
	TxId                string    `json:"txId"`
	PinId               string    `json:"pinId"` //
	MetaId              string    `json:"metaId"`
	Protocol            string    `json:"protocol"`
	SubId               string    `json:"subId"`
	Code                string    `json:"code"`
	CreateTimeStr       string    `json:"createTimeStr"`
	Domain              string    `json:"domain"`
	LuckyBagAddress     string    `json:"luckyBagAddress"`
	GenType             int64     `json:"genType"`  // 0-normal, 1-internal, 2-external
	GenState            int64     `json:"genState"` // 0-normal, 1-success, 2-failed
	Address             string    `json:"address"`
	Index               int64     `json:"index"`
	Amount              string    `json:"amount"`
	PkScript            string    `json:"pkScript"`
	Vins                []*TxIn   `json:"vins"`
	Type                string    `json:"type"`
	RequireTickId       string    `json:"requireTickId"`       // FT-limit requires temporarily mrc20
	RequireCollectionId string    `json:"requireCollectionId"` // NFT-limit requires temporarily mrc721
	LuckyBagTxId        string    `json:"luckyBagTxId"`
	LuckyBagPinId       string    `json:"luckyBagPinId"`
	LuckyBagMetaId      string    `json:"luckyBagMetaId"`
	IsWithdraw          bool      `json:"isWithdraw"`
	Timestamp           int64     `json:"timestamp"`   // Chat record timestamp
	BlockHeight         int64     `json:"blockHeight"` // Block height
	Chain               string    `json:"chain"`       // Chain type
	GrabState           GrabState `json:"grabState"`   // Red envelope status, 0-chain open, 1-centralized open, 2-centralized open and sent, 3-centralized open and sent abnormal, 4-reclaim, 5-reclaim and sent, 6-reclaim and sent abnormal
	GrabTxId            string    `json:"grabTxId"`    //
	GrabMsg             string    `json:"grabMsg"`     //
}
type TxIn struct {
	OutTxID string `json:"outTxId"` // out-txId where it is located
	Index   uint64 `json:"index"`
	//Value       uint64 `json:"value" bson:"value"`
	//Address     string `json:"address" bson:"address"`
	//PublicKey   string `json:"publicKey" bson:"publicKey"`
	//OutTxScript string `json:"outTxScript" bson:"outTxScript"`
}

type TalkGroupResidueLuckyBagV3 struct {
	CommunityId         string            `json:"communityId"` // Room ID unique
	GroupId             string            `json:"groupId"`     // Channel ID unique
	TxId                string            `json:"txId"`
	PinId               string            `json:"pinId"` //
	MetaId              string            `json:"metaId"`
	Address             string            `json:"address"`
	Protocol            string            `json:"protocol"`
	SubId               string            `json:"subId"`
	Code                string            `json:"code"`
	CreateTimeStr       string            `json:"createTimeStr"`
	Domain              string            `json:"domain"`
	LuckyBagAddress     string            `json:"luckyBagAddress"`
	GenType             int64             `json:"genType"`  // 0-normal, 1-internal, 2-external
	GenState            int64             `json:"genState"` // 0-normal, 1-success, 2-failed
	PkScript            string            `json:"pkScript"`
	Amount              string            `json:"amount"`
	Index               int64             `json:"index"`
	UsedList            []*ProInfoPayList `json:"usedList"`
	Vins                []*TxIn           `json:"vins"`
	Type                string            `json:"type"`
	RequireTickId       string            `json:"requireTickId"`       // FT-limit requires temporarily mrc20
	RequireCollectionId string            `json:"requireCollectionId"` // NFT-limit requires temporarily mrc721
	LuckyBagTxId        string            `json:"luckyBagTxId"`
	LuckyBagPinId       string            `json:"luckyBagPinId"`
	LuckyBagMetaId      string            `json:"luckyBagMetaId"`
	Timestamp           int64             `json:"timestamp"`    // Chat record timestamp
	BlockHeight         int64             `json:"blockHeight"`  // Block height
	Chain               string            `json:"chain"`        // Chain type
	ReclaimState        GrabState         `json:"reclaimState"` // Red envelope status, 0-chain open, 1-centralized open, 2-centralized open and sent, 3-centralized open and sent abnormal
	ReclaimTxId         string            `json:"reclaimTxId"`  //
	ReclaimMsg          string            `json:"reclaimMsg"`   //
}

// User group list item + user private chat list item
type MetaIdContextItem struct {
	GroupId          string   `json:"groupId"`          // Group ID
	MetaId           string   `json:"metaId"`           // Other party's MetaId
	Address          string   `json:"address"`          // Other party's address
	Type             string   `json:"type"`             // Type, 1-group chat, 2-private chat
	Timestamp        int64    `json:"timestamp"`        // Latest message timestamp
	ChatType         ChatType `json:"chatType"`         // Message type
	Content          string   `json:"content"`          // Message content summary
	CreateMetaId     string   `json:"createMetaId"`     // MetaId of message creator
	CreateAddress    string   `json:"createAddress"`    // Address of message creator
	LastMessagePinId string   `json:"lastMessagePinId"` // PinId of latest message
	BlockHeight      int64    `json:"blockHeight"`      // Block height
}

// User group list
type MetaIdContextList struct {
	MetaId string               `json:"metaId"` // User MetaId
	Items  []*MetaIdContextItem `json:"items"`  // Group list items
}

// Group latest chat record
type TalkGroupLatestChat struct {
	GroupId          string   `json:"groupId"`          // Group ID
	Timestamp        int64    `json:"timestamp"`        // Latest message timestamp
	ChatType         ChatType `json:"chatType"`         // Message type
	Content          string   `json:"content"`          // Message content summary
	CreateAddress    string   `json:"createAddress"`    // Address of message creator
	LastMessagePinId string   `json:"lastMessagePinId"` // PinId of latest message
	MetaId           string   `json:"metaId"`           // MetaId of message creator
	TxId             string   `json:"txId"`             // Message's TxId
	PinId            string   `json:"pinId"`            // Message's PinId
	Protocol         string   `json:"protocol"`         // Protocol type
	ContentType      string   `json:"contentType"`      // Content type
	Encryption       string   `json:"encryption"`       // Encryption information
	ReplyPin         string   `json:"replyPin"`         // PinId of reply message
	Chain            string   `json:"chain"`            // Chain type
	BlockHeight      int64    `json:"blockHeight"`      // Block height
}

// Private chat message model
type TalkPrivateChatV3 struct {
	From        string     `json:"from"`        // Sender MetaId
	FromAddress string     `json:"fromAddress"` // Sender address
	To          string     `json:"to"`          // Receiver MetaId
	ToAddress   string     `json:"toAddress"`   // Receiver address
	TxId        string     `json:"txId"`
	PinId       string     `json:"pinId"` //
	Protocol    string     `json:"protocol"`
	Content     string     `json:"content"`
	ContentType string     `json:"contentType"`
	Encryption  string     `json:"encryption"`
	ChatType    ChatType   `json:"chatType"` //0-msg, 1-red, 3-img
	ReplyPin    string     `json:"replyPin"`
	ReplyInfo   *ReplyInfo `json:"replyInfo"`
	Timestamp   int64      `json:"timestamp"`   // Chat record timestamp
	Chain       string     `json:"chain"`       // Chain type
	BlockHeight int64      `json:"blockHeight"` // Block height
	Index       int64      `json:"index"`       // Index default -1
}

// Grab lucky bag list item
type OpenLuckyBagListItem struct {
	OpenPinId        string `json:"openPinId"`        // PinId of grabbed lucky bag
	GroupId          string `json:"groupId"`          // Group ID
	Timestamp        int64  `json:"timestamp"`        // Timestamp
	CreateMetaId     string `json:"createMetaId"`     // Creator's MetaId
	CreateAddress    string `json:"createAddress"`    // Creation address
	LuckyBagOutIndex int64  `json:"luckyBagOutIndex"` // Lucky bag output index
}

// Grab lucky bag list
type OpenLuckyBagList struct {
	LuckyBagPinId string                  `json:"luckyBagPinId"` // Lucky bag PinId
	Items         []*OpenLuckyBagListItem `json:"items"`         // Grab lucky bag list items
}

// Reclaim lucky bag list item
type ResidueLuckyBagListItem struct {
	ResiduePinId         string  `json:"residuePinId"`         // PinId of reclaimed lucky bag
	GroupId              string  `json:"groupId"`              // Group ID
	Timestamp            int64   `json:"timestamp"`            // Timestamp
	CreateMetaId         string  `json:"createMetaId"`         // Creator's MetaId
	CreateAddress        string  `json:"createAddress"`        // Creation address
	LuckyBagOutIndexList []int64 `json:"luckyBagOutIndexList"` // Lucky bag output index list
}

// Reclaim lucky bag list
type ResidueLuckyBagList struct {
	LuckyBagPinId string                     `json:"luckyBagPinId"` // Lucky bag PinId
	Items         []*ResidueLuckyBagListItem `json:"items"`         // Reclaim lucky bag list items
}

type UserInfo struct {
	MetaId          string `json:"metaId"`
	Address         string `json:"address"`
	ChatPublicKey   string `json:"chatPublicKey"`
	ChatPublicKeyId string `json:"chatPublicKeyId"`
	Timestamp       int64  `json:"timestamp"`
	BlockHeight     int64  `json:"blockHeight"`
	Chain           string `json:"chain"`
	Operation       string `json:"operation"`
	IsValid         bool   `json:"isValid"`
}

// Group remove user model
type TalkGroupRemoveUserModel struct {
	GroupId         string `json:"groupId"`         // Group ID
	RemoveMetaId    string `json:"removeMetaId"`    // MetaId of user being removed
	RemoveAddress   string `json:"removeAddress"`   // Address of user being removed
	RemoveReason    string `json:"removeReason"`    // Reason for removal
	RemoveByMetaId  string `json:"removeByMetaId"`  // MetaId of user who initiated removal
	RemoveByAddress string `json:"removeByAddress"` // Address of user who initiated removal
	TxId            string `json:"txId"`            // Transaction ID
	PinId           string `json:"pinId"`           // Pin ID
	Chain           string `json:"chain"`           // Chain type
	BlockHeight     int64  `json:"blockHeight"`     // Block height
	ConfirmState    int64  `json:"confirmState"`    // Confirmation state
	Timestamp       int64  `json:"timestamp"`       // Timestamp
}

type SimplePrivateBlock struct {
	PinId       string `json:"pinId"`
	TxId        string `json:"txId"`
	To          string `json:"to"`
	BlockState  int64  `json:"blockState"` // 1:block, -1:unblock
	Protocol    string `json:"protocol"`
	Timestamp   int64  `json:"timestamp"`
	Chain       string `json:"chain"`
	BlockHeight int64  `json:"blockHeight"`
}

type TalkPrivateChatBlock struct {
	BlockPinId     string `json:"blockPinId"`
	BlockMetaId    string `json:"blockMetaId"`
	BlockState     int64  `json:"blockState"` // 1:block, -1:unblock
	BlockTimestamp int64  `json:"blockTimestamp"`
}

// TalkPrivateChatBlockList represents a list of private chat blocks for a user
type TalkPrivateChatBlockList struct {
	MetaId string                  `json:"metaId"` // User MetaId
	Items  []*TalkPrivateChatBlock `json:"items"`  // List of block items
}
