package main

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/chromedp/chromedp"
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
	db, err := sql.Open("mysql", fmt.Sprintf("%s:%s@tcp(%s)/%s?charset=utf8", os.Getenv("database_user"), os.Getenv("database_pass"), os.Getenv("database_ip"), os.Getenv("database_name")))
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

func reloadCounting(user string) {
	// Connect to the MySQL database
	db, err := sql.Open("mysql", fmt.Sprintf("%s:%s@tcp(%s)/%s?charset=utf8", os.Getenv("database_user"), os.Getenv("database_pass"), os.Getenv("database_ip"), os.Getenv("database_name")))
	if err != nil {
		log.Println("error connecting to the database:", err)
		return
	}
	defer db.Close()

	// Delete rows from ITEMS_TO_COUNT
	deleteQuery := fmt.Sprintf("DELETE FROM ITEMS_TO_COUNT WHERE user = '%s'", user)
	_, err = db.Exec(deleteQuery)
	if err != nil {
		log.Println("error executing delete query:", err)
		return
	}

	// Insert new rows into ITEMS_TO_COUNT
	insertQuery := `
		INSERT INTO ITEMS_TO_COUNT (product_id, user, quantity, brand)
		SELECT product_id, ? AS user, 0 AS quantity, brand
		FROM STOCK
		WHERE fabrica > 0 AND brand = 'HONDA'
	`
	_, err = db.Exec(insertQuery, user)
	if err != nil {
		log.Println("error executing insert query:", err)
		return
	}
}

// Scrapes and updates the value of the dollar from BNA, Rumbo, Munditol
func updateUsdPrices() {

	loadConfig()

	log.Print("updating usd prices")

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
	scraped_bna := doc.Find(selector).Text()

	// Clean up the extracted value
	scraped_bna = strings.TrimSpace(scraped_bna)

	// Replace ',' with '.'
	scraped_bna = strings.Replace(scraped_bna, ",", ".", -1)

	// Parse string to float64
	dollar, err := strconv.ParseFloat(scraped_bna, 64)
	if err != nil {
		fmt.Println("Error:", err)
		return
	}

	scraped_bna = convertToString(dollar)

	scraped_rumbo := getRumboUsd()
	scraped_munditol := getMunditolUsd()

	payload := fmt.Sprintf(`
		{
			"Action": "Edit",
			"Properties": {
				"Locale": "en-US",
				"Timezone": "Argentina Standard Time"
			},
			"Rows": [
				{
					"supplier": "HONDA",
					"supplier_usd": %s
				},
				{
					"supplier": "MEGA RED",
					"supplier_usd": %s
				},
				{
					"supplier": "ENERGÍA GLOBAL",
					"supplier_usd": %s
				},
				{
					"supplier": "RUMBO",
					"supplier_usd": %s
				},
				{
					"supplier": "MUNDITOL",
					"supplier_usd": %s
				}
			]
		}`, scraped_bna, scraped_bna, scraped_bna, scraped_rumbo, scraped_munditol)
	log.Print(payload)
	// Create the request
	requestURL := fmt.Sprintf("https://api.appsheet.com/api/v2/apps/%s/tables/suppliers/Action", os.Getenv("appsheet_id"))
	req, err := http.NewRequest(http.MethodPost, requestURL, bytes.NewBufferString(payload))
	if err != nil {
		log.Printf("failed to create request: %v", err)
		return

	}

	// Set request headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("ApplicationAccessKey", os.Getenv("appsheet_key"))

	// Send the request
	client := http.DefaultClient
	resp, err = client.Do(req)
	if err != nil {
		log.Printf("failed to send request: %v", err)
		return
	}
	defer resp.Body.Close()

	// Check the response status code
	if resp.StatusCode != http.StatusOK {
		log.Printf("unexpected status code from appsheet when updating dollar price: %d", resp.StatusCode)
		return
	}

	log.Println("finished updating dollar prices")

}

