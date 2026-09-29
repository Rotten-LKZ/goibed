package database

import "time"

type Images struct {
	ID       string `gorm:"type:varchar(36);not null;primaryKey" json:"ID"`
	Type     string `gorm:"type:varchar(20);not null;default:'image'" json:"type"`
	FileHash string `gorm:"type:varchar(64);uniqueIndex" json:"file_hash"`
	Filename string `gorm:"type:varchar(255);not null" json:"file_name"`
	// 路径均存储相对路径
	ImagePath string  `gorm:"type:varchar(500);not null" json:"-"`
	VideoPath *string `gorm:"type:varchar(500);default:null" json:"-"`
	Width     uint32  `gorm:"default:0" json:"width"`
	Height    uint32  `gorm:"default:0" json:"height"`
	// Byte
	Size      uint64    `gorm:"default:0" json:"size"`
	CreatedAt time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt time.Time `gorm:"autoUpdateTime" json:"updated_at"`
}
