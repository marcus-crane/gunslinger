package db

import (
	"log"
	"os"

	"github.com/marcus-crane/gunslinger/models"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func Initialize() *gorm.DB {
	db, err := gorm.Open(sqlite.Open(os.Getenv("DB_PATH")), &gorm.Config{})
	if err != nil {
		panic(err)
	}
	db.AutoMigrate(&models.DBMediaItem{})
	log.Print("Initialised DB connection")
	return db
}
