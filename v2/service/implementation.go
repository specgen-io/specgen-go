package service

import (
	"fmt"
	"github.com/specgen-io/specgen-golang/v2/goven/generator"
	"github.com/specgen-io/specgen-golang/v2/goven/spec"
	"github.com/specgen-io/specgen-golang/v2/types"
	"github.com/specgen-io/specgen-golang/v2/writer"
)

func (g *Generator) ServicesImpls(version *spec.Version) []generator.CodeFile {
	files := []generator.CodeFile{}
	for _, api := range version.Http.Apis {
		files = append(files, *g.serviceImpl(&api))
	}
	return files
}

func (g *Generator) serviceImpl(api *spec.Api) *generator.CodeFile {
	w := writer.New(g.Modules.ServicesImpl(api.InHttp.InVersion), fmt.Sprintf("%s.go", api.Name.SnakeCase()))

	w.Imports.Add("errors")
	w.Imports.AddApiTypes(api)
	if types.ApiHasBody(api) {
		w.Imports.Module(g.Modules.ServicesApi(api))
	}
	if isContainsModel(api) {
		w.Imports.Module(g.Modules.Models(api.InHttp.InVersion))
	}

	w.EmptyLine()
	w.Line(`type %s struct{}`, serviceTypeName(api))
	w.EmptyLine()
	apiPackage := api.Name.SnakeCase()
	for _, operation := range api.Operations {
		w.Line(`func (service *%s) %s {`, serviceTypeName(api), g.operationSignature(&operation, &apiPackage))
		singleEmptyResponse := len(operation.Responses) == 1 && operation.Responses[0].Type.Definition.IsEmpty()
		if singleEmptyResponse {
			w.Line(`  return errors.NewImports("implementation has not added yet")`)
		} else {
			w.Line(`  return nil, errors.NewImports("implementation has not added yet")`)
		}
		w.Line(`}`)
	}

	return w.ToCodeFile()
}

func isContainsModel(api *spec.Api) bool {
	for _, operation := range api.Operations {
		if operation.Body != nil {
			if types.IsModel(&operation.Body.Type.Definition) {
				return true
			}
		}
		for _, param := range operation.QueryParams {
			if types.IsModel(&param.Type.Definition) {
				return true
			}
		}
		for _, param := range operation.HeaderParams {
			if types.IsModel(&param.Type.Definition) {
				return true
			}
		}
		for _, param := range operation.Endpoint.UrlParams {
			if types.IsModel(&param.Type.Definition) {
				return true
			}
		}
		for _, response := range operation.Responses {
			if types.IsModel(&response.Type.Definition) {
				return true
			}
		}
	}
	return false
}

func serviceTypeName(api *spec.Api) string {
	return fmt.Sprintf(`%sService`, api.Name.PascalCase())
}
