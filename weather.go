package weather

import (
	"context"
	"encoding/json"
	"errors"
	"io/ioutil"
	"math"
	"net/http"
	"net/url"
	"sort"
	"time"
)

const OpenWeatherMapURL = `https://api.openweathermap.org/data/2.5/`

type Weather struct {
	Date           time.Time
	Temperature    float64
	TemperatureMin float64
	TemperatureMax float64
	Humidity       float64
}

type Units string

const (
	Kelvin   Units = "kelvin"
	Imperial Units = "imperial"
	Metric   Units = "metric"
)

type Option func(c *Client)

func WithAPIKey(k string) Option {
	return func(c *Client) {
		c.apiKey = k
	}
}

func WithUnits(units Units) Option {
	return func(c *Client) {
		c.units = units
	}
}

type Client struct {
	apiKey     string
	units      Units
	httpClient *http.Client
}

func NewClient(opts ...Option) Client {
	c := Client{
		units:      Kelvin,
		httpClient: &http.Client{Timeout: 5 * time.Second},
	}
	for _, opt := range opts {
		opt(&c)
	}
	return c
}

func (c Client) makeRequest(ctx context.Context, dest interface{}, endpoint string, queryParams url.Values) error {
	req, err := http.NewRequest("GET", OpenWeatherMapURL+endpoint, nil)
	if err != nil {
		return err
	}

	queryParams.Set("APPID", c.apiKey)
	if c.units != Kelvin {
		queryParams.Set("units", string(c.units))
	}
	req.URL.RawQuery = queryParams.Encode()

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusOK {
		b, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return err
		}
		return errors.New(string(b))
	}

	return json.NewDecoder(resp.Body).Decode(dest)
}

func (c Client) GetForecast(ctx context.Context, zip string) (Forecast, error) {
	var resp struct {
		List []struct {
			Timestamp int64 `json:"dt"`
			Main      struct {
				Temperature    float64 `json:"temp"`
				TemperatureMin float64 `json:"temp_min"`
				TemperatureMax float64 `json:"temp_max"`
				Humidity       float64 `json:"humidity"`
			} `json:"main"`
		} `json:"list"`
	}

	params := make(url.Values)
	params.Set("zip", zip)
	if err := c.makeRequest(ctx, &resp, "forecast", params); err != nil {
		return nil, err
	}

	weathers := make(Forecast, 0, len(resp.List))
	for _, w := range resp.List {
		weathers = append(weathers, Weather{
			Date:           time.Unix(w.Timestamp, 0),
			Humidity:       w.Main.Humidity,
			Temperature:    w.Main.Temperature,
			TemperatureMin: w.Main.TemperatureMin,
			TemperatureMax: w.Main.TemperatureMax,
		})
	}

	return weathers, nil
}

func (c Client) GetCurrentWeather(ctx context.Context, zip string) (Weather, error) {
	var resp struct {
		Timestamp int64 `json:"dt"`
		Main      struct {
			Temperature    float64 `json:"temp"`
			TemperatureMin float64 `json:"temp_min"`
			TemperatureMax float64 `json:"temp_max"`
			Humidity       float64 `json:"humidity"`
		} `json:"main"`
	}

	params := make(url.Values)
	params.Set("zip", zip)
	if err := c.makeRequest(ctx, &resp, "weather", params); err != nil {
		return Weather{}, err
	}

	return Weather{
		Date:           time.Unix(resp.Timestamp, 0),
		Temperature:    resp.Main.Temperature,
		TemperatureMax: resp.Main.TemperatureMax,
		TemperatureMin: resp.Main.TemperatureMin,
	}, nil
}

type Forecast []Weather

func (f Forecast) Daily() Forecast {
	days := make(map[string]Forecast)
	keys := make([]string, 0)
	var loc *time.Location
	for _, w := range f {
		if loc == nil {
			loc = w.Date.Location()
		}
		key := w.Date.Format("20060102")
		if _, seen := days[key]; !seen {
			keys = append(keys, key)
		}
		days[key] = append(days[key], w)
	}
	sort.Strings(keys)

	dailyForecast := make(Forecast, 0, len(days))
	for _, key := range keys {
		hourly := days[key]
		date, _ := time.ParseInLocation("20060102", key, loc)
		dailyForecast = append(dailyForecast, Weather{
			Date:           date,
			Humidity:       hourly.AverageHumidity(),
			Temperature:    hourly.AverageTemperature(),
			TemperatureMin: hourly.MinimumTemperature(),
			TemperatureMax: hourly.MaximumTemperature(),
		})
	}

	return dailyForecast
}

func (f Forecast) MaximumTemperature() float64 {
	max := math.Inf(-1)
	for _, w := range f {
		if w.TemperatureMax > max {
			max = w.TemperatureMax
		}
	}
	return max
}

func (f Forecast) MinimumTemperature() float64 {
	min := math.Inf(1)
	for _, w := range f {
		if w.TemperatureMin < min {
			min = w.TemperatureMin
		}
	}
	return min
}

func (f Forecast) AverageTemperature() float64 {
	temp := 0.0
	for _, w := range f {
		temp += w.Temperature
	}
	return temp / float64(len(f))
}

func (f Forecast) AverageHumidity() float64 {
	hum := 0.0
	for _, w := range f {
		hum += w.Humidity
	}
	return hum / float64(len(f))
}
