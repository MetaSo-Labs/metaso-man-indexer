package request

type FetchGroupListRequest struct {
	MetaId string `form:"metaId"`
	// Page          string `form:"page"`
	// PageSize      string `form:"pageSize"`
	Cursor    int64 `form:"cursor"`
	Size      int64 `form:"size"`
	Timestamp int64 `form:"timestamp"`
}

type FetchGroupInfoRequest struct {
	GroupId string `form:"groupId"`
}

type FetchGroupMemberListRequest struct {
	GroupId string `form:"groupId"`
	// Page      string `form:"page"`
	// PageSize  string `form:"pageSize"`
	Cursor    int64 `form:"cursor"`
	Size      int64 `form:"size"`
	Timestamp int64 `form:"timestamp"`
}

type FetchGroupChatListRequest struct {
	GroupId string `form:"groupId"`
	MetaId  string `form:"metaId"`
	// Page          string `form:"page"`
	// PageSize      string `form:"pageSize"`
	Cursor    int64 `form:"cursor"`
	Size      int64 `form:"size"`
	Timestamp int64 `form:"timestamp"`
	// TimestampType int64 `form:"timestampType"` //0为查历史，1为查最新
}

type FetchLatestChatGroupListRequest struct {
	MetaId string `form:"metaId"`
	// Page      string `form:"page"`
	// PageSize  string `form:"pageSize"`
	Cursor    int64 `form:"cursor"`
	Size      int64 `form:"size"`
	Timestamp int64 `form:"timestamp"`
}

// //红包
// type GetRedEnvelopeInfoRequest struct {
// 	GroupId string `form:"groupId"`
// 	TxId    string `form:"txId"`
// 	Address string `form:"address"`
// }
// type GetRedEnvelopeUnusedRequest struct {
// 	GroupId string `form:"groupId"`
// 	TxId    string `form:"txId"`
// 	Address string `form:"address"`
// }

// FetchGroupPersonRequest 获取群组成员信息请求
type FetchGroupPersonRequest struct {
	MetaId  string `form:"metaId"`
	GroupId string `form:"groupId"`
}

// FetchLatestChatInfoListRequest 获取最新聊天信息列表请求（群聊+私聊）
type FetchLatestChatInfoListRequest struct {
	MetaId string `form:"metaId"`
	// Page      string `form:"page"`
	// PageSize  string `form:"pageSize"`
	Cursor    int64 `form:"cursor"`
	Size      int64 `form:"size"`
	Timestamp int64 `form:"timestamp"`
}

// FetchPrivateChatListRequest 获取私聊记录请求
type FetchPrivateChatListRequest struct {
	MetaId      string `form:"metaId"`      // 当前用户MetaId
	OtherMetaId string `form:"otherMetaId"` // 对方用户MetaId
	// Page          string `form:"page"`
	// PageSize      string `form:"pageSize"`
	Cursor    int64 `form:"cursor"`
	Size      int64 `form:"size"`
	Timestamp int64 `form:"timestamp"`
	// TimestampType int64 `form:"timestampType"` //0为查历史，1为查最新
}
