package grpc

import (
	"context"
	"errors"
	"fmt"

	"github.com/viet34tqc/demo-go-microservice/cmd/user-service/internal/service"
	userpb "github.com/viet34tqc/demo-go-microservice/gen/go/user/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type UserGRPCServer struct {
	userpb.UnimplementedUserServiceServer

	userService *service.UserService
}

func NewUserGRPCServer(userService *service.UserService) *UserGRPCServer {
	return &UserGRPCServer{
		userService: userService,
	}
}

func (s *UserGRPCServer) Register(ctx context.Context, req *userpb.RegisterRequest) (*userpb.AuthResponse, error) {
	result, err := s.userService.Register(ctx, service.RegisterInput{
		Name:     req.GetName(),
		Email:    req.GetEmail(),
		Password: req.GetPassword(),
	})
	if err != nil {
		return nil, mapUserError(err)
	}

	return &userpb.AuthResponse{
		Token: result.Token,
		User:  toProtoUser(result.User),
	}, nil
}

func (s *UserGRPCServer) Login(ctx context.Context, req *userpb.LoginRequest) (*userpb.AuthResponse, error) {
	result, err := s.userService.Login(ctx, service.LoginInput{
		Email:    req.GetEmail(),
		Password: req.GetPassword(),
	})
	if err != nil {
		return nil, mapUserError(err)
	}

	return &userpb.AuthResponse{
		Token: result.Token,
		User:  toProtoUser(result.User),
	}, nil
}

func (s *UserGRPCServer) GetMe(ctx context.Context, req *userpb.GetMeRequest) (*userpb.UserResponse, error) {
	userID, err := userIDFromProto(req.GetUserId())
	if err != nil {
		return nil, err
	}

	user, err := s.userService.GetMe(ctx, userID)
	if err != nil {
		return nil, mapUserError(err)
	}

	return &userpb.UserResponse{
		User: toProtoUser(*user),
	}, nil
}

func toProtoUser(user service.UserOutput) *userpb.User {
	return &userpb.User{
		Id:    uint64(user.ID),
		Email: user.Email,
		Name:  user.Name,
	}
}

func userIDFromProto(userID uint64) (uint, error) {
	if userID > uint64(^uint(0)) {
		return 0, status.Error(codes.InvalidArgument, fmt.Sprintf("user_id %d is out of range", userID))
	}

	return uint(userID), nil
}

func mapUserError(err error) error {
	switch {
	case errors.Is(err, service.ErrInvalidInput):
		return status.Error(codes.InvalidArgument, err.Error())

	case errors.Is(err, service.ErrEmailAlreadyExists):
		return status.Error(codes.AlreadyExists, err.Error())

	case errors.Is(err, service.ErrInvalidCredentials):
		return status.Error(codes.Unauthenticated, err.Error())

	case errors.Is(err, service.ErrUserNotFound):
		return status.Error(codes.NotFound, err.Error())

	default:
		return status.Error(codes.Internal, "internal server error")
	}
}
