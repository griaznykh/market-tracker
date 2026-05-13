package schema

import (
	pb "lib/schema/gen/go/api/core/v1"
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"
)

type ApiToken struct {
	Id        string    `db:"id"`
	UserId    string    `db:"user_id"`
	Token     string    `db:"token"`
	CreatedAt time.Time `db:"created_at"`
}

func (t ApiToken) ToPB() *pb.Token {
	return &pb.Token{
		Id:        t.Id,
		UserId:    t.UserId,
		Token:     t.Token,
		CreatedAt: timestamppb.New(t.CreatedAt),
	}
}
