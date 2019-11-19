package main

import (
	"fmt"
	"log"
	"os"

	_ "github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
	"github.com/joho/godotenv"
)

func init() {
	err := godotenv.Load()
	if err != nil {
		panic(err)
	}
}

func initDB() *sqlx.DB {
	db, err := sqlx.Open("mysql", fmt.Sprintf(`%s:%s@tcp(%s:%s)/%s`,
		os.Getenv("DB_USERNAME"),
		os.Getenv("DB_PASSWORD"),
		os.Getenv("DB_HOST"),
		os.Getenv("DB_PORT"),
		os.Getenv("DB_DATABASE"),
	))
	if err != nil {
		panic(err)
	}

	return db
}

func main() {
	db := initDB()
	s := DatabaseFiller(*db)
	go s.FillDatabaseFromZip(os.Getenv("DATA_ARCHIVE"))

	f, err := os.OpenFile(os.Getenv("APP_LOG"), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0775)
	if err != nil {
		panic(err)
	}
	log.SetOutput(f)
	defer f.Close()

	a := api{storage: db, validator: GetValidator()}
	a.Run()
}
