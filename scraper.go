package main

/*
import (
	"bytes"
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/spf13/viper"

	"github.com/PuerkitoBio/goquery"
	"github.com/chromedp/chromedp"
	_ "github.com/go-sql-driver/mysql"
)

// Sets each enviroment variable declared in the .env file
func loadConfig() {
	viper.SetConfigFile("resources/.env") // Set the path to your configuration file
	viper.SetConfigType("env")
	// Specify the configuration file type (e.g., "env", "json", "yaml", etc.)

	err := viper.ReadInConfig() // Read the configuration file
	if err != nil {
		log.Fatalf("failed to read configuration file: %s", err)
	}

	// Set environment variables based on the configuration values
	for key, value := range viper.AllSettings() {
		//log.Println("adding ", key)
		if err := os.Setenv(key, value.(string)); err != nil {
			log.Fatalf("failed to set environment variable %s: %s", key, err)
		}
	}
}

// Converts a number or string into string
func convertToString(value interface{}) string {
	switch v := value.(type) {
	case string:
		return v
	case int, int64, float32, float64:
		return fmt.Sprintf("%v", v)
	default:
		return ""
	}
}

*/
