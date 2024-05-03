package reader

import (
	"database/sql"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"sync"
	"time"
)

type Monitor struct {
	Name    string `json:"name"`
	Address string `json:"address"`
}

type reader struct {
	duration time.Duration
	mu       sync.RWMutex
	monitors []Monitor
	client   *http.Client
	db       *sql.DB
}

func NewReader(duration time.Duration, db *sql.DB) *reader {
	return &reader{
		duration: duration,
		client:   &http.Client{},
		db:       db,
	}
}

func (r *reader) AddMonitor(monitor Monitor) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.monitors = append(r.monitors, monitor)
}

func (r *reader) read(monitor Monitor) error {
	req, err := http.NewRequest("GET", monitor.Address+"/air-data/latest", nil)
	if err != nil {
		return err
	}

	resp, err := r.client.Do(req)
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

	err = func() error {
		r.mu.Lock()
		defer r.mu.Unlock()
		_, err := r.db.Exec(
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
	}()

	return err
}

func (r *reader) Listen() {
	ticker := time.NewTicker(r.duration)
	defer ticker.Stop()
	for ; true; <-ticker.C {
		for _, monitor := range func() []Monitor {
			r.mu.RLock()
			defer r.mu.RUnlock()
			return r.monitors
		}() {
			go func() {
				if err := r.read(monitor); err != nil {
					log.Println(err)
				}
			}()
		}
	}
}
