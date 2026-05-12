package service

import (
	"context"
	pb "lib/schema/gen/go/api/core/v1"
	"market-service/internal/schema"
)

func (s *ExternalService) GetCandles(
	ctx context.Context,
	req *pb.GetCandlesRequest,
) (*pb.GetCandlesResponse, error) {
	candles := []*pb.Candle{
		schema.Candle{
			Ticker: "APPL",
		}.ToPB(),
	}

	return &pb.GetCandlesResponse{
		Candles: candles,
	}, nil
}
