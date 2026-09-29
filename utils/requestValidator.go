package utils

import (
	"goibed/config"
	"net/http"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func MakeAuthRequiredHandler(fn func(c *Context)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := &Context{W: w, R: r}
		tokenString, err := r.Cookie("JWT_TOKEN")
		if err != nil {
			ctx.Error(http.StatusUnauthorized, "Cookie JWT_TOKEN required")
			return
		}

		token, err := jwt.ParseWithClaims(tokenString.Value, &jwt.RegisteredClaims{}, func(token *jwt.Token) (any, error) {
			return []byte(config.Config.JwtKey), nil
		})
		if err != nil || !token.Valid {
			http.SetCookie(w, &http.Cookie{
				Name:     "JWT_TOKEN",
				Value:    "",
				Path:     "/",
				Domain:   "",
				MaxAge:   -1,
				Expires:  time.Unix(0, 0),
				HttpOnly: true,
				Secure:   false,
			})
			ctx.Error(http.StatusUnauthorized, "Invalid JWT")
			return
		}
		fn(ctx)
	}
}
