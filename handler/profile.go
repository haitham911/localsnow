package handler

import (
	"context"
	"fmt"

	"github.com/gulfcoastdevops/snow/auth"
	pb "github.com/gulfcoastdevops/snow/proto"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// ShowProfile gets a profile
func (h *Handler) ShowProfile(ctx context.Context, req *pb.ShowProfileRequest) (*pb.ProfileResponse, error) {
	h.logger.Infof("show profile req", req)

	session, err := auth.CheckSessionId(ctx)
	if err != nil || session == nil {
		h.logger.Errorf("unauthenticated", err)
		return nil, status.Errorf(codes.Unauthenticated, "unauthenticated")
	}

	currentUser, err := h.us.GetByEmail(session.Login)
	if err != nil {
		h.logger.Errorf("current user not found", err)
		return nil, status.Error(codes.NotFound, "user not found")
	}

	requestUser, err := h.us.GetByUsername(req.GetUsername())
	if err != nil {
		msg := "user was not found"
		h.logger.Errorf(msg, err)
		return nil, status.Error(codes.NotFound, msg)
	}

	following, err := h.us.IsFollowing(currentUser, requestUser)
	if err != nil {
		msg := "failed to get following status"
		h.logger.Errorf(msg, err)
		return nil, status.Error(codes.NotFound, "internal server error")
	}

	return &pb.ProfileResponse{Profile: requestUser.ProtoProfile(following)}, nil
}

// FollowUser follow a user
func (h *Handler) FollowUser(ctx context.Context, req *pb.FollowRequest) (*pb.ProfileResponse, error) {
	h.logger.Infof("follow user req", req)

	session, err := auth.CheckSessionId(ctx)
	if err != nil || session == nil {
		h.logger.Errorf("unauthenticated", err)
		return nil, status.Errorf(codes.Unauthenticated, "unauthenticated")
	}

	currentUser, err := h.us.GetByEmail(session.Login)
	if err != nil {
		h.logger.Errorf("current user not found", err)
		return nil, status.Error(codes.NotFound, "user not found")
	}

	if currentUser.Username == req.GetUsername() {
		h.logger.Error("cannot follow yourself")
		return nil, status.Error(codes.InvalidArgument, "cannot follow yourself")
	}

	requestUser, err := h.us.GetByUsername(req.GetUsername())
	if err != nil {
		h.logger.Error(fmt.Errorf("user not found: %w", err))
		return nil, status.Error(codes.NotFound, "user was not found")
	}

	err = h.us.Follow(currentUser, requestUser)
	if err != nil {
		msg := fmt.Sprintf("failed to follow user: (ID: %d) -> (ID: %d)",
			currentUser.ID, requestUser.ID)
		h.logger.Errorf(msg, err)
		return nil, status.Error(codes.Aborted, "failed to follow user")
	}

	return &pb.ProfileResponse{Profile: requestUser.ProtoProfile(true)}, nil
}

// UnfollowUser unfollow a user
func (h *Handler) UnfollowUser(ctx context.Context, req *pb.UnfollowRequest) (*pb.ProfileResponse, error) {
	h.logger.Infof("unfollow user req", req)
	session, err := auth.CheckSessionId(ctx)
	if err != nil || session == nil {
		h.logger.Errorf("unauthenticated", err)
		return nil, status.Errorf(codes.Unauthenticated, "unauthenticated")
	}

	currentUser, err := h.us.GetByEmail(session.Login)
	if err != nil {
		h.logger.Errorf("current user not found", err)
		return nil, status.Error(codes.NotFound, "user not found")
	}

	if currentUser.Username == req.GetUsername() {
		h.logger.Error("cannot follow yourself")
		return nil, status.Error(codes.InvalidArgument, "cannot follow yourself")
	}

	requestUser, err := h.us.GetByUsername(req.GetUsername())
	if err != nil {
		msg := "user was not found"
		h.logger.Errorf(msg, err)
		return nil, status.Error(codes.NotFound, msg)
	}

	following, err := h.us.IsFollowing(currentUser, requestUser)
	if err != nil {
		msg := "failed to get following status"
		h.logger.Errorf(msg, err)
		return nil, status.Error(codes.NotFound, "internal server error")
	}

	if !following {
		h.logger.Errorf("current user is not following request user", err)
		return nil, status.Errorf(codes.Unauthenticated, "you are not following the user")
	}

	err = h.us.Unfollow(currentUser, requestUser)
	if err != nil {
		msg := fmt.Sprintf("failed to unfollow user: (ID: %d) -> (ID: %d)",
			currentUser.ID, requestUser.ID)
		h.logger.Errorf(msg, err)
		return nil, status.Error(codes.Aborted, "failed to unfollow user")
	}

	return &pb.ProfileResponse{Profile: requestUser.ProtoProfile(false)}, nil
}
