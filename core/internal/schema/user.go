package schema

import (
	pb "lib/schema/gen/go/api/core/v1"
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"
)

type User struct {
	Id        string    `db:"id"`
	Email     string    `db:"email"`
	Password  string    `db:"password"`
	CreatedAt time.Time `db:"created_at"`
}

func (u User) ToPB() *pb.User {
	return &pb.User{
		Id:        u.Id,
		Email:     u.Email,
		Password:  u.Password,
		CreatedAt: timestamppb.New(u.CreatedAt),
	}
}
