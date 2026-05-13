package registry

import (
	"context"
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type (
	JwtClaims struct {
		Id string `json:"id"`

		jwt.RegisteredClaims
	}

	JwtRegistry interface {
		Generate(ctx context.Context, userId string) (token string, err error)
		Verify(ctx context.Context, token string) (claims *JwtClaims, err error)
	}
)

type jwtRegistry struct {
	secret   string
	duration time.Duration
}

func NewJwtRegistry(secret string, duration time.Duration) JwtRegistry {
	return &jwtRegistry{
		secret:   secret,
		duration: duration,
	}
}

func (r *jwtRegistry) Generate(ctx context.Context, userId string) (token string, err error) {
	now := time.Now()

	claims := JwtClaims{
		Id: userId,

		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userId,
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

func (r *jwtRegistry) Verify(ctx context.Context, token string) (claims *JwtClaims, err error) {
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
