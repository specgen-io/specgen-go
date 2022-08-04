package responses

import (
	"fmt"

	"github.com/specgen-io/specgen-go/v2/github.com/specgen-io/specgen-go/v2/goven/github.com/specgen-io/specgen/generator/v2"
	"github.com/specgen-io/specgen-go/v2/types"
	"github.com/specgen-io/specgen-go/v2/writer"
	"github.com/specgen-io/specgen-go/v2/github.com/specgen-io/specgen-go/v2/goven/github.com/specgen-io/specgen/spec/v2"
)

func NewResponse(response *spec.OperationResponse, body string) string {
	return fmt.Sprintf(`%s{%s: &%s}`, ResponseTypeName(response.Operation), response.Name.PascalCase(), body)
}

func GenerateOperationResponseStruct(w *generator.Writer, operation *spec.NamedOperation) {
	w.Line(`type %s struct {`, ResponseTypeName(operation))
	responses := [][]string{}
	for _, response := range operation.Responses {
		responses = append(responses, []string{
			response.Name.PascalCase(),
			types.GoType(spec.Nullable(&response.Type.Definition)),
		})
	}
	writer.WriteAlignedLines(w.Indented(), responses)
	w.Line(`}`)
}

func ResponseTypeName(operation *spec.NamedOperation) string {
	return fmt.Sprintf(`%sResponse`, operation.Name.PascalCase())
}
