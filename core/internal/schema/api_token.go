package schema

import (
	pb "lib/schema/gen/go/api/core/v1"
	"time"

	"github.com/google/uuid"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type ApiToken struct {
	Id        uuid.UUID `db:"id"`
	UserId    uuid.UUID `db:"user_id"`
	Token     string    `db:"token"`
	CreatedAt time.Time `db:"created_at"`
}

func (t ApiToken) ToPB() *pb.Token {
	return &pb.Token{
		Id:        t.Id.String(),
		UserId:    t.UserId.String(),
		Token:     t.Token,
		CreatedAt: timestamppb.New(t.CreatedAt),
	}
}
