package service

import (
	"context"
	pb "lib/schema/gen/go/api/core/v1"
)

func (s *ExternalService) Register(
	ctx context.Context,
	req *pb.RegisterRequest,
) (*pb.RegisterResponse, error) {
	return &pb.RegisterResponse{}, nil
}

func (s *ExternalService) CreateApiToken(
	ctx context.Context,
	req *pb.CreateApiTokenRequest,
) (*pb.CreateApiTokenResponse, error) {
	return &pb.CreateApiTokenResponse{}, nil
}
