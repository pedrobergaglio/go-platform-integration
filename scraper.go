package main

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/chromedp/chromedp"
	_ "github.com/go-sql-driver/mysql"
)

// Converts a number or string into string
func convertToStringg(value interface{}) string {
	switch v := value.(type) {
	case string:
		return v
	case int, int64, float32, float64:
		return fmt.Sprintf("%v", v)
	default:
		return ""
	}
}

// Scrapes and updates the value of the dollar from BNA
func mainxx() {
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

	// Replace ',' with '.'
	value = strings.Replace(value, ",", ".", -1)

	// Parse string to float64
	dollar, err := strconv.ParseFloat(value, 64)
	if err != nil {
		fmt.Println("Error:", err)
		return
	}

	value = convertToString(dollar)

	payload := fmt.Sprintf(`
		{
			"Action": "Edit",
			"Properties": {
				"Locale": "es-US",
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
				}
			]
		}`, value, value, value)
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

	log.Println("finished updating rumbo prices")

}

func mainx() {
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
		log.Fatal(err)
	}

	if err != nil {
		log.Println("Ad window not found:", err)
	}

	// Scrape the value
	var value string
	err = chromedp.Run(ctx,
		chromedp.Text(valueSelector, &value),
	)

	if err != nil {
		log.Fatal(err)
	}

	log.Println("Scraped value:", value)

	// Clean up
	chromedp.Cancel(ctx)
}

func mainnnn() {
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
	//closeAdButtonSelector := "#modal > div > div.cpc"
	//valueSelector := "#general-info > div:nth-child(2) > span:nth-child(3)"

	log.Print(0)

	// Create a context
	ctx, cancel := chromedp.NewContext(context.Background())
	defer cancel()

	log.Print(1)

	// Wait for the guide window to appear and close it
	/*if err := chromedp.Run(ctx,
		chromedp.WaitVisible("#modal > div > div.cpc"),
		chromedp.Click("#modal > div > div.cpc"),
		chromedp.WaitVisible(loginButtonSelector, chromedp.ByID),
	); err != nil {
		log.Println("Guide window not found:", err)
	}*/

	log.Print(1.5)

	// Login
	err := chromedp.Run(ctx,
		chromedp.Navigate(loginURL),
		chromedp.WaitVisible(loginButtonSelector, chromedp.ByID),
	)
	if err != nil {
		log.Fatal(err)
	}

	log.Print(1.75)

	err = chromedp.Run(ctx,
		chromedp.SendKeys(usernameInputSelector, username, chromedp.ByID),
		chromedp.SendKeys(accountNumberInputSelector, accountNumber, chromedp.ByID),
		chromedp.SendKeys(passwordInputSelector, password, chromedp.ByID),
	)
	if err != nil {
		log.Fatal(err)
	}

	log.Print(1.85)

	err = chromedp.Run(ctx,
		chromedp.Click(loginButtonSelector, chromedp.ByID),
	)
	if err != nil {
		log.Fatal(err)
	}

	log.Print(1.88)

	// Define the action to run JavaScript for debugging
	debugAction := chromedp.ActionFunc(func(ctx context.Context) error {
		// Run custom JavaScript to log the inner HTML of the element
		script := `
		var element = document.querySelector("#general-info > div:nth-child(2) > span:nth-child(3)");
		if (element) {
			element.innerHTML;
		} else {
			"Element not found";
		}
	`

		var result string
		return chromedp.EvaluateAsDevTools(script, &result).Do(ctx)
	})

	var result string

	// Execute the debug action
	err = chromedp.Run(ctx, debugAction)
	if err != nil {
		log.Println("Debug error:", err)
	} else {
		log.Println("Scraped value:", result)
	}

	err = chromedp.Run(ctx,
		//chromedp.WaitVisible(closeAdButtonSelector), // Wait for the ad window to appear
		//chromedp.Click(closeAdButtonSelector),       // Close the ad window
		chromedp.WaitVisible("#general-info > div:nth-child(2)"),
	)

	if err != nil {
		log.Fatal(err)
	}

	log.Print(2)

	// Scrape the value
	var value string
	err = chromedp.Run(ctx,
		chromedp.Text("#general-info > div:nth-child(2) > span:nth-child(3)", &value),
	)

	if err != nil {
		log.Fatal(err)
	}

	log.Print(3)

	log.Println("Scraped value:", value)

	// Clean up
	chromedp.Cancel(ctx)
}
