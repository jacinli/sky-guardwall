package middleware

import (
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/jacinli/sky-guardwall/internal/response"
)

func JWTAuth(secret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		raw := c.GetHeader("Authorization")
		tokenStr := strings.TrimPrefix(raw, "Bearer ")
		if tokenStr == "" {
			response.Error(c, 401, "missing authorization token")
			c.Abort()
			return
		}

		token, err := jwt.ParseWithClaims(
			tokenStr,
			&jwt.RegisteredClaims{},
			func(t *jwt.Token) (interface{}, error) {
				if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
					return nil, jwt.ErrSignatureInvalid
				}
				return []byte(secret), nil
			},
		)
		if err != nil || !token.Valid {
			response.Error(c, 401, "invalid or expired token")
			c.Abort()
			return
		}

		claims := token.Claims.(*jwt.RegisteredClaims)
		c.Set("username", claims.Subject)
		c.Next()
	}
}

func GenerateToken(username, secret string, expireHours int) (string, error) {
	claims := jwt.RegisteredClaims{
		Subject:   username,
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Duration(expireHours) * time.Hour)),
		IssuedAt:  jwt.NewNumericDate(time.Now()),
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(secret))
}
