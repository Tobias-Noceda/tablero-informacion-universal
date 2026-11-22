package executer

import (
	"fmt"
	"net/url"
	"regexp"
	"strings"

	"github.com/Secreto31126/tesis/common/infrastructure"
)

var (
	addressPlaceholder = regexp.MustCompile(`\$[A-Za-z][A-Za-z0-9_]*`)
	hostName           = regexp.MustCompile(`^[A-Za-z0-9.-]+(:[0-9]{1,5})?$`)
)

// fillResource puts the card's values in its address, where a card type
// names them ("https://api.github.com/repos/$repo"). A path value is escaped
// segment by segment, so it may hold "owner/name" but never reach the query
// string or climb out with "..". A host value must be a plain host name;
// where it leads is still the SSRF-safe client's call. A placeholder with no
// value stays as written, like in headers and query parameters.
func fillResource(resource *url.URL, params map[string]string) (*url.URL, error) {
	filled := *resource

	if strings.Contains(filled.Host, "$") {
		host, _ := fill(filled.Host, params, func(value string) (string, error) { return value, nil })
		if !hostName.MatchString(host) {
			return nil, refused("the address's host is not a host name")
		}
		filled.Host = host
	}

	if escaped := filled.EscapedPath(); strings.Contains(escaped, "$") {
		path, err := fill(escaped, params, escapePathValue)
		if err != nil {
			return nil, err
		}
		unescaped, err := url.PathUnescape(path)
		if err != nil {
			return nil, refused("the address path is not valid")
		}
		filled.Path, filled.RawPath = unescaped, path
	}

	return &filled, nil
}

// escapePathValue escapes a value segment by segment, so "owner/name" stays
// two segments, and refuses one that climbs out of the path.
func escapePathValue(value string) (string, error) {
	segments := strings.Split(value, "/")
	for i, segment := range segments {
		if segment == "." || segment == ".." {
			return "", refused("a value in the address path climbs out of it")
		}
		segments[i] = url.PathEscape(segment)
	}
	return strings.Join(segments, "/"), nil
}

// fill replaces each placeholder that has a value with what encode makes of
// it, and answers the first error encode gives.
func fill(text string, params map[string]string, encode func(string) (string, error)) (string, error) {
	var problem error
	filled := addressPlaceholder.ReplaceAllStringFunc(text, func(name string) string {
		value, ok := params[name]
		if !ok {
			return name
		}
		encoded, err := encode(value)
		if err != nil && problem == nil {
			problem = err
		}
		return encoded
	})
	return filled, problem
}

func refused(reason string) error {
	return fmt.Errorf("%w: %s", infrastructure.ErrSourceUnreachable, reason)
}
