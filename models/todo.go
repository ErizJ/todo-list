package models

import (
	"errors"
	"smallTodoList/dao"
	"strconv"
)

type Todo struct {
	Id     int    `json:"id" gorm:"primaryKey"`
	Task   string `json:"title" gorm:"size:500;not null"`
	Status bool   `json:"status"`
}

// CreateTodo 新建一个Todo
func CreateTodo(todo *Todo) (err error) {
	if todo.Task == "" {
		return errors.New("待办事项内容不能为空")
	}
	err = dao.DB.Create(todo).Error
	return
}

// DeleteTodo 删除一个Todo
func DeleteTodo(id string) (err error) {
	tid, err := strconv.Atoi(id)
	if err != nil || tid <= 0 {
		return errors.New("无效的 id")
	}
	result := dao.DB.Delete(&Todo{}, tid)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return errors.New("记录不存在")
	}
	return
}

// UpdateTodoStatus 原子操作切换 Todo 状态
func UpdateTodoStatus(id string) (err error) {
	tid, err := strconv.Atoi(id)
	if err != nil || tid <= 0 {
		return errors.New("无效的 id")
	}
	result := dao.DB.Model(&Todo{}).Where("id = ?", tid).
		Update("status", dao.DB.Raw("NOT status"))
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return errors.New("记录不存在")
	}
	return
}

// GetAllTodo 查询所有Todo
func GetAllTodo() (todoList []*Todo, err error) {
	err = dao.DB.Find(&todoList).Error
	return
}