func getMunditolUsd() (scraped_munditol string) {

	scraped_munditol = ""

	// Define login credentials and URL
	loginURL := "https://mundiextra.munditol.com/"
	username := "nicolas.albertoni@energiaglobal.com.ar"
	password := "energia"

	// Define the CSS selectors
	loginButtonSelector := "#homelogin > div.input_bt_box > button"
	//closeAdButtonSelector := "#mundial_pop_box > div > div > img"
	valueSelector := "#multiplicador_id > b"

	// Create a context
	ctx, cancel := chromedp.NewContext(context.Background())
	defer cancel()

	// Login
	err := chromedp.Run(ctx,
		chromedp.Navigate(loginURL),
		chromedp.WaitVisible(loginButtonSelector),
		chromedp.SendKeys("#login", username, chromedp.ByID),
		chromedp.SendKeys("#passwd", password, chromedp.ByID),
		chromedp.Click(loginButtonSelector),
		chromedp.WaitVisible(valueSelector),
	)

	if err != nil {
		log.Println("error munditol:", err)
		return
	}

	err = chromedp.Run(ctx,
		chromedp.Text(valueSelector, &scraped_munditol),
	)

	if err != nil {
		log.Println("error munditol:", err)
		scraped_munditol = ""
		return
	}

	// Clean up
	chromedp.Cancel(ctx)

	// Clean up the extracted value
	scraped_munditol = strings.TrimSpace(scraped_munditol)

	// Replace ',' with '.'
	scraped_munditol = strings.Replace(scraped_munditol, ",", ".", -1)

	// Parse string to float64
	dollar, err := strconv.ParseFloat(scraped_munditol, 64)
	if err != nil {
		log.Println("error munditol:", err)
		scraped_munditol = ""
		return
	}

	scraped_munditol = convertToString(dollar)
	return
}

func getRumboUsd() (scraped_rumbo string) {

	scraped_rumbo = ""

	// Define login credentials and URL
	loginURL := "https://distribuidores.rumbosrl.com.ar/login"
	username := "093163"
	accountNumber := "001"
	password := "compras"

	// Define the CSS selectors
	usernameInputSelector := "username"
	accountNumberInputSelector := "employee_code"
	passwordInputSelector := "password"
	loginButtonSelector := "submitbutton"

	// Create a context
	ctx, cancel := chromedp.NewContext(context.Background())
	defer cancel()

	// Login
	err := chromedp.Run(ctx,
		chromedp.Navigate(loginURL),
		chromedp.WaitVisible("modal", chromedp.ByID),
		chromedp.Click("#modal > div > div.cpc"),
	)
	if err != nil {
		log.Println("error rumbo:", err)
		return
	}

	// Login
	err = chromedp.Run(ctx,
		//chromedp.Navigate(loginURL),
		chromedp.WaitVisible(loginButtonSelector, chromedp.ByID),
	)
	if err != nil {
		log.Println("error rumbo:", err)
		return
	}

	err = chromedp.Run(ctx,
		chromedp.SendKeys(usernameInputSelector, username, chromedp.ByID),
		chromedp.SendKeys(accountNumberInputSelector, accountNumber, chromedp.ByID),
		chromedp.SendKeys(passwordInputSelector, password, chromedp.ByID),
	)
	if err != nil {
		log.Println("error rumbo:", err)
		return
	}

	err = chromedp.Run(ctx,
		chromedp.Click(loginButtonSelector, chromedp.ByID),
	)
	if err != nil {
		log.Println("error rumbo:", err)
		return
	}

	err = chromedp.Run(ctx,
		//chromedp.WaitVisible(closeAdButtonSelector), // Wait for the ad window to appear
		//chromedp.Click(closeAdButtonSelector),       // Close the ad window
		chromedp.WaitVisible("#general-info > div:nth-child(2)"),
	)

	if err != nil {
		log.Println("error rumbo:", err)
		return
	}

	err = chromedp.Run(ctx,
		chromedp.Text("#general-info > div:nth-child(2) > span:nth-child(3)", &scraped_rumbo),
	)

	if err != nil {
		log.Println("error rumbo:", err)
		scraped_rumbo = ""
		return
	}

	// Clean up
	chromedp.Cancel(ctx)

	// Clean up the extracted value
	scraped_rumbo = strings.TrimSpace(scraped_rumbo)

	// Replace ',' with '.'
	scraped_rumbo = strings.Replace(scraped_rumbo, ",", ".", -1)

	// Parse string to float64
	dollar, err := strconv.ParseFloat(scraped_rumbo, 64)
	if err != nil {
		log.Println("error rumbo:", err)
		scraped_rumbo = ""
		return
	}

	scraped_rumbo = convertToString(dollar)
	return
}

