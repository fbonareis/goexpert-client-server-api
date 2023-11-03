package main

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

const (
	internalExchangeRateApiUrl                = "http://localhost:8080/cotacao"
	internalExchangeRateApiUrlTimeoutDuration = 300 * time.Millisecond
	exchangeRateFilePath                      = "./client/cotacao.txt"
)

func main() {
	value, err := GetExchangeRateValue()
	if err != nil {
		panic(err)
	}
	if err := WriteExchangeRate(value); err != nil {
		panic(err)
	}
}

func WriteExchangeRate(value string) error {
	file, err := os.Create(exchangeRateFilePath)
	if err != nil {
		return err
	}
	if _, err = file.WriteString(fmt.Sprintf("Dólar: %s", value)); err != nil {
		return err
	}
	return nil
}

func GetExchangeRateValue() (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), internalExchangeRateApiUrlTimeoutDuration)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, "GET", internalExchangeRateApiUrl, nil)
	if err != nil {
		return "", err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	return string(body), nil
}
