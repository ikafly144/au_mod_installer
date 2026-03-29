package rest

import (
	"fmt"
	"net/url"
	"strings"
)

var (
	EndpointHealth              = NewEndpoint("GET", "/health")
	EndpointGetModList          = NewEndpoint("GET", "/mods")
	EndpointGetModDetail        = NewEndpoint("GET", "/mod/:mod_id")
	EndpointGetModThumbnail     = NewEndpoint("GET", "/mod/:mod_id/thumbnail")
	EndpointGetModVersionList   = NewEndpoint("GET", "/mod/:mod_id/versions")
	EndpointGetModVersionDetail = NewEndpoint("GET", "/mod/:mod_id/version/:version_id")
	EndpointShareGame           = NewEndpoint("POST", "/share_game")
	EndpointDeleteShareGame     = NewEndpoint("DELETE", "/share_game")
	EndpointJoinGame            = NewEndpoint("GET", "/join_game")
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
		start := strings.Index(path, ":")
		if start == -1 {
			break
		}
		end := strings.Index(path[start:], "/") + start
		if end == start-1 {
			end = len(path)
		}
		if start == -1 || end < start {
			break
		}
		paramValue := fmt.Sprint(param)
		path = path[:start] + url.PathEscape(paramValue) + path[end:]
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
