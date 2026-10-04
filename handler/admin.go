package handler

import (
	"context"
	"encoding/json"
	"goibed/config"
	"goibed/database"
	"goibed/imgpool"
	"goibed/utils"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"time"
	"uuid"

	"gorm.io/gorm"
)

type GetImagesRequest struct {
	Page      uint   `json:"page"`
	Step      uint   `json:"step"`
	Keyword   string `json:"keyword"`
	StartTime *int64 `json:"start_time"`
	EndTime   *int64 `json:"end_time"`
}

type UpdateImageRequest struct {
	ID       string `json:"ID"`
	Filename string `json:"file_name"`
}

type DelImageRequest struct {
	ID string `json:"ID"`
}

type ReconvertImagesRequest struct {
	// this argument will be ignored if ID is not empty;
	// if true convert all image,
	// otherwise only convert images whose image_path doesn't end with .avif
	All bool   `json:"all"`
	ID  string `json:"ID"`
}

func GetImagesList(c *utils.Context) {
	defer c.R.Body.Close()
	var t GetImagesRequest
	err := json.NewDecoder(c.R.Body).Decode(&t)
	if err != nil {
		c.Error(http.StatusBadRequest, "Wrong argument")
		return
	}
	if t.StartTime != nil && t.EndTime != nil && *t.EndTime < *t.StartTime {
		c.Error(http.StatusBadRequest, "end_time must be greater than or equal to start_time")
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
	var query gorm.ChainInterface[database.Images] = gorm.G[database.Images](database.DB).Order("created_at DESC")
	if t.Keyword != "" {
		query = query.Where("file_name LIKE ?", "%"+t.Keyword+"%")
	}
	if t.StartTime != nil {
		query = query.Where("created_at >= ?", time.UnixMilli(*t.StartTime).UTC())
	}
	if t.EndTime != nil {
		query = query.Where("created_at <= ?", time.UnixMilli(*t.EndTime).UTC())
	}
	images, err := query.
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

func ReconvertImages(c *utils.Context) {
	defer c.R.Body.Close()
	var t ReconvertImagesRequest
	err := json.NewDecoder(c.R.Body).Decode(&t)
	if err != nil {
		c.Error(http.StatusBadRequest, "Wrong argument")
		return
	}

	if t.ID == "" {
		go func(all bool) {
			ctx := context.Background()
			handleImages := func(data []database.Images, _ int) error {
				imgpool.SubmitImagesBatchAndWait(data)
				return nil
			}

			var err error
			if all {
				err = gorm.G[database.Images](database.DB).FindInBatches(ctx, 200, handleImages)
			} else {
				err = gorm.G[database.Images](database.DB).Where("image_path NOT LIKE ?", "%.avif").FindInBatches(ctx, 200, handleImages)
			}
			if err != nil {
				slog.Error("failed to reconvert images", "err", err)
			}
		}(t.All)
	} else {
		if _, err := uuid.Parse(t.ID); err != nil {
			c.Error(http.StatusBadRequest, "Wrong argument")
			return
		}
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		image, err := gorm.G[database.Images](database.DB).Where("ID = ?", t.ID).First(ctx)
		if err != nil {
			c.Error(http.StatusBadRequest, "Wrong ID")
			return
		}
		if succ := imgpool.Submit(imgpool.Task{
			ID:       image.ID,
			TempPath: utils.ConvertToAbsolute(image.ImagePath),
		}); !succ {
			c.Error(http.StatusInternalServerError, "Failed to queue converting request.")
			return
		}
	}

	c.JSON(http.StatusOK, &utils.Response{
		Code: http.StatusOK,
		Msg:  "Successful",
	})
}
