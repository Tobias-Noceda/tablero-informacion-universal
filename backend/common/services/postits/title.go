package postits

import (
	"regexp"

	"github.com/Secreto31126/tesis/common/models"
)

// titleVar matches a plain mustache variable, "{{name}}" or "{{ name }}".
var titleVar = regexp.MustCompile(`\{\{\s*([A-Za-z_]\w*)\s*\}\}`)

// titleVars reports whether text uses at least one of the post-it's outputs,
// which is what tells the frontend to fetch the data to render the title.
func titleVars(text string, outputs map[string]string) bool {
	for _, match := range titleVar.FindAllStringSubmatch(text, -1) {
		if _, ok := outputs[match[1]]; ok {
			return true
		}
	}
	return false
}

// outputsOf is what a post-it answers with: its query keys, or its params
// when it has no resource to fetch (the executer returns them verbatim).
func outputsOf(postit *models.PostIts) map[string]string {
	if postit.Resource == nil {
		return postit.Params
	}
	return postit.Query
}
