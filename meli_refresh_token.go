package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
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

	err = os.Setenv("MELI_ACCESS_TOKEN", tokenResponse.AccessToken)
	if err != nil {
		return err
	}

	err = os.Setenv("MELI_REFRESH_TOKEN", tokenResponse.RefreshToken)
	if err != nil {
		return err
	}

	fmt.Println("New Refresh Token:", tokenResponse.RefreshToken)
	fmt.Println("New Access Token:", tokenResponse.AccessToken)

	return nil
}
