package service

import (
	"fmt"
	"github.com/specgen-io/specgen-golang/v2/common"
	"strings"

	"github.com/specgen-io/specgen-golang/v2/goven/spec"
	"github.com/specgen-io/specgen-golang/v2/responses"
)

func (g *Generator) OperationSignature(operation *spec.NamedOperation, apiPackage *string) string {
	return fmt.Sprintf(`%s(%s) %s`,
		operation.Name.PascalCase(),
		strings.Join(common.OperationParams(g.Types, operation), ", "),
		g.operationReturn(operation, apiPackage),
	)
}

func (g *Generator) operationReturn(operation *spec.NamedOperation, responsePackageName *string) string {
	if len(operation.Responses) == 1 {
		response := operation.Responses[0]
		if response.Type.Definition.IsEmpty() {
			return `error`
		}
		return fmt.Sprintf(`(*%s, error)`, g.Types.GoType(&response.Type.Definition))
	}
	responseType := responses.ResponseTypeName(operation)
	if responsePackageName != nil {
		responseType = *responsePackageName + "." + responseType
	}
	return fmt.Sprintf(`(*%s, error)`, responseType)
}
