package executer

import (
	"io"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/Secreto31126/tesis/common/models"
)

type HtmlDewIt struct{}

func (this *HtmlDewIt) parse(postit *models.PostIts, body io.Reader) (any, error) {
	data, err := this.decode(body)
	if err != nil {
		return nil, err
	}

	return this.query(data, postit.Query)
}

func (e *HtmlDewIt) decode(body io.Reader) (*goquery.Document, error) {
	doc, err := goquery.NewDocumentFromReader(body)
	if err != nil {
		return nil, err
	}

	return doc, nil
}

func (e *HtmlDewIt) query(doc *goquery.Document, queries map[string]string) (any, error) {
	out := make(map[string]any)

	for key, query := range queries {
		out[key] = strings.TrimSpace(doc.FindMatcher(goquery.Single(query)).Text())
	}

	return out, nil
}
