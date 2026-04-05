package controller

import (
	"github.com/gin-gonic/gin"
	"net/http"
	"smallTodoList/models"
)

const (
	msgFail    = "fail"
	msgSuccess = "success"
)

// InitIndexHtml 初始化 index 页面
func InitIndexHtml(c *gin.Context) {
	c.HTML(http.StatusOK, "index.html", nil)
}

// CreateTodo 新增一个 Todo
func CreateTodo(c *gin.Context) {
	var todo models.Todo
	if err := c.ShouldBindJSON(&todo); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"msg": msgFail, "data": err.Error()})
		return
	}
	if err := models.CreateTodo(&todo); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"msg": msgFail, "data": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"msg": msgSuccess, "data": todo})
}

// DeleteTodo 删除一个 Todo
func DeleteTodo(c *gin.Context) {
	id := c.Param("id")
	if err := models.DeleteTodo(id); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"msg": msgFail, "data": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"msg": msgSuccess, "data": nil})
}

// UpdateTodoStatus 切换 Todo 完成状态
func UpdateTodoStatus(c *gin.Context) {
	id := c.Param("id")
	if err := models.UpdateTodoStatus(id); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"msg": msgFail, "data": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"msg": msgSuccess, "data": nil})
}

// GetAllTodo 查询所有 Todo
func GetAllTodo(c *gin.Context) {
	todos, err := models.GetAllTodo()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"msg": msgFail, "data": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"msg": msgSuccess, "data": todos})
}
