package swagger

import (
	"manindexer/basicprotocols/group_chat/api/swagger/docs"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

// SetupSwagger 设置群聊模块的Swagger路由
func SetupSwagger(router *gin.Engine) {
	// 手动注册 Swagger
	docs.RegisterSwagger()
	// 添加群聊模块的Swagger JSON文档路由
	router.GET("/group-chat/api-docs.json", func(c *gin.Context) {
		// 设置正确的Content-Type
		c.Header("Content-Type", "application/json")

		// 直接读取我们手动更新的 JSON 文件内容
		doc := `{
    "swagger": "2.0",
    "info": {
        "description": "群聊服务 API 文档，包含数据库查询、群组管理、社区管理等功能",
        "title": "Group Chat API",
        "termsOfService": "http://swagger.io/terms/",
        "contact": {
            "name": "API Support",
            "url": "http://www.swagger.io/support",
            "email": "support@swagger.io"
        },
        "license": {
            "name": "Apache 2.0",
            "url": "http://www.apache.org/licenses/LICENSE-2.0.html"
        },
        "version": "1.0"
    },
    "host": "` + c.Request.Host + `",
    "basePath": "/",
    "paths": {
        "/group-chat/group-list": {
            "get": {
                "description": "获取群组列表，支持分页",
                "produces": ["application/json"],
                "tags": ["群组管理"],
                "summary": "获取群组列表",
                "parameters": [
                    {"type": "string", "description": "用户MetaId", "name": "metaId", "in": "query", "required": false},
                    {"type": "integer", "description": "游标，默认为1", "name": "cursor", "in": "query", "required": false},
                    {"type": "integer", "description": "每页大小，默认为20", "name": "size", "in": "query", "required": false},
                    {"type": "integer", "description": "时间戳", "name": "timestamp", "in": "query", "required": false}
                ],
                "responses": {
                    "200": {"description": "成功返回群组列表", "schema": {"type": "object"}}
                }
            }
        },
        "/group-chat/user/latest-group-list": {
            "get": {
                "description": "获取用户的最新聊天群组列表，基于最新聊天时间排序",
                "produces": ["application/json"],
                "tags": ["群组管理"],
                "summary": "获取最新聊天群组列表",
                "parameters": [
                    {"type": "string", "description": "用户MetaId", "name": "metaId", "in": "query", "required": true},
                    {"type": "integer", "description": "游标，默认为1", "name": "cursor", "in": "query", "required": false},
                    {"type": "integer", "description": "每页大小，默认为20", "name": "size", "in": "query", "required": false},
                    {"type": "integer", "description": "时间戳", "name": "timestamp", "in": "query", "required": false}
                ],
                "responses": {
                    "200": {"description": "成功返回最新聊天群组列表", "schema": {"type": "object"}},
                    "400": {"description": "参数错误", "schema": {"type": "object"}}
                }
            }
        },
        "/group-chat/user/latest-chat-info-list": {
            "get": {
                "description": "获取用户的最新聊天信息列表，包括群聊和私聊，基于最新聊天时间排序",
                "produces": ["application/json"],
                "tags": ["群组管理"],
                "summary": "获取最新聊天信息列表（群聊+私聊）",
                "parameters": [
                    {"type": "string", "description": "用户MetaId", "name": "metaId", "in": "query", "required": true},
                    {"type": "integer", "description": "游标，默认为1", "name": "cursor", "in": "query", "required": false},
                    {"type": "integer", "description": "每页大小，默认为20", "name": "size", "in": "query", "required": false},
                    {"type": "integer", "description": "时间戳", "name": "timestamp", "in": "query", "required": false}
                ],
                "responses": {
                    "200": {"description": "成功返回最新聊天信息列表", "schema": {"type": "object"}},
                    "400": {"description": "参数错误", "schema": {"type": "object"}},
                    "500": {"description": "服务器错误", "schema": {"type": "object"}}
                }
            }
        },
        "/group-chat/group-info": {
            "get": {
                "description": "获取指定群组的详细信息",
                "produces": ["application/json"],
                "tags": ["群组管理"],
                "summary": "获取群组信息",
                "parameters": [
                    {"type": "string", "description": "群组ID", "name": "groupId", "in": "query", "required": true}
                ],
                "responses": {
                    "200": {"description": "成功返回群组信息", "schema": {"type": "object"}},
                    "400": {"description": "参数错误", "schema": {"type": "object"}},
                    "404": {"description": "群组不存在", "schema": {"type": "object"}}
                }
            }
        },
        "/group-chat/group-chat-list": {
            "get": {
                "description": "获取群组的聊天记录，支持时间戳分页",
                "produces": ["application/json"],
                "tags": ["群组管理"],
                "summary": "获取群组聊天记录",
                "parameters": [
                    {"type": "string", "description": "群组ID", "name": "groupId", "in": "query", "required": true},
                    {"type": "string", "description": "用户MetaId", "name": "metaId", "in": "query", "required": false},
                    {"type": "integer", "description": "游标，默认为0", "name": "cursor", "in": "query", "required": false},
                    {"type": "integer", "description": "每页大小，默认为20", "name": "size", "in": "query", "required": false},
                    {"type": "integer", "description": "时间戳，用于分页", "name": "timestamp", "in": "query", "required": false}
                ],
                "responses": {
                    "200": {"description": "成功返回群组聊天记录", "schema": {"type": "object"}},
                    "400": {"description": "参数错误", "schema": {"type": "object"}}
                }
            }
        },
        "/group-chat/private-chat-list": {
            "get": {
                "description": "获取两个用户之间的私聊记录，支持时间戳分页",
                "produces": ["application/json"],
                "tags": ["群组管理"],
                "summary": "获取私聊记录",
                "parameters": [
                    {"type": "string", "description": "当前用户MetaId", "name": "metaId", "in": "query", "required": true},
                    {"type": "string", "description": "对方用户MetaId", "name": "otherMetaId", "in": "query", "required": true},
                    {"type": "integer", "description": "游标，默认为0", "name": "cursor", "in": "query", "required": false},
                    {"type": "integer", "description": "每页大小，默认为20", "name": "size", "in": "query", "required": false},
                    {"type": "integer", "description": "时间戳，用于分页", "name": "timestamp", "in": "query", "required": false}
                ],
                "responses": {
                    "200": {"description": "成功返回私聊记录", "schema": {"type": "object"}},
                    "400": {"description": "参数错误", "schema": {"type": "object"}},
                    "500": {"description": "服务器错误", "schema": {"type": "object"}}
                }
            }
        },
        "/group-chat/group-member-list": {
            "get": {
                "description": "获取群组的成员列表，支持分页",
                "produces": ["application/json"],
                "tags": ["群组管理"],
                "summary": "获取群组成员列表",
                "parameters": [
                    {"type": "string", "description": "群组ID", "name": "groupId", "in": "query", "required": true},
                    {"type": "integer", "description": "游标，默认为1", "name": "cursor", "in": "query", "required": false},
                    {"type": "integer", "description": "每页大小，默认为20", "name": "size", "in": "query", "required": false},
                    {"type": "integer", "description": "时间戳", "name": "timestamp", "in": "query", "required": false}
                ],
                "responses": {
                    "200": {"description": "成功返回群组成员列表", "schema": {"type": "object"}},
                    "400": {"description": "参数错误", "schema": {"type": "object"}}
                }
            }
        },
        "/group-chat/group-person": {
            "get": {
                "description": "根据metaId和groupId获取TalkGroupPersonCollection信息，判断用户是否在指定群组中",
                "produces": ["application/json"],
                "tags": ["群组管理"],
                "summary": "获取群组成员信息",
                "parameters": [
                    {"type": "string", "description": "用户MetaId", "name": "metaId", "in": "query", "required": true},
                    {"type": "string", "description": "群组ID", "name": "groupId", "in": "query", "required": true}
                ],
                "responses": {
                    "200": {"description": "成功返回群组成员信息", "schema": {"type": "object"}},
                    "400": {"description": "参数错误", "schema": {"type": "object"}},
                    "500": {"description": "服务器错误", "schema": {"type": "object"}}
                }
            }
        },
        "/api/db/community/version": {
            "get": {
                "description": "根据communityId或pinId查询TalkCommunityVersionInfoCollection数据",
                "produces": ["application/json"],
                "tags": ["数据库查询"],
                "summary": "根据communityId或pinId获取社区版本信息",
                "parameters": [
                    {"type": "string", "description": "社区ID", "name": "communityId", "in": "query", "required": false},
                    {"type": "string", "description": "PinID", "name": "pinId", "in": "query", "required": false},
                    {"type": "integer", "description": "限制数量", "name": "limit", "in": "query", "required": false, "default": 10}
                ],
                "responses": {
                    "200": {"description": "查询结果", "schema": {"type": "object"}},
                    "400": {"description": "参数错误", "schema": {"type": "object"}},
                    "500": {"description": "服务器错误", "schema": {"type": "object"}}
                }
            }
        },
        "/api/db/community/info": {
            "get": {
                "description": "根据communityId查询TalkCommunityInfoCollection数据",
                "produces": ["application/json"],
                "tags": ["数据库查询"],
                "summary": "根据communityId获取社区信息",
                "parameters": [
                    {"type": "string", "description": "社区ID", "name": "communityId", "in": "query", "required": true}
                ],
                "responses": {
                    "200": {"description": "查询结果", "schema": {"type": "object"}},
                    "400": {"description": "参数错误", "schema": {"type": "object"}},
                    "500": {"description": "服务器错误", "schema": {"type": "object"}}
                }
            }
        },
        "/api/db/community/join": {
            "get": {
                "description": "根据communityId或pinId查询TalkCommunityJoinCollection数据",
                "produces": ["application/json"],
                "tags": ["数据库查询"],
                "summary": "根据communityId或pinId获取社区加入记录",
                "parameters": [
                    {"type": "string", "description": "社区ID", "name": "communityId", "in": "query", "required": false},
                    {"type": "string", "description": "PinID", "name": "pinId", "in": "query", "required": false},
                    {"type": "integer", "description": "限制数量", "name": "limit", "in": "query", "required": false, "default": 10}
                ],
                "responses": {
                    "200": {"description": "查询结果", "schema": {"type": "object"}},
                    "400": {"description": "参数错误", "schema": {"type": "object"}},
                    "500": {"description": "服务器错误", "schema": {"type": "object"}}
                }
            }
        },
        "/api/db/community/person": {
            "get": {
                "description": "根据communityId或metaId查询TalkCommunityPersonCollection数据",
                "produces": ["application/json"],
                "tags": ["数据库查询"],
                "summary": "根据communityId或metaId获取社区成员列表",
                "parameters": [
                    {"type": "string", "description": "社区ID", "name": "communityId", "in": "query", "required": false},
                    {"type": "string", "description": "MetaID", "name": "metaId", "in": "query", "required": false},
                    {"type": "integer", "description": "限制数量", "name": "limit", "in": "query", "required": false, "default": 10}
                ],
                "responses": {
                    "200": {"description": "查询结果", "schema": {"type": "object"}},
                    "400": {"description": "参数错误", "schema": {"type": "object"}},
                    "500": {"description": "服务器错误", "schema": {"type": "object"}}
                }
            }
        },
        "/api/db/group/info": {
            "get": {
                "description": "根据groupId查询TalkGroupInfoCollection数据",
                "produces": ["application/json"],
                "tags": ["数据库查询"],
                "summary": "根据groupId获取群组信息",
                "parameters": [
                    {"type": "string", "description": "群组ID", "name": "groupId", "in": "query", "required": true}
                ],
                "responses": {
                    "200": {"description": "查询结果", "schema": {"type": "object"}},
                    "400": {"description": "参数错误", "schema": {"type": "object"}},
                    "500": {"description": "服务器错误", "schema": {"type": "object"}}
                }
            }
        },
        "/api/db/group/version": {
            "get": {
                "description": "根据groupId或pinId查询TalkGroupVersionInfoCollection数据",
                "produces": ["application/json"],
                "tags": ["数据库查询"],
                "summary": "根据groupId或pinId获取群组版本信息",
                "parameters": [
                    {"type": "string", "description": "群组ID", "name": "groupId", "in": "query", "required": false},
                    {"type": "string", "description": "PinID", "name": "pinId", "in": "query", "required": false},
                    {"type": "integer", "description": "限制数量", "name": "limit", "in": "query", "required": false, "default": 10}
                ],
                "responses": {
                    "200": {"description": "查询结果", "schema": {"type": "object"}},
                    "400": {"description": "参数错误", "schema": {"type": "object"}},
                    "500": {"description": "服务器错误", "schema": {"type": "object"}}
                }
            }
        },
        "/api/db/group/join": {
            "get": {
                "description": "根据groupId或pinId查询TalkGroupJoinCollection数据",
                "produces": ["application/json"],
                "tags": ["数据库查询"],
                "summary": "根据groupId或pinId获取群组加入记录",
                "parameters": [
                    {"type": "string", "description": "群组ID", "name": "groupId", "in": "query", "required": false},
                    {"type": "string", "description": "PinID", "name": "pinId", "in": "query", "required": false},
                    {"type": "integer", "description": "限制数量", "name": "limit", "in": "query", "required": false, "default": 10}
                ],
                "responses": {
                    "200": {"description": "查询结果", "schema": {"type": "object"}},
                    "400": {"description": "参数错误", "schema": {"type": "object"}},
                    "500": {"description": "服务器错误", "schema": {"type": "object"}}
                }
            }
        },
        "/api/db/group/person": {
            "get": {
                "description": "根据groupId或metaId查询TalkGroupPersonCollection数据",
                "produces": ["application/json"],
                "tags": ["数据库查询"],
                "summary": "根据groupId或metaId获取群组成员列表",
                "parameters": [
                    {"type": "string", "description": "群组ID", "name": "groupId", "in": "query", "required": false},
                    {"type": "string", "description": "MetaID", "name": "metaId", "in": "query", "required": false},
                    {"type": "integer", "description": "限制数量", "name": "limit", "in": "query", "required": false, "default": 10}
                ],
                "responses": {
                    "200": {"description": "查询结果", "schema": {"type": "object"}},
                    "400": {"description": "参数错误", "schema": {"type": "object"}},
                    "500": {"description": "服务器错误", "schema": {"type": "object"}}
                }
            }
        },
        "/api/db/chat/queue": {
            "get": {
                "description": "根据timestamp查询TalkGroupChatQueueCollection数据，或不传timestamp获取所有数据",
                "produces": ["application/json"],
                "tags": ["数据库查询"],
                "summary": "根据timestamp获取聊天队列列表",
                "parameters": [
                    {"type": "string", "description": "时间戳", "name": "timestamp", "in": "query", "required": false},
                    {"type": "integer", "description": "限制数量", "name": "limit", "in": "query", "required": false, "default": 10}
                ],
                "responses": {
                    "200": {"description": "查询结果", "schema": {"type": "object"}},
                    "400": {"description": "参数错误", "schema": {"type": "object"}},
                    "500": {"description": "服务器错误", "schema": {"type": "object"}}
                }
            }
        },
        "/api/db/chat/pin": {
            "get": {
                "description": "根据pinId查询TalkGroupChatPinCollection数据",
                "produces": ["application/json"],
                "tags": ["数据库查询"],
                "summary": "根据pinId获取聊天消息",
                "parameters": [
                    {"type": "string", "description": "PinID", "name": "pinId", "in": "query", "required": true}
                ],
                "responses": {
                    "200": {"description": "查询结果", "schema": {"type": "object"}},
                    "400": {"description": "参数错误", "schema": {"type": "object"}},
                    "500": {"description": "服务器错误", "schema": {"type": "object"}}
                }
            }
        },
        "/api/db/chat/timestamp": {
            "get": {
                "description": "根据groupId查询TalkGroupChatTimestampCollection数据",
                "produces": ["application/json"],
                "tags": ["数据库查询"],
                "summary": "根据groupId获取聊天时间戳列表",
                "parameters": [
                    {"type": "string", "description": "群组ID", "name": "groupId", "in": "query", "required": true},
                    {"type": "integer", "description": "限制数量", "name": "limit", "in": "query", "required": false, "default": 10}
                ],
                "responses": {
                    "200": {"description": "查询结果", "schema": {"type": "object"}},
                    "400": {"description": "参数错误", "schema": {"type": "object"}},
                    "500": {"description": "服务器错误", "schema": {"type": "object"}}
                }
            }
        },
        "/api/db/user/context": {
            "get": {
                "description": "根据metaId查询TalkMetaIdContextListCollection数据",
                "produces": ["application/json"],
                "tags": ["数据库查询"],
                "summary": "根据metaId获取用户群列表",
                "parameters": [
                    {"type": "string", "description": "MetaID", "name": "metaId", "in": "query", "required": true}
                ],
                "responses": {
                    "200": {"description": "查询结果", "schema": {"type": "object"}},
                    "400": {"description": "参数错误", "schema": {"type": "object"}},
                    "500": {"description": "服务器错误", "schema": {"type": "object"}}
                }
            }
        },
        "/api/db/stats": {
            "get": {
                "description": "获取所有数据库集合的统计信息",
                "produces": ["application/json"],
                "tags": ["数据库查询"],
                "summary": "获取数据库统计信息",
                "responses": {
                    "200": {"description": "统计信息", "schema": {"type": "object"}},
                    "500": {"description": "服务器错误", "schema": {"type": "object"}}
                }
            }
        },
        "/api/db/collections": {
            "get": {
                "description": "获取所有可用的数据库集合名称",
                "produces": ["application/json"],
                "tags": ["数据库查询"],
                "summary": "获取所有可用的数据库集合",
                "responses": {
                    "200": {"description": "集合列表", "schema": {"type": "object"}}
                }
            }
        },
        "/api/db/group/version/all": {
            "get": {
                "description": "获取TalkGroupVersionInfoCollection的所有数据，支持分页",
                "produces": ["application/json"],
                "tags": ["数据库查询"],
                "summary": "获取所有群组版本信息列表（分页）",
                "parameters": [
                    {"type": "integer", "description": "页码，从1开始", "name": "page", "in": "query", "required": false, "default": 1},
                    {"type": "integer", "description": "每页数量", "name": "size", "in": "query", "required": false, "default": 20}
                ],
                "responses": {
                    "200": {"description": "查询结果", "schema": {"type": "object"}},
                    "400": {"description": "参数错误", "schema": {"type": "object"}},
                    "500": {"description": "服务器错误", "schema": {"type": "object"}}
                }
            }
        },
        "/api/db/chat/pin/all": {
            "get": {
                "description": "获取TalkGroupChatPinCollection的所有数据，支持分页",
                "produces": ["application/json"],
                "tags": ["数据库查询"],
                "summary": "获取所有聊天消息列表（分页）",
                "parameters": [
                    {"type": "integer", "description": "页码，从1开始", "name": "page", "in": "query", "required": false, "default": 1},
                    {"type": "integer", "description": "每页数量", "name": "size", "in": "query", "required": false, "default": 20}
                ],
                "responses": {
                    "200": {"description": "查询结果", "schema": {"type": "object"}},
                    "400": {"description": "参数错误", "schema": {"type": "object"}},
                    "500": {"description": "服务器错误", "schema": {"type": "object"}}
                }
            }
        }
    },
    "tags": [
        {
            "description": "数据库查询相关API，用于查看Pebble数据库中的数据",
            "name": "数据库查询"
        },
        {
            "description": "群组管理相关API，包括群组信息、成员管理等",
            "name": "群组管理"
        },
        {
            "description": "社区管理相关API，包括社区信息、成员管理等",
            "name": "社区管理"
        },
        {
            "description": "聊天功能相关API，包括消息、队列等",
            "name": "聊天功能"
        },
        {
            "description": "用户管理相关API，包括用户信息、群列表等",
            "name": "用户管理"
        }
    ]
}`

		swaggerContent := doc

		c.Data(200, "application/json", []byte(swaggerContent))
	})

	// 添加群聊模块的Swagger文档路由
	router.GET("/group-chat/docs/*any", ginSwagger.WrapHandler(swaggerFiles.Handler, ginSwagger.URL("/group-chat/api-docs.json")))
}

// GetSwaggerURL 获取swagger文档的URL
func GetSwaggerURL(host, port string) string {
	return "http://" + host + ":" + port + "/group-chat/docs/index.html"
}
