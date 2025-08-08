package respond

import "manindexer/basicprotocols/group_chat/models"

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
	RoomAvatarUrl       string `json:"roomAvatarUrl"`       //房间头像url
	RoomNinePersonHash  string `json:"roomNinePersonHash"`  //房间前9位人员的metaId总hash值
	RoomNewestTxId      string `json:"roomNewestTxId"`      //房间最新聊天内容的txId
	RoomNewestPinId     string `json:"roomNewestPinId"`     //房间最新聊天内容的pinId
	RoomNewestMetaId    string `json:"roomNewestMetaId"`    //房间最新聊天内容的MetaId
	RoomNewestUserName  string `json:"roomNewestUserName"`  //房间最新聊天内容的MetaId
	RoomNewestProtocol  string `json:"roomNewestProtocol"`  //房间最新聊天内容的协议类型
	RoomNewestContent   string `json:"roomNewestContent"`   //房间最新聊天内容
	RoomNewestTimestamp int64  `json:"roomNewestTimestamp"` //房间最新聊天的时间戳
	CreateUserMetaId    string `json:"createUserMetaId"`    //创建人的metaId
	UserCount           int64  `json:"userCount"`           //房间人数
	ChatSettingType     int64  `json:"chatSettingType"`     //用于设置发言限制， 0-所有人，1-管理员
	DeleteStatus        int64  `json:"deleteStatus"`        //删除状态，0-正常，1-删除
	Timestamp           int64  `json:"timestamp"`           //创建你房间的时间戳
}

type GroupChatResponse struct {
	Total         int64            `json:"total"`
	NextTimestamp int64            `json:"nextTimestamp"`
	List          []*GroupChatItem `json:"list"`
}

type GroupChatItem struct {
	GroupId   string `json:"groupId"`   //房间ID 唯一
	MetanetId string `json:"metanetId"` //
	TxId      string `json:"txId"`
	MetaId    string `json:"metaId"`
	// UserInfo    *model.UserInfoResp `json:"userInfo"`
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
	ReplyTx     string          `json:"replyTx"`
	ReplyInfo   *ReplyInfo      `json:"replyInfo"`
	RedMetaId   string          `json:"redMetaId"`
	Timestamp   int64           `json:"timestamp"` //聊天记录时间戳
	Params      string          `json:"params"`    //通用字段，便于后续新增参数
}

type ReplyInfo struct {
	TxId   string `json:"txId"`
	MetaId string `json:"metaId"`
	// UserInfo    *models.UserInfoResp `json:"userInfo"`
	NickName    string          `json:"nickName"`
	Protocol    string          `json:"protocol"`
	Content     string          `json:"content"`
	ContentType string          `json:"contentType"`
	Encryption  string          `json:"encryption"`
	ChatType    models.ChatType `json:"chatType"`  //0-msg, 1-red, 2-img
	Timestamp   int64           `json:"timestamp"` //聊天记录时间戳
}

type GroupMemberResponse struct {
	Total int64              `json:"total"`
	List  []*GroupMemberItem `json:"list"`
}

type GroupMemberItem struct {
	MetaId    string `json:"metaId"`
	Name      string `json:"name"`
	Address   string `json:"address"`
	TimeStr   string `json:"timeStr"`
	Timestamp int64  `json:"timestamp"`
}
type RoomPersonItem struct {
	GroupId   string `json:"groupId"`   //房间ID 唯一
	MetanetId string `json:"metanetId"` //房间ID 唯一
	MetaId    string `json:"metaId"`    //用户的MetaId
	Address   string `json:"address"`   //用户的00地址
	// UserInfo    *model.UserInfoResp `json:"userInfo"`
	Name        string `json:"name"`        //用户名称
	AvatarTxId  string `json:"avatarTxId"`  //用户当前头像
	AvatarImage string `json:"avatarImage"` //用户当前头像路由
	// AvatarType  model.AvatarType `json:"avatarType"`  //用户当前头像类型
	RoomState models.RoomState `json:"roomState"`
	Timestamp int64            `json:"timestamp"` //加入或离开房间的时间戳
}

type GroupInfoResponse struct {
	CommunityId           string `json:"communityId"`           //社区Id 唯一
	GroupId               string `json:"groupId"`               //房间ID 唯一
	MetanetId             string `json:"metanetId"`             //房间ID 唯一
	TxId                  string `json:"txId"`                  //房间的TxId
	RoomPublicKey         string `json:"roomPublicKey"`         //房间公钥
	RoomName              string `json:"roomName"`              //创建房间的名称
	RoomNote              string `json:"roomNote"`              //创建房间的公告
	RoomType              string `json:"roomType"`              //创建房间的类型 ”1“不加密 “2”加密 加密采用AES加密算法
	RoomStatus            string `json:"roomStatus"`            //"1" 未加密时为“1” 加密时为加密后的信息, 保留字段
	RoomJoinType          string `json:"roomJoinType"`          //加入方式，1为密码，2为nft
	RoomCodeHash          string `json:"roomCodeHash"`          //roomJoinType为2时有值，codeHash
	RoomGenesis           string `json:"roomGenesis"`           //roomJoinType为2时有值，genesis
	RoomLimitAmount       int64  `json:"roomLimitAmount"`       //roomJoinType为2时有值，token的限制
	RoomGenesisSeriesName string `json:"roomGenesisSeriesName"` //
	RoomAvatarUrl         string `json:"roomAvatarUrl"`         //房间头像url
	RoomNinePersonHash    string `json:"roomNinePersonHash"`    //房间前9位人员的metaId总hash值
	RoomNewestTxId        string `json:"roomNewestTxId"`        //房间最新聊天内容的txId
	RoomNewestMetaId      string `json:"roomNewestMetaId"`      //房间最新聊天内容的MetaId
	RoomNewestUserName    string `json:"roomNewestUserName"`    //房间最新聊天内容的MetaId
	RoomNewestProtocol    string `json:"roomNewestProtocol"`    //房间最新聊天内容的协议类型
	RoomNewestContent     string `json:"roomNewestContent"`     //房间最新聊天内容
	RoomNewestTimestamp   int64  `json:"roomNewestTimestamp"`   //房间最新聊天的时间戳
	CreateUserMetaId      string `json:"createUserMetaId"`      //创建人的metaId
	// CreateUserInfo        *model.UserInfoResp `json:"createUserInfo"`
	CreateUserName        string `json:"createUserName"`        //创建人的metaId
	CreateUserAvatarTxId  string `json:"createUserAvatarTxId"`  //创建人的metaId
	CreateUserAvatarImage string `json:"createUserAvatarImage"` //创建人的头像路由
	// CreateUserAvatarType  model.AvatarType    `json:"createUserAvatarType"`  //创建人的metaId
	UserCount       int64 `json:"userCount"`       //房间人数
	ChatSettingType int64 `json:"chatSettingType"` //用于设置发言限制， 0-所有人，1-管理员
	DeleteStatus    int64 `json:"deleteStatus"`    //删除状态，0-正常，1-删除
	Timestamp       int64 `json:"timestamp"`       //创建你房间的时间戳
}
