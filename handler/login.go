package handler

import (
	"encoding/json"
	"goibed/config"
	"goibed/utils"
	"log/slog"
	"net/http"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type LoginRequest struct {
	Token string `json:"token"`
}

func Login(c *utils.Context) {
	defer c.R.Body.Close()
	var t LoginRequest
	err := json.NewDecoder(c.R.Body).Decode(&t)
	if err != nil {
		slog.Debug("login rejected: invalid request", "error", err)
		c.Error(http.StatusBadRequest, "Wrong argument")
		return
	}
	if t.Token != config.Config.Token {
		slog.Debug("login rejected: wrong token")
		c.JSON(http.StatusOK, utils.Response{
			Code: http.StatusOK,
			Msg:  "Wrong token",
		})
		return
	}
	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.RegisteredClaims{
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour * 24 * 30)),
		IssuedAt:  jwt.NewNumericDate(time.Now()),
	}).SignedString([]byte(config.Config.JwtKey))
	if err != nil {
		slog.Error("login signing failed", "error", err)
		c.Error(http.StatusInternalServerError, "Failed to sign JWT token")
		return
	}
	cookie := &http.Cookie{
		Name:     "JWT_TOKEN",
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		Secure:   true,
		MaxAge:   60 * 60 * 24 * 30, // 30 days
	}
	slog.Debug("login succeeded")
	http.SetCookie(c.W, cookie)
	c.JSON(http.StatusOK, utils.Response{
		Code: http.StatusOK,
		Msg:  "Successful",
	})
}
