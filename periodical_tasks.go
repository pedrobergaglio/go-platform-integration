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
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
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

	fmt.Println("meli refresh token:", tokenResponse.RefreshToken, "meli access token:", tokenResponse.AccessToken)

	return nil
}

// checkStock calls a store procedure in the mysql database to check if
// the stock of each product corresponds to the stored movements
func checkStock() {

	log.Print("checking stock values")

	// Connect to the MySQL database
	db, err := sql.Open("mysql", "megared_pedro:Engsu_23@tcp(Mysql4.gohsphere.com)/megared_energiaglobal_23?charset=utf8")
	if err != nil {
		log.Fatal("error connecting to the database:", err)
	}
	defer db.Close()

	// Execute the query
	rows, err := db.Query("CALL CHECK_STOCK()")
	if err != nil {
		log.Fatal("error executing query:", err)
	} else {
		rows.Close()
	}

}

// Scrapes and updates the value of the dollar from BNA
func scrapeBnaDollar() {
	// Send a GET request to the URL
	resp, err := http.Get("https://www.bna.com.ar/Personas")
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()

	// Parse the HTML document
	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		log.Fatal(err)
	}

	// Find the element using the CSS selector
	selector := "#billetes > table > tbody > tr:nth-child(1) > td:nth-child(3)"
	value := doc.Find(selector).Text()

	// Clean up the extracted value
	value = strings.TrimSpace(value)

	fmt.Println("Value:", value)
}

func addNewDate() {

	log.Print("adding new day to dates")

	// Connect to the MySQL database
	db, err := sql.Open("mysql", "megared_pedro:Engsu_23@tcp(Mysql4.gohsphere.com)/megared_energiaglobal_23?charset=utf8")
	if err != nil {
		log.Fatal("error connecting to the database:", err)
	}
	defer db.Close()

	// Execute the query
	rows, err := db.Query("INSERT INTO DATES (date) VALUES (CURDATE())")
	if err != nil {
		log.Fatal("error executing query:", err)
	} else {
		rows.Close()
	}

}

// refreshPeriodically function runs every 5 hours to refresh the meli tokens,
// check stock values, and collect alephee prices with updateRumboPricesAlephee()
func refreshPeriodically() {
	refreshInterval := time.Hour * 5 // Refresh the token every hour (adjust as needed)

	for {
		//updateRumboPricesAlephee()
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

func RunAtMidnight() {
	// Get the current time in UTC
	now := time.Now().UTC()

	// Calculate the duration until the next midnight UTC
	nextMidnight := now.Truncate(24 * time.Hour).Add(24 * time.Hour)
	durationUntilMidnight := nextMidnight.Sub(now)

	// Start a goroutine that runs the scheduled function at the next midnight
	go func() {
		time.Sleep(durationUntilMidnight)
		updateRumboPricesAlephee()
		addNewDate()

		// Set up a ticker to run the function every 24 hours (starting at the next midnight)
		ticker := time.NewTicker(24 * time.Hour)
		defer ticker.Stop()
		// Schedule the daily task to run again every 24 hours
		for range time.Tick(24 * time.Hour) {
			updateRumboPricesAlephee()
			addNewDate()
		}
	}()
}