func addNewDate() {

	log.Print("adding new day to dates")

	// Connect to the MySQL database
	db, err := sql.Open("mysql", fmt.Sprintf("%s:%s@tcp(%s)/%s?charset=utf8", os.Getenv("database_user"), os.Getenv("database_pass"), os.Getenv("database_ip"), os.Getenv("database_name")))
	if err != nil {
		log.Println("error connecting to the database:", err)
		return
	}
	defer db.Close()

	// Execute the query
	rows, err := db.Query("INSERT INTO DATES (date) VALUES (CURDATE())")
	if err != nil {
		log.Println("error executing query:", err)
		return
	} else {
		rows.Close()
	}

}

// refreshPeriodically function runs every 5 hours to refresh the meli tokens,
// check stock values, and collect alephee prices with updateRumboPricesAlephee()
func refreshPeriodically() {
	refreshInterval := time.Hour * 5 // Refresh the token every hour (adjust as needed)

	for {
		err := refreshMeliToken()
		if err != nil {
			log.Println("retrying. there was an error refreshing the meli token:", err)
			err := refreshMeliToken()
			if err != nil {
				log.Println("there was an error refreshing the meli token:", err)
			}
		}

		// Wait for the refresh interval
		time.Sleep(refreshInterval)
	}
}

func RunAtTime() {
	now := time.Now()

	// Schedule task at 18:00 UTC
	go func() {
		today := now.Truncate(24 * time.Hour)

		// se suman 3 porque está en utc
		next18 := today.Add(21*time.Hour + 0*time.Minute)
		if now.After(next18) {
			next18 = next18.Add(24 * time.Hour)
		}
		durationUntil18 := next18.Sub(now)
		log.Println("faltan:", durationUntil18)

		updateCuentasSos()

		time.Sleep(durationUntil18)

		// Set up a ticker to run the function every 24 hours
		ticker := time.NewTicker(24 * time.Hour)
		defer ticker.Stop()

		for range ticker.C {
			updateCuentasSos()
		}
	}()

	// Calculate the duration until the next midnight UTC
	/*nextMidnight := now.Truncate(24 * time.Hour).Add(24 * time.Hour)
	durationUntilMidnight := nextMidnight.Sub(now)

	// Start a goroutine that runs the scheduled functions
	go func() {
		time.Sleep(durationUntilMidnight)
		//updateRumboPricesAlephee()
		checkStock()
		addNewDate()
		//updateRumboPricesAlephee()

		// Set up a ticker to run the function every 24 hours (starting at the next midnight)
		ticker := time.NewTicker(24 * time.Hour)
		defer ticker.Stop()
		// Schedule the daily task to run again every 24 hours
		for range ticker.C {
			//updateRumboPricesAlephee()
			addNewDate()
			checkStock()
			//updateRumboPricesAlephee()
		}
	}()

	// Schedule task at 11:00 UTC
	go func() {
		// Calculate duration until the next 11:00
		next11AM := now.Truncate(24 * time.Hour).Add(11 * time.Hour)
		if now.After(next11AM) {
			next11AM = next11AM.Add(24 * time.Hour)
		}
		durationUntil11AM := next11AM.Sub(now)

		time.Sleep(durationUntil11AM)

		// Set up a ticker to run the function every 24 hours
		ticker := time.NewTicker(24 * time.Hour)
		defer ticker.Stop()

		for range ticker.C {
			// Run the task you want to execute at 11:00
			updateUsdPrices()
		}
	}()

	// Schedule task at 15:30 UTC
	go func() {
		// Calculate duration until the next 15:30
		next1530 := now.Truncate(24 * time.Hour).Add(15*time.Hour + 30*time.Minute)
		if now.After(next1530) {
			next1530 = next1530.Add(24 * time.Hour)
		}
		durationUntil1530 := next1530.Sub(now)

		time.Sleep(durationUntil1530)

		// Set up a ticker to run the function every 24 hours
		ticker := time.NewTicker(24 * time.Hour)
		defer ticker.Stop()

		for range ticker.C {
			// Run the task you want to execute at 15:30
			updateUsdPrices()
		}
	}()*/

}

