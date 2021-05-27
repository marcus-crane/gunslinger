package database

import (
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

var (
	DBConn *gorm.DB
)

func Connect() (err error) {
	DBConn, err = gorm.Open(sqlite.Open("gunslinger.db"), &gorm.Config{})
	if err != nil {
		return err
	}

	sqlDB, err := DBConn.DB()
	sqlDB.SetConnMaxLifetime(time.Hour)

	return nil
}
