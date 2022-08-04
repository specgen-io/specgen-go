package openapi

import (
	"github.com/specgen-io/specgen-go/v2/github.com/specgen-io/specgen-go/v2/goven/github.com/specgen-io/specgen/generator/v2"
	"github.com/specgen-io/specgen-go/v2/github.com/specgen-io/specgen-go/v2/goven/github.com/specgen-io/specgen/spec/v2"
)

var Openapi = generator.Generator{
	"openapi",
	"OpenAPI v3",
	"Generate OpenAPI specification",
	[]generator.GeneratorArg{
		{Arg: generator.ArgSpecFile, Required: true},
		{Arg: generator.ArgOutFile, Required: true},
	},
	func(specification *spec.Spec, params generator.GeneratorArgsValues) *generator.Sources {
		openapiFile := GenerateOpenapi(specification, params[generator.ArgOutFile])
		sources := generator.NewSources()
		sources.AddGenerated(openapiFile)
		return sources
	},
}