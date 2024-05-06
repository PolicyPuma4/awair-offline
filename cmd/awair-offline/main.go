package main

import (
	"awair-offline/internal/reader"
	"database/sql"
	"encoding/json"
	"log"
	"os"
	"strconv"
	"time"

	_ "modernc.org/sqlite"
)

func main() {
	log.SetFlags(log.Flags() | log.Llongfile)

	db, err := sql.Open("sqlite", "./data/database.db")
	if err != nil {
		log.Fatal(err)
	}

	defer func() {
		if err := db.Close(); err != nil {
			log.Fatal(err)
		}
	}()

	// modernc.org/sqlite does not support concurrent writes
	db.SetMaxOpenConns(1)
	if _, err := db.Exec(
		`CREATE TABLE IF NOT EXISTS data(
			name TEXT NOT NULL,
			timestamp TEXT NOT NULL,
			score INTEGER NOT NULL,
			dew_point REAL NOT NULL,
			temp REAL NOT NULL,
			humid REAL NOT NULL,
			abs_humid REAL NOT NULL,
			co2 INTEGER NOT NULL,
			co2_est INTEGER NOT NULL,
			co2_est_baseline INTEGER NOT NULL,
			voc INTEGER NOT NULL,
			voc_baseline INTEGER NOT NULL,
			voc_h2_raw INTEGER NOT NULL,
			voc_ethanol_raw INTEGER NOT NULL,
			pm25 INTEGER NOT NULL,
			pm10_est INTEGER NOT NULL,
			UNIQUE(name, timestamp)
		)`,
	); err != nil {
		log.Fatal(err)
	}

	interval, err := strconv.Atoi(os.Getenv("DURATION"))
	if err != nil {
		log.Fatal(err)
	}

	monitors := []reader.Monitor{}
	if err := json.Unmarshal([]byte(os.Getenv("MONITORS")), &monitors); err != nil {
		log.Fatal(err)
	}

	(&reader.Reader{
		Interval: time.Duration(interval) * time.Second,
		Monitors: monitors,
		DB:       db,
	}).Read()
}
