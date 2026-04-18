package http

import (
	"context"
	"dpm/internal/models"
	"dpm/internal/services"
	"dpm/pkg/api/v1"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
)

type Handler struct {
	Mux       *http.ServeMux
	uServices *services.UserService
	mService  *services.MusicService
	lhService *services.ListeningHistoryService
	fService *services.FavorService
}

func NewHandler(uService *services.UserService, mService *services.MusicService, lhService *services.ListeningHistoryService, fService *services.FavorService) Handler {
	return Handler{
		Mux:       http.NewServeMux(),
		uServices: uService,
		mService:  mService,
		lhService: lhService,
		fService: fService,
	}
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", r.Header.Get("Origin"))
		w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		w.Header().Set("Access-Control-Allow-Credentials", "true")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func (h Handler) RegisterRoutes(strict api.ServerInterface) {
	h.Mux.Handle("GET /ping", http.HandlerFunc(strict.GetPing))
	h.Mux.Handle("POST /login", corsMiddleware(http.HandlerFunc(strict.Login)))
	h.Mux.Handle("OPTIONS /login", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		slog.Info(r.Header.Get("Origin"))
		w.Header().Set("Access-Control-Allow-Origin", r.Header.Get("Origin"))
        w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
        w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		w.Header().Set("Access-Control-Allow-Credentials", "true")
		w.WriteHeader(http.StatusOK)
	}))
	h.Mux.Handle("POST /register", corsMiddleware(http.HandlerFunc(strict.Register)))
	h.Mux.Handle("OPTIONS /register", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		slog.Info(r.Header.Get("Origin"))
		w.Header().Set("Access-Control-Allow-Origin", r.Header.Get("Origin"))
        w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
        w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		w.Header().Set("Access-Control-Allow-Credentials", "true")
		w.WriteHeader(http.StatusOK)
	}))
	h.Mux.Handle("GET /music/{musicID}", corsMiddleware(wrapGetMusic(strict)))
	h.Mux.Handle("GET /music", corsMiddleware(wrapGetAllMusic(strict)))
	h.Mux.Handle("OPTIONS /music", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		slog.Info(r.Header.Get("Origin"))
		w.Header().Set("Access-Control-Allow-Origin", r.Header.Get("Origin"))
        w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
        w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		w.Header().Set("Access-Control-Allow-Credentials", "true")
		w.WriteHeader(http.StatusOK)
	}))
	h.Mux.Handle("POST /listening-history/{userID}", corsMiddleware(wrapAddLToLH(strict)))
	h.Mux.Handle("DELETE /listening-history/{userID}", corsMiddleware(wrapDeleteLFromLH(strict)))
	h.Mux.Handle("GET /listening-history/{userID}", corsMiddleware(wrapGetLH(strict)))
	h.Mux.Handle("OPTIONS /listening-history", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		slog.Info(r.Header.Get("Origin"))
		w.Header().Set("Access-Control-Allow-Origin", r.Header.Get("Origin"))
        w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
        w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		w.Header().Set("Access-Control-Allow-Credentials", "true")
		w.WriteHeader(http.StatusOK)
	}))
	h.Mux.Handle("POST /favor", corsMiddleware(wrapCreateFavor(strict)))
	h.Mux.Handle("GET /favor", corsMiddleware(wrapGetFavor(strict)))
	h.Mux.Handle("DELETE /favor", corsMiddleware(wrapDeleteFavor(strict)))
	h.Mux.Handle("OPTIONS /favor", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		slog.Info(r.Header.Get("Origin"))
		w.Header().Set("Access-Control-Allow-Origin", r.Header.Get("Origin"))
        w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
        w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		w.Header().Set("Access-Control-Allow-Credentials", "true")
		w.WriteHeader(http.StatusOK)
	}))
}

func wrapGetAllMusic(strict api.ServerInterface) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		c, err := r.Cookie("Access-Token")
		if err != nil {
			slog.Info(err.Error())
			c = &http.Cookie{
				Value: "",
			}
			// return
		} else {
			slog.Info(fmt.Sprint(c.Name, " :", c.Value))
		}
		strict.GetAllMusic(w, r, api.GetAllMusicParams{AccessToken: &c.Value})
	}
}

func wrapDeleteFavor(strict api.ServerInterface) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		c, err := r.Cookie("Access-Token")
		if err != nil {
			slog.Info("wrapDeleteFavor error")
			slog.Info(err.Error())
			c = &http.Cookie{
			}
			return
		} else {
			slog.Info(fmt.Sprint(c.Name, " :", c.Value))
		}
		strict.DeleteFavor(w, r, api.DeleteFavorParams{AccessToken: c.Value})
	}
}

