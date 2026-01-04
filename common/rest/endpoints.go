package rest

import (
	"fmt"
	"net/url"
	"strings"
)

var (
	EndpointHealth         = NewEndpoint("GET", "/health")
	EndpointGetModList     = NewEndpoint("GET", "/mods")
	EndpointGetModDetails  = NewEndpoint("GET", "/mods/{modID}")
	EndpointGetModVersions = NewEndpoint("GET", "/mods/{modID}/versions")
	EndpointGetModVersion  = NewEndpoint("GET", "/mods/{modID}/versions/{versionID}")
)

func NewEndpoint(method, path string) *Endpoint {
	return &Endpoint{
		Method: method,
		Route:  path,
	}
}

type Endpoint struct {
	Method string
	Route  string
}

type CompiledEndpoint struct {
	Endpoint *Endpoint

	URL string
}

func (e *Endpoint) Compile(values url.Values, params ...any) *CompiledEndpoint {
	path := e.Route
	for _, param := range params {
		start := strings.Index(path, "{")
		end := strings.Index(path, "}")
		if start == -1 || end == -1 || end < start {
			break
		}
		paramValue := fmt.Sprint(param)
		path = path[:start] + url.PathEscape(paramValue) + path[end+1:]
	}
	query := values.Encode()
	if query != "" {
		query = "?" + query
	}
	return &CompiledEndpoint{
		Endpoint: e,
		URL:      path + query,
	}
}
