package backends

import "lugmac/typechecking"

type Backend interface {
	Generate(module string, in *typechecking.Context) error
}
