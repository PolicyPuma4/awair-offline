package reader

import (
	"database/sql"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"time"
)

type Monitor struct {
	Name    string `json:"name"`
	Address string `json:"address"`
}

type Reader struct {
	Interval time.Duration
	Monitors []Monitor
	DB       *sql.DB
}

var client = &http.Client{}

func (r *Reader) read(monitor Monitor) error {
	req, err := http.NewRequest("GET", monitor.Address+"/air-data/latest", nil)
	if err != nil {
		return err
	}

	resp, err := client.Do(req)
	if err != nil {
		return err
	}

	defer func() {
		if err := resp.Body.Close(); err != nil {
			log.Println(err)
		}
	}()

	if resp.StatusCode != 200 {
		return errors.New(resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	data := struct {
		Timestamp      time.Time `json:"timestamp"`
		Score          int       `json:"score"`
		DewPoint       float64   `json:"dew_point"`
		Temp           float64   `json:"temp"`
		Humid          float64   `json:"humid"`
		AbsHumid       float64   `json:"abs_humid"`
		Co2            int       `json:"co2"`
		Co2Est         int       `json:"co2_est"`
		Co2EstBaseline int       `json:"co2_est_baseline"`
		Voc            int       `json:"voc"`
		VocBaseline    int       `json:"voc_baseline"`
		VocH2Raw       int       `json:"voc_h2_raw"`
		VocEthanolRaw  int       `json:"voc_ethanol_raw"`
		Pm25           int       `json:"pm25"`
		Pm10Est        int       `json:"pm10_est"`
	}{}

	if err := json.Unmarshal(body, &data); err != nil {
		return err
	}

	_, err = r.DB.Exec(
		`INSERT OR IGNORE INTO data(
			name,
			timestamp,
			score,
			dew_point,
			temp,
			humid,
			abs_humid,
			co2,
			co2_est,
			co2_est_baseline,
			voc,
			voc_baseline,
			voc_h2_raw,
			voc_ethanol_raw,
			pm25,
			pm10_est
		) VALUES($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16)`,
		monitor.Name,
		data.Timestamp,
		data.Score,
		data.DewPoint,
		data.Temp,
		data.Humid,
		data.AbsHumid,
		data.Co2,
		data.Co2Est,
		data.Co2EstBaseline,
		data.Voc,
		data.VocBaseline,
		data.VocH2Raw,
		data.VocEthanolRaw,
		data.Pm25,
		data.Pm10Est,
	)

	return err
}

func (r *Reader) Read() {
	ticker := time.NewTicker(r.Interval)
	defer ticker.Stop()
	for ; true; <-ticker.C {
		for _, monitor := range r.Monitors {
			go func() {
				if err := r.read(monitor); err != nil {
					log.Println(err)
				}
			}()
		}
	}
}
