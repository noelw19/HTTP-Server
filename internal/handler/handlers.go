package handler

import (
	"fmt"
	"maps"
	"strings"
)

type Handlers map[string]*Handler

// MatchResult contains the matched handler and extracted path variables
type MatchResult struct {
	Handler HandlerFunc
	Vars    Vars
}

func (h Handlers) Match(route string, method AllowedMethod) (HandlerFunc, error) {
	result, err := h.MatchWithVars(route, method)
	if err != nil {
		return nil, err
	}
	return result.Handler, nil
}

func (h Handlers) MatchWithVars(route string, method AllowedMethod) (*MatchResult, error) {
	if route == "" {
		return nil, fmt.Errorf("Empty route when trying to match")
	}

	// First, try exact matches (static routes)
	if handler, ok := h[route]; ok {
		keys := maps.Keys(handler.MethodFuncs)
		for iter := range keys {
			if iter == method {
				hf := handler.MethodFuncs[method]
				return &MatchResult{Handler: *hf, Vars: make(Vars)}, nil
			}
		}
		if handler.HandleFunc != nil {
			return &MatchResult{Handler: *handler.HandleFunc, Vars: make(Vars)}, nil
		}
	}

	// Then, try dynamic route matching
	for routePath, handler := range h {
		if !strings.Contains(routePath, "{") {
			continue // Skip static routes, already checked above
		}

		vars, matched := matchDynamicRoute(routePath, route)
		if matched {
			keys := maps.Keys(handler.MethodFuncs)
			for iter := range keys {
				if iter == method {
					hf := handler.MethodFuncs[method]
					return &MatchResult{Handler: *hf, Vars: vars}, nil
				}
			}
			if handler.HandleFunc != nil {
				return &MatchResult{Handler: *handler.HandleFunc, Vars: vars}, nil
			}
		}
	}

	return nil, fmt.Errorf("No route match found")
}

// matchDynamicRoute matches a route pattern (e.g., "/wakanda/{id}") against an actual route (e.g., "/wakanda/123")
// Returns the extracted variables and whether there was a match
func matchDynamicRoute(pattern, actualRoute string) (Vars, bool) {
	vars := make(Vars)

	// Split both pattern and actual route into segments
	patternParts := strings.Split(strings.Trim(pattern, "/"), "/")
	actualParts := strings.Split(strings.Trim(actualRoute, "/"), "/")

	// Must have same number of segments
	if len(patternParts) != len(actualParts) {
		return vars, false
	}

	// Match each segment
	for i, patternPart := range patternParts {
		actualPart := actualParts[i]

		// Check if this is a parameter segment (e.g., "{id}")
		if strings.HasPrefix(patternPart, "{") && strings.HasSuffix(patternPart, "}") {
			// Extract parameter name (remove { and })
			paramName := strings.TrimSuffix(strings.TrimPrefix(patternPart, "{"), "}")
			if paramName == "" {
				return vars, false // Invalid parameter name
			}
			vars[paramName] = actualPart
		} else if patternPart != actualPart {
			// Static segment doesn't match
			return vars, false
		}
	}

	return vars, true
}

func (h Handlers) Add(route string, hf HandlerFunc) *Handler {
	if route == "" {
		panic("Empty route when trying to add handler")
	}

	if _, ok := h[route]; ok {
		h[route].HandleFunc = &hf
	} else {
		handle := &Handler{
			route:          route,
			HandleFunc:     &hf,
			MethodFuncs:    map[AllowedMethod]*HandlerFunc{},
			AllowedMethods: []AllowedMethod{},
		}

		h[route] = handle

	}
	return h[route]
}
