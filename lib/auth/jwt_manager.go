package auth

import (
	"context"
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

type (
	JwtClaims struct {
		Id string `json:"id"`

		jwt.RegisteredClaims
	}

	JwtManager interface {
		Generate(ctx context.Context, userId uuid.UUID) (token string, err error)
		Verify(ctx context.Context, token string) (claims *JwtClaims, err error)
	}
)

type jwtManager struct {
	secret   string
	duration time.Duration
}

func NewJwtManager(secret string, duration time.Duration) JwtManager {
	return &jwtManager{
		secret:   secret,
		duration: duration,
	}
}

func (r *jwtManager) Generate(ctx context.Context, userId uuid.UUID) (token string, err error) {
	now := time.Now()

	claims := JwtClaims{
		Id: userId.String(),

		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userId.String(),
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(r.duration)),
			NotBefore: jwt.NewNumericDate(now),
		},
	}

	unsignedToken := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signedToken, err := unsignedToken.SignedString([]byte(r.secret))
	if err != nil {
		return "", err
	}

	return signedToken, nil
}

func (r *jwtManager) Verify(ctx context.Context, token string) (claims *JwtClaims, err error) {
	decode, err := jwt.ParseWithClaims(
		token,
		&JwtClaims{},
		func(token *jwt.Token) (any, error) {
			_, ok := token.Method.(*jwt.SigningMethodHMAC)
			if !ok {
				return nil, errors.New("unexpected signing method")
			}

			return []byte(r.secret), nil
		},
	)

	if err != nil {
		return nil, err
	}

	claims, ok := decode.Claims.(*JwtClaims)
	if !ok {
		return nil, errors.New("invalid claims")
	}

	return claims, nil
}
