package database

import (
  "time"

  "gorm.io/gorm"
  "gorm.io/driver/sqlite"
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
