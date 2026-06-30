package service

import (
	"context"
	"errors"
	"strings"

	"github.com/viet34tqc/demo-go-microservice/cmd/todo-service/internal/model"
	"gorm.io/gorm"
)

var (
	ErrInvalidInput = errors.New("invalid input")
	ErrTodoNotFound = errors.New("todo not found")
	ErrForbidden    = errors.New("forbidden")
)

type TodoService struct {
	db *gorm.DB
}

type CreateTodoInput struct {
	UserID uint
	Title  string
}

type UpdateTodoInput struct {
	UserID    uint
	TodoID    uint
	Title     string
	Completed bool
}

type TodoOutput struct {
	ID        uint
	UserID    uint
	Title     string
	Completed bool
}

func NewTodoService(db *gorm.DB) *TodoService {
	return &TodoService{
		db: db,
	}
}

func (s *TodoService) CreateTodo(ctx context.Context, input CreateTodoInput) (*TodoOutput, error) {
	title := strings.TrimSpace(input.Title)

	if input.UserID == 0 || title == "" {
		return nil, ErrInvalidInput
	}

	todo := model.Todo{
		UserID:    input.UserID,
		Title:     title,
		Completed: false,
	}

	if err := s.db.WithContext(ctx).Create(&todo).Error; err != nil {
		return nil, err
	}

	return toTodoOutput(todo), nil
}

func (s *TodoService) GetTodo(ctx context.Context, userID uint, todoID uint) (*TodoOutput, error) {
	if userID == 0 || todoID == 0 {
		return nil, ErrInvalidInput
	}

	var todo model.Todo
	if err := s.db.WithContext(ctx).First(&todo, todoID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrTodoNotFound
		}

		return nil, err
	}

	if todo.UserID != userID {
		return nil, ErrForbidden
	}

	return toTodoOutput(todo), nil
}

func (s *TodoService) ListTodos(ctx context.Context, userID uint) ([]TodoOutput, error) {
	if userID == 0 {
		return nil, ErrInvalidInput
	}

	var todos []model.Todo
	if err := s.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Order("id DESC").
		Find(&todos).Error; err != nil {
		return nil, err
	}

	results := make([]TodoOutput, 0, len(todos))
	for _, todo := range todos {
		results = append(results, *toTodoOutput(todo))
	}

	return results, nil
}

func (s *TodoService) UpdateTodo(ctx context.Context, input UpdateTodoInput) (*TodoOutput, error) {
	title := strings.TrimSpace(input.Title)

	if input.UserID == 0 || input.TodoID == 0 || title == "" {
		return nil, ErrInvalidInput
	}

	var todo model.Todo
	if err := s.db.WithContext(ctx).First(&todo, input.TodoID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrTodoNotFound
		}

		return nil, err
	}

	if todo.UserID != input.UserID {
		return nil, ErrForbidden
	}

	todo.Title = title
	todo.Completed = input.Completed

	if err := s.db.WithContext(ctx).Save(&todo).Error; err != nil {
		return nil, err
	}

	return toTodoOutput(todo), nil
}

func (s *TodoService) DeleteTodo(ctx context.Context, userID uint, todoID uint) error {
	if userID == 0 || todoID == 0 {
		return ErrInvalidInput
	}

	var todo model.Todo
	if err := s.db.WithContext(ctx).First(&todo, todoID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrTodoNotFound
		}

		return err
	}

	if todo.UserID != userID {
		return ErrForbidden
	}

	if err := s.db.WithContext(ctx).Delete(&todo).Error; err != nil {
		return err
	}

	return nil
}

func toTodoOutput(todo model.Todo) *TodoOutput {
	return &TodoOutput{
		ID:        todo.ID,
		UserID:    todo.UserID,
		Title:     todo.Title,
		Completed: todo.Completed,
	}
}
