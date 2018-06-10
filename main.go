package main

import (
	"encoding/json"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type weatherData struct {
	Name string `json:"name"`
	Main struct {
		Kelvin float64 `json:"temp"`
	} `json:"main"`
}

type apixuData struct {
	Location struct {
		Name string `json:"name"`
	} `json:"location"`
	Current struct {
		Celsius float64 `json:"temp_c"`
	} `json:"current"`
}

type weatherProvider interface {
	temperature(city string) (float64, error)
}

type openWeatherMap struct{}

type apixu struct{}

type multiWeatherProvider []weatherProvider

//http://api.apixu.com/v1/current.json?key=a56cd6be31f140c989f234721182705&q=Buenos%20Aires

func main() {
	mw := multiWeatherProvider{
		openWeatherMap{},
		apixu{},
	}

	http.HandleFunc("/", hello)
	http.HandleFunc("/weather/", func(w http.ResponseWriter, r *http.Request) {
		begin := time.Now()
		city := strings.SplitN(r.URL.Path, "/", 3)[2]

		temp, err := mw.temperature(city)

		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"city": city,
			"temp": temp,
			"took": time.Since(begin).String(),
		})
	})
	http.ListenAndServe(":8080", nil)
}

func (w multiWeatherProvider) temperature(city string) (float64, error) {
	sum := 0.0

	for _, provider := range w {
		temp, err := provider.temperature(city)
		if err != nil {
			return 0, nil
		}

		sum += temp
	}

	return sum / float64(len(w)), nil
}

func (w openWeatherMap) temperature(city string) (float64, error) {
	resp, err := http.Get("http://api.openweathermap.org/data/2.5/weather?APPID=d48cc5e1eaf189bb333fe1cdd892140d&q=" + city)

	if err != nil {
		return 0, err
	}

	defer resp.Body.Close()

	var d weatherData

	if err := json.NewDecoder(resp.Body).Decode(&d); err != nil {
		return 0, err
	}

	log.Printf("openWeatherMap: %s: %.2f", city, d.Main.Kelvin)
	return d.Main.Kelvin, nil
}

func (w apixu) temperature(city string) (float64, error) {
	var Url *url.URL
	Url, err := url.Parse("http://api.apixu.com")
	Url.Path += "/v1/current.json"

	parameters := url.Values{}
	parameters.Add("key", "a56cd6be31f140c989f234721182705")
	parameters.Add("q", city)
	Url.RawQuery = parameters.Encode()

	log.Printf("url: %s", Url.String())

	if err != nil {
		return 0, err
	}

	resp, err := http.Get(Url.String())

	if err != nil {
		return 0, err
	}

	defer resp.Body.Close()

	var d apixuData

	if err := json.NewDecoder(resp.Body).Decode(&d); err != nil {
		return 0, err
	}

	kelvin := d.Current.Celsius + 273.15

	log.Printf("apixu: %s: %.2f", city, kelvin)
	return kelvin, nil
}

func hello(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("Hola"))
}
