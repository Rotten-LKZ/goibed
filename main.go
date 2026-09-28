package main

import (
	"fmt"
	"log/slog"
	"net/http"

	"goibed/handler"
	"goibed/imgpool"
	"goibed/logger"
	"goibed/utils"
)

func helloHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Hello")
}

func main() {
	logger.Init()

	http.HandleFunc("/", helloHandler)
	http.HandleFunc("/api/login", utils.MakeHandler(handler.Login))
	http.HandleFunc("/api/upload", utils.MakeAuthRequiredHandler(handler.UploadImage))

	slog.Info("HTTP server starting", "addr", ":8080")

	imgpool.InitPool()

	if err := http.ListenAndServe(":8080", nil); err != nil {
		slog.Error("HTTP server stopped", "error", err)
	}
}
