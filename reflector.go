package openapi

import impl "github.com/thebases/go-openapi/reflector"

type Reflector = impl.Reflector

func NewReflector() *Reflector {
	return impl.New()
}
