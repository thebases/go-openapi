package openapi

import "github.com/thebases/go-openapi/fiberadapter"

type Handler[I, O any] = fiberadapter.Handler[I, O]

type OperationConfig = fiberadapter.OperationConfig

func GET[I, O any](api *API, path string, config OperationConfig, handler Handler[I, O]) {
	fiberadapter.GET(api, path, config, handler)
}

func POST[I, O any](api *API, path string, config OperationConfig, handler Handler[I, O]) {
	fiberadapter.POST(api, path, config, handler)
}

func PUT[I, O any](api *API, path string, config OperationConfig, handler Handler[I, O]) {
	fiberadapter.PUT(api, path, config, handler)
}

func PATCH[I, O any](api *API, path string, config OperationConfig, handler Handler[I, O]) {
	fiberadapter.PATCH(api, path, config, handler)
}

func DELETE[I, O any](api *API, path string, config OperationConfig, handler Handler[I, O]) {
	fiberadapter.DELETE(api, path, config, handler)
}
