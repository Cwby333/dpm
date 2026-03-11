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
	Mux       *http.ServeMux
	uServices *services.UserService
	mService  *services.MusicService
}

func NewHandler(uService *services.UserService, mService *services.MusicService) Handler {
	return Handler{
		Mux:       http.NewServeMux(),
		uServices: uService,
		mService:  mService,
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
	h.Mux.Handle("GET /music/{musicID}", corsMiddleware(wrapGetMusic(strict)))
	h.Mux.Handle("GET /music", corsMiddleware(http.HandlerFunc(strict.GetAllMusic)))
}

func wrapGetMusic(strict api.ServerInterface) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		strict.GetMusic(w, r, r.PathValue("musicID"))
	}
}

func (h Handler) GetPing(ctx context.Context, request api.GetPingRequestObject) (api.GetPingResponseObject, error) {
	return api.GetPing200JSONResponse("Pong"), nil
}

func (h Handler) Register(ctx context.Context, request api.RegisterRequestObject) (api.RegisterResponseObject, error) {
	const op = "./internal/adapters/http/handlers.go.Login()"

	u := models.User{
		Username: *request.Body.Username,
		Email:    *request.Body.Email,
		HashPsw:  *request.Body.Password,
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
		HashPsw:  *request.Body.Password,
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

func (h Handler) GetAllMusic(ctx context.Context, request api.GetAllMusicRequestObject) (api.GetAllMusicResponseObject, error) {
	const op = "./internal/adapters/http/handler.go.GetAllMusic()"

	slog.Info("Get request")

	p, err := h.mService.GetAllMusic(ctx)
	if err != nil {
		return api.GetAllMusic500JSONResponse(err.Error()), err
	}

	pResp := make([]api.Music, 0, len(p))
	for i := range p {
		pResp = append(pResp, api.Music{
			Id:              p[i].ID,
			Name:            p[i].Name,
			UploaderId:      p[i].UploaderID,
			Likes:           p[i].Likes,
			DurationSeconds: p[i].DurationSec,
		})
	}

	slog.Info("Put response")

	return api.GetAllMusic200JSONResponse{
		pResp,
	}, nil
}

func (h Handler) GetMusic(ctx context.Context, request api.GetMusicRequestObject) (api.GetMusicResponseObject, error) {
	const op = "./internal/adapters/http/handler.go.GetMusic()"

	product, err := h.mService.GetMusic(ctx, request.MusicID)
	if err != nil {
		errMsg := err.Error()
		return api.GetMusic500JSONResponse{
			Message: &errMsg,
		}, err
	}

	return api.GetMusic200JSONResponse{
		GetMusicResponseJSONResponse: api.GetMusicResponseJSONResponse{
			Id:              &product.ID,
			UploaderId:      &product.UploaderID,
			Name:            &product.Name,
			Likes:           &product.Likes,
			DurationSeconds: &product.DurationSec,
		},
	}, nil
}
