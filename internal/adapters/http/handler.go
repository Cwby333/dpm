package http

import (
	"context"
	"dpm/internal/models"
	"dpm/internal/services"
	"dpm/pkg/api/v1"
	"fmt"
	"log/slog"
	"net/http"
)

type Handler struct {
	Mux *http.ServeMux
	uServices *services.UserService
}

func NewHandler(uService *services.UserService) Handler {
	return Handler{
		Mux: http.NewServeMux(),
		uServices: uService,
	}
}

func corsMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("Access-Control-Allow-Origin", "*")
        w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
        w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
        
        if r.Method == "OPTIONS" {
            w.WriteHeader(http.StatusOK)
            return
        }
        
        next.ServeHTTP(w, r)
    })
}


func (h Handler) RegisterRoutes(strict api.ServerInterface) {
	h.Mux.Handle("GET /ping", http.HandlerFunc(strict.GetPing))
	h.Mux.Handle("POST /login", http.HandlerFunc(strict.Login))
	h.Mux.Handle("POST /register", corsMiddleware(http.HandlerFunc(strict.Register)))
}

func (h Handler) GetPing(ctx context.Context, request api.GetPingRequestObject) (api.GetPingResponseObject, error) {
	return api.GetPing200JSONResponse("Pong"), nil
}

func (h Handler) Register(ctx context.Context, request api.RegisterRequestObject) (api.RegisterResponseObject, error) {
	const op = "./internal/adapters/http/handlers.go.Login()"

	u := models.User{
		Username: *request.Body.Username,
		Email: *request.Body.Email,
		HashPsw: *request.Body.Password,
	}

	err := h.uServices.RegisterUser(ctx, u)
	if err != nil {
		slog.Error(fmt.Errorf("%s: %w", op, err).Error())
		msg := err.Error()
		return api.Register500JSONResponse{
			Message: &msg,
		}, err
	}

	msg := "Success register"
	return api.Register200JSONResponse{
		Message: &msg,
	}, nil
}

func (h Handler) Login(ctx context.Context, request api.LoginRequestObject) (api.LoginResponseObject, error) {
	const op = "./internal/adapters/http/handlers.go.Login()"

	user := models.User{
		Username: *request.Body.Username,
		HashPsw: *request.Body.Password,
	}

	token, err := h.uServices.Login(ctx, user)
	if err != nil {
		slog.Error(fmt.Errorf("%s: %w", op, err).Error())
		msg := err.Error()
		return api.Login500JSONResponse{
			Message: &msg,
		}, err
	}

	msg := fmt.Sprintf("Success login, token: %s", token)
	return api.Login200JSONResponse{
		Message: &msg,
	}, nil
}