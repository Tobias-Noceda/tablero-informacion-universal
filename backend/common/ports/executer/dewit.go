package executer

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"time"

	"github.com/Secreto31126/tesis/common/models"
	"github.com/itchyny/gojq"
)

const (
	REQUEST_TIMEOUT = 10 * time.Second
	QUERY_TIMEOUT   = 250 * time.Millisecond
)

type DewIt struct{}

func New() *DewIt {
	return &DewIt{}
}

func (e *DewIt) Execute(postit *models.PostIts) (any, error) {
	if postit.Resource == nil {
		return postit.Params, nil
	}

	if postit.Request.Queries != nil {
		err := e.populate(postit.Params, postit.Request.Queries)
		if err != nil {
			return nil, err
		}
	}

	if postit.Request.Headers != nil {
		err := e.populate(postit.Params, postit.Request.Headers)
		if err != nil {
			return nil, err
		}
	}

	res, err := e.request(postit.Resource, postit.Request.Method, postit.Request.Queries, postit.Request.Headers)
	if err != nil {
		return nil, err
	}

	defer res.Body.Close()

	data, err := e.parse(res.Body)
	if err != nil {
		return nil, err
	}

	return e.query(data, postit.Query)
}

func (*DewIt) populate(input, out map[string]string) error {
	for key, name := range out {
		if value, ok := input[name]; ok {
			out[key] = value
		}
	}

	return nil
}

func (*DewIt) request(url *url.URL, method string, queries, headers map[string]string) (*http.Response, error) {
	ctx, cancel := context.WithTimeout(context.Background(), REQUEST_TIMEOUT)
	defer cancel()

	q := url.Query()
	for k, v := range queries {
		q.Add(k, v)
	}
	url.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, method, url.String(), nil)
	if err != nil {
		return nil, err
	}

	for k, v := range headers {
		req.Header.Add(k, v)
	}

	res, err := (&http.Client{}).Do(req)
	if err != nil {
		return nil, err
	}

	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Resource didn't return 200")
	}

	return res, nil
}

func (e *DewIt) parse(body io.ReadCloser) (any, error) {
	var data any

	if err := json.NewDecoder(body).Decode(&data); err != nil {
		return nil, err
	}

	// TODO: delete this
	switch v := data.(type) {
	case map[string]any:
		log.Println("It's an object!", v)
	case []any:
		log.Println("It's an array!", v)
	default:
		log.Println("It's a primitive type!", v)
	}

	return data, nil
}

func (e *DewIt) query(data any, query string) (any, error) {
	ctx, cancel := context.WithTimeout(context.Background(), QUERY_TIMEOUT)
	defer cancel()

	cmd, err := gojq.Parse(query)
	if err != nil {
		return nil, err
	}

	res, ok := cmd.RunWithContext(ctx, data).Next()
	if !ok {
		return nil, fmt.Errorf("Query matched no results")
	}

	if err, ok := res.(error); ok {
		return nil, err
	}

	return res, nil
}
