package postits

import (
	"fmt"
	"maps"
	"net/url"

	"github.com/Secreto31126/tesis/common/models"
)

// func objectFilter(input map[string]any, keys ...string) map[string]any {
// 	data := make(map[string]any)

// 	for _, key := range keys {
// 		if val, ok := input[key]; ok {
// 			data[key] = val
// 		}
// 	}

// 	return data
// }

func findWellKnown(key string, params map[string]string) (*models.PostIts, error) {
	wk, ok := configuredPostIts[key]
	if !ok {
		return nil, fmt.Errorf("Well Known post-it not found")
	}

	if wk.Request.Queries != nil {
		wk.Request.Queries = maps.Clone(wk.Request.Queries)
	}

	if wk.Request.Headers != nil {
		wk.Request.Headers = maps.Clone(wk.Request.Headers)
	}

	if wk.Params != nil {
		wk.Params = maps.Clone(wk.Params)

		for name, def := range wk.Params {
			value, ok := params[name]

			if ok {
				wk.Params[name] = value
				continue
			}

			if def != "" {
				continue
			}

			return nil, fmt.Errorf("Missing required variable")
		}
	}

	return &wk, nil
}

func getURL(raw string) *url.URL {
	res, _ := url.Parse(raw)
	return res
}

var configuredPostIts = map[string]models.PostIts{
	"static_card": {
		WellKnown: "static_card",
		Params: map[string]string{
			"text": "",
		},
		Resource: nil,
		Rate:     0,
	},
	"temperature": {
		WellKnown: "temperature",
		Params: map[string]string{
			"$latitude":   "-34.6131",
			"$longitude":  "-58.3772",
			"$start_date": "",
			"$end_date":   "",
		},
		Resource: getURL("https://api.open-meteo.com/v1/forecast"),
		Request: models.Request{
			Method: "GET",
			Queries: map[string]string{
				"hourly":     "temperature_2m",
				"latitude":   "$latitude",
				"longitude":  "$longitude",
				"start_date": "$start_date",
				"end_date":   "$end_date",
			},
			Headers: map[string]string{
				"Accept": "application/json",
			},
		},
		Response: "json",
		Query:    "{ min: .hourly.temperature_2m | min, max: .hourly.temperature_2m | max }",
		Rate:     30,
	},
	"events_search": {
		WellKnown: "events_search",
		Params: map[string]string{
			"$keyword": "",
		},
		Resource: getURL("https://app.ticketmaster.com/discovery/v2/events.json"),
		Request: models.Request{
			Method: "GET",
			Queries: map[string]string{
				"keyword": "$keyword",
				"size":    "1",
				"apikey":  "kpGJZiOXIoaBznO4tDvAxXgZN4DGpehP", // TODO: secrets manager
			},
			Headers: map[string]string{
				"Accept": "application/json",
			},
		},
		Response: "json",
		Query:    "{ name: ._embedded.events[0].name, sales: ._embedded.events[0].sales.public.startDateTime, image: ._embedded.events[0].images[0].url }",
		Rate:     120,
	},
	"dog_facts": {
		WellKnown: "dog_facts",
		Resource:  getURL("https://dogapi.dog/api/v2/facts"),
		Request: models.Request{
			Method: "GET",
		},
		Response: "json",
		Query:    ".data[0].attributes",
		Rate:     5,
	},
	"dolar_oficial": {
		WellKnown: "dolar_oficial",
		Resource:  getURL("https://dolarapi.com/v1/dolares/oficial"),
		Request: models.Request{
			Method: "GET",
		},
		Response: "json",
		Query:    "{ compra: .compra, venta: .venta }",
		Rate:     120,
	},
	"riesgo_pais": {
		WellKnown: "riesgo_pais",
		Resource:  getURL("https://api.argentinadatos.com/v1/finanzas/indices/riesgo-pais/ultimo"),
		Request: models.Request{
			Method: "GET",
			Headers: map[string]string{
				"Accept": "application/json",
			},
		},
		Response: "json",
		Query:    "{ valor: .valor, fecha: .fecha }",
		Rate:     3600,
	},
	"crypto_price": {
		WellKnown: "crypto_price",
		Params: map[string]string{
			"$coin":     "bitcoin",
			"$currency": "usd",
		},
		Resource: getURL("https://api.coingecko.com/api/v3/simple/price"),
		Request: models.Request{
			Method: "GET",
			Queries: map[string]string{
				"ids":                 "$coin",
				"vs_currencies":       "$currency",
				"include_24hr_change": "true",
			},
			Headers: map[string]string{
				"Accept": "application/json",
			},
		},
		Response: "json",
		Query:    ".[] | { price: (to_entries[] | select(.key | endswith(\"_24h_change\") | not) | .value), change: (to_entries[] | select(.key | endswith(\"_24h_change\")) | .value) }",
		Rate:     60,
	},
	"air_quality": {
		WellKnown: "air_quality",
		Params: map[string]string{
			"$latitude":  "-34.6131",
			"$longitude": "-58.3772",
		},
		Resource: getURL("https://air-quality-api.open-meteo.com/v1/air-quality"),
		Request: models.Request{
			Method: "GET",
			Queries: map[string]string{
				"latitude":  "$latitude",
				"longitude": "$longitude",
				"current":   "us_aqi,pm2_5",
			},
			Headers: map[string]string{
				"Accept": "application/json",
			},
		},
		Response: "json",
		Query:    "{ aqi: .current.us_aqi, pm25: .current.pm2_5, time: .current.time }",
		Rate:     900,
	},
	"github_repo": {
		WellKnown: "github_repo",
		Params: map[string]string{
			"$query": "",
		},
		Resource: getURL("https://api.github.com/search/repositories"),
		Request: models.Request{
			Method: "GET",
			Queries: map[string]string{
				"q":        "$query",
				"sort":     "stars",
				"order":    "desc",
				"per_page": "1",
			},
			Headers: map[string]string{
				"Accept": "application/vnd.github+json",
			},
		},
		Response: "json",
		Query:    ".items[0] | { name: .full_name, stars: .stargazers_count, forks: .forks_count, issues: .open_issues_count, description: .description }",
		// Unauthenticated search is capped at 10 requests/minute.
		Rate: 300,
	},
}
