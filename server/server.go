package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/google/uuid"
	_ "github.com/mattn/go-sqlite3"
)

type Response struct {
	UsdBrl UsdBrl `json:"USDBRL"`
}

type UsdBrl struct {
	Bid string `json:"bid"`
}

type Dollar struct {
	ID    string
	Price string
}

func NewDollar(price string) *Dollar {
	return &Dollar{
		ID:    uuid.New().String(),
		Price: price,
	}
}

var database *sql.DB

func main() {
	db, err := sql.Open("sqlite3", "sqlite.db")
	if err != nil {
		panic(err)
	}

	_, err = db.Exec("CREATE TABLE IF NOT EXISTS DOLLARS (ID string PRIMARY KEY, PRICE string not null);")
	if err != nil {
		panic(err)
	}
	database = db

	http.HandleFunc("/cotacao", BuscaCotacaoHandler)
	http.ListenAndServe(":8080", nil)
}

func BuscaCotacaoHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/cotacao" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	result, err := BuscaCotacao()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	dollar := NewDollar(result.Bid)
	fmt.Printf("Nova cotação inserida na base: %v\n", dollar.Price)
	err = insertNewDollar(dollar)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(result)
}

func BuscaCotacao() (*UsdBrl, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", "https://economia.awesomeapi.com.br/json/last/USD-BRL", nil)
	if err != nil {
		return nil, err
	}

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	var response Response
	err = json.Unmarshal(body, &response)
	if err != nil {
		return nil, err
	}

	return &response.UsdBrl, nil
}

func insertNewDollar(dollar *Dollar) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	stmt, err := database.Prepare("INSERT INTO DOLLARS(ID, PRICE) VALUES(?, ?)")
	if err != nil {
		fmt.Println(err)
		return err
	}
	defer stmt.Close()

	_, err = stmt.ExecContext(ctx, dollar.ID, dollar.Price)
	if err != nil {
		fmt.Println(err)
		return err
	}

	return nil
}
