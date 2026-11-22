package executer

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/Secreto31126/tesis/common/infrastructure"
	"github.com/Secreto31126/tesis/common/models"
	"github.com/Secreto31126/tesis/common/ports/safehttp"
)

type DewIt struct{}

func New() *DewIt {
	return &DewIt{}
}

type parser interface {
	parse(*models.PostIts, io.Reader) (any, error)
}

func (e *DewIt) Execute(postit *models.PostIts) (any, error) {
	if postit.Resource == nil {
		return postit.Params, nil
	}

	parser, err := e.getParser(postit)
	if err != nil {
		return nil, err
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

	// A POST's JSON body takes the same values, string by string.
	body := ""
	if postit.Request.Method == http.MethodPost && postit.Request.Body != "" {
		body, err = models.ReplaceBodyStrings(postit.Request.Body, postit.Params)
		if err != nil {
			return nil, err
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), safehttp.REQUEST_TIMEOUT)
	defer cancel()

	resource, err := fillResource(postit.Resource, postit.Params)
	if err != nil {
		return nil, err
	}

	res, err := e.request(ctx, resource, postit.Request.Method, postit.Request.Queries, postit.Request.Headers, body)
	if err != nil {
		return nil, err
	}

	defer res.Body.Close()

	return parser.parse(postit, io.LimitReader(res.Body, safehttp.MAX_PAYLOAD_SIZE))
}

func (*DewIt) getParser(postit *models.PostIts) (parser, error) {
	switch postit.Response {
	case "json":
		return &JsonDewIt{}, nil
	case "html":
		return &HtmlDewIt{}, nil
	default:
		return nil, fmt.Errorf("Unsupported content type, no parser implemented")
	}
}

func (*DewIt) populate(input, out map[string]string) error {
	for key, name := range out {
		if value, ok := input[name]; ok {
			out[key] = value
		}
	}

	return nil
}

func (*DewIt) request(ctx context.Context, resource *url.URL, method string, queries, headers map[string]string, body string) (*http.Response, error) {
	target := *resource

	q := target.Query()
	for k, v := range queries {
		q.Set(k, v)
	}
	target.RawQuery = q.Encode()

	var payload io.Reader
	if body != "" {
		payload = strings.NewReader(body)
	}

	req, err := http.NewRequestWithContext(ctx, method, target.String(), payload)
	if err != nil {
		return nil, err
	}

	for k, v := range headers {
		req.Header.Add(k, v)
	}
	if body != "" && req.Header.Get("Content-Type") == "" {
		req.Header.Set("Content-Type", "application/json")
	}

	res, err := safehttp.Client().Do(req)
	if err != nil {
		return nil, transportFailure(err)
	}

	if res.StatusCode != http.StatusOK {
		res.Body.Close()
		return nil, &infrastructure.SourceStatusError{Status: res.StatusCode}
	}

	return res, nil
}
