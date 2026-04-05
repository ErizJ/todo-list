package routers

import (
	"github.com/gin-gonic/gin"
	"os"
	"smallTodoList/controller"
)

// SetupRouter 注册路由并启动服务
func SetupRouter() {
	r := gin.Default()

	r.Static("/static", "statics")
	r.LoadHTMLGlob("templates/*")
	r.GET("/", controller.InitIndexHtml)

	v1 := r.Group("/v1")
	{
		v1.POST("/todo", controller.CreateTodo)
		v1.DELETE("/todo/:id", controller.DeleteTodo)
		v1.PUT("/todo/:id", controller.UpdateTodoStatus)
		v1.GET("/todo", controller.GetAllTodo)
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "9090"
	}
	if err := r.Run(":" + port); err != nil {
		panic(err)
	}
}
