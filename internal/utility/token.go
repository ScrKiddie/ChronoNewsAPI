package utility

import (
	"ChronoverseAPI/internal/model"
	"fmt"
	"github.com/golang-jwt/jwt/v5"
	"time"
)

func CreateJWT(secret string, exp int, id int32) (string, error) {
	claims := jwt.MapClaims{
		"sub": id,
		"exp": time.Now().Add(time.Duration(exp) * time.Hour).Unix(),
	}

	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(secret))
	if err != nil {
		return "", err
	}
	return token, nil
}

func ValidateJWT(secret string, token string) (*model.UserAuthorization, error) {
	t, err := jwt.Parse(token, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(secret), nil
	})

	if err != nil {
		return nil, err
	}

	if !t.Valid {
		return nil, fmt.Errorf("invalid token")
	}

	claims, ok := t.Claims.(jwt.MapClaims)
	if !ok {
		return nil, fmt.Errorf("failed to parse claims")
	}

	sub, ok := claims["sub"].(float64)
	if !ok {
		return nil, fmt.Errorf("invalid sub claim type")
	}

	return &model.UserAuthorization{ID: int32(sub)}, nil
}
