package main

import (
	"context"
	"log"
	"os"

	"github.com/davecgh/go-spew/spew"

	"github.com/haleyrc/weather"
)

func main() {
	ctx := context.Background()

	key := os.Getenv("OPENWEATHERMAP_APIKEY")
	c := weather.NewClient(
		weather.WithAPIKey(key),
		weather.WithUnits(weather.Imperial),
	)
	w, err := c.GetCurrentWeather(ctx, os.Args[1])
	if err != nil {
		log.Fatalln(err)
	}
	spew.Dump(w)

	ws, err := c.GetForecast(ctx, os.Args[1])
	if err != nil {
		log.Fatalln(err)
	}
	spew.Dump(ws)

	daily := ws.Daily()
	spew.Dump(daily)
}
