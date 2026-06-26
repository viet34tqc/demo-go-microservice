package handler

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/viet34tqc/demo-go-microservice/cmd/todo-service/internal/model"
	"github.com/viet34tqc/demo-go-microservice/cmd/todo-service/middleware"
	"github.com/viet34tqc/demo-go-microservice/cmd/todo-service/repository"
	"gorm.io/gorm"
)

type TodoHandler struct {
	repo *repository.TodoRepository
}

func NewTodoHandler(repo *repository.TodoRepository) *TodoHandler {
	return &TodoHandler{repo: repo}
}

type createTodoRequest struct {
	Title string `json:"title" binding:"required"`
}

type updateTodoRequest struct {
	Title     *string `json:"title"`
	Completed *bool   `json:"completed"`
}

func (h *TodoHandler) Create(c *gin.Context) {
	userID := middleware.GetUserID(c)

	var req createTodoRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "title is required",
		})
		return
	}

	todo := model.Todo{
		UserID:    userID,
		Title:     req.Title,
		Completed: false,
	}

	if err := h.repo.Create(&todo); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to create todo",
		})
		return
	}

	c.JSON(http.StatusCreated, todo)
}

func (h *TodoHandler) List(c *gin.Context) {
	userID := middleware.GetUserID(c)
	todos, err := h.repo.FindAllByUserID(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to get todos",
		})
		return
	}

	c.JSON(http.StatusOK, todos)
}

func (h *TodoHandler) GetByID(c *gin.Context) {
	userID := middleware.GetUserID(c)

	id, ok := parseIDParam(c)
	if !ok {
		return
	}

	todo, err := h.repo.FindByIDAndUserID(id, userID)
	if err != nil {
		handleTodoNotFoundOrError(c, err)
		return
	}

	c.JSON(http.StatusOK, todo)
}

func (h *TodoHandler) Update(c *gin.Context) {
	userID := middleware.GetUserID(c)

	id, ok := parseIDParam(c)
	if !ok {
		return
	}

	var req updateTodoRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid request body",
		})
		return
	}

	todo, err := h.repo.FindByIDAndUserID(id, userID)
	if err != nil {
		handleTodoNotFoundOrError(c, err)
		return
	}

	if req.Title != nil {
		todo.Title = *req.Title
	}

	if req.Completed != nil {
		todo.Completed = *req.Completed
	}

	if err := h.repo.Update(todo); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to update todo",
		})
		return
	}

	c.JSON(http.StatusOK, todo)
}

func (h *TodoHandler) Delete(c *gin.Context) {
	userID := middleware.GetUserID(c)

	id, ok := parseIDParam(c)
	if !ok {
		return
	}

	todo, err := h.repo.FindByIDAndUserID(id, userID)
	if err != nil {
		handleTodoNotFoundOrError(c, err)
		return
	}

	if err := h.repo.Delete(todo); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to delete todo",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "todo deleted",
	})
}

func parseIDParam(c *gin.Context) (uint, bool) {
	idParam := c.Param("id")

	id, err := strconv.ParseUint(idParam, 10, 64)
	if err != nil || id == 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid todo id",
		})
		return 0, false
	}

	return uint(id), true
}

func handleTodoNotFoundOrError(c *gin.Context, err error) {
	if errors.Is(err, gorm.ErrRecordNotFound) {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "todo not found",
		})
		return
	}

	c.JSON(http.StatusInternalServerError, gin.H{
		"error": "failed to get todo",
	})
}
