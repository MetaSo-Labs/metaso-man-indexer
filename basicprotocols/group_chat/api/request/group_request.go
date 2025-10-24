package request

type FetchGroupListRequest struct {
	MetaId    string `form:"metaId"`
	Cursor    int64  `form:"cursor"`
	Size      int64  `form:"size"`
	Timestamp int64  `form:"timestamp"`
}

type FetchGroupInfoRequest struct {
	GroupId string `form:"groupId"`
}

type FetchGroupMemberListRequest struct {
	GroupId   string `form:"groupId"`
	Cursor    int64  `form:"cursor"`
	Size      int64  `form:"size"`
	Timestamp int64  `form:"timestamp"`
	OrderBy   string `form:"orderBy"`
	OrderType string `form:"orderType"`
}

type FetchGroupChatListRequest struct {
	GroupId   string `form:"groupId"`
	MetaId    string `form:"metaId"`
	Cursor    int64  `form:"cursor"`
	Size      int64  `form:"size"`
	Timestamp int64  `form:"timestamp"`
}

type FetchLatestChatGroupListRequest struct {
	MetaId    string `form:"metaId"`
	Cursor    int64  `form:"cursor"`
	Size      int64  `form:"size"`
	Timestamp int64  `form:"timestamp"`
}

// FetchGroupPersonRequest Get group member info request
type FetchGroupPersonRequest struct {
	MetaId  string `form:"metaId"`
	GroupId string `form:"groupId"`
}

// FetchLatestChatInfoListRequest Get latest chat info list request (group chat + private chat)
type FetchLatestChatInfoListRequest struct {
	MetaId    string `form:"metaId"`
	Cursor    int64  `form:"cursor"`
	Size      int64  `form:"size"`
	Timestamp int64  `form:"timestamp"`
}

// FetchPrivateChatListRequest Get private chat records request
type FetchPrivateChatListRequest struct {
	MetaId      string `form:"metaId"`      // Current user MetaId
	OtherMetaId string `form:"otherMetaId"` // Other user MetaId
	Cursor      int64  `form:"cursor"`
	Size        int64  `form:"size"`
	Timestamp   int64  `form:"timestamp"`
}

// FetchPrivateChatListByIndexRequest Get private chat list by index range request
type FetchPrivateChatListByIndexRequest struct {
	MetaId      string `form:"metaId"`      // Current user MetaId
	OtherMetaId string `form:"otherMetaId"` // Other user MetaId
	StartIndex  int64  `form:"startIndex"`  // Start index for pagination
	Size        int64  `form:"size"`        // Page size
}

// FetchLuckyBagInfoRequest Get lucky bag info request
type FetchLuckyBagInfoRequest struct {
	GroupId string `form:"groupId"` // Group ID
	PinId   string `form:"pinId"`   // Lucky bag PinId
}

// GrabLuckyBagRequest Grab lucky bag request
type GrabLuckyBagRequest struct {
	GroupId string `form:"groupId"` // Group ID
	PinId   string `form:"pinId"`   // Lucky bag PinId
	MetaId  string `form:"metaId"`  // User MetaId
	Address string `form:"address"` // User address
}

// ReclaimLuckyBagRequest Reclaim lucky bag request
type ReclaimLuckyBagRequest struct {
	GroupId string `form:"groupId"` // Group ID
	PinId   string `form:"pinId"`   // Lucky bag PinId
	MetaId  string `form:"metaId"`  // User MetaId
	Address string `form:"address"` // User address
}

// FetchGroupChatListByIndexRequest Get group chat list by index range request
type FetchGroupChatListByIndexRequest struct {
	GroupId    string `form:"groupId"`    // Group ID
	StartIndex int64  `form:"startIndex"` // Start index for pagination
	Size       int64  `form:"size"`       // Page size
}

// FetchGroupChatListByStartTimeRequest Get group chat list by start timestamp range request
type FetchGroupChatListByStartTimeRequest struct {
	GroupId        string `form:"groupId"`        // Group ID
	StartTimestamp int64  `form:"startTimestamp"` // Start timestamp for pagination
	Size           int64  `form:"size"`           // Page size
}

// SearchGroupRequest Search groups by name or ID request
type SearchGroupAndUserRequest struct {
	Query string `form:"query"` // Search query (group name or ID)
	Size  int64  `form:"size"`  // Page size
}

// SearchGroupRequest Search groups by name or ID request
type SearchGroupRequest struct {
	Query string `form:"query"` // Search query (group name or ID)
	Size  int64  `form:"size"`  // Page size
}

// SearchGroupMembersRequest Search group members request
type SearchGroupMembersRequest struct {
	GroupId string `form:"groupId"` // Group ID
	Query   string `form:"query"`   // Search query (user name, metaId)
	Size    int64  `form:"size"`    // Page size
}

// BatchUserInfoRequest Batch user info request
type BatchUserInfoRequest struct {
	Addresses []string `json:"addresses"` // List of user addresses
	MetaIds   []string `json:"metaIds"`   // List of user metaIds
}

// ==================== Channel Chat Request Types ====================

// FetchChannelChatListRequest Get channel chat records request
type FetchChannelChatListRequest struct {
	ChannelId string `form:"channelId"` // Channel ID
	// Cursor    int64  `form:"cursor"`    // Cursor for pagination
	Size      int64 `form:"size"`      // Page size
	Timestamp int64 `form:"timestamp"` // Timestamp for pagination
}

// FetchChannelChatListByIndexRequest Get channel chat records by index range request
type FetchChannelChatListByIndexRequest struct {
	ChannelId  string `form:"channelId"`  // Channel ID
	StartIndex int64  `form:"startIndex"` // Start index for pagination
	Size       int64  `form:"size"`       // Page size
}

// FetchChannelChatListByStartTimeRequest Get channel chat records by start timestamp range request
type FetchChannelChatListByStartTimeRequest struct {
	ChannelId      string `form:"channelId"`      // Channel ID
	StartTimestamp int64  `form:"startTimestamp"` // Start timestamp for pagination
	Size           int64  `form:"size"`           // Page size
}

// ==================== Group Channel Request Types ====================

// FetchGroupChannelListRequest Get group channel list request
type FetchGroupChannelListRequest struct {
	GroupId string `form:"groupId"` // Group ID
	Cursor  int64  `form:"cursor"`  // Cursor for pagination
	Size    int64  `form:"size"`    // Page size
}

// FetchGroupUserRoleInfoRequest Get group user role info request
type FetchGroupUserRoleInfoRequest struct {
	GroupId   string `form:"groupId"`   // Group ID
	ChannelId string `form:"channelId"` // Channel ID (optional)
	MetaId    string `form:"metaId"`    // User MetaId
}

// ProcessLuckyBagMetaContractFtManualExtraGasRequest Process lucky bag meta contract FT manual extra gas request
type ProcessLuckyBagMetaContractFtManualExtraGasRequest struct {
	LuckyBagPinId        string `json:"luckyBagPinId"`        // Lucky bag pin ID
	OutSidePrivateKeyHex string `json:"outSidePrivateKeyHex"` // Outside private key hex
	OutSideAddress       string `json:"outSideAddress"`       // Outside address
	OutSideTxId          string `json:"outSideTxId"`          // Outside transaction ID
	OutSideIndex         int64  `json:"outSideIndex"`         // Outside output index
	OutSideAmount        uint64 `json:"outSideAmount"`        // Outside amount
	PerAmount            uint64 `json:"perAmount"`            // Amount per output
	ChangeAddress        string `json:"changeAddress"`        // Change address
}
