package handler

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"goibed/config"
	"goibed/database"
	"goibed/imgpool"
	"goibed/utils"
	"io"
	"log/slog"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
	"uuid"

	"gorm.io/gorm"
)

const maxUploadSize = 32 << 20

func UploadImage(c *utils.Context) {
	slog.Debug("upload parsing started")
	reader, err := c.R.MultipartReader()
	if err != nil {
		c.Error(http.StatusBadRequest, "Parse form failed")
		return
	}
	var file *multipart.Part
	for {
		part, err := reader.NextPart()
		if err != nil {
			c.Error(http.StatusBadRequest, "Cannot found uploaded file")
			return
		}
		if part.FormName() == "file" && part.FileName() != "" {
			file = part
			break
		}
		part.Close()
	}
	defer file.Close()
	filename := file.FileName()
	slog.Debug("upload received", "filename", filename)

	uploadDir, err := utils.GetUploadFolder()
	if err != nil {
		slog.Error(err.Error())
		c.Error(http.StatusInternalServerError, "Failed to create folder to save file")
		return
	}

	fileUUID := uuid.New().String()
	ext := filepath.Ext(filename)

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
	size, err := io.CopyN(writer, file, maxUploadSize+1)
	if size > maxUploadSize {
		os.Remove(tempPath)
		c.Error(http.StatusRequestEntityTooLarge, "File exceeds 32 MiB limit")
		return
	}
	if err != nil && err != io.EOF {
		os.Remove(tempPath)
		slog.Error(err.Error())
		c.Error(http.StatusInternalServerError, "Failed to save file")
		return
	}
	fileHash := hex.EncodeToString(hash.Sum(nil))
	slog.Debug("upload saved and hashed", "ID", fileUUID, "bytes", size, "hash", fileHash)

	tempPathRelative, err := utils.ConvertToRelative(tempPath)
	if err != nil {
		c.Error(http.StatusInternalServerError, "Failed to convert to relative path")
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
	defer cancel()
	var image database.Images
	image, err = gorm.G[database.Images](database.DB).Where("file_hash = ?", fileHash).First(ctx)

	slog.Debug("upload duplicate lookup completed", "ID", fileUUID, "error", err)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		ctx, cancel = context.WithTimeout(context.Background(), time.Second*3)
		defer cancel()
		image = database.Images{
			ID:        fileUUID,
			FileHash:  fileHash,
			Filename:  filename,
			ImagePath: tempPathRelative,
		}
		err = gorm.G[database.Images](database.DB).Create(ctx, &image)
		if err != nil {
			os.Remove(tempPath)
			slog.Error(err.Error())
			c.Error(http.StatusInternalServerError, "Failed to write to db")
			return
		}
		slog.Debug("upload recorded", "ID", fileUUID)
		if succ := imgpool.Submit(imgpool.Task{
			ID:       fileUUID,
			TempPath: tempPath,
		}); !succ {
			c.Error(http.StatusInternalServerError, "Failed to queue converting request.")
			return
		}
		slog.Debug("upload conversion queued", "ID", fileUUID)
	} else if err != nil {
		c.Error(http.StatusInternalServerError, "Failed to convert to relative path")
		return
	} else {
		// exists
		err := os.Remove(tempPath)
		if err != nil {
			c.Error(http.StatusInternalServerError, "Failed to remove temp img file")
			return
		}
		slog.Debug("upload deduplicated", "ID", image.ID)
	}

	slog.Debug("upload response ready", "ID", image.ID)
	c.JSON(http.StatusOK, utils.Response{
		Code: http.StatusOK,
		Msg:  "Successful",
		Data: image,
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
	slog.Debug("image requested", "filename", imageName)
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
	file, err := os.Open(filepath.Join(config.Config.BasePath, image.ImagePath))
	if err != nil {
		c.Error(http.StatusNotFound, "No such image")
		return
	}
	defer file.Close()

	modified := time.Time(image.UpdatedAt)
	c.W.Header().Set("ETag", `"`+image.ID+"-"+strconv.FormatInt(modified.UnixNano(), 10)+`"`)
	if strings.HasSuffix(image.ImagePath, ".avif") {
		c.W.Header().Set("Cache-Control", "public, max-age=31536000")
	} else {
		c.W.Header().Set("Cache-Control", "no-cache")
	}
	http.ServeContent(c.W, c.R, imageName, modified, file)
	slog.Debug("image served", "ID", ID, "path", image.ImagePath)
}

func GetImageInfo(c *utils.Context) {
	c.W.Header().Set("Access-Control-Allow-Origin", "*")
	imageName := c.R.PathValue("filename")
	ID, isValid := getUUIDFromFilename(imageName)
	slog.Debug("image info requested", "filename", imageName)
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
	slog.Debug("image info found", "ID", ID)
	c.JSON(http.StatusOK, &utils.Response{
		Code: http.StatusOK,
		Msg:  "Successful",
		Data: image,
	})
}
