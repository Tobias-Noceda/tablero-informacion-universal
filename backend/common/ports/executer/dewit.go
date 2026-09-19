package executer

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"

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

	res, cancel, err := e.request(postit.Resource, postit.Request.Method, postit.Request.Queries, postit.Request.Headers)
	if err != nil {
		return nil, err
	}

	// The request context must outlive the body read below, so the parser
	// sees the whole payload rather than a "context canceled" mid-stream.
	defer cancel()
	defer res.Body.Close()
	body := io.LimitReader(res.Body, safehttp.MAX_PAYLOAD_SIZE)

	return parser.parse(postit, body)
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

func (*DewIt) request(resource *url.URL, method string, queries, headers map[string]string) (*http.Response, context.CancelFunc, error) {
	ctx, cancel := context.WithTimeout(context.Background(), safehttp.REQUEST_TIMEOUT)

	target := *resource

	q := target.Query()
	for k, v := range queries {
		q.Set(k, v)
	}
	target.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, method, target.String(), nil)
	if err != nil {
		cancel()
		return nil, nil, err
	}

	for k, v := range headers {
		req.Header.Add(k, v)
	}

	res, err := safehttp.Client().Do(req)
	if err != nil {
		cancel()
		return nil, nil, err
	}

	if res.StatusCode != http.StatusOK {
		res.Body.Close()
		cancel()
		return nil, nil, fmt.Errorf("Resource returned %d", res.StatusCode)
	}

	return res, cancel, nil
}
