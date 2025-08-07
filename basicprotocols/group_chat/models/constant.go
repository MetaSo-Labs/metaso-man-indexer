package models

type RoomState int64

const (
	RoomStateIn  RoomState = 1
	RoomStateOut RoomState = -1
)

type ChatType int64

const (
	ChatTypeMsg        ChatType = 0  //群聊 - 信息类型
	ChatTypeEmoji      ChatType = 6  //群聊 - 表情类型
	ChatTypeRed        ChatType = 1  //群聊 - 红包类型
	ChatTypeOpenRed    ChatType = 2  //群聊 - 抢红包类型
	ChatTypeRecycleRed ChatType = 22 //群聊 - 回收红包类型
	ChatTypeFile       ChatType = 3  //群聊 - 文件类型
	ChatTypeJoin       ChatType = 4  //群聊 - 加入类型
	ChatTypeLeave      ChatType = 5  //群聊 - 离开类型
)
