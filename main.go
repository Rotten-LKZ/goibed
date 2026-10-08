package main

import (
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"slices"
	"time"

	"goibed/config"
	"goibed/database"
	"goibed/handler"
	"goibed/imgpool"
	"goibed/logger"
	"goibed/utils"
)

var version = "dev"

func helloHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Hello")
}

func main() {
	if slices.Contains(os.Args[1:], "--version") {
		fmt.Println(version)
		return
	}

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

	if err := http.ListenAndServe(":8080", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		slog.Debug("request received", "method", r.Method, "path", r.URL.Path, "remote", r.RemoteAddr)
		mux.ServeHTTP(w, r)
		slog.Debug("request completed", "method", r.Method, "path", r.URL.Path, "duration", time.Since(start))
	})); err != nil {
		slog.Error("HTTP server stopped", "error", err)
	}
}
