package http

import (
	"dpm/pkg/api/v1"
	"log/slog"
	"net/http"

	nethttpmiddleware "github.com/oapi-codegen/nethttp-middleware"
)

func NewServer(handler Handler) *http.Server {
	strictServer := api.NewStrictHandler(handler, nil)

	handler.RegisterRoutes(strictServer)

	swagger, err := api.GetSwagger()
	if err != nil {
		slog.Error("cannot get swagger spec for validate middleware")
		return nil
	}

	mw := nethttpmiddleware.OapiRequestValidator(swagger)

	hand := api.HandlerWithOptions(strictServer, api.StdHTTPServerOptions{
		BaseURL:     "0.0.0.0:3003",
		BaseRouter:  handler.Mux,
		Middlewares: []api.MiddlewareFunc{mw},
	})

	server := &http.Server{
		Handler: hand,
		Addr:    "0.0.0.0:3003",
	}

	return server
}
