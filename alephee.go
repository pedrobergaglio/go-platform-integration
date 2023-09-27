package main

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"time"
)

/*
https://api.alephcrm.com/v2/products?API_KEY=8F509A97-B5C8-4B9E-8148-07C055C54C05&accountId=3319

[
    {
      "sku": "1025700",
      "stock": {
        "quantity": 2}
    }
  ]
*/

// Updates the stock of an alephee publication
func updateAlephee(alephee_id, stock string) string {

	// Convert the total_stock value to an integer
	alephee_id_int, err := strconv.Atoi(alephee_id)
	if err != nil {
		return fmt.Sprintln("error converting publication id to int:", err)
	}
	// Format the total_stock with leading zeros (7 characters)
	alephee_id = fmt.Sprintf("%07d", alephee_id_int)

	URL := fmt.Sprintf("https://api.alephcrm.com/v2/products?API_KEY=%s&accountId=%s", os.Getenv("alephee_api_key"), os.Getenv("alephee_account_id"))
	payload := fmt.Sprintf(`
	[
		{
		  "sku": "%s",
		  "stock": {
			"quantity": %s
			}
		}
	  ]`, alephee_id, stock)

	req, err := http.NewRequest(http.MethodPost, URL, bytes.NewBufferString(payload))
	if err != nil {
		return "error creating request for alephee:" + fmt.Sprint(err)
	}

	req.Header.Set("Content-Type", "application/json")

	client := http.DefaultClient
	resp, err := client.Do(req)
	if err != nil {
		return "error updating product in alephee:" + fmt.Sprint(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != 412 && resp.StatusCode != 429 {
		errorBody, _ := io.ReadAll(resp.Body)
		return "error updating product in alephee: " + string(errorBody)

	} else if resp.StatusCode == 412 {
		return "412: no changes processed, possibly sku (alephee_id) not found"

	} else if resp.StatusCode == 429 {
		fmt.Println("error too many requests. waiting for 1 minute...")
		time.Sleep(1*time.Minute + 5*time.Second) // Wait for 1 minute and 5 seconds
		return updateAlephee(alephee_id, stock)
	}

	return ""
}
