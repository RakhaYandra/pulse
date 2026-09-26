package security

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type JWTService struct {
	Secret string
}

func (j JWTService) Issue(userID string) (string, error) {
	t := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub": userID,
		"exp": time.Now().Add(24 * time.Hour).Unix(),
		"iat": time.Now().Unix(),
	})
	return t.SignedString([]byte(j.Secret))
}

func (j JWTService) Parse(token string) (string, error) {
	tok, err := jwt.Parse(token, func(t *jwt.Token) (any, error) {
		if t.Method.Alg() != jwt.SigningMethodHS256.Alg() {
			return nil, errors.New("bad alg")
		}
		return []byte(j.Secret), nil
	})
	if err != nil || !tok.Valid {
		return "", errors.New("invalid token")
	}
	sub, _ := tok.Claims.(jwt.MapClaims)["sub"].(string)
	if sub == "" {
		return "", errors.New("invalid claims")
	}
	return sub, nil
}
