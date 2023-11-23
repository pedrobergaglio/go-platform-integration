package main

/*
import (
	"log"
	"time"
)

func chau() {
	now := time.Now()

	log.Println("hola")

	// Schedule task at 18:00 UTC
	go func() {
		today := now.Truncate(24 * time.Hour)

		// se suman 3 a las 18 porque está en utc
		next18 := today.Add(21*time.Hour + 0*time.Minute)
		if now.After(next18) {
			next18 = next18.Add(24 * time.Hour)
		}
		durationUntil18 := next18.Sub(now)
		log.Println("faltan:", durationUntil18)

		log.Println("hola")

		time.Sleep(durationUntil18)

		// Set up a ticker to run the function every 24 hours
		ticker := time.NewTicker(2 * time.Second)
		defer ticker.Stop()

		for range ticker.C {
			log.Println("hola")
		}
	}()

}

func main() {

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
*/
