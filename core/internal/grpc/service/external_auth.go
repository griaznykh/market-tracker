package service

import (
	"context"
	"errors"
	pb "lib/schema/gen/go/api/core/v1"

	"market-service/internal/db"
	"market-service/internal/schema"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (s *ExternalService) Register(
	ctx context.Context,
	req *pb.RegisterRequest,
) (*pb.RegisterResponse, error) {

	passwordHash, err := bcrypt.GenerateFromPassword(
		[]byte(req.Password),
		bcrypt.DefaultCost,
	)
	if err != nil {
		return nil, status.Error(
			codes.Internal,
			"failed to hash password",
		)
	}

	user := &schema.User{
		Id:       uuid.New(),
		Email:    req.Email,
		Password: string(passwordHash),
	}

	err = s.db.CreateUser(
		ctx,
		user,
	)
	if err != nil {
		if errors.Is(err, db.ErrUserAlreadyExist) {
			return nil, status.Error(
				codes.AlreadyExists,
				"user already exist",
			)
		}

		return nil, status.Error(
			codes.Internal,
			"failed to create user",
		)
	}

	return &pb.RegisterResponse{
		User: user.ToPB(),
	}, nil
}

func (s *ExternalService) Login(
	ctx context.Context,
	req *pb.LoginRequest,
) (*pb.LoginResponse, error) {
	user, err := s.db.GetUserByEmail(ctx, req.Email)
	if err != nil {
		return nil, status.Error(
			codes.NotFound,
			"user not found",
		)
	}

	err = bcrypt.CompareHashAndPassword(
		[]byte(user.Password),
		[]byte(req.Password),
	)
	if err != nil {
		return nil, status.Error(
			codes.Unauthenticated,
			"invalid credentials",
		)
	}

	token, err := s.jwtManager.Generate(ctx, user.Id)
	if err != nil {
		return nil, status.Error(
			codes.Unauthenticated,
			"generate token faild",
		)
	}

	return &pb.LoginResponse{
		User:        user.ToPB(),
		AccessToken: token,
	}, nil
}