func updateSosAt18() {

	now := time.Now()

	updateCuentasSos()

	today := now.Truncate(24 * time.Hour)

	// se suman 3 a las 18 porque está en utc (-3 horas)
	next18 := today.Add(21*time.Hour + 0*time.Minute)
	if now.After(next18) {
		next18 = next18.Add(24 * time.Hour)
	}
	durationUntil18 := next18.Sub(now)
	log.Println("para las 18hs falta:", durationUntil18)

	time.Sleep(durationUntil18)

	for {

		updateCuentasSos()

		time.Sleep(24 * time.Hour)

	}
}

// check stock values, and collect alephee prices with updateRumboPricesAlephee()
func refreshPedidosProduccion() {
	refreshInterval := time.Minute * 20 // Refresh the token every hour (adjust as needed)

	for {
		err := refreshPedidosProduccionFunc()
		if err != nil {
			log.Println("retrying. there was an error sending update start:", err)
			time.Sleep(time.Minute)
			err := refreshPedidosProduccionFunc()
			if err != nil {
				log.Println("there was an error sending update start:", err)
			}
		}

		// Wait for the refresh interval
		time.Sleep(refreshInterval)
	}
}

type PedidosTincho struct {
	RowNumber      string `json:"_RowNumber"`
	NumeroPedido   string `json:"Nº PEDIDO"`
	Vendedor       string `json:"VENDEDOR"`
	Cliente        string `json:"CLIENTE"`
	Equipo         string `json:"EQUIPO"`
	QE             string `json:"Q/E"`
	Observaciones  string `json:"OBSERVACIONES"`
	FechaPedido    string `json:"F.PEDIDO"`
	NumeroTactica  string `json:"N° TACTICA"`
	Estado         string `json:"ESTADO"`
	FechaTerminado string `json:"F.TERMINADO"`
	NumeroDeSerie  string `json:"Nº DE SERIE"`
	Updatecol      string `json:"updatecol"`
}

