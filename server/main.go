package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"io"
	"net/http"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

const (
	webServerPort                            = ":8080"
	exchangeRateApiUrl                       = "https://economia.awesomeapi.com.br/json/last/USD-BRL"
	exchangeRateApiRequestTimeoutDuration    = 200 * time.Millisecond
	exchangeRateDBFilePath                   = "./server/db/exchanges.db"
	exchangeRateDBTransactionTimeoutDuration = 10 * time.Millisecond
)

type ExchangeRateApiResponse struct {
	USDBRL struct {
		Code       string `json:"code"`
		Codein     string `json:"codein"`
		Name       string `json:"name"`
		High       string `json:"high"`
		Low        string `json:"low"`
		VarBid     string `json:"varBid"`
		PctChange  string `json:"pctChange"`
		Bid        string `json:"bid"`
		Ask        string `json:"ask"`
		Timestamp  string `json:"timestamp"`
		CreateDate string `json:"create_date"`
	} `json:"USDBRL"`
}

func main() {
	db, err := DatabaseFactory()
	if err != nil {
		panic(err)
	}
	defer db.Close()

	http.HandleFunc("/cotacao", ExchangeHandler(db))
	http.ListenAndServe(webServerPort, nil)
}

// ExchangeHandler handles the request and returns the exchange rate
func ExchangeHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		exchangeRate, err := GetExchangeRate(ctx)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		err = SaveExchangeRateInDatabase(ctx, db, exchangeRate)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		err = json.NewEncoder(w).Encode(exchangeRate.USDBRL.Bid)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
}

// GetExchangeRate gets the exchange rate from the external API
func GetExchangeRate(ctx context.Context) (*ExchangeRateApiResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, exchangeRateApiRequestTimeoutDuration)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", exchangeRateApiUrl, nil)
	if err != nil {
		return nil, err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	var e ExchangeRateApiResponse
	if err = json.Unmarshal(body, &e); err != nil {
		return nil, err
	}

	return &e, nil
}

// DatabaseFactory creates the database and the table if they don't exist yet and returns the database connection
func DatabaseFactory() (*sql.DB, error) {
	db, err := sql.Open("sqlite3", exchangeRateDBFilePath)
	if err != nil {
		return nil, err
	}
	const createDatabase = `CREATE TABLE IF NOT EXISTS exchanges (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		code VARCHAR(3) NOT NULL,
		codein VARCHAR(3) NOT NULL,
		name VARCHAR(100) NOT NULL,
		high VARCHAR(10) NOT NULL,
		low VARCHAR(10) NOT NULL,
		varBid VARCHAR(10) NOT NULL,
		pctChange VARCHAR(10) NOT NULL,
		bid VARCHAR(10) NOT NULL,
		ask VARCHAR(10) NOT NULL,
		create_date TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
	);`
	if _, err := db.Exec(createDatabase); err != nil {
		return nil, err
	}
	return db, nil
}

// SaveExchangeRateInDatabase saves the exchange rate in the database
func SaveExchangeRateInDatabase(ctx context.Context, db *sql.DB, exchangeRate *ExchangeRateApiResponse) error {
	ctx, cancel := context.WithTimeout(ctx, exchangeRateDBTransactionTimeoutDuration)
	defer cancel()
	stmt, err := db.PrepareContext(ctx, "INSERT INTO exchanges(code, codein, name, high, low, varBid, pctChange, bid, ask) values(?,?,?,?,?,?,?,?,?)")
	if err != nil {
		return err
	}
	_, err = stmt.ExecContext(
		ctx,
		exchangeRate.USDBRL.Code,
		exchangeRate.USDBRL.Codein,
		exchangeRate.USDBRL.Name,
		exchangeRate.USDBRL.High,
		exchangeRate.USDBRL.Low,
		exchangeRate.USDBRL.VarBid,
		exchangeRate.USDBRL.PctChange,
		exchangeRate.USDBRL.Bid,
		exchangeRate.USDBRL.Ask,
	)
	if err != nil {
		return err
	}
	return nil
}
