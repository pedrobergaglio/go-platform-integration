package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

type TokenResponse struct {
	RefreshToken string `json:"refresh_token"`
	AccessToken  string `json:"access_token"`
}

// since meli access token expire every 6 hours,
// refreshMeliToken function gets the last given refresh token to get a new access token
func refreshMeliToken() error {

	refreshToken := os.Getenv("meli_refresh_token")

	// Define the request URL and payload
	url := "https://api.mercadolibre.com/oauth/token"
	payload := fmt.Sprintf(`{
		"grant_type": "refresh_token",
		"client_id": "3917704976553080",
		"client_secret": "6VuqfhmGawIqjEmp7pzgFhyTSChQjhl4",
		"refresh_token": "%s"
	}`, refreshToken)

	// Create the HTTP request
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewBufferString(payload))
	if err != nil {
		return err
	}

	// Set the request headers
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	// Send the request
	client := http.DefaultClient
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Read the response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	// Unmarshal the JSON response into a TokenResponse struct
	var tokenResponse TokenResponse
	err = json.Unmarshal(body, &tokenResponse)
	if err != nil {
		return err
	}

	if tokenResponse.AccessToken == "" || tokenResponse.RefreshToken == "" {
		return errors.New("empty variables returned")
	}

	err = os.Setenv("MELI_ACCESS_TOKEN", tokenResponse.AccessToken)
	if err != nil {
		return err
	}

	err = os.Setenv("MELI_REFRESH_TOKEN", tokenResponse.RefreshToken)
	if err != nil {
		return err
	}

	//fmt.Println("new refresh token:", tokenResponse.RefreshToken)
	//fmt.Println("new access token:", tokenResponse.AccessToken)

	return nil
}

// checkStock calls a store procedure in the mysql database to check if
// the stock of each product corresponds to the stored movements
func checkStock() {

	// Connect to the MySQL database
	db, err := sql.Open("mysql", "megared_pedro:Engsu_23@tcp(Mysql4.gohsphere.com)/megared_energiaglobal_23?charset=utf8")
	if err != nil {
		log.Fatal("Error connecting to the database:", err)
	}
	defer db.Close()

	// Execute the query
	rows, err := db.Query("CALL CHECK_STOCK()")
	if err != nil {
		log.Fatal("Error executing query:", err)
	} else {
		rows.Close()
	}

}

// refreshPeriodically function runs every 5 hours to refresh the meli tokens,
// check stock values, and collect alephee prices with updateRumboPricesAlephee()
func refreshPeriodically() {
	refreshInterval := time.Hour * 5 // Refresh the token every hour (adjust as needed)

	for {
		updateRumboPricesAlephee()
		checkStock()
		err := refreshMeliToken()
		if err != nil {
			log.Println("retrying. there was an error refreshing the meli token:", err)
			err := refreshMeliToken()
			if err != nil {
				log.Fatal("there was an error refreshing the meli token:", err)
			}
		}

		// Wait for the refresh interval
		time.Sleep(refreshInterval)
	}
}
