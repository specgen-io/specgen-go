package models

import (
	"github.com/specgen-io/specgen-golang/v2/goven/generator"
	"github.com/specgen-io/specgen-golang/v2/writer"
)

func (g *EncodingJsonGenerator) EnumsHelperFunctions() *generator.CodeFile {
	w := writer.New(g.Modules.Enums, `helpers.go`)
	w.Lines(`
import (
	"encoding/json"
	"errors"
	"fmt"
)

func contains(lookFor string, arr []string) bool {
	for _, value := range arr {
		if lookFor == value {
			return true
		}
	}
	return false
}

func ReadStringValue(b []byte, values []string) (string, error) {
	var str string
	if err := json.Unmarshal(b, &str); err != nil {
		return "", err
	}
	if !contains(str, values) {
		return "", errors.New(fmt.Sprintf("Unknown enum value: %s", str))
	}
	return str, nil
}
`)
	return w.ToCodeFile()
}
