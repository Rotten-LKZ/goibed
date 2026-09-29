package config

import (
	"encoding/json"
	"flag"
	"log/slog"
	"os"
)

type ConfigList struct {
	Token          string `json:"token"`
	BasePath       string `json:"base_path"`
	MagickPath     string `json:"magick_path"`
	MaxImgPool     int    `json:"max_img_pool"`
	MaxQueueLength int    `json:"max_queue_length"`
	JwtKey         string `json:"jwt_key"`
}

// Ensure not to write to this var after Init
// otherwise may cause data race
var Config *ConfigList

func Init() {
	configPath := flag.String("config", "./config.json", "配置文件路径")
	flag.Parse()

	data, err := os.ReadFile(*configPath)
	if err != nil {
		slog.Error("Failed to read config file")
		os.Exit(1)
	}
	if err := json.Unmarshal(data, Config); err != nil {
		slog.Error("Failed to parse config file")
		os.Exit(1)
	}

	if Config.Token == "" {
		slog.Warn("LOGIN TOKEN HAVEN'T BE SET, REMEMBER TO SET LOGIN TOKEN IN THE PRODUCTION ENV OR YOUR DATA MAY BE STOLEN.")
	}
	if Config.JwtKey == "" {
		slog.Warn("JWT KEY HAVEN'T BE SET, REMEMBER TO SET JWT KEY IN THE PRODUCTION ENV OR YOUR DATA MAY BE STOLEN.")
	}
	if info, err := os.Stat(Config.BasePath); err != nil || !info.IsDir() {
		slog.Info("The specified directory does not exist, and the 'data' folder in the working directory will be used as the default data saving directory.")
		Config.BasePath, err = os.Getwd()
		if err != nil {
			slog.Error("failed to get the working directory as the default data saving directory")
			os.Exit(1)
		}
	}
	if Config.MaxImgPool <= 0 {
		Config.MaxImgPool = 4
		slog.Warn("You don't set MaxImgPool or set it to non-positive value, change it to default value 4")
	}
	if Config.MaxQueueLength <= 0 {
		Config.MaxQueueLength = 500
		slog.Warn("You don't set MaxQueueLength or set it to non-positive value, change it to default value 500")
	}
	if Config.MagickPath == "" {
		Config.MagickPath = "magick"
		slog.Warn("You don't set MagickPath, program will run magick directly.")
	}
}
