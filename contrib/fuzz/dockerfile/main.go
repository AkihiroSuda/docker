package dockerfile

import (
	"bytes"

	"github.com/docker/docker/builder/dockerfile/parser"
)

func Fuzz(data []byte) int {
	d := parser.Directive{LookingForDirectives: true}
	parser.SetEscapeToken(parser.DefaultEscapeToken, &d)
	_, err := parser.Parse(bytes.NewReader(data), &d)
	if err != nil {
		return 0
	}
	return 1
}
