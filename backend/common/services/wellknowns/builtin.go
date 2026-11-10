package wellknowns

import "github.com/Secreto31126/tesis/common/models"

// Builtin is what the platform seeds on boot. A key already stored, edited or
// deleted by an admin, is never overwritten: this is a starting point, the
// well_knowns collection is the source of truth.
func Builtin() []models.WellKnown {
	defs := []models.WellKnown{
		{
			Key:    "static_card",
			NameEn: "Note", NameEs: "Nota",
			TitleEn: "{{text}}",
			TitleEs: "{{text}}",
			Parameters: []models.WellKnownParameter{
				{Key: "text", LabelEn: "Content", LabelEs: "Contenido", Type: models.ParameterString, PlaceholderEn: new("Enter text"), PlaceholderEs: new("Ingrese texto"), Default: new("")},
			},
			Outputs: []models.WellKnownOutput{
				{Key: "text", LabelEn: "Text", LabelEs: "Texto", ExampleEn: "Hello!", ExampleEs: "Hola!"},
			},
		},
		{
			Key:    "temperature",
			NameEn: "Temperature", NameEs: "Temperatura",
			TitleEn:  "Today's temps are between {{min}}°C and {{max}}°C",
			TitleEs:  "Hoy la temperatura va de {{min}}°C a {{max}}°C",
			Resource: "https://api.open-meteo.com/v1/forecast",
			Request: models.Request{
				Method: "GET",
				Queries: map[string]string{
					"hourly":     "temperature_2m",
					"latitude":   "$latitude",
					"longitude":  "$longitude",
					"start_date": "$start_date",
					"end_date":   "$end_date",
				},
				Headers: map[string]string{"Accept": "application/json"},
			},
			Response: "json",
			Query: map[string]string{
				"min": ".hourly.temperature_2m | min",
				"max": ".hourly.temperature_2m | max",
			},
			Rate: 30,
			Parameters: []models.WellKnownParameter{
				{Key: "$latitude", LabelEn: "Latitude", LabelEs: "Latitud", Type: models.ParameterNumber, Default: new("-34.6131")},
				{Key: "$longitude", LabelEn: "Longitude", LabelEs: "Longitud", Type: models.ParameterNumber, Default: new("-58.3772")},
				{Key: "$start_date", LabelEn: "Start date", LabelEs: "Fecha de inicio", Type: models.ParameterString, PlaceholderEn: new("YYYY-MM-DD"), PlaceholderEs: new("AAAA-MM-DD")},
				{Key: "$end_date", LabelEn: "End date", LabelEs: "Fecha de fin", Type: models.ParameterString, PlaceholderEn: new("YYYY-MM-DD"), PlaceholderEs: new("AAAA-MM-DD")},
			},
			Outputs: []models.WellKnownOutput{
				{Key: "min", LabelEn: "Minimum °C", LabelEs: "Mínimo °C", ExampleEn: 12.4, ExampleEs: 12.4},
				{Key: "max", LabelEn: "Maximum °C", LabelEs: "Máximo °C", ExampleEn: 21.9, ExampleEs: 21.9},
			},
		},
		{
			Key:    "events_search",
			NameEn: "Events Search", NameEs: "Búsqueda de Eventos",
			TitleEn:  "{{sales}}: {{name}}",
			TitleEs:  "{{sales}}: {{name}}",
			Resource: "https://app.ticketmaster.com/discovery/v2/events.json",
			Request: models.Request{
				Method: "GET",
				Queries: map[string]string{
					"keyword": "$keyword",
					"size":    "1",
					"apikey":  "$credential",
				},
				Headers: map[string]string{"Accept": "application/json"},
			},
			Response: "json",
			Query: map[string]string{
				"name":  "._embedded.events[0].name",
				"sales": "._embedded.events[0].sales.public.startDateTime",
				"image": "._embedded.events[0].images[0].url",
				"date":  "._embedded.events[0].dates.start.localDate",
				"venue": "._embedded.events[0]._embedded.venues[0].name",
				"city":  "._embedded.events[0]._embedded.venues[0].city.name",
				"url":   "._embedded.events[0].url",
			},
			Rate: 120,
			Parameters: []models.WellKnownParameter{
				{Key: "$keyword", LabelEn: "Keyword", LabelEs: "Palabra clave", Type: models.ParameterString, PlaceholderEn: new("Enter keyword"), PlaceholderEs: new("Ingrese palabra clave")},
				{Key: "$credential", LabelEn: "Credential", LabelEs: "Credencial", Type: models.ParameterSecret},
			},
			Outputs: []models.WellKnownOutput{
				{Key: "name", LabelEn: "Event name", LabelEs: "Nombre del evento", ExampleEn: "Coldplay - Music of the Spheres", ExampleEs: "Coldplay - Música de las Esferas"},
				{Key: "sales", LabelEn: "Sales start", LabelEs: "Inicio de ventas", ExampleEn: "2026-03-01T13:00:00Z", ExampleEs: "2026-03-01T13:00:00Z"},
				{Key: "image", LabelEn: "Image URL", LabelEs: "URL de la imagen", ExampleEn: "https://s1.ticketm.net/img.jpg", ExampleEs: "https://s1.ticketm.net/img.jpg"},
				{Key: "date", LabelEn: "Event date", LabelEs: "Fecha del evento", ExampleEn: "2026-08-01", ExampleEs: "2026-08-01"},
				{Key: "venue", LabelEn: "Venue name", LabelEs: "Nombre del lugar", ExampleEn: "Movistar Arena", ExampleEs: "Movistar Arena"},
				{Key: "city", LabelEn: "Event's city", LabelEs: "Ciudad del evento", ExampleEn: "Buenos Aires", ExampleEs: "Buenos Aires"},
				{Key: "url", LabelEn: "Event's url", LabelEs: "URL del evento", ExampleEn: "https://www.ticketmaster.com", ExampleEs: "https://www.ticketmaster.com"},
			},
		},
		{
			Key:    "dog_facts",
			NameEn: "Dog Facts", NameEs: "Hechos sobre Perros",
			TitleEn:  "{{body}}",
			TitleEs:  "{{body}}",
			Resource: "https://dogapi.dog/api/v2/facts",
			Request:  models.Request{Method: "GET"},
			Response: "json",
			Query:    map[string]string{"body": ".data[0].attributes.body"},
			Rate:     5,
			Outputs: []models.WellKnownOutput{
				{Key: "body", LabelEn: "Fact", LabelEs: "Hecho", ExampleEn: "Dogs have three eyelids.", ExampleEs: "Los perros tienen tres párpados."},
			},
		},
		{
			Key:    "dolar_oficial",
			NameEn: "Official Dollar", NameEs: "Dólar Oficial",
			TitleEn:  "Dolar at {{compra}}",
			TitleEs:  "Dólar a {{compra}}",
			Resource: "https://dolarapi.com/v1/dolares/oficial",
			Request:  models.Request{Method: "GET"},
			Response: "json",
			Query: map[string]string{
				"compra": ".compra",
				"venta":  ".venta",
			},
			Rate: 120,
			Outputs: []models.WellKnownOutput{
				{Key: "compra", LabelEn: "Buy", LabelEs: "Comprar", ExampleEn: 1420, ExampleEs: 1420},
				{Key: "venta", LabelEn: "Sell", LabelEs: "Vender", ExampleEn: 1470, ExampleEs: 1470},
			},
		},
		{
			Key:    "riesgo_pais",
			NameEn: "Country Risk", NameEs: "Riesgo País",
			TitleEn:  "Argentine's country risk: {{valor}}",
			TitleEs:  "Riesgo país: {{valor}}",
			Resource: "https://api.argentinadatos.com/v1/finanzas/indices/riesgo-pais/ultimo",
			Request: models.Request{
				Method:  "GET",
				Headers: map[string]string{"Accept": "application/json"},
			},
			Response: "json",
			Query: map[string]string{
				"valor": ".valor",
				"fecha": ".fecha",
			},
			Rate: 3600,
			Outputs: []models.WellKnownOutput{
				{Key: "valor", LabelEn: "Value", LabelEs: "Valor", ExampleEn: 612, ExampleEs: 612},
				{Key: "fecha", LabelEn: "Date", LabelEs: "Fecha", ExampleEn: "2026-10-03", ExampleEs: "2026-10-03"},
			},
		},
		{
			Key:    "crypto_price",
			NameEn: "Crypto Price", NameEs: "Precio Cripto",
			TitleEn:  "Let's go gambling! {{coin}} at {{price}}",
			TitleEs:  "¡A apostar! {{coin}} a {{price}}",
			Resource: "https://api.coingecko.com/api/v3/simple/price",
			Request: models.Request{
				Method: "GET",
				Queries: map[string]string{
					"ids":                 "$coin",
					"vs_currencies":       "$currency",
					"include_24hr_change": "true",
				},
				Headers: map[string]string{"Accept": "application/json"},
			},
			Response: "json",
			Query: map[string]string{
				"coin":   "keys_unsorted[0]",
				"price":  ".[] | to_entries[] | select(.key | endswith(\"_24h_change\") | not) | .value",
				"change": ".[] | to_entries[] | select(.key | endswith(\"_24h_change\")) | .value",
			},
			Rate: 60,
			Parameters: []models.WellKnownParameter{
				{Key: "$coin", LabelEn: "Coin", LabelEs: "Moneda", Type: models.ParameterString, PlaceholderEn: new("bitcoin"), PlaceholderEs: new("bitcoin"), Default: new("bitcoin")},
				{Key: "$currency", LabelEn: "Currency", LabelEs: "Moneda", Type: models.ParameterString, PlaceholderEn: new("usd"), PlaceholderEs: new("usd"), Default: new("usd")},
			},
			Outputs: []models.WellKnownOutput{
				{Key: "coin", LabelEn: "Coin", LabelEs: "Moneda", ExampleEn: "bitcoin", ExampleEs: "bitcoin"},
				{Key: "price", LabelEn: "Price", LabelEs: "Precio", ExampleEn: 62150, ExampleEs: 62150},
				{Key: "change", LabelEn: "24h change", LabelEs: "Cambio 24h", ExampleEn: -1.23, ExampleEs: -1.23},
			},
		},
		{
			Key:    "air_quality",
			NameEn: "Air Quality", NameEs: "Calidad del Aire",
			TitleEn:  "Today's air is {{pm25}}",
			TitleEs:  "Calidad del aire hoy: {{pm25}}",
			Resource: "https://air-quality-api.open-meteo.com/v1/air-quality",
			Request: models.Request{
				Method: "GET",
				Queries: map[string]string{
					"latitude":  "$latitude",
					"longitude": "$longitude",
					"current":   "us_aqi,pm2_5",
				},
				Headers: map[string]string{"Accept": "application/json"},
			},
			Response: "json",
			Query: map[string]string{
				"aqi":  ".current.us_aqi",
				"pm25": ".current.pm2_5",
				"time": ".current.time",
			},
			Rate: 900,
			Parameters: []models.WellKnownParameter{
				{Key: "$latitude", LabelEn: "Latitude", LabelEs: "Latitud", Type: models.ParameterNumber, Default: new("-34.6131")},
				{Key: "$longitude", LabelEn: "Longitude", LabelEs: "Longitud", Type: models.ParameterNumber, Default: new("-58.3772")},
			},
			Outputs: []models.WellKnownOutput{
				{Key: "aqi", LabelEn: "US AQI", LabelEs: "AQI de EE.UU.", ExampleEn: 42, ExampleEs: 42},
				{Key: "pm25", LabelEn: "PM2.5", LabelEs: "PM2.5", ExampleEn: 8.3, ExampleEs: 8.3},
				{Key: "time", LabelEn: "Measured at", LabelEs: "Medido a las", ExampleEn: "2026-10-04T12:00", ExampleEs: "2026-10-04T12:00"},
			},
		},
		{
			Key:    "github_repo",
			NameEn: "GitHub Repository", NameEs: "Repositorio GitHub",
			TitleEn:  "The repo {{name}} has {{stars}} stars and {{issues}} open issues",
			TitleEs:  "El repo {{name}} tiene {{stars}} estrellas y {{issues}} issues abiertos",
			Resource: "https://api.github.com/search/repositories",
			Request: models.Request{
				Method: "GET",
				Queries: map[string]string{
					"q":        "$query",
					"sort":     "stars",
					"order":    "desc",
					"per_page": "1",
				},
				Headers: map[string]string{"Accept": "application/vnd.github+json"},
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
			Parameters: []models.WellKnownParameter{
				{Key: "$query", LabelEn: "Repository", LabelEs: "Repositorio", Type: models.ParameterString, PlaceholderEn: new("itchyny/gojq"), PlaceholderEs: new("itchyny/gojq")},
			},
			Outputs: []models.WellKnownOutput{
				{Key: "name", LabelEn: "Repository", LabelEs: "Repositorio", ExampleEn: "itchyny/gojq", ExampleEs: "itchyny/gojq"},
				{Key: "stars", LabelEn: "Stars", LabelEs: "Estrellas", ExampleEn: 3412, ExampleEs: 3412},
				{Key: "forks", LabelEn: "Forks", LabelEs: "Bifurcaciones", ExampleEn: 245, ExampleEs: 245},
				{Key: "issues", LabelEn: "Open issues", LabelEs: "Problemas abiertos", ExampleEn: 12, ExampleEs: 12},
				{Key: "description", LabelEn: "Description", LabelEs: "Descripción", ExampleEn: "Pure Go implementation of jq", ExampleEs: "Pure Go implementation of jq"},
			},
		},
		{
			Key:    "exchange_rate",
			NameEn: "Exchange Rate", NameEs: "Cotización",
			TitleEn:  "{{code}} at {{value}}",
			TitleEs:  "{{code}} a {{value}}",
			Resource: "https://api.currencyapi.com/v3/latest",
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
			Parameters: []models.WellKnownParameter{
				{Key: "$base", LabelEn: "Base currency", LabelEs: "Moneda base", Type: models.ParameterString, PlaceholderEn: new("USD"), PlaceholderEs: new("USD"), Default: new("USD")},
				{Key: "$currency", LabelEn: "Quote currency", LabelEs: "Moneda de cotización", Type: models.ParameterString, PlaceholderEn: new("ARS"), PlaceholderEs: new("ARS"), Default: new("ARS")},
				{Key: "$credential", LabelEn: "Credential", LabelEs: "Credencial", Type: models.ParameterSecret},
			},
			Outputs: []models.WellKnownOutput{
				{Key: "code", LabelEn: "Currency code", LabelEs: "Código de moneda", ExampleEn: "ARS", ExampleEs: "ARS"},
				{Key: "value", LabelEn: "Rate", LabelEs: "Tasa", ExampleEn: 1452.5, ExampleEs: 1452.5},
				{Key: "updated", LabelEn: "Last update", LabelEs: "Última actualización", ExampleEn: "2026-10-04T00:00:00Z", ExampleEs: "2026-10-04T00:00:00Z"},
			},
		},
		{
			Key:    "gmail_inbox",
			NameEn: "Gmail Inbox", NameEs: "Bandeja de Gmail",
			TitleEn:  "You have {{unread}} new emails",
			TitleEs:  "Tenés {{unread}} mails nuevos",
			Resource: "https://gmail.googleapis.com/gmail/v1/users/me/labels/INBOX",
			Request: models.Request{
				Method: "GET",
				Headers: map[string]string{
					"Accept":        "application/json",
					"Authorization": "$credential",
				},
			},
			Response: "json",
			Query: map[string]string{
				"unread": ".messagesUnread",
				"total":  ".messagesTotal",
			},
			Rate: 60,
			Parameters: []models.WellKnownParameter{
				{Key: "$credential", LabelEn: "Google account", LabelEs: "Cuenta de Google", Type: models.ParameterSecret},
			},
			Outputs: []models.WellKnownOutput{
				{Key: "unread", LabelEn: "Unread", LabelEs: "Sin leer", ExampleEn: 7, ExampleEs: 7},
				{Key: "total", LabelEn: "Total", LabelEs: "Total", ExampleEn: 1532, ExampleEs: 1532},
			},
		},
		{
			Key:    "google_calendar",
			NameEn: "Google Calendar", NameEs: "Calendario de Google",
			TitleEn:  "{{summary}} starts at {{start}}",
			TitleEs:  "{{summary}} empieza a las {{start}}",
			Resource: "https://www.googleapis.com/calendar/v3/calendars/primary/events",
			Request: models.Request{
				Method: "GET",
				Queries: map[string]string{
					"singleEvents": "true",
					"orderBy":      "startTime",
					"maxResults":   "50",
					"timeMin":      "$time_min",
				},
				Headers: map[string]string{
					"Accept":        "application/json",
					"Authorization": "$credential",
				},
			},
			Response: "json",
			Query: map[string]string{
				"summary": "[.items[] | select((.start.dateTime // .start.date) >= (now | todate))] | first | .summary",
				"start":   "[.items[] | select((.start.dateTime // .start.date) >= (now | todate))] | first | .start.dateTime // .start.date",
			},
			Rate: 300,
			Parameters: []models.WellKnownParameter{
				{Key: "$time_min", LabelEn: "From", LabelEs: "De", Type: models.ParameterString, PlaceholderEn: new("2026-01-01T00:00:00Z"), PlaceholderEs: new("2026-01-01T00:00:00Z")},
				{Key: "$credential", LabelEn: "Google account", LabelEs: "Cuenta de Google", Type: models.ParameterSecret},
			},
			Outputs: []models.WellKnownOutput{
				{Key: "summary", LabelEn: "Next event", LabelEs: "Próximo evento", ExampleEn: "Thesis meeting", ExampleEs: "Reunión de tesis"},
				{Key: "start", LabelEn: "Starts at", LabelEs: "Comienza a las", ExampleEn: "2026-10-05T15:00:00-03:00", ExampleEs: "2026-10-05T15:00:00-03:00"},
			},
		},
		{
			Key:    "nasa_apod",
			NameEn: "NASA Picture of the Day", NameEs: "Foto del Día de la NASA",
			TitleEn:  "Today's NASA's picture: {{title}}: {{explanation}}",
			TitleEs:  "La foto de hoy de la NASA: {{title}}: {{explanation}}",
			Resource: "https://api.nasa.gov/planetary/apod",
			Request: models.Request{
				Method:  "GET",
				Queries: map[string]string{"api_key": "$api_key"},
				Headers: map[string]string{"Accept": "application/json"},
			},
			Response: "json",
			Query: map[string]string{
				"title":       ".title",
				"image":       ".url",
				"explanation": ".explanation",
				"date":        ".date",
			},
			Rate: 3600,
			// The API key is the platform's own: nothing to ask the user for.
			SystemSecrets: map[string]models.SystemSecretName{"$api_key": models.SystemNasaApiKey},
			Outputs: []models.WellKnownOutput{
				{Key: "title", LabelEn: "Title", LabelEs: "Título", ExampleEn: "The Horsehead Nebula", ExampleEs: "La Nebulosa del Caballo"},
				{Key: "image", LabelEn: "Image URL", LabelEs: "URL de la imagen", ExampleEn: "https://apod.nasa.gov/apod/image/horsehead.jpg", ExampleEs: "https://apod.nasa.gov/apod/image/horsehead.jpg"},
				{Key: "explanation", LabelEn: "Explanation", LabelEs: "Explicación", ExampleEn: "A dark cloud of dust...", ExampleEs: "Una nube oscura de polvo..."},
				{Key: "date", LabelEn: "Date", LabelEs: "Fecha", ExampleEn: "2026-10-04", ExampleEs: "2026-10-04"},
			},
		},
	}

	for i := range defs {
		defs[i].Builtin = true
		defs[i].Order = (i + 1) * 10
	}
	return defs
}
