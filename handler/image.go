package handler

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"goibed/database"
	"goibed/imgpool"
	"goibed/utils"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"
	"uuid"

	"gorm.io/gorm"
)

func UploadImage(c *utils.Context) {
	if !utils.IsPOST(c) {
		return
	}

	if err := c.R.ParseMultipartForm(32 << 20); err != nil {
		c.Error(http.StatusBadRequest, "Parse form failed or too large file")
		return
	}

	file, header, err := c.R.FormFile("file")
	if err != nil {
		c.Error(http.StatusBadRequest, "Cannot found uploaded file")
		return
	}
	defer file.Close()

	uploadDir, err := utils.GetUploadFolder()
	if err != nil {
		slog.Error(err.Error())
		c.Error(http.StatusInternalServerError, "Failed to create folder to save file")
		return
	}

	fileUUID := uuid.New().String()
	ext := filepath.Ext(header.Filename)

	tempFileName := fileUUID + ext
	tempPath := filepath.Join(uploadDir, tempFileName)

	dst, err := os.Create(tempPath)
	if err != nil {
		slog.Error(err.Error())
		c.Error(http.StatusInternalServerError, "Failed to save file")
		return
	}
	defer dst.Close()

	hash := sha256.New()
	writer := io.MultiWriter(dst, hash)
	if _, err := io.Copy(writer, file); err != nil {
		os.Remove(tempPath)
		slog.Error(err.Error())
		c.Error(http.StatusInternalServerError, "Failed to save file")
		return
	}
	fileHash := hex.EncodeToString(hash.Sum(nil))

	tempPathRelative, err := utils.ConvertToRelative(tempPath)
	if err != nil {
		c.Error(http.StatusInternalServerError, "Failed to convert to relative path")
		return
	}

	ctx := context.Background()
	err = gorm.G[database.Images](database.DB).Create(ctx, &database.Images{
		ID:        fileUUID,
		FileHash:  fileHash,
		Filename:  header.Filename,
		ImagePath: tempPathRelative,
	})
	if err != nil {
		os.Remove(tempPath)
		slog.Error(err.Error())
		c.Error(http.StatusInternalServerError, "Failed to write to db")
		return
	}
	if succ := imgpool.Submit(imgpool.Task{
		ID:       fileUUID,
		TempPath: tempPath,
	}); !succ {
		c.Error(http.StatusInternalServerError, "Failed to queue converting request.")
		return
	}

	c.JSON(http.StatusOK, utils.Response{
		Code: http.StatusOK,
		Msg:  "Uploaded Successful",
		Data: map[string]string{
			"hash": fileHash,
			"url":  "/uploads/" + strconv.Itoa(time.Now().Year()) + "/" + fileUUID + ".avif",
		},
	})
}

func GetImage(w http.ResponseWriter, r *http.Request) {

}
