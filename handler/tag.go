package handler

import (
	"context"

	pb "github.com/gulfcoastdevops/snow/proto"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// GetTags returns all of tags
func (h *Handler) GetTags(ctx context.Context, req *pb.Empty) (*pb.TagsResponse, error) {
	h.logger.Infof("get tags req", req)
	tags, err := h.as.GetTags()
	if err != nil {
		h.logger.Errorf("faield to get tags", err)
		return nil, status.Error(codes.Aborted, "internal server error")
	}

	tagNames := make([]string, 0, len(tags))
	for _, t := range tags {
		tagNames = append(tagNames, t.Name)
	}

	return &pb.TagsResponse{Tags: tagNames}, nil
}