func wrapGetFavor(strict api.ServerInterface) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		c, err := r.Cookie("Access-Token")
		if err != nil {
			slog.Info("wrapGetFavor error")
			slog.Info(err.Error())
			c = &http.Cookie{
			}
			return
		} else {
			slog.Info(fmt.Sprint(c.Name, " :", c.Value))
		}
		strict.GetFavor(w, r, api.GetFavorParams{AccessToken: c.Value})	
	}
}

func wrapCreateFavor(strict api.ServerInterface) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		c, err := r.Cookie("Access-Token")
		if err != nil {
			slog.Info("wrapCreateFavor error")
			slog.Info(err.Error())
			c = &http.Cookie{
			}
			return
		} else {
			slog.Info(fmt.Sprint(c.Name, " :", c.Value))
		}
		strict.AddFavor(w, r, api.AddFavorParams{AccessToken: c.Value})
	}
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
		Body: struct{Message *string "json:\"message,omitempty\""}{&msg},
		Headers: api.Login200ResponseHeaders{
			AccessToken: access,
			RefreshToken: refresh,
		},
	}, nil
}

func (h Handler) GetAllMusic(ctx context.Context, request api.GetAllMusicRequestObject) (api.GetAllMusicResponseObject, error) {
	const op = "./internal/adapters/http/handler.go.GetAllMusic()"

	if request.Params.AccessToken != nil {
		slog.Info(*request.Params.AccessToken)
	}

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
	slog.Info(fmt.Sprintf("%+v", lhi))
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

func (h Handler) AddFavor(ctx context.Context, request api.AddFavorRequestObject) (api.AddFavorResponseObject, error) {
	const op = "./internal/adapters/http/handler.go.AddFavor()"

	t := request.Params.AccessToken

	if t == "" {
		slog.Info("DeleteFavor token empty")
		return api.AddFavor500JSONResponse("Access-Token empty, please login and retry action"), errors.New("Token empty")
	}

	claims, err := h.uServices.CheckAccessToken(ctx, t)

	f := models.ListeningHistory{
		UserID: claims["sub"].(string),
		MusicID: request.Body.MusicID,
	}
	slog.Info(fmt.Sprintf("%+v", f))
	err = h.fService.CreateFavor(ctx, f)
	if err != nil {
		return api.AddFavor500JSONResponse(err.Error()), fmt.Errorf("%s: %w", op, err)
	}

	return api.AddFavor200JSONResponse("Success"), nil
}

func (h Handler) GetFavor(ctx context.Context, request api.GetFavorRequestObject) (api.GetFavorResponseObject, error) {
	const op = "./internal/adapters/http/handler.go.GetFavor()"

	t := request.Params.AccessToken

	if t == "" {
		slog.Info("DeleteFavor token empty")
		return api.GetFavor500JSONResponse("Access-Token empty, please login and retry action"), errors.New("Token empty")
	}

	claims, err := h.uServices.CheckAccessToken(ctx, t)

	f := models.ListeningHistory{
		UserID: claims["sub"].(string),
	}
	slog.Info(fmt.Sprintf("%+v", f))
	favor, err := h.fService.ReadFavor(ctx, f)
	if err != nil {
		slog.Info(err.Error())
		return api.GetFavor500JSONResponse(err.Error()), fmt.Errorf("%s: %w", op, err)
	}

	fAPI := make([]api.Music, 0, len(favor))
	
	for i := range favor {
		fAPI = append(fAPI, api.Music{
			Id: favor[i].MusicID,
			Name: favor[i].MusicName,
			MusicCover: &favor[i].MusicCover,
			SongUrl: favor[i].MusicSongURL,
			UploaderId: favor[i].MusicUploaderID,
			Likes: favor[i].MusicLikes,
		})
	}

	return api.GetFavor200JSONResponse{
		GetMusicJSONResponse: fAPI,
	}, nil
}

func (h Handler) DeleteFavor(ctx context.Context, request api.DeleteFavorRequestObject) (api.DeleteFavorResponseObject, error) {
	const op = "./internal/adapters/http/handler.go.DeleteFavor()"

	t := request.Params.AccessToken

	if t == "" {
		slog.Info("DeleteFavor token empty")
		return api.DeleteFavor500JSONResponse("Access-Token empty, please login and retry action"), errors.New("Token empty")
	}

	claims, err := h.uServices.CheckAccessToken(ctx, t)

	lhi := models.ListeningHistory{
		UserID: claims["sub"].(string),
		MusicID: request.Body.MusicID,
	}
	err = h.fService.DeleteFavor(ctx, lhi)
	if err != nil {
		return api.DeleteFavor500JSONResponse(err.Error()), fmt.Errorf("%s: %w", op, err)
	}

	return api.DeleteFavor200JSONResponse("Success"), nil
}