package schema

import (
	pb "lib/schema/gen/go/api/core/v1"
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"
)

type Token struct {
	Id        string
	UserId    string
	Token     string
	CreatedAt time.Time
}

func (t Token) ToPB() *pb.Token {
	return &pb.Token{
		Id:        t.Id,
		UserId:    t.UserId,
		Token:     t.Token,
		CreatedAt: timestamppb.New(t.CreatedAt),
	}
}