// Adds a movement in appsheet with the product_id, stock in each location, and movement_type
func refreshPedidosProduccionFunc() error {

	appsheet_id := "970ce665-46a8-4c61-87fa-1e7ea7211db8"
	appsheet_key := "V2-OhxkI-pPuLC-MW362-Bzj0Q-FtTd0-RCJq8-RXZpD-DXDDa"

	//*********************
	//OBTENER TODAS LAS FILAS DEL EXCEL DE TINCHO QUE ESTAN EN LA INTERNA Y ACTUALIZARLO

	payload := `{
		"Action": "Find",
		"Properties": {
			"Locale": "en-US",
			"Timezone": "Argentina Standard Time",
			"Selector": 'Filter(PEDIDOS AÑO ACTUAL, AND(OR(ISNOTBLANK([N° TACTICA]), ISNOTBLANK([CLIENTE])), IN([Nº PEDIDO], EQUIPOS PEDIDOS INTERNA[Nº PEDIDO])))',
		},
		"Rows": []
		}`

	// Create the request
	requestURL := fmt.Sprintf("https://api.appsheet.com/api/v2/apps/%s/tables/PEDIDOS AÑO ACTUAL/Action", appsheet_id)
	req, err := http.NewRequest(http.MethodPost, requestURL, bytes.NewBufferString(payload))
	if err != nil {
		return fmt.Errorf("failed to create request: %v", err)
	}

	// Set request headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("ApplicationAccessKey", appsheet_key)

	// Send the request
	client := http.DefaultClient
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %v", err)
	}
	defer resp.Body.Close()

	// Check the response status code
	if resp.StatusCode != http.StatusOK {
		fmt.Println(payload)
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %v", err)
	}

	// Convert the response body to a string
	responseString := string(body)

	// Make the update
	payload = fmt.Sprintf(`
	{
		"Action": "Edit",
		"Properties": {"Locale": "en-US"},
		"Rows": %s
	}`, responseString)

	// Create the request
	requestURL = fmt.Sprintf("https://api.appsheet.com/api/v2/apps/%s/tables/EQUIPOS PEDIDOS INTERNA/Action", appsheet_id)
	req, err = http.NewRequest(http.MethodPost, requestURL, bytes.NewBufferString(payload))
	if err != nil {
		return fmt.Errorf("failed to create request: %v", err)
	}

	// Set request headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("ApplicationAccessKey", appsheet_key)

	log.Println("actualizando excel de pedidos a producción interno")

	// Send the request
	client = http.DefaultClient
	resp, err = client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %v", err)
	}
	defer resp.Body.Close()

	// Check the response status code
	if resp.StatusCode != http.StatusOK && resp.StatusCode != 504 {
		fmt.Println(payload)
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	//*********************
	//OBTENER TODAS LAS FILAS DEL EXCEL DE TINCHO QUE NO ESTAN EN LA INTERNA Y AGREGARLAS

	payload = `{
		"Action": "Find",
		"Properties": {
			"Locale": "en-US",
			"Timezone": "Argentina Standard Time",
			"Selector": 'Filter(PEDIDOS AÑO ACTUAL, AND(OR(ISNOTBLANK([N° TACTICA]), ISNOTBLANK([CLIENTE])), NOT(IN([Nº PEDIDO], EQUIPOS PEDIDOS INTERNA[Nº PEDIDO]))))',
		},
		"Rows": []
		}`

	// Create the request
	requestURL = fmt.Sprintf("https://api.appsheet.com/api/v2/apps/%s/tables/PEDIDOS AÑO ACTUAL/Action", appsheet_id)
	req, err = http.NewRequest(http.MethodPost, requestURL, bytes.NewBufferString(payload))
	if err != nil {
		return fmt.Errorf("failed to create request: %v", err)
	}

	// Set request headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("ApplicationAccessKey", appsheet_key)

	// Send the request
	client = http.DefaultClient
	resp, err = client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %v", err)
	}
	defer resp.Body.Close()

	// Check the response status code
	if resp.StatusCode != http.StatusOK {
		fmt.Println(payload)
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	body, err = io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %v", err)
	}

	// Convert the response body to a string
	responseString = string(body)

	// Make the update
	payload = fmt.Sprintf(`
	{
		"Action": "Add",
		"Properties": {"Locale": "en-US"},
		"Rows": %s
	}`, responseString)

	// Create the request
	requestURL = fmt.Sprintf("https://api.appsheet.com/api/v2/apps/%s/tables/EQUIPOS PEDIDOS INTERNA/Action", appsheet_id)
	req, err = http.NewRequest(http.MethodPost, requestURL, bytes.NewBufferString(payload))
	if err != nil {
		return fmt.Errorf("failed to create request: %v", err)
	}

	// Set request headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("ApplicationAccessKey", appsheet_key)

	log.Println("agregando filas al excel de pedidos a producción interno")

	// Send the request
	client = http.DefaultClient
	resp, err = client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %v", err)
	}
	defer resp.Body.Close()

	// Check the response status code
	if resp.StatusCode != http.StatusOK && resp.StatusCode != 504 {
		fmt.Println(payload)
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	//********************************************************************************

	//*********************
	//OBTENER TODAS LAS FILAS INTERNAS QUE NO ESTÁN EN EL EXCEL DE TINCHO Y ELIMINARLAS

	payload = `{
		"Action": "Find",
		"Properties": {
			"Locale": "en-US",
			"Timezone": "Argentina Standard Time",
			"Selector": 'Filter(EQUIPOS PEDIDOS INTERNA, NOT(IN([Nº PEDIDO], PEDIDOS AÑO ACTUAL[Nº PEDIDO])))',
		},
		"Rows": []
		}`

	// Create the request
	requestURL = fmt.Sprintf("https://api.appsheet.com/api/v2/apps/%s/tables/EQUIPOS PEDIDOS INTERNA/Action", appsheet_id)
	req, err = http.NewRequest(http.MethodPost, requestURL, bytes.NewBufferString(payload))
	if err != nil {
		return fmt.Errorf("failed to create request: %v", err)
	}

	// Set request headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("ApplicationAccessKey", appsheet_key)

	// Send the request
	client = http.DefaultClient
	resp, err = client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %v", err)
	}
	defer resp.Body.Close()

	// Check the response status code
	if resp.StatusCode != http.StatusOK {
		fmt.Println(payload)
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	body, err = io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %v", err)
	}

	// Convert the response body to a string
	responseString = string(body)

	// Make the update
	payload = fmt.Sprintf(`
	{
		"Action": "Delete",
		"Properties": {"Locale": "en-US"},
		"Rows": %s
	}`, responseString)

	// Create the request
	requestURL = fmt.Sprintf("https://api.appsheet.com/api/v2/apps/%s/tables/EQUIPOS PEDIDOS INTERNA/Action", appsheet_id)
	req, err = http.NewRequest(http.MethodPost, requestURL, bytes.NewBufferString(payload))
	if err != nil {
		return fmt.Errorf("failed to create request: %v", err)
	}

	// Set request headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("ApplicationAccessKey", appsheet_key)

	log.Println("eliminando filas internas que no están en el excel de tincho")

	// Send the request
	client = http.DefaultClient
	resp, err = client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %v", err)
	}
	defer resp.Body.Close()

	// Check the response status code
	if resp.StatusCode != http.StatusOK && resp.StatusCode != 504 {
		fmt.Println(payload)
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}
	log.Println("excel de pedidos a producción interno actualizado")
	//********************************************************************************

	return nil
}

