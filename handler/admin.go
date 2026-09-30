package handler

import (
	"context"
	"encoding/json"
	"goibed/config"
	"goibed/database"
	"goibed/utils"
	"net/http"
	"os"
	"path/filepath"
	"time"
	"uuid"

	"gorm.io/gorm"
)

type GetImagesRequest struct {
	Page uint `json:"page"`
	Step uint `json:"step"`
}

type UpdateImageRequest struct {
	ID       string `json:"ID"`
	Filename string `json:"file_name"`
}

type DelImageRequest struct {
	ID string `json:"ID"`
}

func GetImagesList(c *utils.Context) {
	defer c.R.Body.Close()
	var t GetImagesRequest
	err := json.NewDecoder(c.R.Body).Decode(&t)
	if err != nil {
		c.Error(http.StatusBadRequest, "Wrong argument")
		return
	}

	if t.Page == 0 {
		t.Page = 1
	}
	if t.Step == 0 {
		t.Step = 20
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	images, err := gorm.G[database.Images](database.DB).
		Offset(int((t.Page - 1) * t.Step)).
		Limit(int(t.Step)).
		Find(ctx)
	if err != nil {
		c.Error(http.StatusInternalServerError, "Failed to get images list")
		return
	}
	c.JSON(http.StatusOK, &utils.Response{
		Code: http.StatusOK,
		Msg:  "Successful",
		Data: images,
	})
}

func UpdateImage(c *utils.Context) {
	defer c.R.Body.Close()
	var t UpdateImageRequest
	err := json.NewDecoder(c.R.Body).Decode(&t)
	if err != nil {
		c.Error(http.StatusBadRequest, "Wrong argument")
		return
	}

	if _, err := uuid.Parse(t.ID); err != nil || t.Filename == "" {
		c.Error(http.StatusBadRequest, "Wrong argument")
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	rowsAffected, err := gorm.G[database.Images](database.DB).Where("ID = ?", t.ID).Update(ctx, "filename", t.Filename)
	if err != nil {
		c.Error(http.StatusInternalServerError, "Failed to update img info")
		return
	}
	if rowsAffected == 0 {
		c.Error(http.StatusBadRequest, "Wrong ID")
		return
	}
	c.JSON(http.StatusOK, &utils.Response{
		Code: http.StatusOK,
		Msg:  "Successful",
	})
}

func DelImage(c *utils.Context) {
	defer c.R.Body.Close()
	var t DelImageRequest
	err := json.NewDecoder(c.R.Body).Decode(&t)
	if err != nil {
		c.Error(http.StatusBadRequest, "Wrong argument")
		return
	}

	if _, err := uuid.Parse(t.ID); err != nil {
		c.Error(http.StatusBadRequest, "Wrong argument")
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	image, err := gorm.G[database.Images](database.DB).Where("ID = ?", t.ID).First(ctx)
	if err != nil {
		c.Error(http.StatusBadRequest, "Wrong ID")
		return
	}
	rowsAffected, err := gorm.G[database.Images](database.DB).Where("ID = ?", t.ID).Delete(ctx)

	if rowsAffected == 0 || err != nil {
		c.Error(http.StatusInternalServerError, "Failed to delete img")
		return
	}
	if err := os.Remove(filepath.Join(config.Config.BasePath, image.ImagePath)); err != nil {
		c.Error(http.StatusInternalServerError, "Failed to delete img file")
		return
	}

	c.JSON(http.StatusOK, &utils.Response{
		Code: http.StatusOK,
		Msg:  "Successful",
	})
}
