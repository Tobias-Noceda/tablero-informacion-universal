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
		Title: models.Title{
			Text: "{{}}",
			Vars: []string{"text"},
		},
		Params: map[string]string{
			"text": "",
		},
		Resource: nil,
		Rate:     0,
	},
	"temperature": {
		WellKnown: "temperature",
		Title: models.Title{
			Text: "Today's temps are between {{}}°C and {{}}°C",
			Vars: []string{"min", "max"},
		},
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
		Query: map[string]string{
			"min": ".hourly.temperature_2m | min",
			"max": ".hourly.temperature_2m | max",
		},
		Rate: 30,
	},
	"events_search": {
		WellKnown: "events_search",
		Title: models.Title{
			Text: "{{}}: {{}}",
			Vars: []string{"sales", "name"},
		},
		Params: map[string]string{
			"$keyword":    "",
			"$credential": "",
		},
		Resource: getURL("https://app.ticketmaster.com/discovery/v2/events.json"),
		Request: models.Request{
			Method: "GET",
			Queries: map[string]string{
				"keyword": "$keyword",
				"size":    "1",
				"apikey":  "$credential",
			},
			Headers: map[string]string{
				"Accept": "application/json",
			},
		},
		Response: "json",
		Query: map[string]string{
			"name":  "._embedded.events[0].name",
			"sales": "._embedded.events[0].sales.public.startDateTime",
			"image": "._embedded.events[0].images[0].url",
		},
		Rate: 120,
	},
	"dog_facts": {
		WellKnown: "dog_facts",
		Title: models.Title{
			Text: "{{}}",
			Vars: []string{"body"},
		},
		Resource: getURL("https://dogapi.dog/api/v2/facts"),
		Request: models.Request{
			Method: "GET",
		},
		Response: "json",
		Query: map[string]string{
			"body": ".data[0].attributes.body",
		},
		Rate: 5,
	},
	"dolar_oficial": {
		WellKnown: "dolar_oficial",
		Title: models.Title{
			Text: "Dolar at {{}}",
			Vars: []string{"compra"},
		},
		Resource: getURL("https://dolarapi.com/v1/dolares/oficial"),
		Request: models.Request{
			Method: "GET",
		},
		Response: "json",
		Query: map[string]string{
			"compra": ".compra",
			"venta":  ".venta",
		},
		Rate: 120,
	},
	"exchange_rate": {
		WellKnown: "exchange_rate",
		Title: models.Title{
			Text: "{{}} at {{}}",
			Vars: []string{"code", "value"},
		},
		Params: map[string]string{
			"$base":       "USD",
			"$currency":   "ARS",
			"$credential": "",
		},
		Resource: getURL("https://api.currencyapi.com/v3/latest"),
		Request: models.Request{
			Method: "GET",
			Queries: map[string]string{
				"base_currency": "$base",
				"currencies":    "$currency",
			},
			Headers: map[string]string{
				"Accept": "application/json",
				"apikey": "$credential",
			},
		},
		Response: "json",
		Query: map[string]string{
			"code":    ".data[].code",
			"value":   ".data[].value",
			"updated": ".meta.last_updated_at",
		},
		Rate: 3600,
	},
	"riesgo_pais": {
		WellKnown: "riesgo_pais",
		Title: models.Title{
			Text: "Argentine's country risk: {{}}",
			Vars: []string{"valor"},
		},
		Resource: getURL("https://api.argentinadatos.com/v1/finanzas/indices/riesgo-pais/ultimo"),
		Request: models.Request{
			Method: "GET",
			Headers: map[string]string{
				"Accept": "application/json",
			},
		},
		Response: "json",
		Query: map[string]string{
			"valor": ".valor",
			"fecha": ".fecha",
		},
		Rate: 3600,
	},
	"crypto_price": {
		WellKnown: "crypto_price",
		Title: models.Title{
			Text: "Let's go gambling! {{}} at {{}}",
			Vars: []string{"coin", "price"},
		},
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
		Query: map[string]string{
			"coin":   "keys_unsorted[0]",
			"price":  ".[] | to_entries[] | select(.key | endswith(\"_24h_change\") | not) | .value",
			"change": ".[] | to_entries[] | select(.key | endswith(\"_24h_change\")) | .value",
		},
		Rate: 60,
	},
	"air_quality": {
		WellKnown: "air_quality",
		Title: models.Title{
			Text: "Today's air is {{}}",
			Vars: []string{"pm25"},
		},
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
		Query: map[string]string{
			"aqi":  ".current.us_aqi",
			"pm25": ".current.pm2_5",
			"time": ".current.time",
		},
		Rate: 900,
	},
	"github_repo": {
		WellKnown: "github_repo",
		Title: models.Title{
			Text: "The repo {{}} has {{}} stars and {{}} open issues",
			Vars: []string{"name", "stars", "issues"},
		},
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
		Query: map[string]string{
			"name":        ".items[0].full_name",
			"stars":       ".items[0].stargazers_count",
			"forks":       ".items[0].forks_count",
			"issues":      ".items[0].open_issues_count",
			"description": ".items[0].description",
		},
		Rate: 300,
	},
}
