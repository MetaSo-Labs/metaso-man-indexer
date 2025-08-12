package swagger

import (
	"manindexer/basicprotocols/group_chat/api/swagger/docs"
	"strings"

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

		// 使用group_chat模块的docs包，指定实例名称
		// 延迟加载，避免在包初始化时注册
		var doc string
		func() {
			defer func() {
				if r := recover(); r != nil {
					// 如果注册失败，使用默认文档
					doc = `{"swagger":"2.0","info":{"title":"Group Chat API","version":"1.0"},"paths":{}}`
				}
			}()
			doc = docs.SwaggerInfogroup_chat_swagger.ReadDoc()
		}()

		// 替换host为当前请求的host
		swaggerContent := strings.Replace(doc, `"host": "0.0.0.0:7568"`, `"host": "`+c.Request.Host+`"`, 1)

		c.Data(200, "application/json", []byte(swaggerContent))
	})

	// 添加群聊模块的Swagger文档路由
	router.GET("/group-chat/docs/*any", ginSwagger.WrapHandler(swaggerFiles.Handler, ginSwagger.URL("/group-chat/api-docs.json")))
}

// GetSwaggerURL 获取swagger文档的URL
func GetSwaggerURL(host, port string) string {
	return "http://" + host + ":" + port + "/group-chat/docs/index.html"
}
