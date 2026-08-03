package executer

import (
	"context"
	"encoding/json"
	"fmt"
	"io"

	"github.com/Secreto31126/tesis/common/models"
	"github.com/itchyny/gojq"
)

type JsonDewIt struct{}

func (this *JsonDewIt) parse(postit *models.PostIts, body io.Reader) (any, error) {
	data, err := this.decode(body)
	if err != nil {
		return nil, err
	}

	return this.query(data, postit.Query)
}

func (this *JsonDewIt) decode(body io.Reader) (any, error) {
	var data any
	if err := json.NewDecoder(body).Decode(&data); err != nil {
		return nil, err
	}

	return data, nil
}

func (this *JsonDewIt) query(data any, queries map[string]string) (any, error) {
	ctx, cancel := context.WithTimeout(context.Background(), QUERY_TIMEOUT)
	defer cancel()

	out := make(map[string]any)

	for key, query := range queries {
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

		out[key] = res
	}

	return out, nil
}
