package utils

import (
	"goibed/config"
	"os"
	"path/filepath"
	"strconv"
	"time"
)

// 获取上传图片存放的文件夹，这里采用 BASEPATH/data/uploads/YYYY/UUID(with -).avif 的形式存储
// 在 database 包的 init 下已保证 uploads 文件夹已创建
func GetUploadFolder() (string, error) {
	uploadDir := filepath.Join(config.Config.BasePath, "data", "uploads", strconv.Itoa(time.Now().Year()))
	if err := os.MkdirAll(uploadDir, 0755); err != nil {
		return "", err
	}
	return uploadDir, nil
}

// 将绝对路径转换成相对路径，统一转换成正斜杠存储
func ConvertToRelative(absPath string) (string, error) {
	cleanBase := filepath.Clean(config.Config.BasePath)
	cleanTarget := filepath.Clean(absPath)

	rel, err := filepath.Rel(cleanBase, cleanTarget)
	if err != nil {
		return "", err
	}

	return filepath.ToSlash(rel), nil
}

// 将相对路径转换成绝对路径以便访问文件
func ConvertToAbsolute(relPath string) string {
	return filepath.Join(config.Config.BasePath, relPath)
}
