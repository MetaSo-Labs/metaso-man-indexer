package swagger

import (
	"manindexer/basicprotocols/group_chat/api/swagger/docs"
	"strings"

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

		// Detect proxy path prefix from request headers or URL
		basePath := "/"
		host := c.Request.Host

		// Check for common proxy headers
		if xForwardedPath := c.GetHeader("X-Forwarded-Path"); xForwardedPath != "" {
			basePath = xForwardedPath + "/"
		} else if xOriginalURI := c.GetHeader("X-Original-URI"); xOriginalURI != "" {
			// Extract base path from X-Original-URI
			// Example: /chat-api-test/group-chat/api-docs.json -> /chat-api-test/
			if len(xOriginalURI) > 0 && xOriginalURI != "/group-chat/api-docs.json" {
				// Find the position where /group-chat/api-docs.json starts
				if idx := len(xOriginalURI) - len("/group-chat/api-docs.json"); idx > 0 {
					basePath = xOriginalURI[:idx] + "/"
				}
			}
		} else {
			// Try to detect from Referer header
			if referer := c.GetHeader("Referer"); referer != "" {
				// Example: https://www.show.now/chat-api-test/group-chat/docs/index.html
				if idx := strings.Index(referer, "/group-chat/docs/"); idx != -1 {
					// Extract everything before /group-chat/docs/
					if protocolIdx := strings.Index(referer, "://"); protocolIdx != -1 {
						hostStart := protocolIdx + 3
						if hostEnd := strings.Index(referer[hostStart:], "/"); hostEnd != -1 {
							pathStart := hostStart + hostEnd
							basePath = referer[pathStart:idx] + "/"
							if basePath == "/" {
								basePath = "/"
							}
						}
					}
				}
			}
		}

		// Directly read our manually updated JSON file content with dynamic path adjustment
		docTemplate := `{
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
    "host": "` + host + `",
    "basePath": "` + basePath + `",
    "paths": {
        "group-chat/group-list": {
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
        "/group-chat/group-chat-list-v3": {
            "get": {
                "description": "Get chat records of a group using GetChatsByGroupIdAndTimestampRange3 (test version with IterOptions for improved performance)",
                "produces": ["application/json"],
                "tags": ["Group Management"],
                "summary": "Get group chat records (test version with IterOptions)",
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
        "/group-chat/group-chat-list-by-index": {
            "get": {
                "description": "Get group chat records by index range (ascending order) using TalkGroupChatIndexCollection",
                "produces": ["application/json"],
                "tags": ["Group Management"],
                "summary": "Get group chat records by index range",
                "parameters": [
                    {"type": "string", "description": "Group ID", "name": "groupId", "in": "query", "required": true},
                    {"type": "integer", "description": "Start index for pagination, default is 0", "name": "startIndex", "in": "query", "required": false},
                    {"type": "integer", "description": "Page size, default is 20", "name": "size", "in": "query", "required": false}
                ],
                "responses": {
                    "200": {
                        "description": "Successfully return group chat records by index", 
                        "schema": {
                            "type": "object",
                            "properties": {
                                "code": {"type": "integer", "description": "Response code"},
                                "message": {"type": "string", "description": "Response message"},
                                "data": {"$ref": "#/definitions/GroupChatResponse"},
                                "timestamp": {"type": "integer", "description": "Response timestamp"}
                            }
                        }
                    },
                    "400": {"description": "Parameter error", "schema": {"$ref": "#/definitions/Message"}},
                    "500": {"description": "Server error", "schema": {"$ref": "#/definitions/Message"}}
                }
            }
        },
        "/group-chat/group-chat-list-by-start-time": {
            "get": {
                "description": "Get group chat records by start timestamp range (ascending order) using TalkGroupChatTimestamp2Collection",
                "produces": ["application/json"],
                "tags": ["Group Management"],
                "summary": "Get group chat records by start timestamp range",
                "parameters": [
                    {"type": "string", "description": "Group ID", "name": "groupId", "in": "query", "required": true},
                    {"type": "integer", "description": "Start timestamp for pagination, default is 0", "name": "startTimestamp", "in": "query", "required": false},
                    {"type": "integer", "description": "Page size, default is 20", "name": "size", "in": "query", "required": false}
                ],
                "responses": {
                    "200": {
                        "description": "Successfully return group chat records by start timestamp", 
                        "schema": {
                            "type": "object",
                            "properties": {
                                "code": {"type": "integer", "description": "Response code"},
                                "message": {"type": "string", "description": "Response message"},
                                "data": {"$ref": "#/definitions/GroupChatResponse"},
                                "timestamp": {"type": "integer", "description": "Response timestamp"}
                            }
                        }
                    },
                    "400": {"description": "Parameter error", "schema": {"$ref": "#/definitions/Message"}},
                    "500": {"description": "Server error", "schema": {"$ref": "#/definitions/Message"}}
                }
            }
        },
        "/group-chat/channel-chat-list-v3": {
            "get": {
                "description": "Get channel chat records using GetChatsByChannelIdAndEndTimestampRange3 (test version with IterOptions for improved performance)",
                "produces": ["application/json"],
                "tags": ["Group Management"],
                "summary": "Get channel chat records (V3 with improved performance)",
                "parameters": [
                    {"type": "string", "description": "Channel ID", "name": "channelId", "in": "query", "required": true},
                    {"type": "string", "description": "User MetaId", "name": "metaId", "in": "query", "required": false},
                    {"type": "integer", "description": "Cursor, default is 0", "name": "cursor", "in": "query", "required": false},
                    {"type": "integer", "description": "Page size, default is 20", "name": "size", "in": "query", "required": false},
                    {"type": "integer", "description": "Timestamp for pagination", "name": "timestamp", "in": "query", "required": false}
                ],
                "responses": {
                    "200": {"description": "Successfully return channel chat records", "schema": {"type": "object"}},
                    "400": {"description": "Parameter error", "schema": {"type": "object"}},
                    "500": {"description": "Server error", "schema": {"type": "object"}}
                }
            }
        },
        "/group-chat/channel-chat-list-by-index": {
            "get": {
                "description": "Get channel chat records by index range (ascending order) using TalkGroupChannelChatIndexCollection",
                "produces": ["application/json"],
                "tags": ["Group Management"],
                "summary": "Get channel chat records by index range",
                "parameters": [
                    {"type": "string", "description": "Channel ID", "name": "channelId", "in": "query", "required": true},
                    {"type": "integer", "description": "Start index for pagination, default is 0", "name": "startIndex", "in": "query", "required": false},
                    {"type": "integer", "description": "Page size, default is 20", "name": "size", "in": "query", "required": false}
                ],
                "responses": {
                    "200": {
                        "description": "Successfully return channel chat records by index", 
                        "schema": {
                            "type": "object",
                            "properties": {
                                "code": {"type": "integer", "description": "Response code"},
                                "message": {"type": "string", "description": "Response message"},
                                "data": {"$ref": "#/definitions/GroupChatResponse"},
                                "timestamp": {"type": "integer", "description": "Response timestamp"}
                            }
                        }
                    },
                    "400": {"description": "Parameter error", "schema": {"$ref": "#/definitions/Message"}},
                    "500": {"description": "Server error", "schema": {"$ref": "#/definitions/Message"}}
                }
            }
        },
        "/group-chat/channel-chat-list-by-start-time": {
            "get": {
                "description": "Get channel chat records by start timestamp range (ascending order) using TalkGroupChannelChatTimestamp2Collection",
                "produces": ["application/json"],
                "tags": ["Group Management"],
                "summary": "Get channel chat records by start timestamp range",
                "parameters": [
                    {"type": "string", "description": "Channel ID", "name": "channelId", "in": "query", "required": true},
                    {"type": "integer", "description": "Start timestamp for pagination, default is 0", "name": "startTimestamp", "in": "query", "required": false},
                    {"type": "integer", "description": "Page size, default is 20", "name": "size", "in": "query", "required": false}
                ],
                "responses": {
                    "200": {
                        "description": "Successfully return channel chat records by start timestamp", 
                        "schema": {
                            "type": "object",
                            "properties": {
                                "code": {"type": "integer", "description": "Response code"},
                                "message": {"type": "string", "description": "Response message"},
                                "data": {"$ref": "#/definitions/GroupChatResponse"},
                                "timestamp": {"type": "integer", "description": "Response timestamp"}
                            }
                        }
                    },
                    "400": {"description": "Parameter error", "schema": {"$ref": "#/definitions/Message"}},
                    "500": {"description": "Server error", "schema": {"$ref": "#/definitions/Message"}}
                }
            }
        },
        "/group-chat/group-channel-list": {
            "get": {
                "description": "Get group channel list by group ID from TalkGroupChannelCollection",
                "produces": ["application/json"],
                "tags": ["Group Management"],
                "summary": "Get group channel list",
                "parameters": [
                    {"type": "string", "description": "Group ID", "name": "groupId", "in": "query", "required": true},
                    {"type": "integer", "description": "Cursor for pagination, default is 0", "name": "cursor", "in": "query", "required": false},
                    {"type": "integer", "description": "Page size, default is 20", "name": "size", "in": "query", "required": false}
                ],
                "responses": {
                    "200": {
                        "description": "Successfully return group channel list", 
                        "schema": {
                            "type": "object",
                            "properties": {
                                "code": {"type": "integer", "description": "Response code"},
                                "message": {"type": "string", "description": "Response message"},
                                "data": {"$ref": "#/definitions/GroupChannelResponse"},
                                "timestamp": {"type": "integer", "description": "Response timestamp"}
                            }
                        }
                    },
                    "400": {"description": "Parameter error", "schema": {"$ref": "#/definitions/Message"}},
                    "500": {"description": "Server error", "schema": {"$ref": "#/definitions/Message"}}
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
        "/group-chat/user-info": {
            "get": {
                "description": "Get user information by address or metaId. If address is provided, it will be used; if address is empty but metaId is provided, metaId will be used; if both are empty, an error will be returned.",
                "produces": ["application/json"],
                "tags": ["Group Management"],
                "summary": "Get user info by address or metaId",
                "parameters": [
                    {"type": "string", "description": "User address", "name": "address", "in": "query", "required": false},
                    {"type": "string", "description": "User MetaId", "name": "metaId", "in": "query", "required": false}
                ],
                "responses": {
                    "200": {"description": "Successfully return user information", "schema": {"type": "object"}},
                    "400": {"description": "Parameter error", "schema": {"type": "object"}},
                    "500": {"description": "Server error", "schema": {"type": "object"}}
                }
            }
        },
        "/group-chat/batch-user-info": {
            "post": {
                "description": "Get user information by multiple addresses or metaIds in batch. Maximum total count is 100 (addresses + metaIds).",
                "consumes": ["application/json"],
                "produces": ["application/json"],
                "tags": ["Group Management"],
                "summary": "Get batch user info by addresses or metaIds (max 100)",
                "parameters": [
                    {
                        "name": "request",
                        "in": "body",
                        "required": true,
                        "schema": {
                            "type": "object",
                            "properties": {
                                "addresses": {
                                    "type": "array",
                                    "items": {"type": "string"},
                                    "description": "List of user addresses"
                                },
                                "metaIds": {
                                    "type": "array",
                                    "items": {"type": "string"},
                                    "description": "List of user metaIds"
                                }
                            }
                        }
                    }
                ],
                "responses": {
                    "200": {"description": "Successfully return batch user information", "schema": {"type": "object"}},
                    "400": {"description": "Parameter error", "schema": {"type": "object"}},
                    "500": {"description": "Server error", "schema": {"type": "object"}}
                }
            }
        },
        "/group-chat/search-groups": {
            "get": {
                "description": "Search groups by name or ID using fuzzy search",
                "produces": ["application/json"],
                "tags": ["Group Management"],
                "summary": "Search groups by name or ID",
                "parameters": [
                    {"type": "string", "description": "Search query (group name or ID)", "name": "query", "in": "query", "required": true},
                    {"type": "integer", "description": "Page size, default is 20", "name": "size", "in": "query", "required": false}
                ],
                "responses": {
                    "200": {
                        "description": "Successfully return search results", 
                        "schema": {
                            "type": "object",
                            "properties": {
                                "code": {"type": "integer", "description": "Response code"},
                                "message": {"type": "string", "description": "Response message"},
                                "data": {"$ref": "#/definitions/GroupSearchResponse"},
                                "timestamp": {"type": "integer", "description": "Response timestamp"}
                            }
                        }
                    },
                    "400": {"description": "Parameter error", "schema": {"$ref": "#/definitions/Message"}},
                    "500": {"description": "Server error", "schema": {"$ref": "#/definitions/Message"}}
                }
            }
        },
        "/group-chat/search-groups-and-users": {
            "get": {
                "description": "Search both groups and users by name or ID using fuzzy search",
                "produces": ["application/json"],
                "tags": ["Group Management"],
                "summary": "Search groups and users by name or ID",
                "parameters": [
                    {"type": "string", "description": "Search query (group name, group ID, user name, or metaId)", "name": "query", "in": "query", "required": true},
                    {"type": "integer", "description": "Page size, default is 5", "name": "size", "in": "query", "required": false}
                ],
                "responses": {
                    "200": {
                        "description": "Successfully return combined search results", 
                        "schema": {
                            "type": "object",
                            "properties": {
                                "code": {"type": "integer", "description": "Response code"},
                                "message": {"type": "string", "description": "Response message"},
                                "data": {"$ref": "#/definitions/GroupAndUserSearchResponse"},
                                "timestamp": {"type": "integer", "description": "Response timestamp"}
                            }
                        }
                    },
                    "400": {"description": "Parameter error", "schema": {"$ref": "#/definitions/Message"}},
                    "500": {"description": "Server error", "schema": {"$ref": "#/definitions/Message"}}
                }
            }
        },
        "/group-chat/search-groups-cache-stats": {
            "get": {
                "description": "Get group search cache statistics",
                "produces": ["application/json"],
                "tags": ["Group Management"],
                "summary": "Get group search cache statistics",
                "responses": {
                    "200": {
                        "description": "Successfully return cache statistics", 
                        "schema": {
                            "type": "object",
                            "properties": {
                                "code": {"type": "integer", "description": "Response code"},
                                "message": {"type": "string", "description": "Response message"},
                                "data": {
                                    "type": "object",
                                    "properties": {
                                        "totalGroups": {"type": "integer", "description": "Total number of groups in cache"},
                                        "lastUpdate": {"type": "integer", "description": "Last update timestamp"}
                                    }
                                },
                                "timestamp": {"type": "integer", "description": "Response timestamp"}
                            }
                        }
                    },
                    "500": {"description": "Server error", "schema": {"$ref": "#/definitions/Message"}}
                }
            }
        },
        "/group-chat/chat-sendable": {
            "get": {
                "description": "Check if the current system allows sending chat messages",
                "produces": ["application/json"],
                "tags": ["Group Management"],
                "summary": "Check if chat is sendable",
                "responses": {
                    "200": {
                        "description": "Successfully return chat sendable status",
                        "schema": {
                            "type": "object",
                            "properties": {
                                "code": {"type": "integer", "description": "Response code"},
                                "message": {"type": "string", "description": "Response message"},
                                "data": {"$ref": "#/definitions/ChatSendableResponse"},
                                "timestamp": {"type": "integer", "description": "Response timestamp"}
                            }
                        }
                    },
                    "500": {"description": "Server error", "schema": {"$ref": "#/definitions/Message"}}
                }
            }
        },
        "/group-chat/search-group-members": {
            "get": {
                "description": "Search group members by name, metaId, or address using fuzzy search",
                "produces": ["application/json"],
                "tags": ["Group Management"],
                "summary": "Search group members",
                "parameters": [
                    {"type": "string", "description": "Group ID", "name": "groupId", "in": "query", "required": true},
                    {"type": "string", "description": "Search query (user name, metaId, or address)", "name": "query", "in": "query", "required": true},
                    {"type": "integer", "description": "Page size, default is 20", "name": "size", "in": "query", "required": false}
                ],
                "responses": {
                    "200": {
                        "description": "Successfully return search results",
                        "schema": {
                            "type": "object",
                            "properties": {
                                "code": {"type": "integer", "description": "Response code"},
                                "message": {"type": "string", "description": "Response message"},
                                "data": {"$ref": "#/definitions/GroupMemberSearchResponse"},
                                "timestamp": {"type": "integer", "description": "Response timestamp"}
                            }
                        }
                    },
                    "400": {"description": "Parameter error", "schema": {"$ref": "#/definitions/Message"}},
                    "500": {"description": "Server error", "schema": {"$ref": "#/definitions/Message"}}
                }
            }
        },
        "/group-chat/max-group-chat-index": {
            "get": {
                "description": "Get the current maximum index for a group's chat records",
                "produces": ["application/json"],
                "tags": ["Group Management"],
                "summary": "Get current maximum group chat index",
                "parameters": [
                    {"type": "string", "description": "Group ID", "name": "groupId", "in": "query", "required": true}
                ],
                "responses": {
                    "200": {"description": "Successfully return maximum group chat index", "schema": {"type": "object"}},
                    "400": {"description": "Parameter error", "schema": {"type": "object"}},
                    "500": {"description": "Server error", "schema": {"type": "object"}}
                }
            }
        },
        "/group-chat/max-group-channel-chat-index": {
            "get": {
                "description": "Get the current maximum index for a channel's chat records",
                "produces": ["application/json"],
                "tags": ["Group Management"],
                "summary": "Get current maximum group channel chat index",
                "parameters": [
                    {"type": "string", "description": "Channel ID", "name": "channelId", "in": "query", "required": true}
                ],
                "responses": {
                    "200": {"description": "Successfully return maximum group channel chat index", "schema": {"type": "object"}},
                    "400": {"description": "Parameter error", "schema": {"type": "object"}},
                    "500": {"description": "Server error", "schema": {"type": "object"}}
                }
            }
        },
        "/group-chat/max-private-chat-index": {
            "get": {
                "description": "Get the current maximum index for a private conversation between two users",
                "produces": ["application/json"],
                "tags": ["Group Management"],
                "summary": "Get current maximum private chat index",
                "parameters": [
                    {"type": "string", "description": "From user MetaId", "name": "fromMetaId", "in": "query", "required": true},
                    {"type": "string", "description": "To user MetaId", "name": "toMetaId", "in": "query", "required": true}
                ],
                "responses": {
                    "200": {"description": "Successfully return maximum private chat index", "schema": {"type": "object"}},
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
        "/group-chat/generate-lucky-bag-code": {
            "get": {
                "description": "Generate a new lucky bag code address key for frontend to use before creating a lucky bag",
                "produces": ["application/json"],
                "tags": ["Group Management"],
                "summary": "Generate lucky bag code address key",
                "responses": {
                    "200": {
                        "description": "Successfully return code and address",
                        "schema": {
                            "type": "object",
                            "properties": {
                                "code": {"type": "integer", "description": "Response code"},
                                "message": {"type": "string", "description": "Response message"},
                                "data": {"$ref": "#/definitions/LuckyBagCodeAddressKeyResponse"},
                                "timestamp": {"type": "integer", "description": "Response timestamp"}
                            }
                        }
                    },
                    "500": {"description": "Server error", "schema": {"$ref": "#/definitions/Message"}}
                }
            }
        },
        "/group-chat/lucky-bag-info-v2": {
            "get": {
                "description": "Get lucky bag object and claimed list based on groupId and pinId using cache optimization (V2)",
                "produces": ["application/json"],
                "tags": ["Group Management"],
                "summary": "Get lucky bag info (V2 with cache)",
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
        "/group-chat/lucky-bag-unused-info-v2": {
            "get": {
                "description": "Get lucky bag object and unclaimed list based on groupId and pinId using cache optimization (V2)",
                "produces": ["application/json"],
                "tags": ["Group Management"],
                "summary": "Get lucky bag unused info (V2 with cache)",
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
        "/group-chat/grab-lucky-bag-v2": {
            "post": {
                "description": "Grab lucky bag based on groupId, pinId, metaId, and address using cache optimization (V2)",
                "consumes": ["application/json"],
                "produces": ["application/json"],
                "tags": ["Group Management"],
                "summary": "Grab lucky bag (V2 with cache)",
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
        "api/db/community/version": {
            "get": {
                "description": "Query TalkCommunityVersionInfoCollection data by communityId or pinId",
                "produces": ["application/json"],
                "tags": ["Database Operations"],
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
                "tags": ["Database Operations"],
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
                "tags": ["Database Operations"],
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
                "tags": ["Database Operations"],
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
                "tags": ["Database Operations"],
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
                "tags": ["Database Operations"],
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
                "tags": ["Database Operations"],
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
                "tags": ["Database Operations"],
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
                "tags": ["Database Operations"],
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
                "tags": ["Database Operations"],
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
                "tags": ["Database Operations"],
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
                "tags": ["Database Operations"],
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
                "tags": ["Database Operations"],
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
                "tags": ["Database Operations"],
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
                "tags": ["Database Operations"],
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
                "tags": ["Statistics"],
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
        "/api/db/luckybag/lock-stats": {
            "get": {
                "description": "Get comprehensive statistics about lucky bag locks including total locks, active locks, and inactive locks",
                "produces": ["application/json"],
                "tags": ["Database Operations"],
                "summary": "Get lucky bag lock statistics",
                "responses": {
                    "200": {"description": "Lucky bag lock statistics", "schema": {"type": "object"}},
                    "500": {"description": "Server error", "schema": {"type": "object"}}
                }
            }
        },
        "/api/db/group/version/all": {
            "get": {
                "description": "Get all data of TalkGroupVersionInfoCollection, support pagination",
                "produces": ["application/json"],
                "tags": ["Database Operations"],
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
                "tags": ["Database Operations"],
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
        "/api/db/chat/timestamp2/out/channel": {
            "get": {
                "description": "Get channel chat timestamp2 out collection list with pagination and reverse order",
                "produces": ["application/json"],
                "tags": ["Database Operations"],
                "summary": "Get channel chat timestamp2 out list",
                "parameters": [
                    {"type": "integer", "description": "Cursor position, default is 0", "name": "cursor", "in": "query", "required": false},
                    {"type": "integer", "description": "Page size, default is 20", "name": "size", "in": "query", "required": false},
                    {"type": "string", "description": "Channel ID for filtering", "name": "channelId", "in": "query", "required": false}
                ],
                "responses": {
                    "200": {"description": "Successfully return channel chat timestamp2 out list", "schema": {"type": "object"}},
                    "400": {"description": "Parameter error", "schema": {"type": "object"}},
                    "500": {"description": "Server error", "schema": {"type": "object"}}
                }
            }
        },
        "/api/db/chat/index/channel": {
            "get": {
                "description": "Get channel chat index collection list with pagination and reverse order",
                "produces": ["application/json"],
                "tags": ["Database Operations"],
                "summary": "Get channel chat index list",
                "parameters": [
                    {"type": "integer", "description": "Cursor position, default is 0", "name": "cursor", "in": "query", "required": false},
                    {"type": "integer", "description": "Page size, default is 20", "name": "size", "in": "query", "required": false},
                    {"type": "string", "description": "Channel ID for filtering", "name": "channelId", "in": "query", "required": false}
                ],
                "responses": {
                    "200": {"description": "Successfully return channel chat index list", "schema": {"type": "object"}},
                    "400": {"description": "Parameter error", "schema": {"type": "object"}},
                    "500": {"description": "Server error", "schema": {"type": "object"}}
                }
            }
        },
        "/api/db/chat/index/channel/keys": {
            "get": {
                "description": "Get channel chat index collection key list with pagination",
                "produces": ["application/json"],
                "tags": ["Database Operations"],
                "summary": "Get channel chat index keys",
                "parameters": [
                    {"type": "integer", "description": "Cursor position, default is 0", "name": "cursor", "in": "query", "required": false},
                    {"type": "integer", "description": "Page size, default is 20", "name": "size", "in": "query", "required": false},
                    {"type": "string", "description": "Channel ID for filtering", "name": "channelId", "in": "query", "required": false}
                ],
                "responses": {
                    "200": {"description": "Successfully return channel chat index keys", "schema": {"type": "object"}},
                    "400": {"description": "Parameter error", "schema": {"type": "object"}},
                    "500": {"description": "Server error", "schema": {"type": "object"}}
                }
            }
        },
        "/health": {
            "get": {
                "description": "Check if the group chat service is running properly",
                "produces": ["application/json"],
                "tags": ["System"],
                "summary": "Health check endpoint",
                "responses": {
                    "200": {
                        "description": "Service is healthy",
                        "schema": {
                            "type": "object",
                            "properties": {
                                "code": {"type": "integer"},
                                "message": {"type": "string"},
                                "data": {
                                    "type": "object",
                                    "properties": {
                                        "status": {"type": "string"},
                                        "service": {"type": "string"},
                                        "timestamp": {"type": "integer"},
                                        "uptime": {"type": "string"}
                                    }
                                }
                            }
                        }
                    }
                }
            }
        },
        "/api/db/chat/index/group": {
            "get": {
                "description": "Get TalkGroupChatIndexCollection list with cursor pagination and reverse order",
                "produces": ["application/json"],
                "tags": ["Database Operations"],
                "summary": "Get group chat index list",
                "parameters": [
                    {"type": "integer", "description": "Cursor, starting from 0", "name": "cursor", "in": "query", "required": false, "default": 0},
                    {"type": "integer", "description": "Number of items per page", "name": "size", "in": "query", "required": false, "default": 20},
                    {"type": "string", "description": "Group ID for filtering (optional)", "name": "groupId", "in": "query", "required": false}
                ],
                "responses": {
                    "200": {"description": "Group chat index list", "schema": {"type": "object"}},
                    "400": {"description": "Parameter error", "schema": {"type": "object"}},
                    "500": {"description": "Server error", "schema": {"type": "object"}}
                }
            }
        },
        "/api/db/chat/index/group/keys": {
            "get": {
                "description": "Get TalkGroupChatIndexCollection key list with cursor pagination",
                "produces": ["application/json"],
                "tags": ["Database Operations"],
                "summary": "Get group chat index keys",
                "parameters": [
                    {"type": "integer", "description": "Cursor, starting from 0", "name": "cursor", "in": "query", "required": false, "default": 0},
                    {"type": "integer", "description": "Number of items per page", "name": "size", "in": "query", "required": false, "default": 20},
                    {"type": "string", "description": "Group ID for filtering (optional)", "name": "groupId", "in": "query", "required": false}
                ],
                "responses": {
                    "200": {"description": "Group chat index keys", "schema": {"type": "object"}},
                    "400": {"description": "Parameter error", "schema": {"type": "object"}},
                    "500": {"description": "Server error", "schema": {"type": "object"}}
                }
            }
        },
        "/api/db/chat/index/private": {
            "get": {
                "description": "Get TalkPrivateChatIndexCollection list with cursor pagination and reverse order",
                "produces": ["application/json"],
                "tags": ["Database Operations"],
                "summary": "Get private chat index list",
                "parameters": [
                    {"type": "integer", "description": "Cursor, starting from 0", "name": "cursor", "in": "query", "required": false, "default": 0},
                    {"type": "integer", "description": "Number of items per page", "name": "size", "in": "query", "required": false, "default": 20},
                    {"type": "string", "description": "From MetaId to To MetaId for filtering (format: fromMetaId_toMetaId, optional)", "name": "fromTo", "in": "query", "required": false}
                ],
                "responses": {
                    "200": {"description": "Private chat index list", "schema": {"type": "object"}},
                    "400": {"description": "Parameter error", "schema": {"type": "object"}},
                    "500": {"description": "Server error", "schema": {"type": "object"}}
                }
            }
        },
        "/api/db/chat/timestamp2/out": {
            "get": {
                "description": "Get TalkGroupChatTimestamp2OutCollection list with cursor pagination and reverse order",
                "produces": ["application/json"],
                "tags": ["Database Operations"],
                "summary": "Get group chat timestamp2 out list",
                "parameters": [
                    {"type": "integer", "description": "Cursor, starting from 0", "name": "cursor", "in": "query", "required": false, "default": 0},
                    {"type": "integer", "description": "Number of items per page", "name": "size", "in": "query", "required": false, "default": 20},
                    {"type": "string", "description": "Group ID for filtering (optional)", "name": "groupId", "in": "query", "required": false}
                ],
                "responses": {
                    "200": {"description": "Group chat timestamp2 out list", "schema": {"type": "object"}},
                    "400": {"description": "Parameter error", "schema": {"type": "object"}},
                    "500": {"description": "Server error", "schema": {"type": "object"}}
                }
            }
        },
        "/api/db/luckybag/open/list": {
            "get": {
                "description": "Get detailed open lucky bag list with grab state, user info, and lucky bag details",
                "produces": ["application/json"],
                "tags": ["Database Operations"],
                "summary": "Get detailed open lucky bag list by lucky bag PinId",
                "parameters": [
                    {"type": "string", "description": "Lucky bag PinId", "name": "luckyBagPinId", "in": "query", "required": true}
                ],
                "responses": {
                    "200": {"description": "Detailed query result with grab state, user info, and lucky bag details", "schema": {"type": "object"}},
                    "400": {"description": "Parameter error", "schema": {"type": "object"}},
                    "500": {"description": "Server error", "schema": {"type": "object"}}
                }
            }
        },
        "/api/db/luckybag/update-validation": {
            "get": {
                "description": "Update lucky bag validation counts and lists by lucky bag PinId",
                "produces": ["application/json"],
                "tags": ["Database Operations"],
                "summary": "Update lucky bag validation",
                "parameters": [
                    {"type": "string", "description": "Lucky bag PinId", "name": "luckyBagPinId", "in": "query", "required": true}
                ],
                "responses": {
                    "200": {"description": "Update result with validation counts and lists", "schema": {"type": "object"}},
                    "400": {"description": "Parameter error", "schema": {"type": "object"}},
                    "500": {"description": "Server error", "schema": {"type": "object"}}
                }
            }
        },
        "/api/db/luckybag/process-expired": {
            "get": {
                "description": "Process a specific lucky bag by pinId as if it were expired, simulating the expired lucky bag processing logic",
                "produces": ["application/json"],
                "tags": ["Database Operations"],
                "summary": "Process expired lucky bag by pinId",
                "parameters": [
                    {"type": "string", "description": "Lucky bag PinId", "name": "pinId", "in": "query", "required": true}
                ],
                "responses": {
                    "200": {"description": "Process result with source collection, target collection, and new state", "schema": {"type": "object"}},
                    "400": {"description": "Parameter error", "schema": {"type": "object"}},
                    "500": {"description": "Server error", "schema": {"type": "object"}}
                }
            }
        },
        "/api/db/luckybag/collection/list": {
            "get": {
                "description": "Get lucky bag collection list with pagination support for pending, completed, timeout residue, error pending, and error timeout residue collections",
                "produces": ["application/json"],
                "tags": ["Database Operations"],
                "summary": "Get lucky bag collection list with pagination",
                "parameters": [
                    {"type": "string", "description": "Collection name (talk_group_lucky_bag_pin_pending, talk_group_lucky_bag_pin_completed, talk_group_lucky_bag_pin_timeout_residue, talk_group_lucky_bag_pin_err_pending, talk_group_lucky_bag_pin_err_timeout_residue)", "name": "collection", "in": "query", "required": true},
                    {"type": "integer", "description": "Cursor, starting from 0", "name": "cursor", "in": "query", "required": false, "default": 0},
                    {"type": "integer", "description": "Number of items per page", "name": "size", "in": "query", "required": false, "default": 20}
                ],
                "responses": {
                    "200": {"description": "Lucky bag collection list with pagination", "schema": {"type": "object"}},
                    "400": {"description": "Parameter error", "schema": {"type": "object"}},
                    "500": {"description": "Server error", "schema": {"type": "object"}}
                }
            }
        },
        "/api/db/luckybag/collection/pinid": {
            "get": {
                "description": "Get lucky bag collection data by specific pinId from any of the lucky bag collections",
                "produces": ["application/json"],
                "tags": ["Database Operations"],
                "summary": "Get lucky bag collection data by specific pinId",
                "parameters": [
                    {"type": "string", "description": "Collection name (talk_group_lucky_bag_pin_pending, talk_group_lucky_bag_pin_completed, talk_group_lucky_bag_pin_timeout_residue, talk_group_lucky_bag_pin_err_pending, talk_group_lucky_bag_pin_err_timeout_residue)", "name": "collection", "in": "query", "required": true},
                    {"type": "string", "description": "Lucky bag PinId", "name": "pinId", "in": "query", "required": true}
                ],
                "responses": {
                    "200": {"description": "Lucky bag collection data by pinId", "schema": {"type": "object"}},
                    "400": {"description": "Parameter error", "schema": {"type": "object"}},
                    "500": {"description": "Server error", "schema": {"type": "object"}}
                }
            }
        },
        "/api/db/luckybag/queue/list": {
            "get": {
                "description": "Get lucky bag queue collection list with pagination support for open lucky bag queue and residue lucky bag queue collections",
                "produces": ["application/json"],
                "tags": ["Database Operations"],
                "summary": "Get lucky bag queue collection list with pagination",
                "parameters": [
                    {"type": "string", "description": "Collection name (talk_group_open_lucky_bag_queue, talk_group_residue_lucky_bag_queue)", "name": "collection", "in": "query", "required": true},
                    {"type": "integer", "description": "Cursor, starting from 0", "name": "cursor", "in": "query", "required": false, "default": 0},
                    {"type": "integer", "description": "Number of items per page", "name": "size", "in": "query", "required": false, "default": 20}
                ],
                "responses": {
                    "200": {"description": "Lucky bag queue collection list with pagination", "schema": {"type": "object"}},
                    "400": {"description": "Parameter error", "schema": {"type": "object"}},
                    "500": {"description": "Server error", "schema": {"type": "object"}}
                }
            }
        },
        "/api/db/residue-luckybag/pinid": {
            "get": {
                "description": "Get residue lucky bag data by specific pinId from TalkGroupResidueLuckyBagPinCollection",
                "produces": ["application/json"],
                "tags": ["Database Operations"],
                "summary": "Get residue lucky bag data by pinId",
                "parameters": [
                    {"type": "string", "description": "Residue lucky bag PinId", "name": "pinId", "in": "query", "required": true}
                ],
                "responses": {
                    "200": {"description": "Residue lucky bag data", "schema": {"type": "object"}},
                    "400": {"description": "Parameter error", "schema": {"type": "object"}},
                    "500": {"description": "Server error", "schema": {"type": "object"}}
                }
            }
        },
        "/api/db/residue-luckybag/list": {
            "get": {
                "description": "Get residue lucky bag collection list with pagination support",
                "produces": ["application/json"],
                "tags": ["Database Operations"],
                "summary": "Get residue lucky bag list with pagination",
                "parameters": [
                    {"type": "integer", "description": "Cursor, starting from 0", "name": "cursor", "in": "query", "required": false, "default": 0},
                    {"type": "integer", "description": "Number of items per page", "name": "size", "in": "query", "required": false, "default": 20}
                ],
                "responses": {
                    "200": {"description": "Residue lucky bag collection list with pagination", "schema": {"type": "object"}},
                    "400": {"description": "Parameter error", "schema": {"type": "object"}},
                    "500": {"description": "Server error", "schema": {"type": "object"}}
                }
            }
        },
        "/api/db/private-chat/timestamp/list": {
            "get": {
                "description": "Get private chat timestamp collection list with pagination support for specific from and to users",
                "produces": ["application/json"],
                "tags": ["Database Operations"],
                "summary": "Get private chat timestamp list with pagination",
                "parameters": [
                    {"type": "string", "description": "From user MetaId", "name": "from", "in": "query", "required": true},
                    {"type": "string", "description": "To user MetaId", "name": "to", "in": "query", "required": true},
                    {"type": "integer", "description": "Cursor, starting from 0", "name": "cursor", "in": "query", "required": false, "default": 0},
                    {"type": "integer", "description": "Number of items per page", "name": "size", "in": "query", "required": false, "default": 20}
                ],
                "responses": {
                    "200": {"description": "Private chat timestamp collection list with pagination", "schema": {"type": "object"}},
                    "400": {"description": "Parameter error", "schema": {"type": "object"}},
                    "500": {"description": "Server error", "schema": {"type": "object"}}
                }
            }
        },
        "/api/db/metaid/join": {
            "get": {
                "description": "Query TalkGroupMetaIdJoinCollection data by metaId",
                "produces": ["application/json"],
                "tags": ["Database Operations"],
                "summary": "Get MetaId join list by metaId",
                "parameters": [
                    {"type": "string", "description": "MetaId", "name": "metaId", "in": "query", "required": true}
                ],
                "responses": {
                    "200": {"description": "MetaId join list with detailed information", "schema": {"type": "object"}},
                    "400": {"description": "Parameter error", "schema": {"type": "object"}},
                    "500": {"description": "Server error", "schema": {"type": "object"}}
                }
            }
        },
        "/api/db/metaid/context": {
            "get": {
                "description": "Get TalkMetaIdContextListCollection data by metaId",
                "produces": ["application/json"],
                "tags": ["Database Operations"],
                "summary": "Get MetaId context list by metaId",
                "parameters": [
                    {"type": "string", "description": "MetaId", "name": "metaId", "in": "query", "required": true}
                ],
                "responses": {
                    "200": {"description": "MetaId context list data", "schema": {"type": "object"}},
                    "400": {"description": "Parameter error", "schema": {"type": "object"}},
                    "500": {"description": "Server error", "schema": {"type": "object"}}
                }
            }
        },
        "/api/db/group/member-list": {
            "get": {
                "description": "Get group member list with pagination support",
                "produces": ["application/json"],
                "tags": ["Database Operations"],
                "summary": "Get group member list",
                "parameters": [
                    {"type": "string", "description": "Group ID", "name": "groupId", "in": "query", "required": true},
                    {"type": "integer", "description": "Cursor, starting from 0", "name": "cursor", "in": "query", "required": false, "default": 0},
                    {"type": "integer", "description": "Number of items per page", "name": "size", "in": "query", "required": false, "default": 20},
                    {"type": "string", "description": "Order by field, use 'timestamp' for timestamp descending order", "name": "orderBy", "in": "query", "required": false},
                    {"type": "string", "description": "Order type, use 'desc' for descending order", "name": "orderType", "in": "query", "required": false}
                ],
                "responses": {
                    "200": {"description": "Group member list", "schema": {"type": "object"}},
                    "400": {"description": "Parameter error", "schema": {"type": "object"}},
                    "500": {"description": "Server error", "schema": {"type": "object"}}
                }
            }
        },
        "/api/db/group/member-list-v2": {
            "get": {
                "description": "Get group member list using TalkGroupPersonListCollection (already sorted by timestamp descending)",
                "produces": ["application/json"],
                "tags": ["Database Operations"],
                "summary": "Get group member list V2",
                "parameters": [
                    {"type": "string", "description": "Group ID", "name": "groupId", "in": "query", "required": true},
                    {"type": "integer", "description": "Cursor, starting from 0", "name": "cursor", "in": "query", "required": false, "default": 0},
                    {"type": "integer", "description": "Number of items per page", "name": "size", "in": "query", "required": false, "default": 20}
                ],
                "responses": {
                    "200": {"description": "Group member list V2", "schema": {"type": "object"}},
                    "400": {"description": "Parameter error", "schema": {"type": "object"}},
                    "500": {"description": "Server error", "schema": {"type": "object"}}
                }
            }
        },
        "/api/db/group/person-list-collection": {
            "get": {
                "description": "Get TalkGroupPersonListCollection data with pagination support, returns groupId and member count",
                "produces": ["application/json"],
                "tags": ["Database Operations"],
                "summary": "Get group person list collection with pagination",
                "parameters": [
                    {"type": "integer", "description": "Cursor, starting from 0", "name": "cursor", "in": "query", "required": false, "default": 0},
                    {"type": "integer", "description": "Number of items per page", "name": "size", "in": "query", "required": false, "default": 20}
                ],
                "responses": {
                    "200": {"description": "Group person list collection with pagination", "schema": {"type": "object"}},
                    "400": {"description": "Parameter error", "schema": {"type": "object"}},
                    "500": {"description": "Server error", "schema": {"type": "object"}}
                }
            }
        },
        "/api/db/group/admin-collection": {
            "get": {
                "description": "Get TalkGroupAdminCollection data with pagination support",
                "produces": ["application/json"],
                "tags": ["Database Operations"],
                "summary": "Get group admin collection",
                "parameters": [
                    {"type": "integer", "description": "Cursor, starting from 0", "name": "cursor", "in": "query", "required": false, "default": 0},
                    {"type": "integer", "description": "Number of items per page", "name": "size", "in": "query", "required": false, "default": 20}
                ],
                "responses": {
                    "200": {"description": "Group admin collection", "schema": {"type": "object"}},
                    "400": {"description": "Parameter error", "schema": {"type": "object"}},
                    "500": {"description": "Server error", "schema": {"type": "object"}}
                }
            }
        },
        "/api/db/group/block-collection": {
            "get": {
                "description": "Get TalkGroupBlockCollection data with pagination support",
                "produces": ["application/json"],
                "tags": ["Database Operations"],
                "summary": "Get group block collection",
                "parameters": [
                    {"type": "integer", "description": "Cursor, starting from 0", "name": "cursor", "in": "query", "required": false, "default": 0},
                    {"type": "integer", "description": "Number of items per page", "name": "size", "in": "query", "required": false, "default": 20}
                ],
                "responses": {
                    "200": {"description": "Group block collection", "schema": {"type": "object"}},
                    "400": {"description": "Parameter error", "schema": {"type": "object"}},
                    "500": {"description": "Server error", "schema": {"type": "object"}}
                }
            }
        },
        "/api/db/group/whitelist-collection": {
            "get": {
                "description": "Get TalkGroupWhitelistCollection data with pagination support",
                "produces": ["application/json"],
                "tags": ["Database Operations"],
                "summary": "Get group whitelist collection",
                "parameters": [
                    {"type": "integer", "description": "Cursor, starting from 0", "name": "cursor", "in": "query", "required": false, "default": 0},
                    {"type": "integer", "description": "Number of items per page", "name": "size", "in": "query", "required": false, "default": 20}
                ],
                "responses": {
                    "200": {"description": "Group whitelist collection", "schema": {"type": "object"}},
                    "400": {"description": "Parameter error", "schema": {"type": "object"}},
                    "500": {"description": "Server error", "schema": {"type": "object"}}
                }
            }
        },
        "/api/db/group/admin/{groupId}": {
            "get": {
                "description": "Get group admin data by specific groupId",
                "produces": ["application/json"],
                "tags": ["Database Operations"],
                "summary": "Get group admin by groupId",
                "parameters": [
                    {"type": "string", "description": "Group ID", "name": "groupId", "in": "path", "required": true}
                ],
                "responses": {
                    "200": {"description": "Group admin data", "schema": {"type": "object"}},
                    "400": {"description": "Parameter error", "schema": {"type": "object"}},
                    "404": {"description": "Group admin not found", "schema": {"type": "object"}},
                    "500": {"description": "Server error", "schema": {"type": "object"}}
                }
            }
        },
        "/api/db/group/block/{groupId}": {
            "get": {
                "description": "Get group block data by specific groupId",
                "produces": ["application/json"],
                "tags": ["Database Operations"],
                "summary": "Get group block by groupId",
                "parameters": [
                    {"type": "string", "description": "Group ID", "name": "groupId", "in": "path", "required": true}
                ],
                "responses": {
                    "200": {"description": "Group block data", "schema": {"type": "object"}},
                    "400": {"description": "Parameter error", "schema": {"type": "object"}},
                    "404": {"description": "Group block not found", "schema": {"type": "object"}},
                    "500": {"description": "Server error", "schema": {"type": "object"}}
                }
            }
        },
        "/api/db/group/whitelist/{groupId}": {
            "get": {
                "description": "Get group whitelist data by specific groupId",
                "produces": ["application/json"],
                "tags": ["Database Operations"],
                "summary": "Get group whitelist by groupId",
                "parameters": [
                    {"type": "string", "description": "Group ID", "name": "groupId", "in": "path", "required": true}
                ],
                "responses": {
                    "200": {"description": "Group whitelist data", "schema": {"type": "object"}},
                    "400": {"description": "Parameter error", "schema": {"type": "object"}},
                    "404": {"description": "Group whitelist not found", "schema": {"type": "object"}},
                    "500": {"description": "Server error", "schema": {"type": "object"}}
                }
            }
        },
        "/api/db/luckybag/error-keys": {
            "get": {
                "description": "Get keys from lucky bag error collections with pagination",
                "produces": ["application/json"],
                "tags": ["Database Operations"],
                "summary": "Get lucky bag error collection keys",
                "parameters": [
                    {"type": "string", "description": "Collection name (talk_group_open_lucky_bag_err or talk_group_residue_lucky_bag_err)", "name": "collection", "in": "query", "required": true},
                    {"type": "integer", "description": "Cursor for pagination (default: 0)", "name": "cursor", "in": "query", "required": false},
                    {"type": "integer", "description": "Page size (default: 20)", "name": "size", "in": "query", "required": false}
                ],
                "responses": {
                    "200": {
                        "description": "Successfully return keys list",
                        "schema": {
                            "type": "object",
                            "properties": {
                                "success": {"type": "boolean"},
                                "data": {
                                    "type": "object",
                                    "properties": {
                                        "collection": {"type": "string"},
                                        "keys": {"type": "array", "items": {"type": "string"}},
                                        "total": {"type": "integer"},
                                        "cursor": {"type": "integer"},
                                        "size": {"type": "integer"},
                                        "hasMore": {"type": "boolean"},
                                        "nextCursor": {"type": "integer"}
                                    }
                                }
                            }
                        }
                    },
                    "400": {"description": "Bad request", "schema": {"type": "object"}},
                    "500": {"description": "Internal server error", "schema": {"type": "object"}}
                }
            }
        },
        "/api/db/luckybag/pin": {
            "get": {
                "description": "Get lucky bag pin data by pinId from specified collection",
                "produces": ["application/json"],
                "tags": ["Database Operations"],
                "summary": "Get lucky bag pin data by pinId",
                "parameters": [
                    {"type": "string", "description": "Collection name (talk_group_open_lucky_bag_pin or talk_group_residue_lucky_bag_pin)", "name": "collection", "in": "query", "required": true},
                    {"type": "string", "description": "PinId to search for", "name": "pinId", "in": "query", "required": true}
                ],
                "responses": {
                    "200": {
                        "description": "Successfully return pin data",
                        "schema": {
                            "type": "object",
                            "properties": {
                                "success": {"type": "boolean"},
                                "data": {
                                    "type": "object",
                                    "properties": {
                                        "collection": {"type": "string"},
                                        "pinId": {"type": "string"},
                                        "data": {"type": "object"}
                                    }
                                }
                            }
                        }
                    },
                    "400": {"description": "Bad request", "schema": {"type": "object"}},
                    "404": {"description": "PinId not found", "schema": {"type": "object"}},
                    "500": {"description": "Internal server error", "schema": {"type": "object"}}
                }
            }
        },
        "/api/db/luckybag/code-address-key": {
            "get": {
                "description": "Get lucky bag code address key from completed collection by code and address",
                "produces": ["application/json"],
                "tags": ["Database Operations"],
                "summary": "Get lucky bag code address key from completed collection",
                "parameters": [
                    {"type": "string", "description": "Lucky bag code", "name": "code", "in": "query", "required": true},
                    {"type": "string", "description": "Lucky bag address", "name": "address", "in": "query", "required": true}
                ],
                "responses": {
                    "200": {
                        "description": "Successfully return code address key data",
                        "schema": {
                            "type": "object",
                            "properties": {
                                "success": {"type": "boolean"},
                                "data": {
                                    "type": "object",
                                    "properties": {
                                        "code": {"type": "string"},
                                        "luckyBagAddress": {"type": "string"},
                                        "timestamp": {"type": "integer"}
                                    }
                                }
                            }
                        }
                    },
                    "400": {"description": "Bad request", "schema": {"type": "object"}},
                    "404": {"description": "Code address key not found", "schema": {"type": "object"}},
                    "500": {"description": "Internal server error", "schema": {"type": "object"}}
                }
            }
        },
        "/api/db/luckybag/retry": {
            "post": {
                "description": "Retry failed lucky bag operation by pinId from error collections",
                "produces": ["application/json"],
                "tags": ["Database Operations"],
                "summary": "Retry failed lucky bag operation",
                "parameters": [
                    {"type": "string", "description": "PinId of the failed lucky bag operation", "name": "pinId", "in": "query", "required": true}
                ],
                "responses": {
                    "200": {
                        "description": "Successfully retried lucky bag operation",
                        "schema": {
                            "type": "object",
                            "properties": {
                                "success": {"type": "boolean"},
                                "data": {
                                    "type": "object",
                                    "properties": {
                                        "type": {"type": "string", "description": "Operation type: open or residue"},
                                        "pinId": {"type": "string"},
                                        "luckyBagPinId": {"type": "string"},
                                        "message": {"type": "string"}
                                    }
                                }
                            }
                        }
                    },
                    "400": {"description": "Bad request", "schema": {"type": "object"}},
                    "404": {"description": "PinId not found in error collections", "schema": {"type": "object"}},
                    "500": {"description": "Internal server error", "schema": {"type": "object"}}
                }
            }
        },
        "/api/db/luckybag/retry-by-luckybag-id": {
            "post": {
                "description": "Retry all failed lucky bag operations for a specific lucky bag ID from error collections",
                "produces": ["application/json"],
                "tags": ["Database Operations"],
                "summary": "Retry failed lucky bag operations by lucky bag ID",
                "parameters": [
                    {"type": "string", "description": "Lucky bag ID to retry failed operations for", "name": "luckyBagId", "in": "query", "required": true}
                ],
                "responses": {
                    "200": {
                        "description": "Successfully retried lucky bag operations",
                        "schema": {
                            "type": "object",
                            "properties": {
                                "success": {"type": "boolean"},
                                "data": {
                                    "type": "object",
                                    "properties": {
                                        "luckyBagId": {"type": "string", "description": "Lucky bag ID that was processed"},
                                        "totalFound": {"type": "integer", "description": "Total number of failed operations found"},
                                        "successCount": {"type": "integer", "description": "Number of operations successfully retried"},
                                        "errorCount": {"type": "integer", "description": "Number of operations that failed to retry"},
                                        "retryResults": {
                                            "type": "array",
                                            "items": {
                                                "type": "object",
                                                "properties": {
                                                    "type": {"type": "string", "description": "Operation type: open or residue"},
                                                    "pinId": {"type": "string"},
                                                    "luckyBagPinId": {"type": "string"},
                                                    "message": {"type": "string"}
                                                }
                                            }
                                        },
                                        "retryErrors": {
                                            "type": "array",
                                            "items": {"type": "string", "description": "Error messages for failed retry attempts"}
                                        },
                                        "message": {"type": "string", "description": "Summary message of the retry operation"}
                                    }
                                }
                            }
                        }
                    },
                    "400": {"description": "Bad request", "schema": {"type": "object"}},
                    "500": {"description": "Internal server error", "schema": {"type": "object"}}
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
        },
        "/api/db/global-block/set": {
            "post": {
                "description": "Set an address to the global block list",
                "produces": ["application/json"],
                "tags": ["Global Block"],
                "summary": "Set global block address",
                "parameters": [
                    {
                        "name": "request",
                        "in": "body",
                        "required": true,
                        "schema": {
                            "type": "object",
                            "properties": {
                                "address": {
                                    "type": "string",
                                    "description": "Address to block"
                                },
                                "reason": {
                                    "type": "string",
                                    "description": "Reason for blocking"
                                }
                            },
                            "required": ["address"]
                        }
                    }
                ],
                "responses": {
                    "200": {"description": "Success response", "schema": {"type": "object"}},
                    "400": {"description": "Bad request", "schema": {"type": "object"}},
                    "500": {"description": "Internal server error", "schema": {"type": "object"}}
                }
            }
        },
        "/api/db/global-block/delete": {
            "post": {
                "description": "Remove an address from the global block list",
                "produces": ["application/json"],
                "tags": ["Global Block"],
                "summary": "Delete global block address",
                "parameters": [
                    {
                        "name": "request",
                        "in": "body",
                        "required": true,
                        "schema": {
                            "type": "object",
                            "properties": {
                                "address": {
                                    "type": "string",
                                    "description": "Address to unblock"
                                }
                            },
                            "required": ["address"]
                        }
                    }
                ],
                "responses": {
                    "200": {"description": "Success response", "schema": {"type": "object"}},
                    "400": {"description": "Bad request", "schema": {"type": "object"}},
                    "500": {"description": "Internal server error", "schema": {"type": "object"}}
                }
            }
        },
        "/api/db/global-block/stats": {
            "get": {
                "description": "Get statistics about the global block list",
                "produces": ["application/json"],
                "tags": ["Global Block"],
                "summary": "Get global block statistics",
                "responses": {
                    "200": {"description": "Success response with statistics", "schema": {"type": "object"}},
                    "500": {"description": "Internal server error", "schema": {"type": "object"}}
                }
            }
        },
        "/api/db/global-block/addresses": {
            "get": {
                "description": "Get paginated list of all global block addresses",
                "produces": ["application/json"],
                "tags": ["Global Block"],
                "summary": "Get global block addresses",
                "parameters": [
                    {"type": "integer", "description": "Cursor for pagination", "name": "cursor", "in": "query", "required": false},
                    {"type": "integer", "description": "Page size", "name": "size", "in": "query", "required": false}
                ],
                "responses": {
                    "200": {"description": "Success response with paginated data", "schema": {"type": "object"}},
                    "400": {"description": "Bad request", "schema": {"type": "object"}},
                    "500": {"description": "Internal server error", "schema": {"type": "object"}}
                }
            }
        },
        "/api/db/global-block/check": {
            "get": {
                "description": "Check if an address is in the global block list",
                "produces": ["application/json"],
                "tags": ["Global Block"],
                "summary": "Check global block status",
                "parameters": [
                    {"type": "string", "description": "Address to check", "name": "address", "in": "query", "required": true}
                ],
                "responses": {
                    "200": {"description": "Success response with block status", "schema": {"type": "object"}},
                    "400": {"description": "Bad request", "schema": {"type": "object"}},
                    "500": {"description": "Internal server error", "schema": {"type": "object"}}
                }
            }
        },
        "/api/db/chat-statistics": {
            "get": {
                "description": "Get comprehensive chat statistics including group chat, private chat, channel chat, group creation, and total counts within a time range",
                "produces": ["application/json"],
                "tags": ["Statistics"],
                "summary": "Get chat statistics",
                "parameters": [
                    {"type": "integer", "description": "Start timestamp (Unix timestamp in milliseconds)", "name": "startTime", "in": "query", "required": true},
                    {"type": "integer", "description": "End timestamp (Unix timestamp in milliseconds)", "name": "endTime", "in": "query", "required": true},
                    {"type": "string", "description": "Group ID (leave empty to get statistics for all groups)", "name": "groupId", "in": "query", "required": false}
                ],
                "responses": {
                    "200": {"description": "Success response with chat statistics", "schema": {"type": "object"}},
                    "400": {"description": "Bad request", "schema": {"type": "object"}},
                    "500": {"description": "Internal server error", "schema": {"type": "object"}}
                }
            }
        }
    },
    "tags": [
        {
            "description": "Group management related APIs, including group information, member management, etc.",
            "name": "Group Management"
        },
        {
            "description": "Socket management related APIs, including connection statistics and user online status",
            "name": "Socket Management"
        },
        {
            "description": "Database query related APIs for viewing data in Pebble database",
            "name": "Database Operations"
        },
        {
            "description": "Statistics related APIs for getting various statistics and analytics",
            "name": "Statistics"
        },
        {
            "description": "Global block list management APIs for managing globally blocked addresses",
            "name": "Global Block"
        },
        {
            "description": "System health and status related APIs",
            "name": "System"
        }
    ],
    "definitions": {
        "Message": {
            "type": "object",
            "properties": {
                "code": {
                    "type": "integer",
                    "description": "Response code"
                },
                "message": {
                    "type": "string",
                    "description": "Response message"
                },
                "data": {
                    "type": "object",
                    "description": "Response data"
                },
                "timestamp": {
                    "type": "integer",
                    "description": "Response timestamp"
                }
            }
        },
        "GroupResponse": {
            "type": "object",
            "properties": {
                "total": {
                    "type": "integer",
                    "description": "Total count"
                },
                "list": {
                    "type": "array",
                    "items": {
                        "$ref": "#/definitions/GroupItem"
                    },
                    "description": "Group list"
                }
            }
        },
        "GroupItem": {
            "type": "object",
            "properties": {
                "communityId": {
                    "type": "string",
                    "description": "Community ID"
                },
                "groupId": {
                    "type": "string",
                    "description": "Group ID"
                },
                "txId": {
                    "type": "string",
                    "description": "Transaction ID"
                },
                "pinId": {
                    "type": "string",
                    "description": "Pin ID"
                },
                "roomName": {
                    "type": "string",
                    "description": "Room name"
                },
                "roomNote": {
                    "type": "string",
                    "description": "Room note"
                },
                "roomIcon": {
                    "type": "string",
                    "description": "Room icon"
                },
                "roomType": {
                    "type": "string",
                    "description": "Room type"
                },
                "roomStatus": {
                    "type": "string",
                    "description": "Room status"
                },
                "roomJoinType": {
                    "type": "string",
                    "description": "Room join type"
                },
                "roomAvatarUrl": {
                    "type": "string",
                    "description": "Room avatar URL"
                },
                "roomNinePersonHash": {
                    "type": "string",
                    "description": "Room nine person hash"
                },
                "roomNewestTxId": {
                    "type": "string",
                    "description": "Room newest transaction ID"
                },
                "roomNewestPinId": {
                    "type": "string",
                    "description": "Room newest pin ID"
                },
                "roomNewestMetaId": {
                    "type": "string",
                    "description": "Room newest meta ID"
                },
                "roomNewestUserName": {
                    "type": "string",
                    "description": "Room newest user name"
                },
                "roomNewestProtocol": {
                    "type": "string",
                    "description": "Room newest protocol"
                },
                "roomNewestContent": {
                    "type": "string",
                    "description": "Room newest content"
                },
                "roomNewestTimestamp": {
                    "type": "integer",
                    "description": "Room newest timestamp"
                },
                "createUserMetaId": {
                    "type": "string",
                    "description": "Create user meta ID"
                },
                "createUserAddress": {
                    "type": "string",
                    "description": "Create user address"
                },
                "createUserInfo": {
                    "$ref": "#/definitions/UserInfo"
                },
                "userCount": {
                    "type": "integer",
                    "description": "User count"
                },
                "chatSettingType": {
                    "type": "string",
                    "description": "Chat setting type"
                },
                "deleteStatus": {
                    "type": "string",
                    "description": "Delete status"
                },
                "timestamp": {
                    "type": "integer",
                    "description": "Timestamp"
                },
                "chain": {
                    "type": "string",
                    "description": "Chain"
                },
                "blockHeight": {
                    "type": "integer",
                    "description": "Block height"
                },
                "index": {
                    "type": "integer",
                    "description": "Index"
                }
            }
        },
        "GroupChatResponse": {
            "type": "object",
            "properties": {
                "total": {
                    "type": "integer",
                    "description": "Total count"
                },
                "nextTimestamp": {
                    "type": "integer",
                    "description": "Next timestamp for pagination"
                },
                "lastIndex": {
                    "type": "integer",
                    "description": "Last index for pagination"
                },
                "lastTimestamp": {
                    "type": "integer",
                    "description": "Last timestamp for pagination"
                },
                "list": {
                    "type": "array",
                    "items": {
                        "$ref": "#/definitions/GroupChatItem"
                    },
                    "description": "Chat list"
                }
            }
        },
        "GroupChatItem": {
            "type": "object",
            "properties": {
                "groupId": {
                    "type": "string",
                    "description": "Group ID"
                },
                "metanetId": {
                    "type": "string",
                    "description": "Metanet ID"
                },
                "txId": {
                    "type": "string",
                    "description": "Transaction ID"
                },
                "pinId": {
                    "type": "string",
                    "description": "Pin ID"
                },
                "address": {
                    "type": "string",
                    "description": "User address"
                },
                "metaId": {
                    "type": "string",
                    "description": "User meta ID"
                },
                "userInfo": {
                    "$ref": "#/definitions/UserInfo"
                },
                "nickName": {
                    "type": "string",
                    "description": "Nick name"
                },
                "protocol": {
                    "type": "string",
                    "description": "Protocol"
                },
                "content": {
                    "type": "string",
                    "description": "Content"
                },
                "contentType": {
                    "type": "string",
                    "description": "Content type"
                },
                "encryption": {
                    "type": "string",
                    "description": "Encryption"
                },
                "chatType": {
                    "type": "integer",
                    "description": "Chat type: 0-msg, 1-red, 2-img"
                },
                "replyPin": {
                    "type": "string",
                    "description": "Reply pin"
                },
                "replyInfo": {
                    "$ref": "#/definitions/ReplyInfo"
                },
                "redMetaId": {
                    "type": "string",
                    "description": "Red meta ID"
                },
                "timestamp": {
                    "type": "integer",
                    "description": "Timestamp"
                },
                "chain": {
                    "type": "string",
                    "description": "Chain"
                },
                "blockHeight": {
                    "type": "integer",
                    "description": "Block height"
                },
                "index": {
                    "type": "integer",
                    "description": "Index"
                }
            }
        },
        "ReplyInfo": {
            "type": "object",
            "properties": {
                "pinId": {
                    "type": "string",
                    "description": "Pin ID"
                },
                "metaId": {
                    "type": "string",
                    "description": "Meta ID"
                },
                "address": {
                    "type": "string",
                    "description": "Address"
                },
                "userInfo": {
                    "$ref": "#/definitions/UserInfo"
                },
                "nickName": {
                    "type": "string",
                    "description": "Nick name"
                },
                "protocol": {
                    "type": "string",
                    "description": "Protocol"
                },
                "content": {
                    "type": "string",
                    "description": "Content"
                },
                "contentType": {
                    "type": "string",
                    "description": "Content type"
                },
                "encryption": {
                    "type": "string",
                    "description": "Encryption"
                },
                "chatType": {
                    "type": "integer",
                    "description": "Chat type"
                },
                "timestamp": {
                    "type": "integer",
                    "description": "Timestamp"
                },
                "chain": {
                    "type": "string",
                    "description": "Chain"
                },
                "index": {
                    "type": "integer",
                    "description": "Index"
                }
            }
        },
        "UserInfo": {
            "type": "object",
            "properties": {
                "address": {
                    "type": "string",
                    "description": "User address"
                },
                "metaid": {
                    "type": "string",
                    "description": "User meta ID"
                },
                "name": {
                    "type": "string",
                    "description": "User name"
                },
                "avatar": {
                    "type": "string",
                    "description": "User avatar"
                },
                "bio": {
                    "type": "string",
                    "description": "User bio"
                },
                "chatPublicKey": {
                    "type": "string",
                    "description": "Chat public key"
                },
                "chatPublicKeyId": {
                    "type": "string",
                    "description": "Chat public key ID"
                }
            }
        },
        "GroupMemberResponse": {
            "type": "object",
            "properties": {
                "total": {
                    "type": "integer",
                    "description": "Total count"
                },
                "list": {
                    "type": "array",
                    "items": {
                        "$ref": "#/definitions/GroupMemberItem"
                    },
                    "description": "Member list"
                }
            }
        },
        "GroupMemberItem": {
            "type": "object",
            "properties": {
                "metaId": {
                    "type": "string",
                    "description": "Meta ID"
                },
                "address": {
                    "type": "string",
                    "description": "Address"
                },
                "userInfo": {
                    "$ref": "#/definitions/UserInfo"
                },
                "timeStr": {
                    "type": "string",
                    "description": "Time string"
                },
                "timestamp": {
                    "type": "integer",
                    "description": "Timestamp"
                }
            }
        },
        "GroupPersonResponse": {
            "type": "object",
            "properties": {
                "isInGroup": {
                    "type": "boolean",
                    "description": "Whether in the group"
                },
                "person": {
                    "$ref": "#/definitions/GroupPersonItem"
                }
            }
        },
        "GroupPersonItem": {
            "type": "object",
            "properties": {
                "groupIdMetaIdHash": {
                    "type": "string",
                    "description": "Group ID and member unique identifier"
                },
                "groupId": {
                    "type": "string",
                    "description": "Group ID"
                },
                "metaId": {
                    "type": "string",
                    "description": "Meta ID"
                },
                "address": {
                    "type": "string",
                    "description": "Address"
                },
                "userInfo": {
                    "$ref": "#/definitions/UserInfo"
                },
                "avatarTxId": {
                    "type": "string",
                    "description": "Avatar transaction ID"
                },
                "userName": {
                    "type": "string",
                    "description": "User name"
                },
                "userNickName": {
                    "type": "string",
                    "description": "User nick name"
                },
                "groupState": {
                    "type": "integer",
                    "description": "Group state: 1-in group, -1-left"
                },
                "timestamp": {
                    "type": "integer",
                    "description": "Timestamp"
                },
                "blockHeight": {
                    "type": "integer",
                    "description": "Block height"
                },
                "pinId": {
                    "type": "string",
                    "description": "Pin ID"
                }
            }
        },
        "UserInfoResponse": {
            "type": "object",
            "properties": {
                "address": {
                    "type": "string",
                    "description": "User address"
                },
                "metaId": {
                    "type": "string",
                    "description": "User meta ID"
                },
                "userInfo": {
                    "$ref": "#/definitions/UserInfo"
                }
            }
        },
        "MaxIndexResponse": {
            "type": "object",
            "properties": {
                "groupId": {
                    "type": "string",
                    "description": "Group ID"
                },
                "fromMetaId": {
                    "type": "string",
                    "description": "From meta ID"
                },
                "toMetaId": {
                    "type": "string",
                    "description": "To meta ID"
                },
                "maxIndex": {
                    "type": "integer",
                    "description": "Maximum index"
                }
            }
        },
        "PrivateChatResponse": {
            "type": "object",
            "properties": {
                "total": {
                    "type": "integer",
                    "description": "Total count"
                },
                "nextTimestamp": {
                    "type": "integer",
                    "description": "Next timestamp for pagination"
                },
                "list": {
                    "type": "array",
                    "items": {
                        "$ref": "#/definitions/PrivateChatItem"
                    },
                    "description": "Private chat list"
                }
            }
        },
        "PrivateChatItem": {
            "type": "object",
            "properties": {
                "from": {
                    "type": "string",
                    "description": "From user"
                },
                "fromUserInfo": {
                    "$ref": "#/definitions/UserInfo"
                },
                "to": {
                    "type": "string",
                    "description": "To user"
                },
                "toUserInfo": {
                    "$ref": "#/definitions/UserInfo"
                },
                "txId": {
                    "type": "string",
                    "description": "Transaction ID"
                },
                "pinId": {
                    "type": "string",
                    "description": "Pin ID"
                },
                "metaId": {
                    "type": "string",
                    "description": "Meta ID"
                },
                "address": {
                    "type": "string",
                    "description": "Address"
                },
                "userInfo": {
                    "$ref": "#/definitions/UserInfo"
                },
                "nickName": {
                    "type": "string",
                    "description": "Nick name"
                },
                "protocol": {
                    "type": "string",
                    "description": "Protocol"
                },
                "content": {
                    "type": "string",
                    "description": "Content"
                },
                "contentType": {
                    "type": "string",
                    "description": "Content type"
                },
                "encryption": {
                    "type": "string",
                    "description": "Encryption"
                },
                "chatType": {
                    "type": "integer",
                    "description": "Chat type"
                },
                "replyPin": {
                    "type": "string",
                    "description": "Reply pin"
                },
                "replyInfo": {
                    "$ref": "#/definitions/ReplyInfo"
                },
                "replyMetaId": {
                    "type": "string",
                    "description": "Reply meta ID"
                },
                "timestamp": {
                    "type": "integer",
                    "description": "Timestamp"
                },
                "chain": {
                    "type": "string",
                    "description": "Chain"
                },
                "blockHeight": {
                    "type": "integer",
                    "description": "Block height"
                },
                "index": {
                    "type": "integer",
                    "description": "Index"
                }
            }
        },
        "ChatInfoResponse": {
            "type": "object",
            "properties": {
                "total": {
                    "type": "integer",
                    "description": "Total count"
                },
                "list": {
                    "type": "array",
                    "items": {
                        "$ref": "#/definitions/ChatInfoItem"
                    },
                    "description": "Chat info list"
                }
            }
        },
        "ChatInfoItem": {
            "type": "object",
            "properties": {
                "type": {
                    "type": "string",
                    "description": "Type: 1-group chat, 2-private chat"
                },
                "groupId": {
                    "type": "string",
                    "description": "Group ID (for group chat)"
                },
                "metaId": {
                    "type": "string",
                    "description": "Other party MetaId (for private chat)"
                },
                "address": {
                    "type": "string",
                    "description": "Other party address (for private chat)"
                },
                "timestamp": {
                    "type": "integer",
                    "description": "Latest message timestamp"
                },
                "chatType": {
                    "type": "integer",
                    "description": "Message type 0-msg, 1-red, 2-img"
                },
                "content": {
                    "type": "string",
                    "description": "Message content summary"
                },
                "createMetaId": {
                    "type": "string",
                    "description": "Create meta ID"
                },
                "createAddress": {
                    "type": "string",
                    "description": "Create address"
                },
                "lastMessagePinId": {
                    "type": "string",
                    "description": "Last message pin ID"
                },
                "blockHeight": {
                    "type": "integer",
                    "description": "Block height"
                },
                "index": {
                    "type": "integer",
                    "description": "Index"
                },
                "communityId": {
                    "type": "string",
                    "description": "Community ID"
                },
                "roomName": {
                    "type": "string",
                    "description": "Room name"
                },
                "roomNote": {
                    "type": "string",
                    "description": "Room note"
                },
                "roomType": {
                    "type": "string",
                    "description": "Room type"
                },
                "roomStatus": {
                    "type": "string",
                    "description": "Room status"
                },
                "roomJoinType": {
                    "type": "string",
                    "description": "Room join type"
                },
                "roomAvatarUrl": {
                    "type": "string",
                    "description": "Room avatar URL"
                },
                "createUserMetaId": {
                    "type": "string",
                    "description": "Create user meta ID"
                },
                "createUserAddress": {
                    "type": "string",
                    "description": "Create user address"
                },
                "createUserInfo": {
                    "$ref": "#/definitions/UserInfo"
                },
                "userCount": {
                    "type": "integer",
                    "description": "User count"
                },
                "chatSettingType": {
                    "type": "string",
                    "description": "Chat setting type"
                },
                "deleteStatus": {
                    "type": "string",
                    "description": "Delete status"
                },
                "chain": {
                    "type": "string",
                    "description": "Chain"
                },
                "userInfo": {
                    "$ref": "#/definitions/UserInfo"
                }
            }
        },
        "GroupSearchResponse": {
            "type": "object",
            "properties": {
                "total": {
                    "type": "integer",
                    "description": "Total number of groups found"
                },
                "list": {
                    "type": "array",
                    "items": {
                        "$ref": "#/definitions/GroupSearchItem"
                    },
                    "description": "List of groups"
                }
            }
        },
        "GroupSearchItem": {
            "type": "object",
            "properties": {
                "groupId": {
                    "type": "string",
                    "description": "Group ID"
                },
                "groupName": {
                    "type": "string",
                    "description": "Group name"
                },
                "pinId": {
                    "type": "string",
                    "description": "Pin ID"
                },
                "timestamp": {
                    "type": "integer",
                    "description": "Timestamp"
                }
            }
        },
        "ChatSendableResponse": {
            "type": "object",
            "properties": {
                "sendable": {
                    "type": "boolean",
                    "description": "Whether chat is sendable"
                }
            }
        },
        "GroupMemberSearchItem": {
            "type": "object",
            "properties": {
                "metaId": {
                    "type": "string",
                    "description": "User MetaId"
                },
                "address": {
                    "type": "string",
                    "description": "User address"
                },
                "userInfo": {
                    "$ref": "#/definitions/UserInfo"
                },
                "userName": {
                    "type": "string",
                    "description": "User name from group"
                },
                "userNickName": {
                    "type": "string",
                    "description": "User nickname from group"
                },
                "timestamp": {
                    "type": "integer",
                    "description": "Join timestamp"
                }
            }
        },
        "GroupMemberSearchResponse": {
            "type": "object",
            "properties": {
                "total": {
                    "type": "integer",
                    "description": "Total number of results"
                },
                "list": {
                    "type": "array",
                    "items": {
                        "$ref": "#/definitions/GroupMemberSearchItem"
                    },
                    "description": "Search results"
                }
            }
        },
        "LuckyBagCodeAddressKeyResponse": {
            "type": "object",
            "properties": {
                "code": {
                    "type": "string",
                    "description": "6-digit random code"
                },
                "luckyBagAddress": {
                    "type": "string",
                    "description": "Lucky bag address"
                },
                "timestamp": {
                    "type": "integer",
                    "description": "Creation timestamp"
                }
            }
        },
        "GroupAndUserSearchResponse": {
            "type": "object",
            "properties": {
                "total": {
                    "type": "integer",
                    "description": "Total number of groups and users found"
                },
                "list": {
                    "type": "array",
                    "items": {
                        "$ref": "#/definitions/GroupAndUserSearchItem"
                    },
                    "description": "List of groups and users"
                }
            }
        },
        "GroupAndUserSearchItem": {
            "type": "object",
            "properties": {
                "groupId": {
                    "type": "string",
                    "description": "Group ID"
                },
                "groupName": {
                    "type": "string",
                    "description": "Group name"
                },
                "pinId": {
                    "type": "string",
                    "description": "Pin ID"
                },
                "timestamp": {
                    "type": "integer",
                    "description": "Timestamp"
                },
                "userName": {
                    "type": "string",
                    "description": "User name"
                },
                "userNickName": {
                    "type": "string",
                    "description": "User nickname"
                }
            }
        },
        "GroupChannelResponse": {
            "type": "object",
            "properties": {
                "total": {
                    "type": "integer",
                    "description": "Total count"
                },
                "list": {
                    "type": "array",
                    "items": {
                        "$ref": "#/definitions/GroupChannelItem"
                    },
                    "description": "Channel list"
                }
            }
        },
        "GroupChannelItem": {
            "type": "object",
            "properties": {
                "channelId": {
                    "type": "string",
                    "description": "Channel ID"
                },
                "groupId": {
                    "type": "string",
                    "description": "Group ID"
                },
                "channelName": {
                    "type": "string",
                    "description": "Channel name"
                },
                "channelIcon": {
                    "type": "string",
                    "description": "Channel icon"
                },
                "channelNote": {
                    "type": "string",
                    "description": "Channel note"
                },
                "channelType": {
                    "type": "integer",
                    "description": "Channel type: 0-normal, 1-launch"
                },
                "createUserMetaId": {
                    "type": "string",
                    "description": "Create user MetaId"
                },
                "createUserAddress": {
                    "type": "string",
                    "description": "Create user address"
                },
                "timestamp": {
                    "type": "integer",
                    "description": "Timestamp"
                },
                "chain": {
                    "type": "string",
                    "description": "Chain"
                },
                "blockHeight": {
                    "type": "integer",
                    "description": "Block height"
                },
                "index": {
                    "type": "integer",
                    "description": "Index"
                }
            }
        }
    }
}`

		// Process the document template to adjust paths if needed
		doc := docTemplate

		// Convert absolute paths to relative paths for proper proxy support
		// Replace all "/api/ with "api/ and "/health with "health
		doc = strings.ReplaceAll(doc, `"/api/`, `"api/`)
		doc = strings.ReplaceAll(doc, `"/health`, `"health`)

		c.Data(200, "application/json", []byte(doc))
	})

	// Add group chat module Swagger documentation route with dynamic URL
	router.GET("/group-chat/docs/*any", ginSwagger.WrapHandler(
		swaggerFiles.Handler,
		ginSwagger.URL("../api-docs.json"), // Use relative path to support proxy
	))
}

// GetSwaggerURL Get swagger documentation URL
func GetSwaggerURL(host, port string) string {
	return "http://" + host + ":" + port + "/group-chat/docs/index.html"
}
