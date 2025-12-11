package templates

import (
	"encoding/json"
	"html/template"
)

// jsonMarshal converts a value to JSON string for use in templates
func jsonMarshal(v any) template.JS {
	b, err := json.Marshal(v)
	if err != nil {
		return template.JS("null")
	}
	return template.JS(b)
}
