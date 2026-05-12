package schema

import (
	pb "lib/schema/gen/go/api/core/v1"
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"
)

type Candle struct {
	Ticker   string    `db:"ticker"`
	Interval string    `db:"interval"`
	Open     float64   `db:"open"`
	High     float64   `db:"high"`
	Low      float64   `db:"low"`
	Close    float64   `db:"close"`
	Volume   float64   `db:"volume"`
	Time     time.Time `db:"time"`
}

func (c Candle) ToPB() *pb.Candle {
	return &pb.Candle{
		Ticker:   c.Ticker,
		Interval: c.Interval,
		Open:     c.Open,
		High:     c.High,
		Low:      c.Low,
		Close:    c.Close,
		Volume:   c.Volume,
		Time:     timestamppb.New(c.Time),
	}
}
