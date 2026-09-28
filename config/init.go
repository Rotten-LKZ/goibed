package config

import (
	"log/slog"
	"os"
	"strconv"
)

type ConfigList struct {
	Token          string
	BasePath       string
	MaxImgPool     int
	MaxQueueLength int
	JwtKey         []byte
}

// Ensure not to write to this var after Init
// otherwise may cause data race
var Config *ConfigList

func Init() {
	token := os.Getenv("GOIBED_TOKEN")
	if token == "" {
		slog.Warn("LOGIN TOKEN HAVEN'T BE SET, REMEMBER TO SET LOGIN TOKEN IN THE PRODUCTION ENV OR YOUR DATA MAY BE STOLEN.")
	}
	jwtKey := os.Getenv("GOIBED_JWT_KEY")
	if jwtKey == "" {
		slog.Warn("JWT KEY HAVEN'T BE SET, REMEMBER TO SET JWT KEY IN THE PRODUCTION ENV OR YOUR DATA MAY BE STOLEN.")
	}
	basePath := os.Getenv("GOIBED_BASEPATH")
	if info, err := os.Stat(basePath); err != nil || !info.IsDir() {
		slog.Info("The specified directory does not exist, and the 'data' folder in the working directory will be used as the default data saving directory.")
		basePath, err = os.Getwd()
		if err != nil {
			slog.Error("failed to get the working directory as the default data saving directory")
			os.Exit(1)
		}
	}
	maxImgPoolStr := os.Getenv("GOIBED_MAX_IMGPOOL")
	maxImgPool := 4
	if maxImgPoolStr != "" {
		parseI, err := strconv.Atoi(maxImgPoolStr)
		if err != nil {
			slog.Error("WRONG FORMAT GOIBED_MAX_IMGPOOL, SWITCH TO DEFAULT VALUE 4.")
		} else {
			maxImgPool = parseI
		}
	}
	maxQueueLengthStr := os.Getenv("GOIBED_MAX_QUEUE_LENGTH")
	maxQueueLength := 4
	if maxImgPoolStr != "" {
		parseI, err := strconv.Atoi(maxQueueLengthStr)
		if err != nil {
			slog.Error("WRONG FORMAT GOIBED_MAX_QUEUE_LENGTH, SWITCH TO DEFAULT VALUE 500.")
		} else {
			maxQueueLength = parseI
		}
	}

	Config = &ConfigList{
		Token:          token,
		BasePath:       basePath,
		MaxImgPool:     maxImgPool,
		MaxQueueLength: maxQueueLength,
		JwtKey:         []byte(jwtKey),
	}
}
