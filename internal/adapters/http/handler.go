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
	lhService *services.ListeningHistoryService
}

func NewHandler(uService *services.UserService, mService *services.MusicService, lhService *services.ListeningHistoryService) Handler {
	return Handler{
		Mux:       http.NewServeMux(),
		uServices: uService,
		mService:  mService,
		lhService: lhService,
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
	h.Mux.Handle("POST /listening-history/{userID}", corsMiddleware(wrapAddLToLH(strict)))
	h.Mux.Handle("DELETE /listening-history/{userID}", corsMiddleware(wrapDeleteLFromLH(strict)))
	h.Mux.Handle("GET /listening-history/{userID}", corsMiddleware(wrapGetLH(strict)))
}

func wrapGetLH(strict api.ServerInterface) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		strict.GetLH(w, r, r.PathValue("userID"))
	}
}

func wrapDeleteLFromLH(strict api.ServerInterface) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		strict.DeleteListeningFromLH(w, r, r.PathValue("userID"))
	}
}

func wrapAddLToLH(strict api.ServerInterface) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		strict.AddListeningToLH(w, r, r.PathValue("userID"))
	}
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

	access, refresh, err := h.uServices.Login(ctx, user)
	if err != nil {
		slog.Error(fmt.Errorf("%s: %w", op, err).Error())
		msg := err.Error()
		return api.Login500JSONResponse{
			Message: &msg,
		}, err
	}

	msg := "Success"

	return api.Login200JSONResponse{
		Headers: api.Login200ResponseHeaders{
			AccessToken: access,
			RefreshToken: refresh,
		},
		Body: struct{Message *string "json:\"message,omitempty\""}{Message: &msg},
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
			MusicCover: &p[i].CoverURL,
			SongUrl: p[i].SongURL,
		})
	}

	slog.Info("Put response")

	return api.GetAllMusic200JSONResponse{
		GetMusicJSONResponse: pResp,
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
			MusicCover: &product.CoverURL,
			SongUrl: &product.SongURL,
		},
	}, nil
}

func (h Handler) AddListeningToLH(ctx context.Context, request api.AddListeningToLHRequestObject) (api.AddListeningToLHResponseObject, error) {
	const op = "./internal/adapters/http/handler.go.AddListeningToLH"

	lhi := models.ListeningHistory{
		UserID: request.UserID,
		MusicID: request.Body.MusicID,
	}
	err := h.lhService.CreateListeningHistoryItem(ctx, lhi)
	if err != nil {
		return api.AddListeningToLH500JSONResponse(err.Error()), fmt.Errorf("%s: %w", op, err)
	}

	return api.AddListeningToLH200JSONResponse("Success"), nil
}

func (h Handler) GetLH(ctx context.Context, request api.GetLHRequestObject) (api.GetLHResponseObject, error) {
	const op = "./internal/adapters/http/handler.go.GetLH()"
	
	lhi := models.ListeningHistory{
		UserID: request.UserID,
	}
	lh, err := h.lhService.ReadListeningHistory(ctx, lhi)
	if err != nil {
		return api.GetLH500JSONResponse(err.Error()), nil
	}

	lhr := make([]api.ListeningHistoryResponse, 0, len(lh))

	for i := range lh {
		lhr = append(lhr, api.ListeningHistoryResponse{
			MusicId: &lh[i].MusicID,
			MusicName: &lh[i].MusicName,
			MusicCover: &lh[i].MusicCover,
			SongUrl: &lh[i].MusicSongURL,
			MusicDuration: &lh[i].MusicDurationSeconds,
			MusicLikes: &lh[i].MusicLikes,
			UploaderId: &lh[i].MusicUploaderID,
			UploaderUsername: &lh[i].UserUsername,
			ListeningDate: &lh[i].ListeningDate,
		})
	}

	return api.GetLH200JSONResponse{
		GetListeningHistoryJSONResponse: lhr,
	}, nil
}	

func (h Handler) DeleteListeningFromLH(ctx context.Context, request api.DeleteListeningFromLHRequestObject) (api.DeleteListeningFromLHResponseObject, error) {	
	const op = "./internal/adapters/http/handler.go.DeleteListingFromLH()"

	slog.Info(request.UserID)
	slog.Info(request.Body.MusicId)
	lhi := models.ListeningHistory{
		UserID: request.UserID,
		MusicID: request.Body.MusicId,
	}
	err := h.lhService.DeleteListeningHistoryItem(ctx, lhi)
	if err != nil {
		return api.DeleteListeningFromLH500JSONResponse(err.Error()), fmt.Errorf("%s: %w", op, err)
	}

	return api.DeleteListeningFromLH200JSONResponse("Success"), nil
}