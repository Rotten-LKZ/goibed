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
	mux.HandleFunc("GET /{$}", helloHandler)
	mux.HandleFunc("GET /i/{filename}", utils.MakeHandler(handler.GetImage))
	mux.HandleFunc("GET /info/{filename}", utils.MakeHandler(handler.GetImageInfo))
	mux.HandleFunc("POST /api/login", utils.MakeHandler(handler.Login))
	mux.HandleFunc("POST /api/upload", utils.MakeAuthRequiredHandler(handler.UploadImage))
	mux.HandleFunc("POST /api/manage/list", utils.MakeAuthRequiredHandler(handler.GetImagesList))
	mux.HandleFunc("POST /api/manage/update", utils.MakeAuthRequiredHandler(handler.UpdateImage))
	mux.HandleFunc("POST /api/manage/delete", utils.MakeAuthRequiredHandler(handler.DelImage))
	mux.HandleFunc("POST /api/manage/reconvert", utils.MakeAuthRequiredHandler(handler.ReconvertImages))

	slog.Info("HTTP server starting", "addr", ":8080")

	if err := http.ListenAndServe(":8080", mux); err != nil {
		slog.Error("HTTP server stopped", "error", err)
	}
}
