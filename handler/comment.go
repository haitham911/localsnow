package handler

import (
	"context"
	"fmt"
	"strconv"

	"github.com/gulfcoastdevops/snow/auth"
	"github.com/gulfcoastdevops/snow/model"
	pb "github.com/gulfcoastdevops/snow/proto"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// CreateComment create a comment for an article
func (h *Handler) CreateComment(ctx context.Context, req *pb.CreateCommentRequest) (*pb.CommentResponse, error) {
	h.logger.Infof("Create comment | req: %+v", req)

	// get current user
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

	// get article
	articleID, err := strconv.Atoi(req.GetSlug())
	if err != nil {
		msg := fmt.Sprintf("cannot convert slug (%s) into integer", req.GetSlug())
		h.logger.Errorf(msg, err)
		return nil, status.Error(codes.InvalidArgument, "invalid article id")
	}

	article, err := h.as.GetByID(uint(articleID))
	if err != nil {
		msg := fmt.Sprintf("requested article (slug=%d) not found", articleID)
		h.logger.Errorf(msg, err)
		return nil, status.Error(codes.InvalidArgument, "invalid article id")
	}

	// new comment
	comment := model.Comment{
		Body:      req.GetComment().GetBody(),
		Author:    *currentUser,
		ArticleID: article.ID,
	}

	err = comment.Validate()
	if err != nil {
		err = fmt.Errorf("validation error: %w", err)
		h.logger.Errorf("validation error", err)
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	// create comment
	err = h.as.CreateComment(&comment)
	if err != nil {
		msg := "failed to create comment."
		h.logger.Errorf(msg, err)
		return nil, status.Error(codes.Aborted, msg)
	}

	// map model.Comment to pb.Comment
	pc := comment.ProtoComment()
	pc.Author = currentUser.ProtoProfile(false)

	return &pb.CommentResponse{Comment: pc}, nil
}

// GetComments gets comments of the article
func (h *Handler) GetComments(ctx context.Context, req *pb.GetCommentsRequest) (*pb.CommentsResponse, error) {
	h.logger.Infof("Get comments | req: %+v", req)
	// get article
	articleID, err := strconv.Atoi(req.GetSlug())
	if err != nil {
		msg := fmt.Sprintf("cannot convert slug (%s) into integer", req.GetSlug())
		h.logger.Errorf(msg, err)
		return nil, status.Error(codes.InvalidArgument, "invalid article id")
	}

	article, err := h.as.GetByID(uint(articleID))
	if err != nil {
		msg := fmt.Sprintf("requested article (slug=%d) not found", articleID)
		h.logger.Errorf(msg, err)
		return nil, status.Error(codes.InvalidArgument, "invalid article id")
	}

	comments, err := h.as.GetComments(article)
	if err != nil {
		msg := "failed to get comments"
		h.logger.Errorf(msg, err)
		return nil, status.Error(codes.Aborted, msg)
	}

	var currentUser *model.User
	userID, err := auth.GetUserID(ctx)
	if err == nil {
		currentUser, err = h.us.GetByID(userID)
		if err != nil {
			h.logger.Errorf("current user not found", err)

			return nil, status.Error(codes.NotFound, "user not found")
		}
	}

	pcs := make([]*pb.Comment, 0, len(comments))
	for _, c := range comments {
		pc := c.ProtoComment()

		// get whether current user follows article author
		following, err := h.us.IsFollowing(currentUser, &c.Author)
		if err != nil {
			msg := "failed to get following status"
			h.logger.Errorf(msg, err)
			return nil, status.Error(codes.NotFound, "internal server error")
		}
		pc.Author = c.Author.ProtoProfile(following)

		pcs = append(pcs, pc)
	}

	return &pb.CommentsResponse{Comments: pcs}, nil
}

// DeleteComment delete a commnet of the article
func (h *Handler) DeleteComment(ctx context.Context, req *pb.DeleteCommentRequest) (*pb.Empty, error) {
	h.logger.Infof("Delete comment | req: %+v", req)

	// get current user
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

	commentID, err := strconv.Atoi(req.GetId())
	if err != nil {
		msg := fmt.Sprintf("cannot convert id (%s) into integer", req.GetId())
		h.logger.Errorf(msg, err)
		return nil, status.Error(codes.InvalidArgument, "invalid article id")
	}

	comment, err := h.as.GetCommentByID(uint(commentID))
	if err != nil {
		msg := "failed to get comment"
		h.logger.Errorf(msg, err)
		return nil, status.Error(codes.InvalidArgument, msg)
	}

	if req.GetSlug() != fmt.Sprintf("%d", comment.ArticleID) {
		msg := "the comment is not in the article"
		h.logger.Errorf(msg, err)
		return nil, status.Error(codes.InvalidArgument, msg)
	}

	if comment.UserID != currentUser.ID {
		msg := "forbidden"
		h.logger.Errorf(msg, err)
		return nil, status.Error(codes.InvalidArgument, msg)
	}

	err = h.as.DeleteComment(comment)
	if err != nil {
		msg := "failed to delete comment"
		h.logger.Errorf(msg, err)
		return nil, status.Error(codes.InvalidArgument, msg)
	}

	return &pb.Empty{}, nil
}
