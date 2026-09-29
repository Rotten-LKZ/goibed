package main

import (
	"fmt"
	"log/slog"
	"net/http"

	"goibed/config"
	"goibed/database"
	"goibed/handler"
	"goibed/imgpool"
	"goibed/logger"
	"goibed/utils"
)

func helloHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Hello")
}

func main() {
	config.Init()
	database.Init()
	logger.Init()
	imgpool.InitPool()

	mux := http.NewServeMux()
	mux.HandleFunc("GET /", helloHandler)
	mux.HandleFunc("GET /i/{filename}", utils.MakeHandler(handler.GetImage))
	mux.HandleFunc("GET /info/{filename}", utils.MakeHandler(handler.GetImageInfo))
	mux.HandleFunc("POST /api/login", utils.MakeHandler(handler.Login))
	mux.HandleFunc("POST /api/upload", utils.MakeAuthRequiredHandler(handler.UploadImage))

	slog.Info("HTTP server starting", "addr", ":8080")

	if err := http.ListenAndServe(":8080", mux); err != nil {
		slog.Error("HTTP server stopped", "error", err)
	}
}
