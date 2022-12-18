package db

import (
	"log"

	"github.com/marcus-crane/gunslinger/models"
	"github.com/marcus-crane/gunslinger/utils"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func Initialize() *gorm.DB {
	db, err := gorm.Open(sqlite.Open(utils.MustEnv("DB_PATH")), &gorm.Config{})
	if err != nil {
		panic(err)
	}
	db.AutoMigrate(&models.DBMediaItem{})
	log.Print("Initialised DB connection")
	return db
}