// check stock values, and collect alephee prices with updateRumboPricesAlephee()
func refreshResumenBanco() {
	refreshInterval := time.Hour * 2 // Refresh the token every hour (adjust as needed)

	for {

		fmt.Println("updating resumen formulas")

		err := sendUpdateBanco()
		if err != nil {
			log.Println("retrying. there was an error sending update start:", err)
			time.Sleep(time.Minute)
			err := sendUpdateBanco()
			if err != nil {
				log.Println("there was an error sending update start:", err)
			}
		}

		// Wait for the refresh interval
		time.Sleep(refreshInterval)
	}
}

// Adds a movement in appsheet with the product_id, stock in each location, and movement_type
func sendUpdateBanco() error {

	appsheet_id := "acf512aa-6952-4aaf-8d17-c200fefa116b"
	appsheet_key := "V2-RIUo6-uKEV7-puGvy-TeVYT-K2ag9-85j8j-6IaP2-ZX7Rr"

	//*********************
	//SEND UPDATE

	payload := `{
		"Action": "Actualizar3",
		"Properties": {
			"Locale": "en-US",
			"Timezone": "Argentina Standard Time"
		},
		"Rows": [
			{
				"ID": "hola"
			}
		]
		}`

	// Create the request
	requestURL := fmt.Sprintf("https://api.appsheet.com/api/v2/apps/%s/tables/RESUMEN/Action", appsheet_id)
	req, err := http.NewRequest(http.MethodPost, requestURL, bytes.NewBufferString(payload))
	if err != nil {
		return fmt.Errorf("failed to create request: %v", err)
	}

	// Set request headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("ApplicationAccessKey", appsheet_key)

	// Send the request
	client := http.DefaultClient
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %v", err)
	}
	defer resp.Body.Close()

	// Check the response status code
	if resp.StatusCode != http.StatusOK {
		fmt.Println(payload)
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	return nil
}
