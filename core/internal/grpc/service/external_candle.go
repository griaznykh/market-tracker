package service

import (
	"context"
	pb "lib/schema/gen/go/api/core/v1"
)

func (s *ExternalService) GetCandles(
	ctx context.Context,
	req *pb.GetCandlesRequest,
) (*pb.GetCandlesResponse, error) {
	return &pb.GetCandlesResponse{}, nil
}
