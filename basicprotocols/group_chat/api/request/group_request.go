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
