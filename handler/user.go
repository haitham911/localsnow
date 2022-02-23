package handler

import (
	"context"
	"fmt"

	"github.com/gulfcoastdevops/snow/auth"
	"github.com/gulfcoastdevops/snow/model"
	pb "github.com/gulfcoastdevops/snow/proto"
	"github.com/opentracing/opentracing-go"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// LoginUser is existing user login
func (h *Handler) LoginUser(ctx context.Context, req *pb.LoginUserRequest) (*pb.UserResponse, error) {
	span, _ := opentracing.StartSpanFromContext(ctx, "users.LoginUser")
	defer span.Finish()
	h.logger.Info("login user req", req)
	u, err := h.us.GetByEmail(req.GetUser().GetEmail())
	if err != nil {
		msg := "invalid email or password"
		err = fmt.Errorf("failed to login due to wrong email: %w", err)
		h.logger.Errorf(msg, err)
		return nil, status.Error(codes.InvalidArgument, msg)
	}

	if !u.CheckPassword(req.GetUser().GetPassword()) {
		h.logger.Errorf("failed to login due to receive wrong password: %s", u.Email)
		return nil, status.Error(codes.InvalidArgument, "invalid email or password")
	}

	token, err := auth.GenerateToken(u.ID)
	if err != nil {
		msg := "internal server error"
		err := fmt.Errorf("Failed to create token. %w", err)
		h.logger.Errorf(msg, err)
		return nil, status.Error(codes.Aborted, msg)
	}

	return &pb.UserResponse{User: u.ProtoUser(token)}, nil
}

// CreateUser registers a new user
func (h *Handler) CreateUser(ctx context.Context, req *pb.CreateUserRequest) (*pb.UserResponse, error) {
	h.logger.Infof("create user req", req)
	u := model.User{
		Username: req.User.GetUsername(),
		Email:    req.User.GetEmail(),
		Password: req.User.GetPassword(),
	}

	err := u.Validate()
	if err != nil {
		msg := "validation error"
		err = fmt.Errorf("validation error: %w", err)
		h.logger.Errorf(msg, err)
		return nil, status.Error(codes.InvalidArgument, msg)
	}

	err = u.HashPassword()
	if err != nil {
		msg := "internal server error"
		err := fmt.Errorf("Failed to hash password, %w", err)
		h.logger.Errorf(msg, err)
		return nil, status.Error(codes.Aborted, err.Error())
	}

	err = h.us.Create(&u)
	if err != nil {
		msg := "internal server error"
		err := fmt.Errorf("Failed to create user. %w", err)
		h.logger.Errorf(msg, err)
		return nil, status.Error(codes.Canceled, msg)
	}

	token, err := auth.GenerateToken(u.ID)
	if err != nil {
		msg := "internal server error"
		err := fmt.Errorf("Failed to create token. %w", err)
		h.logger.Errorf(msg, err)
		return nil, status.Error(codes.Aborted, msg)
	}

	return &pb.UserResponse{User: u.ProtoUser(token)}, nil
}

// CurrentUser gets a current user
func (h *Handler) CurrentUser(ctx context.Context, req *pb.Empty) (*pb.UserResponse, error) {
	span, _ := opentracing.StartSpanFromContext(ctx, "users.createUser")
	defer span.Finish()
	h.logger.Infof("get current user req", req)

	userID, err := auth.GetUserID(ctx)
	if err != nil {
		msg := "unauthenticated"
		h.logger.Errorf(msg, err)
		return nil, status.Errorf(codes.Unauthenticated, msg)
	}

	u, err := h.us.GetByID(userID)
	if err != nil {
		msg := "user not found"
		err = fmt.Errorf("token is valid but the user not found: %w", err)
		h.logger.Errorf(msg, err)
		return nil, status.Error(codes.NotFound, msg)
	}

	token, err := auth.GenerateToken(u.ID)
	if err != nil {
		msg := "internal server error"
		err := fmt.Errorf("Failed to create token. %w", err)
		h.logger.Error(err)

		return nil, status.Error(codes.Aborted, msg)
	}

	return &pb.UserResponse{User: u.ProtoUser(token)}, nil
}

// UpdateUser updates current user
func (h *Handler) UpdateUser(ctx context.Context, req *pb.UpdateUserRequest) (*pb.UserResponse, error) {
	h.logger.Info("update user request")
	userID, err := auth.GetUserID(ctx)
	if err != nil {
		msg := "unauthenticated"
		h.logger.Errorf(msg, err)
		return nil, status.Errorf(codes.Unauthenticated, msg)
	}

	u, err := h.us.GetByID(userID)
	if err != nil {
		msg := "not user found"
		err = fmt.Errorf("token is valid but the user not found: %w", err)
		h.logger.Errorf(msg, err)
		return nil, status.Error(codes.NotFound, msg)
	}

	// update non zero-valu fields eonly
	username := req.GetUser().GetUsername()
	if username != "" {
		u.Username = username
	}

	email := req.GetUser().GetEmail()
	if email != "" {
		u.Email = email
	}

	password := req.GetUser().GetPassword()
	if password != "" {
		u.Password = password
	}

	image := req.GetUser().GetImage()
	if image != "" {
		u.Image = image
	}

	bio := req.GetUser().GetBio()
	if bio != "" {
		u.Bio = bio
	}

	// validation
	err = u.Validate()
	if err != nil {
		err = fmt.Errorf("validation error: %w", err)
		h.logger.Errorf("validation error", err)
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	if req.GetUser().GetPassword() != "" {
		err = u.HashPassword()
		if err != nil {
			msg := "internal server error"
			err := fmt.Errorf("Failed to hash password, %w", err)
			h.logger.Errorf(msg, err)
			return nil, status.Error(codes.Aborted, msg)
		}
	}

	err = h.us.Update(u)
	if err != nil {
		msg := "internal server error"
		err = fmt.Errorf("failed to update user: %w", err)
		h.logger.Errorf(msg, err)
		return nil, status.Error(codes.InvalidArgument, msg)
	}

	token, err := auth.GenerateToken(u.ID)
	if err != nil {
		msg := "internal server error"
		err := fmt.Errorf("Failed to create token. %w", err)
		h.logger.Errorf(msg, err)
		return nil, status.Error(codes.Aborted, msg)
	}

	return &pb.UserResponse{User: u.ProtoUser(token)}, nil
}
