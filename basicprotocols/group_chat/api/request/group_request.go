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
