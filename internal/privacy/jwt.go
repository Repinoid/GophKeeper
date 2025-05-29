package privacy

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v4"
)

type Claims struct {
	jwt.RegisteredClaims
	UserID int
}

const TokenExp = time.Hour * 3
const SecretKey = "supersecretkey"

func BuildJWTString(id string, secret []byte) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(TokenExp)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    "ЧК",
			Subject:   "Чтоб ништяк",
			ID:        id,
		},
		UserID: 1,
	})
	tokenString, err := token.SignedString(secret)
	if err != nil {
		return "", err
	}
	return tokenString, nil
}

func CheckToken(tokenString string, secret []byte) (*Claims, error) {
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenString, claims,
		func(t *jwt.Token) (interface{}, error) {
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
			}
			return secret, nil
		})
	if err != nil {
		return nil, fmt.Errorf("bad jwt.ParseWithClaims, err %w", err)
	}
	if !token.Valid {
		return nil, fmt.Errorf("token is not valid, err %w", err)
	}
	return claims, nil
}
