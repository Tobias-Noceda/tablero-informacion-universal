package models

import (
	"bytes"
	"encoding/json"
	"errors"
)

// MaxRequestBody is how large a card's JSON request body may be.
const MaxRequestBody = 16 << 10

// ErrInvalidRequestBody reports a request body that is not a JSON document.
var ErrInvalidRequestBody = errors.New("request body must be JSON")

// BodyStrings lists every string value in a JSON body, at any depth (object
// keys excluded): where a card may write $NAME for a credential.
func BodyStrings(body string) ([]string, error) {
	document, err := decodeBody(body)
	if err != nil {
		return nil, err
	}
	var found []string
	walkStrings(document, func(value string) string {
		found = append(found, value)
		return value
	})
	return found, nil
}

// ReplaceBodyStrings returns body with every string value that is a key of
// values replaced by its value. It works on the decoded document, so a
// replacement holding quotes or backslashes stays one JSON string.
func ReplaceBodyStrings(body string, values map[string]string) (string, error) {
	document, err := decodeBody(body)
	if err != nil {
		return "", err
	}
	document = walkStrings(document, func(value string) string {
		if replacement, ok := values[value]; ok {
			return replacement
		}
		return value
	})
	encoded, err := json.Marshal(document)
	if err != nil {
		return "", err
	}
	return string(encoded), nil
}

func decodeBody(body string) (any, error) {
	decoder := json.NewDecoder(bytes.NewReader([]byte(body)))
	// Numbers stay as written: a float64 would round large integers.
	decoder.UseNumber()
	var document any
	if err := decoder.Decode(&document); err != nil {
		return nil, ErrInvalidRequestBody
	}
	if decoder.More() {
		return nil, ErrInvalidRequestBody
	}
	return document, nil
}

func walkStrings(node any, visit func(string) string) any {
	switch value := node.(type) {
	case string:
		return visit(value)
	case map[string]any:
		for key, child := range value {
			value[key] = walkStrings(child, visit)
		}
	case []any:
		for i, child := range value {
			value[i] = walkStrings(child, visit)
		}
	}
	return node
}
