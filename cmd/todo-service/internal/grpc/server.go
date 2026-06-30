package grpc

import (
	"context"
	"errors"

	"github.com/viet34tqc/demo-go-microservice/cmd/todo-service/service"
	todopb "github.com/viet34tqc/demo-go-microservice/gen/go/todo/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type TodoGRPCServer struct {
	todopb.UnimplementedTodoServiceServer

	todoService *service.TodoService
}

func NewTodoGRPCServer(todoService *service.TodoService) *TodoGRPCServer {
	return &TodoGRPCServer{
		todoService: todoService,
	}
}

func (s *TodoGRPCServer) CreateTodo(ctx context.Context, req *todopb.CreateTodoRequest) (*todopb.TodoResponse, error) {
	todo, err := s.todoService.CreateTodo(ctx, service.CreateTodoInput{
		UserID: uint(req.GetUserId()),
		Title:  req.GetTitle(),
	})
	if err != nil {
		return nil, mapTodoError(err)
	}

	return toProtoTodoResponse(todo), nil
}

func (s *TodoGRPCServer) GetTodo(ctx context.Context, req *todopb.GetTodoRequest) (*todopb.TodoResponse, error) {
	todo, err := s.todoService.GetTodo(
		ctx,
		uint(req.GetUserId()),
		uint(req.GetTodoId()),
	)
	if err != nil {
		return nil, mapTodoError(err)
	}

	return toProtoTodoResponse(todo), nil
}

func (s *TodoGRPCServer) ListTodos(ctx context.Context, req *todopb.ListTodosRequest) (*todopb.ListTodosResponse, error) {
	todos, err := s.todoService.ListTodos(ctx, uint(req.GetUserId()))
	if err != nil {
		return nil, mapTodoError(err)
	}

	protoTodos := make([]*todopb.Todo, 0, len(todos))
	for i := range todos {
		protoTodos = append(protoTodos, toProtoTodo(&todos[i]))
	}

	return &todopb.ListTodosResponse{
		Todos: protoTodos,
	}, nil
}

func (s *TodoGRPCServer) UpdateTodo(ctx context.Context, req *todopb.UpdateTodoRequest) (*todopb.TodoResponse, error) {
	todo, err := s.todoService.UpdateTodo(ctx, service.UpdateTodoInput{
		UserID:    uint(req.GetUserId()),
		TodoID:    uint(req.GetTodoId()),
		Title:     req.GetTitle(),
		Completed: req.GetCompleted(),
	})
	if err != nil {
		return nil, mapTodoError(err)
	}

	return toProtoTodoResponse(todo), nil
}

func (s *TodoGRPCServer) DeleteTodo(ctx context.Context, req *todopb.DeleteTodoRequest) (*todopb.DeleteTodoResponse, error) {
	err := s.todoService.DeleteTodo(
		ctx,
		uint(req.GetUserId()),
		uint(req.GetTodoId()),
	)
	if err != nil {
		return nil, mapTodoError(err)
	}

	return &todopb.DeleteTodoResponse{
		Success: true,
	}, nil
}

func toProtoTodo(todo *service.TodoOutput) *todopb.Todo {
	return &todopb.Todo{
		Id:        uint64(todo.ID),
		UserId:    uint64(todo.UserID),
		Title:     todo.Title,
		Completed: todo.Completed,
	}
}

func toProtoTodoResponse(todo *service.TodoOutput) *todopb.TodoResponse {
	return &todopb.TodoResponse{
		Todo: toProtoTodo(todo),
	}
}

func mapTodoError(err error) error {
	switch {
	case errors.Is(err, service.ErrInvalidInput):
		return status.Error(codes.InvalidArgument, err.Error())

	case errors.Is(err, service.ErrTodoNotFound):
		return status.Error(codes.NotFound, err.Error())

	case errors.Is(err, service.ErrForbidden):
		return status.Error(codes.PermissionDenied, err.Error())

	default:
		return status.Error(codes.Internal, "internal server error")
	}
}
