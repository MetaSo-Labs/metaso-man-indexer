package swagger

import (
	"manindexer/basicprotocols/group_chat/api/swagger/docs"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

// SetupSwagger Setup Swagger routes for group chat module
func SetupSwagger(router *gin.Engine) {
	// Manually register Swagger
	docs.RegisterSwagger()
	// Add group chat module Swagger JSON documentation route
	router.GET("/group-chat/api-docs.json", func(c *gin.Context) {
		// Set correct Content-Type
		c.Header("Content-Type", "application/json")

		// Directly read our manually updated JSON file content
		doc := `{
    "swagger": "2.0",
    "info": {
        "description": "Group Chat Service API Documentation, including database queries, group management, community management and other functions",
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
                "description": "Get group list with pagination support",
                "produces": ["application/json"],
                "tags": ["Group Management"],
                "summary": "Get group list",
                "parameters": [
                    {"type": "string", "description": "User MetaId", "name": "metaId", "in": "query", "required": false},
                    {"type": "integer", "description": "Cursor, default is 0", "name": "cursor", "in": "query", "required": false},
                    {"type": "integer", "description": "Page size, default is 20", "name": "size", "in": "query", "required": false},
                    {"type": "integer", "description": "Timestamp", "name": "timestamp", "in": "query", "required": false}
                ],
                "responses": {
                    "200": {"description": "Successfully return group list", "schema": {"type": "object"}}
                }
            }
        },
        "/group-chat/user/latest-group-list": {
            "get": {
                "description": "Get user's latest chat group list, sorted by latest chat time",
                "produces": ["application/json"],
                "tags": ["Group Management"],
                "summary": "Get latest chat group list",
                "parameters": [
                    {"type": "string", "description": "User MetaId", "name": "metaId", "in": "query", "required": true},
                    {"type": "integer", "description": "Cursor, default is 0", "name": "cursor", "in": "query", "required": false},
                    {"type": "integer", "description": "Page size, default is 20", "name": "size", "in": "query", "required": false},
                    {"type": "integer", "description": "Timestamp", "name": "timestamp", "in": "query", "required": false}
                ],
                "responses": {
                    "200": {"description": "Successfully return latest chat group list", "schema": {"type": "object"}},
                    "400": {"description": "Parameter error", "schema": {"type": "object"}}
                }
            }
        },
        "/group-chat/user/latest-chat-info-list": {
            "get": {
                "description": "Get user's latest chat info list, including group chats and private chats, sorted by latest chat time",
                "produces": ["application/json"],
                "tags": ["Group Management"],
                "summary": "Get latest chat info list (group chat + private chat)",
                "parameters": [
                    {"type": "string", "description": "User MetaId", "name": "metaId", "in": "query", "required": true},
                    {"type": "integer", "description": "Cursor, default is 0", "name": "cursor", "in": "query", "required": false},
                    {"type": "integer", "description": "Page size, default is 20", "name": "size", "in": "query", "required": false},
                    {"type": "integer", "description": "Timestamp", "name": "timestamp", "in": "query", "required": false}
                ],
                "responses": {
                    "200": {"description": "Successfully return latest chat info list", "schema": {"type": "object"}},
                    "400": {"description": "Parameter error", "schema": {"type": "object"}},
                    "500": {"description": "Server error", "schema": {"type": "object"}}
                }
            }
        },
        "/group-chat/group-info": {
            "get": {
                "description": "Get detailed information of a specified group",
                "produces": ["application/json"],
                "tags": ["Group Management"],
                "summary": "Get group info",
                "parameters": [
                    {"type": "string", "description": "Group ID", "name": "groupId", "in": "query", "required": true}
                ],
                "responses": {
                    "200": {"description": "Successfully return group information", "schema": {"type": "object"}},
                    "400": {"description": "Parameter error", "schema": {"type": "object"}},
                    "404": {"description": "Group not found", "schema": {"type": "object"}}
                }
            }
        },
        "/group-chat/group-chat-list": {
            "get": {
                "description": "Get chat records of a group, support timestamp pagination",
                "produces": ["application/json"],
                "tags": ["Group Management"],
                "summary": "Get group chat records",
                "parameters": [
                    {"type": "string", "description": "Group ID", "name": "groupId", "in": "query", "required": true},
                    {"type": "string", "description": "User MetaId", "name": "metaId", "in": "query", "required": false},
                    {"type": "integer", "description": "Cursor, default is 0", "name": "cursor", "in": "query", "required": false},
                    {"type": "integer", "description": "Page size, default is 20", "name": "size", "in": "query", "required": false},
                    {"type": "integer", "description": "Timestamp for pagination", "name": "timestamp", "in": "query", "required": false}
                ],
                "responses": {
                    "200": {"description": "Successfully return group chat records", "schema": {"type": "object"}},
                    "400": {"description": "Parameter error", "schema": {"type": "object"}}
                }
            }
        },
        "/group-chat/group-chat-list-v2": {
            "get": {
                "description": "Get chat records of a group using TalkGroupChatTimestamp2Collection with improved key format (groupId_timestamp_pinId)",
                "produces": ["application/json"],
                "tags": ["Group Management"],
                "summary": "Get group chat records (new format)",
                "parameters": [
                    {"type": "string", "description": "Group ID", "name": "groupId", "in": "query", "required": true},
                    {"type": "string", "description": "User MetaId", "name": "metaId", "in": "query", "required": false},
                    {"type": "integer", "description": "Cursor, default is 0", "name": "cursor", "in": "query", "required": false},
                    {"type": "integer", "description": "Page size, default is 20", "name": "size", "in": "query", "required": false},
                    {"type": "integer", "description": "Timestamp for pagination", "name": "timestamp", "in": "query", "required": false}
                ],
                "responses": {
                    "200": {"description": "Successfully return group chat records", "schema": {"type": "object"}},
                    "400": {"description": "Parameter error", "schema": {"type": "object"}}
                }
            }
        },
        "/group-chat/private-chat-list": {
            "get": {
                "description": "Get private chat records between two users, support timestamp pagination",
                "produces": ["application/json"],
                "tags": ["Group Management"],
                "summary": "Get private chat records",
                "parameters": [
                    {"type": "string", "description": "Current user MetaId", "name": "metaId", "in": "query", "required": true},
                    {"type": "string", "description": "Other user MetaId", "name": "otherMetaId", "in": "query", "required": true},
                    {"type": "integer", "description": "Cursor, default is 0", "name": "cursor", "in": "query", "required": false},
                    {"type": "integer", "description": "Page size, default is 20", "name": "size", "in": "query", "required": false},
                    {"type": "integer", "description": "Timestamp for pagination", "name": "timestamp", "in": "query", "required": false}
                ],
                "responses": {
                    "200": {"description": "Successfully return private chat records", "schema": {"type": "object"}},
                    "400": {"description": "Parameter error", "schema": {"type": "object"}},
                    "500": {"description": "Server error", "schema": {"type": "object"}}
                }
            }
        },
        "/group-chat/group-member-list": {
            "get": {
                "description": "Get member list of a group, support pagination",
                "produces": ["application/json"],
                "tags": ["Group Management"],
                "summary": "Get group member list",
                "parameters": [
                    {"type": "string", "description": "Group ID", "name": "groupId", "in": "query", "required": true},
                    {"type": "integer", "description": "Cursor, default is 0", "name": "cursor", "in": "query", "required": false},
                    {"type": "integer", "description": "Page size, default is 20", "name": "size", "in": "query", "required": false},
                    {"type": "integer", "description": "Timestamp", "name": "timestamp", "in": "query", "required": false},
                    {"type": "string", "description": "Order by field, use 'timestamp' for timestamp descending order", "name": "orderBy", "in": "query", "required": false},
                    {"type": "string", "description": "Order type, use 'desc' for descending order", "name": "orderType", "in": "query", "required": false}
                ],
                "responses": {
                    "200": {"description": "Successfully return group member list", "schema": {"type": "object"}},
                    "400": {"description": "Parameter error", "schema": {"type": "object"}}
                }
            }
        },
        "/group-chat/group-person": {
            "get": {
                "description": "Get TalkGroupPersonCollection information based on metaId and groupId to determine if the user is in the specified group",
                "produces": ["application/json"],
                "tags": ["Group Management"],
                "summary": "Get group member info",
                "parameters": [
                    {"type": "string", "description": "User MetaId", "name": "metaId", "in": "query", "required": true},
                    {"type": "string", "description": "Group ID", "name": "groupId", "in": "query", "required": true}
                ],
                "responses": {
                    "200": {"description": "Successfully return group member information", "schema": {"type": "object"}},
                    "400": {"description": "Parameter error", "schema": {"type": "object"}},
                    "500": {"description": "Server error", "schema": {"type": "object"}}
                }
            }
        },
        "/group-chat/lucky-bag-info": {
            "get": {
                "description": "Get lucky bag object and claimed list based on groupId and pinId",
                "produces": ["application/json"],
                "tags": ["Group Management"],
                "summary": "Get lucky bag info",
                "parameters": [
                    {"type": "string", "description": "Group ID", "name": "groupId", "in": "query", "required": true},
                    {"type": "string", "description": "Lucky bag PinId", "name": "pinId", "in": "query", "required": true}
                ],
                "responses": {
                    "200": {"description": "Successfully return lucky bag info", "schema": {"type": "object"}},
                    "400": {"description": "Parameter error", "schema": {"type": "object"}},
                    "500": {"description": "Server error", "schema": {"type": "object"}}
                }
            }
        },
        "/group-chat/lucky-bag-unused-info": {
            "get": {
                "description": "Get lucky bag object and unclaimed list based on groupId and pinId",
                "produces": ["application/json"],
                "tags": ["Group Management"],
                "summary": "Get lucky bag unused info",
                "parameters": [
                    {"type": "string", "description": "Group ID", "name": "groupId", "in": "query", "required": true},
                    {"type": "string", "description": "Lucky bag PinId", "name": "pinId", "in": "query", "required": true}
                ],
                "responses": {
                    "200": {"description": "Successfully return lucky bag unused info", "schema": {"type": "object"}},
                    "400": {"description": "Parameter error", "schema": {"type": "object"}},
                    "500": {"description": "Server error", "schema": {"type": "object"}}
                }
            }
        },
        "/group-chat/grab-lucky-bag": {
            "post": {
                "description": "Grab lucky bag based on groupId, pinId, metaId, and address",
                "consumes": ["application/json"],
                "produces": ["application/json"],
                "tags": ["Group Management"],
                "summary": "Grab lucky bag",
                "parameters": [
                    {
                        "description": "Grab lucky bag request parameters",
                        "name": "request",
                        "in": "body",
                        "required": true,
                        "schema": {
                            "type": "object",
                            "properties": {
                                "groupId": {
                                    "type": "string",
                                    "description": "Group ID"
                                },
                                "pinId": {
                                    "type": "string",
                                    "description": "Lucky bag PinId"
                                },
                                "metaId": {
                                    "type": "string",
                                    "description": "User MetaId"
                                },
                                "address": {
                                    "type": "string",
                                    "description": "User address"
                                }
                            },
                            "required": ["groupId", "pinId", "metaId", "address"]
                        }
                    }
                ],
                "responses": {
                    "200": {"description": "Successfully return grab lucky bag result", "schema": {"type": "object"}},
                    "400": {"description": "Parameter error", "schema": {"type": "object"}},
                    "500": {"description": "Server error", "schema": {"type": "object"}}
                }
            }
        },
        "/group-chat/reclaim-lucky-bag": {
            "post": {
                "description": "Reclaim UTXOs of expired lucky bags remaining for the person who sent the lucky bag",
                "consumes": ["application/json"],
                "produces": ["application/json"],
                "tags": ["Group Management"],
                "summary": "Reclaim lucky bag",
                "parameters": [
                    {
                        "description": "Reclaim lucky bag request parameters",
                        "name": "request",
                        "in": "body",
                        "required": true,
                        "schema": {
                            "type": "object",
                            "properties": {
                                "groupId": {
                                    "type": "string",
                                    "description": "Group ID"
                                },
                                "pinId": {
                                    "type": "string",
                                    "description": "Lucky bag PinId"
                                },
                                "metaId": {
                                    "type": "string",
                                    "description": "User MetaId"
                                },
                                "address": {
                                    "type": "string",
                                    "description": "User address"
                                }
                            },
                            "required": ["groupId", "pinId", "metaId", "address"]
                        }
                    }
                ],
                "responses": {
                    "200": {"description": "Successfully return reclaim lucky bag result", "schema": {"type": "object"}},
                    "400": {"description": "Parameter error", "schema": {"type": "object"}},
                    "500": {"description": "Server error", "schema": {"type": "object"}}
                }
            }
        },
        "/api/db/community/version": {
            "get": {
                "description": "Query TalkCommunityVersionInfoCollection data by communityId or pinId",
                "produces": ["application/json"],
                "tags": ["Database Query"],
                "summary": "Get community version info by communityId or pinId",
                "parameters": [
                    {"type": "string", "description": "Community ID", "name": "communityId", "in": "query", "required": false},
                    {"type": "string", "description": "Pin ID", "name": "pinId", "in": "query", "required": false},
                    {"type": "integer", "description": "Limit count", "name": "limit", "in": "query", "required": false, "default": 10}
                ],
                "responses": {
                    "200": {"description": "Query result", "schema": {"type": "object"}},
                    "400": {"description": "Parameter error", "schema": {"type": "object"}},
                    "500": {"description": "Server error", "schema": {"type": "object"}}
                }
            }
        },
        "/api/db/community/info": {
            "get": {
                "description": "Query TalkCommunityInfoCollection data by communityId",
                "produces": ["application/json"],
                "tags": ["Database Query"],
                "summary": "Get community info by communityId",
                "parameters": [
                    {"type": "string", "description": "Community ID", "name": "communityId", "in": "query", "required": true}
                ],
                "responses": {
                    "200": {"description": "Query result", "schema": {"type": "object"}},
                    "400": {"description": "Parameter error", "schema": {"type": "object"}},
                    "500": {"description": "Server error", "schema": {"type": "object"}}
                }
            }
        },
        "/api/db/community/join": {
            "get": {
                "description": "Query TalkCommunityJoinCollection data by communityId or pinId",
                "produces": ["application/json"],
                "tags": ["Database Query"],
                "summary": "Get community join records by communityId or pinId",
                "parameters": [
                    {"type": "string", "description": "Community ID", "name": "communityId", "in": "query", "required": false},
                    {"type": "string", "description": "Pin ID", "name": "pinId", "in": "query", "required": false},
                    {"type": "integer", "description": "Limit count", "name": "limit", "in": "query", "required": false, "default": 10}
                ],
                "responses": {
                    "200": {"description": "Query result", "schema": {"type": "object"}},
                    "400": {"description": "Parameter error", "schema": {"type": "object"}},
                    "500": {"description": "Server error", "schema": {"type": "object"}}
                }
            }
        },
        "/api/db/community/person": {
            "get": {
                "description": "Query TalkCommunityPersonCollection data by communityId or metaId",
                "produces": ["application/json"],
                "tags": ["Database Query"],
                "summary": "Get community person list by communityId or metaId",
                "parameters": [
                    {"type": "string", "description": "Community ID", "name": "communityId", "in": "query", "required": false},
                    {"type": "string", "description": "Meta ID", "name": "metaId", "in": "query", "required": false},
                    {"type": "integer", "description": "Limit count", "name": "limit", "in": "query", "required": false, "default": 10}
                ],
                "responses": {
                    "200": {"description": "Query result", "schema": {"type": "object"}},
                    "400": {"description": "Parameter error", "schema": {"type": "object"}},
                    "500": {"description": "Server error", "schema": {"type": "object"}}
                }
            }
        },
        "/api/db/group/info": {
            "get": {
                "description": "Query TalkGroupInfoCollection data by groupId",
                "produces": ["application/json"],
                "tags": ["Database Query"],
                "summary": "Get group info by groupId",
                "parameters": [
                    {"type": "string", "description": "Group ID", "name": "groupId", "in": "query", "required": true}
                ],
                "responses": {
                    "200": {"description": "Query result", "schema": {"type": "object"}},
                    "400": {"description": "Parameter error", "schema": {"type": "object"}},
                    "500": {"description": "Server error", "schema": {"type": "object"}}
                }
            }
        },
        "/api/db/group/version": {
            "get": {
                "description": "Query TalkGroupVersionInfoCollection data by groupId or pinId",
                "produces": ["application/json"],
                "tags": ["Database Query"],
                "summary": "Get group version info by groupId or pinId",
                "parameters": [
                    {"type": "string", "description": "Group ID", "name": "groupId", "in": "query", "required": false},
                    {"type": "string", "description": "Pin ID", "name": "pinId", "in": "query", "required": false},
                    {"type": "integer", "description": "Limit count", "name": "limit", "in": "query", "required": false, "default": 10}
                ],
                "responses": {
                    "200": {"description": "Query result", "schema": {"type": "object"}},
                    "400": {"description": "Parameter error", "schema": {"type": "object"}},
                    "500": {"description": "Server error", "schema": {"type": "object"}}
                }
            }
        },
        "/api/db/group/join": {
            "get": {
                "description": "Query TalkGroupJoinCollection data by groupId or pinId",
                "produces": ["application/json"],
                "tags": ["Database Query"],
                "summary": "Get group join records by groupId or pinId",
                "parameters": [
                    {"type": "string", "description": "Group ID", "name": "groupId", "in": "query", "required": false},
                    {"type": "string", "description": "Pin ID", "name": "pinId", "in": "query", "required": false},
                    {"type": "integer", "description": "Limit count", "name": "limit", "in": "query", "required": false, "default": 10}
                ],
                "responses": {
                    "200": {"description": "Query result", "schema": {"type": "object"}},
                    "400": {"description": "Parameter error", "schema": {"type": "object"}},
                    "500": {"description": "Server error", "schema": {"type": "object"}}
                }
            }
        },
        "/api/db/group/person": {
            "get": {
                "description": "Query TalkGroupPersonCollection data by groupId or metaId",
                "produces": ["application/json"],
                "tags": ["Database Query"],
                "summary": "Get group person list by groupId or metaId",
                "parameters": [
                    {"type": "string", "description": "Group ID", "name": "groupId", "in": "query", "required": false},
                    {"type": "string", "description": "Meta ID", "name": "metaId", "in": "query", "required": false},
                    {"type": "integer", "description": "Limit count", "name": "limit", "in": "query", "required": false, "default": 10}
                ],
                "responses": {
                    "200": {"description": "Query result", "schema": {"type": "object"}},
                    "400": {"description": "Parameter error", "schema": {"type": "object"}},
                    "500": {"description": "Server error", "schema": {"type": "object"}}
                }
            }
        },
        "/api/db/chat/queue": {
            "get": {
                "description": "Query TalkGroupChatQueueCollection data by timestamp, or get all data without timestamp",
                "produces": ["application/json"],
                "tags": ["Database Query"],
                "summary": "Get chat queue list by timestamp",
                "parameters": [
                    {"type": "string", "description": "Timestamp", "name": "timestamp", "in": "query", "required": false},
                    {"type": "integer", "description": "Limit count", "name": "limit", "in": "query", "required": false, "default": 10}
                ],
                "responses": {
                    "200": {"description": "Query result", "schema": {"type": "object"}},
                    "400": {"description": "Parameter error", "schema": {"type": "object"}},
                    "500": {"description": "Server error", "schema": {"type": "object"}}
                }
            }
        },
        "/api/db/chat/pin": {
            "get": {
                "description": "Query TalkGroupChatPinCollection data by pinId",
                "produces": ["application/json"],
                "tags": ["Database Query"],
                "summary": "Get chat message by pinId",
                "parameters": [
                    {"type": "string", "description": "Pin ID", "name": "pinId", "in": "query", "required": true}
                ],
                "responses": {
                    "200": {"description": "Query result", "schema": {"type": "object"}},
                    "400": {"description": "Parameter error", "schema": {"type": "object"}},
                    "500": {"description": "Server error", "schema": {"type": "object"}}
                }
            }
        },
        "/api/db/chat/timestamp": {
            "get": {
                "description": "Query TalkGroupChatTimestampCollection data by groupId",
                "produces": ["application/json"],
                "tags": ["Database Query"],
                "summary": "Get chat timestamp list by groupId",
                "parameters": [
                    {"type": "string", "description": "Group ID", "name": "groupId", "in": "query", "required": true},
                    {"type": "integer", "description": "Limit count", "name": "limit", "in": "query", "required": false, "default": 10}
                ],
                "responses": {
                    "200": {"description": "Query result", "schema": {"type": "object"}},
                    "400": {"description": "Parameter error", "schema": {"type": "object"}},
                    "500": {"description": "Server error", "schema": {"type": "object"}}
                }
            }
        },
        "/api/db/user/context": {
            "get": {
                "description": "Query TalkMetaIdContextListCollection data by metaId",
                "produces": ["application/json"],
                "tags": ["Database Query"],
                "summary": "Get user group list by metaId",
                "parameters": [
                    {"type": "string", "description": "Meta ID", "name": "metaId", "in": "query", "required": true}
                ],
                "responses": {
                    "200": {"description": "Query result", "schema": {"type": "object"}},
                    "400": {"description": "Parameter error", "schema": {"type": "object"}},
                    "500": {"description": "Server error", "schema": {"type": "object"}}
                }
            }
        },
        "/api/db/stats": {
            "get": {
                "description": "Get statistics for all database collections",
                "produces": ["application/json"],
                "tags": ["Database Query"],
                "summary": "Get database statistics",
                "responses": {
                    "200": {"description": "Statistics", "schema": {"type": "object"}},
                    "500": {"description": "Server error", "schema": {"type": "object"}}
                }
            }
        },
        "/api/db/collections": {
            "get": {
                "description": "Get all available database collection names",
                "produces": ["application/json"],
                "tags": ["Database Query"],
                "summary": "Get all available database collections",
                "responses": {
                    "200": {"description": "Collection list", "schema": {"type": "object"}}
                }
            }
        },
        "/api/db/migration/info": {
            "get": {
                "description": "Get comprehensive database migration information including current status, supported migrations, and migration history",
                "produces": ["application/json"],
                "tags": ["Database Query"],
                "summary": "Get database migration information",
                "responses": {
                    "200": {"description": "Migration information", "schema": {"type": "object"}},
                    "500": {"description": "Server error", "schema": {"type": "object"}}
                }
            }
        },
        "/api/db/luckybag/statistics": {
            "get": {
                "description": "Get comprehensive lucky bag statistics for a specific group or all groups within a time range",
                "produces": ["application/json"],
                "tags": ["Database Query"],
                "summary": "Get lucky bag statistics by group and time range",
                "parameters": [
                    {"type": "string", "description": "Group ID (leave empty to get statistics for all groups)", "name": "groupId", "in": "query", "required": false},
                    {"type": "integer", "description": "Start timestamp (Unix timestamp)", "name": "startTime", "in": "query", "required": true},
                    {"type": "integer", "description": "End timestamp (Unix timestamp)", "name": "endTime", "in": "query", "required": true}
                ],
                "responses": {
                    "200": {"description": "Lucky bag statistics", "schema": {"type": "object"}},
                    "400": {"description": "Parameter error", "schema": {"type": "object"}},
                    "500": {"description": "Server error", "schema": {"type": "object"}}
                }
            }
        },
        "/api/db/group/version/all": {
            "get": {
                "description": "Get all data of TalkGroupVersionInfoCollection, support pagination",
                "produces": ["application/json"],
                "tags": ["Database Query"],
                "summary": "Get all group version info list (pagination)",
                "parameters": [
                    {"type": "integer", "description": "Page number, starting from 1", "name": "page", "in": "query", "required": false, "default": 1},
                    {"type": "integer", "description": "Number of items per page", "name": "size", "in": "query", "required": false, "default": 20}
                ],
                "responses": {
                    "200": {"description": "Query result", "schema": {"type": "object"}},
                    "400": {"description": "Parameter error", "schema": {"type": "object"}},
                    "500": {"description": "Server error", "schema": {"type": "object"}}
                }
            }
        },
        "/api/db/chat/pin/all": {
            "get": {
                "description": "Get all data of TalkGroupChatPinCollection, support pagination",
                "produces": ["application/json"],
                "tags": ["Database Query"],
                "summary": "Get all chat message list (pagination)",
                "parameters": [
                    {"type": "integer", "description": "Page number, starting from 1", "name": "page", "in": "query", "required": false, "default": 1},
                    {"type": "integer", "description": "Number of items per page", "name": "size", "in": "query", "required": false, "default": 20}
                ],
                "responses": {
                    "200": {"description": "Query result", "schema": {"type": "object"}},
                    "400": {"description": "Parameter error", "schema": {"type": "object"}},
                    "500": {"description": "Server error", "schema": {"type": "object"}}
                }
            }
        },
        "/group-chat/socket/stats": {
            "get": {
                "description": "Get Socket connection statistics including total connections, active connections, etc.",
                "produces": ["application/json"],
                "tags": ["Socket Management"],
                "summary": "Get connection statistics",
                "responses": {
                    "200": {"description": "Connection statistics", "schema": {"type": "object"}},
                    "500": {"description": "Server error", "schema": {"type": "object"}}
                }
            }
        },
        "/group-chat/socket/user-online": {
            "get": {
                "description": "Check if a specific user is currently online based on MetaId",
                "produces": ["application/json"],
                "tags": ["Socket Management"],
                "summary": "Check if user is online",
                "parameters": [
                    {"type": "string", "description": "User MetaId", "name": "metaId", "in": "query", "required": true}
                ],
                "responses": {
                    "200": {"description": "User online status", "schema": {"type": "object"}},
                    "400": {"description": "Parameter error", "schema": {"type": "object"}},
                    "500": {"description": "Server error", "schema": {"type": "object"}}
                }
            }
        }
    },
    "tags": [
        {
            "description": "Database query related APIs for viewing data in Pebble database",
            "name": "Database Query"
        },
        {
            "description": "Group management related APIs, including group information, member management, etc.",
            "name": "Group Management"
        },
        {
            "description": "Socket management related APIs, including connection statistics and user online status",
            "name": "Socket Management"
        }
    ]
}`

		swaggerContent := doc

		c.Data(200, "application/json", []byte(swaggerContent))
	})

	// Add group chat module Swagger documentation route
	router.GET("/group-chat/docs/*any", ginSwagger.WrapHandler(swaggerFiles.Handler, ginSwagger.URL("/group-chat/api-docs.json")))
}

// GetSwaggerURL Get swagger documentation URL
func GetSwaggerURL(host, port string) string {
	return "http://" + host + ":" + port + "/group-chat/docs/index.html"
}
