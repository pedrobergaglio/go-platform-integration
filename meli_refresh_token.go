package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"time"
)

type TokenResponse struct {
	RefreshToken string `json:"refresh_token"`
	AccessToken  string `json:"access_token"`
}

func refreshToken() error {

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
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	// Unmarshal the JSON response into a TokenResponse struct
	var tokenResponse TokenResponse
	err = json.Unmarshal(body, &tokenResponse)
	if err != nil {
		return err
	}

	if "" == tokenResponse.AccessToken || "" == tokenResponse.RefreshToken {
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

	fmt.Println("new refresh token:", tokenResponse.RefreshToken)
	fmt.Println("new access token:", tokenResponse.AccessToken)

	return nil
}

func refreshTokenPeriodically() {
	refreshInterval := time.Hour * 5 // Refresh the token every hour (adjust as needed)

	for {
		err := refreshToken()
		if err != nil {
			log.Println("retrying. there was an error refreshing the meli token:", err)
			err := refreshToken()
			if err != nil {
				log.Fatal("there was an error refreshing the meli token:", err)
			}
		}

		// Wait for the refresh interval
		time.Sleep(refreshInterval)
	}
}
