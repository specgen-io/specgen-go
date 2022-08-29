package types

import (
	"fmt"
	"strings"

	"github.com/specgen-io/specgen-golang/v2/goven/generator"
	"github.com/specgen-io/specgen-golang/v2/goven/spec"
	"github.com/specgen-io/specgen-golang/v2/module"
)

var ModelsPackage = "models"
var ErrorsModelsPackage = "errmodels"

func GoType(typ *spec.TypeDef) string {
	return goType(typ, ModelsPackage)
}

func GoErrType(typ *spec.TypeDef) string {
	return goType(typ, ErrorsModelsPackage)
}

func GoTypeSamePackage(typ *spec.TypeDef) string {
	return goType(typ, "")
}

func goType(typ *spec.TypeDef, modelsPackage string) string {
	switch typ.Node {
	case spec.PlainType:
		return PlainGoType(typ.Plain, modelsPackage)
	case spec.NullableType:
		child := goType(typ.Child, modelsPackage)
		if typ.Child.Node == spec.PlainType {
			return "*" + child
		}
		return child
	case spec.ArrayType:
		child := goType(typ.Child, modelsPackage)
		result := "[]" + child
		return result
	case spec.MapType:
		child := goType(typ.Child, modelsPackage)
		result := "map[string]" + child
		return result
	default:
		panic(fmt.Sprintf("Unknown type: %v", typ))
	}
}

func PlainGoType(typ string, modelsPackage string) string {
	switch typ {
	case spec.TypeInt32:
		return "int"
	case spec.TypeInt64:
		return "int64"
	case spec.TypeFloat:
		return "float32"
	case spec.TypeDouble:
		return "float64"
	case spec.TypeDecimal:
		return "decimal.Decimal"
	case spec.TypeBoolean:
		return "bool"
	case spec.TypeString:
		return "string"
	case spec.TypeUuid:
		return "uuid.UUID"
	case spec.TypeDate:
		return "civil.Date"
	case spec.TypeDateTime:
		return "civil.DateTime"
	case spec.TypeJson:
		return "json.RawMessage"
	case spec.TypeEmpty:
		return EmptyType
	default:
		if modelsPackage != "" {
			return fmt.Sprintf("%s.%s", modelsPackage, typ)
		}
		return typ
	}
}

const EmptyType = `empty.Type`

func GenerateEmpty(module module.Module) *generator.CodeFile {
	code := `
package empty

type Type struct{}

var Value = Type{}
`
	code, _ = generator.ExecuteTemplate(code, struct{ PackageName string }{module.Name})
	return &generator.CodeFile{module.GetPath("empty.go"), strings.TrimSpace(code)}
}
