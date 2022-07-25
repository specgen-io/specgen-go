package openapi

import "github.com/specgen-io/specgen-go/v2/spec"

func name(source string) spec.Name {
	return spec.Name{source, nil}
}

var emptyType = spec.Type{*spec.Plain(spec.TypeEmpty), nil}
