package main

//https://articulo.mercadolibre.com.ar/MLA-1449065018-item-de-prueba-por-favor-no-ofertar-_JM

/*
COMPRADOR
{
	"id":1422812884,
	"email":"test_user_941471677@testuser.com",
	"nickname":"TESTUSER941471677",
	"site_status":"active",
	"password":"nT3xqyQeup"
	}
*/

/*
VENDEDOR
{
	"id":1423001750,
	"email":"test_user_275613916@testuser.com",
	"nickname":"TESTUSER275613916",
	"site_status":"active",
	"password":"gzoLcRK5SJ"
}
*/

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

import (
	"fmt"
	"log"
	"net/http"
	"os"
)

/*
func createUser() {

	// Define the request URL and payload
	url := "https://api.mercadolibre.com/users/test_user"
	payload := `{
		"site_id": "MLA"
	}`

	// Create the HTTP request
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewBufferString(payload))
	if err != nil {
		fmt.Println(req)
		return
	}

	fmt.Println(os.Getenv("MELI_ACCESS_TOKEN"))

	// Set the request headers
	req.Header.Set("Authorization", "Bearer "+os.Getenv("MELI_ACCESS_TOKEN"))
	req.Header.Set("Content-type", "application/json")

	// Send the request
	client := http.DefaultClient
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("do")
		return
	}
	defer resp.Body.Close()

	// Read the response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("body")
		return
	}

	fmt.Println(string(body))
}*/

func main() {

	loadConfig()

	//os.Setenv("meli_refresh_token", "TG-64b08f898b72c50001cbe90e-1423001750")

	// Start a background goroutine to periodically check token expiration and refresh if needed
	go refreshPeriodically()

	// Register the webhook handler functions with the default server mux
	http.HandleFunc("/movement", handleASMovementWebhook)
	http.HandleFunc("/woocommerce", handleWCWebhook)
	http.HandleFunc("/price", handleASPriceWebhook)
	http.HandleFunc("/meli", handleMeliWebhook)

	// Root route handler
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// Write the desired text response
		fmt.Fprintf(w, "ENERGÍA GLOBAL INTEGRATION SERVICE")
	})

	// Use PORT environment variable provided by Railway or default to 8080
	port := ":" + os.Getenv("PORT")
	if port == ":" {
		port = ":8080"
	}

	// Start the server and specify the host and port
	log.Println("server listening on", port)
	log.Fatal(http.ListenAndServe("0.0.0.0"+port, nil))

}

/*
func getItems() {

	accessToken := os.Getenv("meli_access_token")

	// Define the request URL and payload
	url := "https://api.mercadolibre.com/items/MLA1449065018"
	payload := fmt.Sprintf(`{
		"grant_type": "refresh_token",
		"client_id": "3917704976553080",
		"client_secret": "6VuqfhmGawIqjEmp7pzgFhyTSChQjhl4",
		"refresh_token": "%s"
	}`, accessToken)

}
*/
