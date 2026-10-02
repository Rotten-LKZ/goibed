package handler

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"goibed/config"
	"goibed/database"
	"goibed/imgpool"
	"goibed/utils"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
	"uuid"

	"gorm.io/gorm"
)

func UploadImage(c *utils.Context) {
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

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
	defer cancel()
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

	resourceURL, err := utils.GetResourceURL(c.R, fmt.Sprintf("/i/%s.avif", fileUUID))
	if err != nil {
		c.Error(http.StatusInternalServerError, "Failed to get img resource url")
		return
	}
	c.JSON(http.StatusOK, utils.Response{
		Code: http.StatusOK,
		Msg:  "Uploaded Successful",
		Data: map[string]string{
			"hash": fileHash,
			"url":  resourceURL,
		},
	})
}

func getUUIDFromFilename(filename string) (string, bool) {
	if len(filename) != 36+5 || !strings.HasSuffix(filename, ".avif") {
		return "", false
	}
	possibleUUID := filename[0:36]
	_, err := uuid.Parse(possibleUUID)
	if err != nil {
		return "", false
	}
	return possibleUUID, true
}

func GetImage(c *utils.Context) {
	imageName := c.R.PathValue("filename")
	ID, isValid := getUUIDFromFilename(imageName)
	if !isValid {
		c.Error(http.StatusNotFound, "No such image")
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
	defer cancel()
	image, err := gorm.G[database.Images](database.DB).Where("ID = ?", ID).First(ctx)
	if err != nil {
		c.Error(http.StatusNotFound, "No such image")
		return
	}
	http.ServeFile(c.W, c.R, filepath.Join(config.Config.BasePath, image.ImagePath))
}

func GetImageInfo(c *utils.Context) {
	imageName := c.R.PathValue("filename")
	ID, isValid := getUUIDFromFilename(imageName)
	if !isValid {
		c.Error(http.StatusNotFound, "No such image")
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
	defer cancel()
	image, err := gorm.G[database.Images](database.DB).Where("ID = ?", ID).First(ctx)
	if err != nil {
		c.Error(http.StatusNotFound, "No such image")
		return
	}
	c.JSON(http.StatusOK, &utils.Response{
		Code: http.StatusOK,
		Msg:  "Successful",
		Data: image,
	})
}
