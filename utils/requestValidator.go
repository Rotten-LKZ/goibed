package utils

import (
	"goibed/config"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func IsPOST(c *Context) bool {
	if c.R.Method != http.MethodPost {
		c.Error(http.StatusMethodNotAllowed, "Method not allowed")
		return false
	}
	return true
}

func MakeAuthRequiredHandler(fn func(c *Context)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := &Context{W: w, R: r}
		cookie, err := r.Cookie("JWT_TOKEN")
		if err != nil {
			ctx.Error(http.StatusUnauthorized, "Cookie JWT_TOKEN required")
			return
		}

		parts := strings.SplitN(cookie.Value, " ", 2)
		if !(len(parts) == 2 && parts[0] == "Bearer") {
			ctx.Error(http.StatusUnauthorized, "Wrong Authorization format")
			return
		}
		tokenString := parts[1]

		token, err := jwt.ParseWithClaims(tokenString, &jwt.RegisteredClaims{}, func(token *jwt.Token) (any, error) {
			return config.Config.JwtKey, nil
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
