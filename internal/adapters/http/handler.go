package http

import (
	"context"
	"dpm/internal/models"
	"dpm/internal/services"
	"dpm/pkg/api/v1"
	"encoding/json"
	"time"

	// "encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"math"
	_ "mime/multipart"
	"net/http"

	// "time"

	"github.com/tcolgate/mp3"
)

type Handler struct {
	Mux         *http.ServeMux
	uServices   *services.UserService
	mService    *services.MusicService
	lhService   *services.ListeningHistoryService
	fService    *services.FavorService
	likeService *services.LikeService
	aService *services.AlbumsService
}

func NewHandler(uService *services.UserService, mService *services.MusicService, lhService *services.ListeningHistoryService, fService *services.FavorService, likeService *services.LikeService, aService *services.AlbumsService) Handler {
	return Handler{
		Mux:         http.NewServeMux(),
		uServices:   uService,
		mService:    mService,
		lhService:   lhService,
		fService:    fService,
		likeService: likeService,
		aService: aService,
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

func recoveryMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				slog.Error(fmt.Sprintf("Recover middleware: %v", err))
			}
		}()

		next.ServeHTTP(w, r)
	})
}

func (h Handler) RegisterRoutes(strict api.ServerInterface) {
	h.Mux.Handle("GET /ping", http.HandlerFunc(strict.GetPing))
	h.Mux.Handle("OPTIONS /ping", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		slog.Info(r.Header.Get("Origin"))
		w.Header().Set("Access-Control-Allow-Origin", r.Header.Get("Origin"))
		w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		w.Header().Set("Access-Control-Allow-Credentials", "true")
		w.WriteHeader(http.StatusOK)
	}))
	h.Mux.Handle("POST /login", corsMiddleware(http.HandlerFunc(h.Login)))
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
	h.Mux.Handle("OPTIONS /music/{musicID}", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		slog.Info(r.Header.Get("Origin"))
		w.Header().Set("Access-Control-Allow-Origin", r.Header.Get("Origin"))
		w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		w.Header().Set("Access-Control-Allow-Credentials", "true")
		w.WriteHeader(http.StatusOK)
	}))
	h.Mux.Handle("GET /music", corsMiddleware(wrapGetAllMusic(strict)))
	h.Mux.Handle("OPTIONS /music", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		slog.Info(r.Header.Get("Origin"))
		w.Header().Set("Access-Control-Allow-Origin", r.Header.Get("Origin"))
		w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		w.Header().Set("Access-Control-Allow-Credentials", "true")
		w.WriteHeader(http.StatusOK)
	}))
	h.Mux.Handle("POST /listening-history", corsMiddleware(wrapAddLToLH(strict)))
	h.Mux.Handle("DELETE /listening-history", corsMiddleware(wrapDeleteLFromLH(strict)))
	h.Mux.Handle("GET /listening-history", corsMiddleware(wrapGetLH(strict)))
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
	h.Mux.Handle("GET /profile", corsMiddleware(wrapGetProfile(strict)))
	h.Mux.Handle("OPTIONS /profile", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		slog.Info(r.Header.Get("Origin"))
		w.Header().Set("Access-Control-Allow-Origin", r.Header.Get("Origin"))
		w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		w.Header().Set("Access-Control-Allow-Credentials", "true")
		w.WriteHeader(http.StatusOK)
	}))
	h.Mux.Handle("POST /music/like", corsMiddleware(wrapPostLike(strict)))
	h.Mux.Handle("DELETE /music/like", corsMiddleware(wrapDeleteLike(strict)))
	h.Mux.Handle("OPTIONS /music/like", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		slog.Info(r.Header.Get("Origin"))
		w.Header().Set("Access-Control-Allow-Origin", r.Header.Get("Origin"))
		w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		w.Header().Set("Access-Control-Allow-Credentials", "true")
		w.WriteHeader(http.StatusOK)
	}))
	h.Mux.Handle("GET /likes", corsMiddleware(wrapGetLikedTracks(strict)))
	h.Mux.Handle("OPTIONS /likes", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		slog.Info(r.Header.Get("Origin"))
		w.Header().Set("Access-Control-Allow-Origin", r.Header.Get("Origin"))
		w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		w.Header().Set("Access-Control-Allow-Credentials", "true")
		w.WriteHeader(http.StatusOK)
	}))
	h.Mux.Handle("POST /logout", corsMiddleware(http.HandlerFunc(h.Logout)))
	h.Mux.Handle("OPTIONS /logout", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		slog.Info(r.Header.Get("Origin"))
		w.Header().Set("Access-Control-Allow-Origin", r.Header.Get("Origin"))
		w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		w.Header().Set("Access-Control-Allow-Credentials", "true")
		w.WriteHeader(http.StatusOK)
	}))
	h.Mux.Handle("POST /music/upload", corsMiddleware(http.HandlerFunc(h.MusicUpload)))
	h.Mux.Handle("OPTIONS /music/upload", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		slog.Info(r.Header.Get("Origin"))
		w.Header().Set("Access-Control-Allow-Origin", r.Header.Get("Origin"))
		w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		w.Header().Set("Access-Control-Allow-Credentials", "true")
		w.WriteHeader(http.StatusOK)
	}))
	h.Mux.Handle("POST /music/play", corsMiddleware(http.HandlerFunc(strict.PostMusicPlay)))
	h.Mux.Handle("OPTIONS /music/play", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		slog.Info(r.Header.Get("Origin"))
		w.Header().Set("Access-Control-Allow-Origin", r.Header.Get("Origin"))
		w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		w.Header().Set("Access-Control-Allow-Credentials", "true")
		w.WriteHeader(http.StatusOK)
	}))
	h.Mux.Handle("GET /album", corsMiddleware(http.HandlerFunc(strict.GetAlbums)))
	h.Mux.Handle("POST /album", corsMiddleware(http.HandlerFunc(h.UploadAlbum)))
	h.Mux.Handle("OPTIONS /album", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		slog.Info(r.Header.Get("Origin"))
		w.Header().Set("Access-Control-Allow-Origin", r.Header.Get("Origin"))
		w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		w.Header().Set("Access-Control-Allow-Credentials", "true")
		w.WriteHeader(http.StatusOK)
	}))
	h.Mux.Handle("GET /album/{albumID}", corsMiddleware(wrapGetAlbum(strict)))
	h.Mux.Handle("OPTIONS /album/{albumID}", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		slog.Info(r.Header.Get("Origin"))
		w.Header().Set("Access-Control-Allow-Origin", r.Header.Get("Origin"))
		w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		w.Header().Set("Access-Control-Allow-Credentials", "true")
		w.WriteHeader(http.StatusOK)
	}))
}

func wrapGetAlbum(strict api.ServerInterface) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		strict.GetAlbumID(w, r, r.PathValue("albumID"))
	}
}

func wrapGetLikedTracks(strict api.ServerInterface) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		c, err := r.Cookie("Access-Token")
		if err != nil {
			slog.Info("wrapGetLikedTracks")
			slog.Error(err.Error())
			c = &http.Cookie{
				Value: "",
			}
		} else {
			slog.Info(fmt.Sprintf("%v: %v", c.Name, c.Value))
		}

		strict.GetLikes(w, r, api.GetLikesParams{AccessToken: c.Value})
	}
}

func wrapDeleteLike(strict api.ServerInterface) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		c, err := r.Cookie("Access-Token")
		if err != nil {
			slog.Info("wrapDeleteLike")
			slog.Error(err.Error())
			c = &http.Cookie{}
		} else {
			slog.Info(fmt.Sprintf("%v: %v", c.Name, c.Value))
		}

		strict.DeleteMusicLike(w, r, api.DeleteMusicLikeParams{AccessToken: c.Value})
	}
}

func wrapPostLike(strict api.ServerInterface) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		c, err := r.Cookie("Access-Token")
		if err != nil {
			slog.Info("wrapPostLike")
			slog.Error(err.Error())
			c = &http.Cookie{}
		} else {
			slog.Info(fmt.Sprintf("%v: %v", c.Name, c.Value))
		}

		strict.PostMusicLike(w, r, api.PostMusicLikeParams{AccessToken: c.Value})
	}
}

func wrapGetProfile(strict api.ServerInterface) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		c, err := r.Cookie("Access-Token")
		if err != nil {
			slog.Info("wrapGetProfile")
			slog.Error(err.Error())
			c = &http.Cookie{}
		} else {
			slog.Info(fmt.Sprintf("%v: %v", c.Name, c.Value))
		}

		strict.GetProfile(w, r, api.GetProfileParams{AccessToken: c.Value})
	}
}

func wrapGetAllMusic(strict api.ServerInterface) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		c, err := r.Cookie("Access-Token")
		if err != nil {
			slog.Info(err.Error())
			c = &http.Cookie{
				Value: "",
			}
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
			c = &http.Cookie{}
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
			c = &http.Cookie{}
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
			c = &http.Cookie{}
		} else {
			slog.Info(fmt.Sprint(c.Name, " :", c.Value))
		}
		strict.AddFavor(w, r, api.AddFavorParams{AccessToken: c.Value})
	}
}

func wrapGetLH(strict api.ServerInterface) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		c, err := r.Cookie("Access-Token")
		if err != nil {
			slog.Info("wrapGetLH error")
			slog.Info(err.Error())
			c = &http.Cookie{}
		} else {
			slog.Info(fmt.Sprint(c.Name, " :", c.Value))
		}
		strict.GetLH(w, r, api.GetLHParams{AccessToken: c.Value})
	}
}

func wrapDeleteLFromLH(strict api.ServerInterface) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		c, err := r.Cookie("Access-Token")
		if err != nil {
			slog.Info("wrapDelLFromLH error")
			slog.Info(err.Error())
			c = &http.Cookie{}
		} else {
			slog.Info(fmt.Sprint(c.Name, " :", c.Value))
		}
		strict.DeleteListeningFromLH(w, r, api.DeleteListeningFromLHParams{AccessToken: c.Value})
	}
}

func wrapAddLToLH(strict api.ServerInterface) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		c, err := r.Cookie("Access-Token")
		if err != nil {
			slog.Info("wrapAddLToLH error")
			slog.Info(err.Error())
			c = &http.Cookie{}
			return
		} else {
			slog.Info(fmt.Sprint(c.Name, " :", c.Value))
		}
		strict.AddListeningToLH(w, r, api.AddListeningToLHParams{AccessToken: c.Value})
	}
}

func wrapGetMusic(strict api.ServerInterface) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		c, err := r.Cookie("Access-Token")
		if err != nil {
			slog.Info(err.Error())
			c = &http.Cookie{
				Value: "",
			}
		}
		strict.GetMusic(w, r, r.PathValue("musicID"), api.GetMusicParams{AccessToken: &c.Value})
	}
}

type LoginReq struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func (h Handler) Login(w http.ResponseWriter, r *http.Request) {
	const op = "./internal/adapters/http/handler.go.Login()"

	slog.Info("Login200ReqNativeHandler")

	data, err := io.ReadAll(r.Body)
	if err != nil {
		slog.Error(fmt.Errorf("%s: %w", op, err).Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	user := LoginReq{}
	err = json.Unmarshal(data, &user)

	access, refresh, err := h.uServices.Login(r.Context(), models.User{
		Username: user.Username,
		HashPsw: user.Password,
	})
	if err != nil {
		slog.Error(fmt.Errorf("%s: %w", op, err).Error())
		http.Error(w, err.Error(), http.StatusBadRequest)
		return		
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "Access-Token",
		Value:    access.Sign,
		Expires:  time.Now().Add(time.Hour * 1),
		Secure:   true,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteNoneMode,
		Domain:   "",
	})
	http.SetCookie(w, &http.Cookie{
		Name:     "Refresh-Token",
		Value:    refresh.Sign,
		Expires:  time.Now().Add(time.Hour * 24),
		Secure:   true,
		HttpOnly: true,
		Domain:   "",
		Path:     "/refresh",
		SameSite: http.SameSiteNoneMode,
	})
	http.SetCookie(w, &http.Cookie{
		Name:     "Refresh-Token-Logout",
		Value:    refresh.Sign,
		Expires:  time.Now().Add(time.Hour * 24),
		Secure:   true,
		HttpOnly: true,
		Domain:   "",
		Path:     "/logout",
		SameSite: http.SameSiteNoneMode,
	})

	w.WriteHeader(200)
}

func (h Handler) Logout(w http.ResponseWriter, r *http.Request) {
	const op = "./internal/adapters/http/handler.go.Logout()"

	slog.Info("Logout200NativeHandler")

	_, err := r.Cookie("Access-Token")
	if err != nil {
		slog.Error(fmt.Errorf("%s: %w", op, err).Error())
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	_, err = r.Cookie("Refresh-Token-Logout")
	if err != nil {
		slog.Error(fmt.Errorf("%s: %w", op, err).Error())
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "Access-Token",
		Value:    "",
		Expires:  time.Now().Add(time.Second * 3),
		Secure:   true,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteNoneMode,
		Domain:   "",
	})
	http.SetCookie(w, &http.Cookie{
		Name:     "Refresh-Token",
		Value:    "",
		Expires:  time.Now().Add(time.Second * 3),
		Secure:   true,
		HttpOnly: true,
		Domain:   "",
		Path:     "/refresh",
		SameSite: http.SameSiteNoneMode,
	})
	http.SetCookie(w, &http.Cookie{
		Name:     "Refresh-Token-Logout",
		Value:    "",
		Expires:  time.Now().Add(time.Second * 3),
		Secure:   true,
		HttpOnly: true,
		Domain:   "",
		Path:     "/logout",
		SameSite: http.SameSiteNoneMode,
	})

	w.WriteHeader(200)
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

func (h Handler) GetAllMusic(ctx context.Context, request api.GetAllMusicRequestObject) (api.GetAllMusicResponseObject, error) {
	const op = "./internal/adapters/http/handler.go.GetAllMusic()"

	if request.Params.AccessToken != nil {
		slog.Info(*request.Params.AccessToken)
	}

	t := request.Params.AccessToken
	u := models.User{}

	if t != nil && *t != "" {
		slog.Info("Token not nil and not empty")

		claims, err := h.uServices.CheckAccessToken(ctx, *t)
		if err != nil {
			slog.Error(err.Error())
		} else {
			u.ID = claims["sub"].(string)
		}
	}

	slog.Info("Get request")

	p, l, err := h.mService.GetAllMusic(ctx, u)
	if err != nil {
		return api.GetAllMusic500JSONResponse(err.Error()), err
	}

	pResp := make([]api.Music, 0, len(p))
	for i := range p {
		urlCover := p[i].CoverURL
		if p[i].CoverURL != "" {
			urlCover, err = h.mService.GetPresignURLSong(ctx, p[i].CoverURL)
			if err != nil {
				slog.Error(err.Error())
				return api.GetAllMusic500JSONResponse(err.Error()), fmt.Errorf("%s: %w", op, err)
			}
		}

		pResp = append(pResp, api.Music{
			Id:              p[i].ID,
			Name:            p[i].Name,
			UploaderId:      p[i].UploaderID,
			Likes:           p[i].Likes,
			DurationSeconds: p[i].DurationSec,
			MusicCover:      &urlCover,
			SongUrl:         p[i].SongURL,
		})
	}

	lResp := make([]api.MusicLikes, 0, len(p))
	for i := range l {
		lResp = append(lResp, api.MusicLikes{
			MusicId: &l[i].MusicID,
		})
	}

	slog.Info("Put response")

	return api.GetAllMusic200JSONResponse{
		GetMusicJSONResponse: api.GetMusicJSONResponse{
			Music:      pResp,
			MusicLikes: &lResp,
		},
	}, nil
}

func (h Handler) GetMusic(ctx context.Context, request api.GetMusicRequestObject) (api.GetMusicResponseObject, error) {
	const op = "./internal/adapters/http/handler.go.GetMusic()"

	if request.Params.AccessToken != nil {
		slog.Info(*request.Params.AccessToken)
	}

	t := request.Params.AccessToken
	u := models.User{}

	if t != nil && *t != "" {
		slog.Info("Token not nil and not empty")

		claims, err := h.uServices.CheckAccessToken(ctx, *t)
		if err != nil {
			slog.Error(err.Error())
		} else {
			u.ID = claims["sub"].(string)
		}
	}

	slog.Info("GetMusic GET /music/{musicID} req", slog.String("reqID", request.MusicID))

	product, like, err := h.mService.GetMusic(ctx, request.MusicID, u.ID)
	if err != nil {
		slog.Error(err.Error())
		errMsg := err.Error()
		return api.GetMusic500JSONResponse{
			Message: &errMsg,
		}, err
	}

	slog.Info("GetMusic GET /music/{musicID}, Get music", slog.String("music_cover", product.CoverURL))

	urlCover, err := h.mService.GetPresignURLSong(ctx, product.CoverURL)
	if err != nil {
		msg := err.Error()
		slog.Error(err.Error())
		return api.GetMusic500JSONResponse{Message: &msg}, fmt.Errorf("%s: %w", op, err)
	}

	slog.Info("GetMusic GET /music/{musicID}, get presign URL", slog.String("URL", urlCover))

	return api.GetMusic200JSONResponse{
		GetMusicResponseJSONResponse: api.GetMusicResponseJSONResponse{
			Music: api.Music{
				Id:              product.ID,
				UploaderId:      product.UploaderID,
				Name:            product.Name,
				Likes:           product.Likes,
				DurationSeconds: product.DurationSec,
				MusicCover:      &urlCover,
				SongUrl:         product.SongURL,
			},
			MusicFavor: &api.MusicLikes{
				MusicId: &like.MusicID,
			},
		},
	}, nil
}

func (h Handler) AddListeningToLH(ctx context.Context, request api.AddListeningToLHRequestObject) (api.AddListeningToLHResponseObject, error) {
	const op = "./internal/adapters/http/handler.go.AddListeningToLH"

	t := request.Params.AccessToken

	if t == "" {
		slog.Info("DeleteFavor token empty")
		return api.AddListeningToLH500JSONResponse("Access-Token empty, please login and retry action"), errors.New("Token empty")
	}

	claims, err := h.uServices.CheckAccessToken(ctx, t)
	if err != nil {
		slog.Error(err.Error())
		return api.AddListeningToLH500JSONResponse(err.Error()), nil
	}

	lhi := models.ListeningHistory{
		UserID:  claims["sub"].(string),
		MusicID: request.Body.MusicID,
	}
	slog.Info(fmt.Sprintf("%+v", lhi))
	err = h.lhService.CreateListeningHistoryItem(ctx, lhi)
	if err != nil {
		slog.Error(err.Error())
		return api.AddListeningToLH500JSONResponse(err.Error()), fmt.Errorf("%s: %w", op, err)
	}

	return api.AddListeningToLH200JSONResponse("Success"), nil
}

func (h Handler) GetLH(ctx context.Context, request api.GetLHRequestObject) (api.GetLHResponseObject, error) {
	const op = "./internal/adapters/http/handler.go.GetLH()"

	t := request.Params.AccessToken

	slog.Info("GetLH Token: " + t)

	if t == "" {
		slog.Info("GetLH token empty")
		return api.GetLH500JSONResponse("Access-Token empty, please login and retry action"), errors.New("Token empty")
	}

	claims, err := h.uServices.CheckAccessToken(ctx, t)
	if err != nil {
		slog.Error(err.Error())
		return api.GetLH500JSONResponse(err.Error()), nil
	}

	slog.Info(fmt.Sprintf("Claims is nil: %v", claims == nil))

	lhi := models.ListeningHistory{
		UserID: claims["sub"].(string),
	}

	lh, err := h.lhService.ReadListeningHistory(ctx, lhi)
	if err != nil {
		return api.GetLH500JSONResponse(err.Error()), nil
	}

	lhr := make([]api.ListeningHistoryResponse, 0, len(lh))

	for i := range lh {
		lhr = append(lhr, api.ListeningHistoryResponse{
			MusicId:          &lh[i].MusicID,
			MusicName:        &lh[i].MusicName,
			MusicCover:       &lh[i].MusicCover,
			SongUrl:          &lh[i].MusicSongURL,
			MusicDuration:    &lh[i].MusicDurationSeconds,
			MusicLikes:       &lh[i].MusicLikes,
			UploaderId:       &lh[i].MusicUploaderID,
			UploaderUsername: &lh[i].UserUsername,
			ListeningDate:    &lh[i].ListeningDate,
		})
	}

	return api.GetLH200JSONResponse{
		GetListeningHistoryJSONResponse: lhr,
	}, nil
}

func (h Handler) DeleteListeningFromLH(ctx context.Context, request api.DeleteListeningFromLHRequestObject) (api.DeleteListeningFromLHResponseObject, error) {
	const op = "./internal/adapters/http/handler.go.DeleteListingFromLH()"

	t := request.Params.AccessToken

	if t == "" {
		slog.Info("DeleteFavor token empty")
		return api.DeleteListeningFromLH500JSONResponse("Access-Token empty, please login and retry action"), errors.New("Token empty")
	}

	claims, err := h.uServices.CheckAccessToken(ctx, t)
	if err != nil {
		slog.Error(err.Error())
		return api.DeleteListeningFromLH500JSONResponse(err.Error()), nil
	}

	slog.Info(t)
	slog.Info(request.Body.MusicId)
	lhi := models.ListeningHistory{
		UserID:        claims["sub"].(string),
		MusicID:       request.Body.MusicId,
		ListeningDate: *request.Body.ListeningDate,
	}
	err = h.lhService.DeleteListeningHistoryItem(ctx, lhi)
	if err != nil {
		slog.Error(err.Error())
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
	if err != nil {
		slog.Error(err.Error())
		return api.AddFavor500JSONResponse(err.Error()), nil
	}

	f := models.ListeningHistory{
		UserID:  claims["sub"].(string),
		MusicID: request.Body.MusicID,
	}
	slog.Info(fmt.Sprintf("%+v", f))
	err = h.fService.CreateFavor(ctx, f)
	if err != nil {
		slog.Error(err.Error())
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
	if err != nil {
		slog.Error(err.Error())
		return api.GetFavor500JSONResponse(err.Error()), nil
	}

	f := models.ListeningHistory{
		UserID: claims["sub"].(string),
	}
	slog.Info(fmt.Sprintf("%+v", f))
	favor, err := h.fService.ReadFavor(ctx, f)
	if err != nil {
		slog.Info(err.Error())
		return api.GetFavor500JSONResponse(err.Error()), fmt.Errorf("%s: %w", op, err)
	}

	fAPI := make([]api.Favor, 0, len(favor))

	for i := range favor {
		fAPI = append(fAPI, api.Favor{
			Id:         favor[i].MusicID,
			Name:       favor[i].MusicName,
			MusicCover: &favor[i].MusicCover,
			SongUrl:    favor[i].MusicSongURL,
			UploaderId: favor[i].MusicUploaderID,
			Likes:      favor[i].MusicLikes,
		})
	}

	return api.GetFavor200JSONResponse{
		GetFavorJSONResponse: fAPI,
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
	if err != nil {
		slog.Error(err.Error())
		return api.DeleteFavor500JSONResponse(err.Error()), nil
	}

	lhi := models.ListeningHistory{
		UserID:  claims["sub"].(string),
		MusicID: request.Body.MusicID,
	}
	err = h.fService.DeleteFavor(ctx, lhi)
	if err != nil {
		return api.DeleteFavor500JSONResponse(err.Error()), fmt.Errorf("%s: %w", op, err)
	}

	return api.DeleteFavor200JSONResponse("Success"), nil
}

func (h Handler) GetProfile(ctx context.Context, request api.GetProfileRequestObject) (api.GetProfileResponseObject, error) {
	const op = "./internal/adapters/http/handler.go.GetProfile()"

	t := request.Params.AccessToken

	if t == "" {
		slog.Info("GET /profile token empty")
		return api.GetProfile500JSONResponse("Token empty"), fmt.Errorf("%s: %w", op, errors.New("Token empty"))
	}

	claims, err := h.uServices.CheckAccessToken(ctx, t)
	if err != nil {
		slog.Error(err.Error())
		return api.GetProfile500JSONResponse(err.Error()), fmt.Errorf("%s: %w", op, err)
	}

	u := models.User{
		ID: claims["sub"].(string),
	}

	us, err := h.uServices.ReadUser(ctx, u)
	if err != nil {
		slog.Error(err.Error())
		return api.GetProfile500JSONResponse(err.Error()), fmt.Errorf("%s: %w", op, err)
	}

	sTime := fmt.Sprint(us.RegisterAt)
	return api.GetProfile200JSONResponse{
		GetProfileJSONResponse: api.GetProfileJSONResponse{
			Email:          &us.Email,
			Username:       &us.Username,
			RegisterAt:     &sTime,
			Likes:          &us.Likes,
			ListeningCount: &us.ListeningCount,
			FavorCount:     &us.FavorCount,
		},
	}, nil
}

func (h Handler) PostMusicLike(ctx context.Context, request api.PostMusicLikeRequestObject) (api.PostMusicLikeResponseObject, error) {
	const op = "./internal/adapters/http/handler.go.PostMusicLike()"

	t := request.Params.AccessToken

	if t == "" {
		slog.Info("Token empty")
		return api.PostMusicLike500JSONResponse("Token empty"), fmt.Errorf("%s: %w", op, errors.New("Token empty"))
	}

	claims, err := h.uServices.CheckAccessToken(ctx, t)
	if err != nil {
		slog.Error(err.Error())
		return api.PostMusicLike500JSONResponse(err.Error()), fmt.Errorf("%s: %w", op, err)
	}

	l := models.Like{
		UserID:  claims["sub"].(string),
		MusicID: *request.Body.MusicID,
	}

	err = h.likeService.CreateLike(ctx, l)
	if err != nil {
		slog.Error(err.Error())
		return api.PostMusicLike500JSONResponse(err.Error()), fmt.Errorf("%s: %w", op, err)
	}

	return api.PostMusicLike200JSONResponse("Success"), nil
}

func (h Handler) DeleteMusicLike(ctx context.Context, request api.DeleteMusicLikeRequestObject) (api.DeleteMusicLikeResponseObject, error) {
	const op = "./internal/adapters/http/handler.go.DeleteMusic()"

	t := request.Params.AccessToken

	if t == "" {
		slog.Info("Token empty")
		return api.DeleteMusicLike500JSONResponse("Token empty"), fmt.Errorf("%s: %w", op, errors.New("Token empty"))
	}

	claims, err := h.uServices.CheckAccessToken(ctx, t)
	if err != nil {
		slog.Error(err.Error())
		return api.DeleteMusicLike500JSONResponse(err.Error()), fmt.Errorf("%s: %w", op, err)
	}

	l := models.Like{
		UserID:  claims["sub"].(string),
		MusicID: *request.Body.MusicID,
	}

	err = h.likeService.DeleteLike(ctx, l)
	if err != nil {
		slog.Error(err.Error())
		return api.DeleteMusicLike200JSONResponse(err.Error()), fmt.Errorf("%s: %w", op, err)
	}

	return api.DeleteMusicLike200JSONResponse("Success"), nil
}

func (h Handler) GetLikes(ctx context.Context, request api.GetLikesRequestObject) (api.GetLikesResponseObject, error) {
	const op = "./internal/adapters/http/handler.go.GetLikes()"

	t := request.Params.AccessToken

	if t == "" {
		slog.Info("Token empty")
		return api.GetLikes500JSONResponse("Token empty"), fmt.Errorf("%s: %w", op, errors.New("Token empty"))
	}

	claims, err := h.uServices.CheckAccessToken(ctx, t)
	if err != nil {
		slog.Error(err.Error())
		return api.GetLikes500JSONResponse(err.Error()), fmt.Errorf("%s: %w", op, err)
	}

	u := models.User{
		ID: claims["sub"].(string),
	}

	l, err := h.likeService.ReadLikedTracks(ctx, u)
	if err != nil {
		slog.Error(err.Error())
		return api.GetLikes500JSONResponse(err.Error()), fmt.Errorf("%s: %w", op, err)
	}

	lR := make([]api.LikedTrack, 0, len(l))

	for i := range l {
		lR = append(lR, api.LikedTrack{
			MusicId:          &l[i].MusicID,
			UploaderId:       &l[i].MusicUploaderID,
			UploaderUsername: &l[i].UserUsername,
			MusicName:        &l[i].MusicName,
			MusicDuration:    &l[i].MusicDurationSeconds,
			MusicLikes:       &l[i].MusicLikes,
			MusicCover:       &l[i].MusicCover,
			SongUrl:          &l[i].MusicSongURL,
		})
	}

	return api.GetLikes200JSONResponse{
		GetLikedTracksJSONResponse: lR,
	}, nil
}

func (h Handler) MusicUpload(w http.ResponseWriter, r *http.Request) {
	const op = "./internal/adapters/http/handler.go.MusicUpload()"

	t, err := r.Cookie("Access-Token")
	if err != nil {
		slog.Error(err.Error())
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if t.Value == "" {
		slog.Info("Token empty")
		http.Error(w, "Empty token", http.StatusBadRequest)
		return
	}

	claims, err := h.uServices.CheckAccessToken(r.Context(), t.Value)
	if err != nil {
		slog.Error(err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, 50<<20)

	if err := r.ParseMultipartForm(32 << 20); err != nil {
		slog.Error(fmt.Sprint(op, err.Error()))
		http.Error(w, "Failed to parse form: "+err.Error(), http.StatusBadRequest)
		return
	}

	m := make(map[string]models.DataAndCT)

	name := r.FormValue("name")
	if name == "" {
		slog.Warn("Name field empty, please")
		http.Error(w, "Name field on form empty", http.StatusBadRequest)
		return
	}

	file, header, err := r.FormFile("music")
	if err != nil {
		slog.Error(fmt.Sprint(op, err.Error()))
		http.Error(w, "Failed to get file: "+err.Error(), http.StatusBadRequest)
		return
	}
	defer file.Close()

	slog.Info("File:", slog.String("filename", header.Filename), slog.Int64("size", header.Size), slog.String("CT", header.Header.Get("Content-Type")))

	songData, err := io.ReadAll(file)
	if err != nil {
		slog.Error(fmt.Sprint(op, err.Error()))
		http.Error(w, "Failed to read song file", http.StatusInternalServerError)
		return
	}

	slog.Info("First 100 file's ch", slog.String("value", string(songData[:100])))

	m["songData"] = models.DataAndCT{
		Name:        "songData",
		Data:        songData,
		ContentType: header.Header.Get("Content-Type"),
	}

	_, err = file.Seek(0, 0)
	if err != nil {
		slog.Error(err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	skipped := 0
	dec := mp3.NewDecoder(file)

	var f mp3.Frame
	count := 0
	for {
		if err := dec.Decode(&f, &skipped); err != nil {
			slog.Error(err.Error())
			break
		}

		count += 1
	}

	slog.Info("frames", slog.Int("count", count), slog.Int("dur seconds", (count*26)/1000))

	musicCoverUploaded := true
	musicCover, header, err := r.FormFile("music_cover")
	if err != nil && errors.Is(err, http.ErrMissingFile) {
		slog.Info(fmt.Sprint(op, "Missing song's cover file"))
		slog.Error(op + " " + err.Error())
		musicCoverUploaded = false
	} else if err != nil {
		slog.Error("error here")
		musicCoverUploaded = false
		http.Error(w, "Failed to get song's cover file", http.StatusBadRequest)
		return
	}

	songCoverData := make([]byte, 0)
	if musicCoverUploaded {
		slog.Info("Song' cover file:", slog.String("filename", header.Filename), slog.Int64("size", header.Size), slog.String("CT", header.Header.Get("Content-Type")))

		songCoverData, err = io.ReadAll(musicCover)
		if err != nil {
			slog.Error(fmt.Sprint(op, err.Error()))
			http.Error(w, "Failed to read song's cover file", http.StatusInternalServerError)
			return
		}

		slog.Info("First 100 symbols song's cover file", slog.String("value", string(songCoverData[:100])))
	}

	if len(songCoverData) != 0 {
		m["coverData"] = models.DataAndCT{
			Name:        "coverData",
			Data:        songCoverData,
			ContentType: header.Header.Get("Content-Type"),
		}
	}

	music := models.Music{
		Name:        name,
		UploaderID:  claims["sub"].(string),
		DurationSec: int(math.Round((float64(count) * 26.0) / 1000.0)),
	}

	err = h.mService.UploadMusic(r.Context(), m, music)
	if err != nil {
		slog.Error(fmt.Sprint(op, err.Error()))
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(200)
	w.Write([]byte("Success"))
}

func (h Handler) UploadAlbum(w http.ResponseWriter, r *http.Request) {
	const op = "./internal/adapters/http/handler.go.UploadAlbum()"

	t, err := r.Cookie("Access-Token")
	if err != nil {
		slog.Error(err.Error())
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if t.Value == "" {
		slog.Info("Token empty")
		http.Error(w, "Empty token", http.StatusBadRequest)
		return
	}

	claims, err := h.uServices.CheckAccessToken(r.Context(), t.Value)
	if err != nil {
		slog.Error(err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, 50<<20)

	if err := r.ParseMultipartForm(32 << 20); err != nil {
		slog.Error(fmt.Sprint(op, err.Error()))
		http.Error(w, "Failed to parse form: "+err.Error(), http.StatusBadRequest)
		return
	}

	albumName := r.FormValue("album_name")
	if albumName == "" {
		slog.Warn("album_name field empty")
		http.Error(w, "album_name field is required", http.StatusBadRequest)
		return
	}

	uploaderID := claims["sub"].(string)

	var coverData []byte
	coverContentType := ""
	coverFile, _, err := r.FormFile("album_cover")
	if err == nil {
		defer coverFile.Close()
		coverData, err = io.ReadAll(coverFile)
		if err != nil {
			slog.Error(fmt.Sprint(op, err.Error()))
			http.Error(w, "Failed to read cover file", http.StatusInternalServerError)
			return
		}
		coverContentType = "image/jpeg"
	} else if !errors.Is(err, http.ErrMissingFile) {
		slog.Error(fmt.Sprint(op, err.Error()))
		http.Error(w, "Failed to get cover file", http.StatusBadRequest)
		return
	}

	var songs []services.SongUpload
	for i := 0; ; i++ {
		songName := r.FormValue(fmt.Sprintf("song_%d_name", i))
		if songName == "" && i > 0 {
			break
		}
		if songName == "" {
			if i == 0 {
				http.Error(w, "song_0_name is required", http.StatusBadRequest)
				return
			}
			break
		}

		songFile, songHeader, err := r.FormFile(fmt.Sprintf("song_%d_music", i))
		if err != nil {
			slog.Error(fmt.Sprint(op, err.Error()))
			http.Error(w, fmt.Sprintf("Failed to get song_%d_music: %s", i, err.Error()), http.StatusBadRequest)
			return
		}
		defer songFile.Close()

		songData, err := io.ReadAll(songFile)
		if err != nil {
			slog.Error(fmt.Sprint(op, err.Error()))
			http.Error(w, "Failed to read song file", http.StatusInternalServerError)
			return
		}

		songs = append(songs, services.SongUpload{
			Name:        songName,
			Data:        songData,
			ContentType: songHeader.Header.Get("Content-Type"),
		})
	}

	albumID, err := h.aService.UploadAlbum(r.Context(), albumName, uploaderID, coverData, coverContentType, songs)
	if err != nil {
		slog.Error(fmt.Sprint(op, err.Error()))
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(200)
	json.NewEncoder(w).Encode(map[string]string{"album_id": albumID})
}

func (h Handler) PostMusicPlay(ctx context.Context, request api.PostMusicPlayRequestObject) (api.PostMusicPlayResponseObject, error) {
	const op = "./internal/adapters/http/handler.go.PlayMusic()"

	url, err := h.mService.GetPresignURLSong(ctx, *request.Body.MusicId+"-song")
	if err != nil {
		slog.Error(err.Error())
		return api.PostMusicPlay500JSONResponse(err.Error()), fmt.Errorf("%s: %w", op, err)
	}

	return api.PostMusicPlay200JSONResponse{
		PresignUrl: &url,
	}, nil
}

func (h Handler) GetAlbums(ctx context.Context, request api.GetAlbumsRequestObject) (api.GetAlbumsResponseObject, error) {
	const op = "./internal/adapters/http/handler.go.GetAlbums()"

	a, err := h.aService.GetAlbumsInfo(ctx)
	if err != nil {
		slog.Error(fmt.Errorf("%s: %w", op, err).Error())
		return api.GetAlbums500JSONResponse(err.Error()), err
	}

	al := make([]api.Album, 0, len(a))
	for i := range a {
		coverURL := a[i].Cover
		if a[i].Cover != "" {
			url, err := h.aService.GetAlbumCoverPresignURL(ctx, a[i].Cover)
			if err == nil {
				coverURL = url
			}
		}

		al = append(al, api.Album{
			Id: &a[i].ID,
			Name: &a[i].Name,
			UploaderId: &a[i].UploaderID,
			UploaderUsername: &a[i].Username,
			Cover: &coverURL,
		})
	}

	return api.GetAlbums200JSONResponse(al), nil
}

func (h Handler) GetAlbumID(ctx context.Context, request api.GetAlbumIDRequestObject) (api.GetAlbumIDResponseObject, error) {
	const op = "./internal/adapters/http/handler.go.GetAlbumID()"

	a, err := h.aService.GetAlbumsMusic(ctx, request.AlbumID)
	if err != nil {
		slog.Error(fmt.Errorf("%s: %w", op, err).Error())
		return api.GetAlbumID500JSONResponse(err.Error()), err
	}

	al := make([]api.LikedTrack, 0, len(a))
	for i := range a {
		coverURL := a[i].MusicCover
		if a[i].MusicCover != "" {
			url, err := h.aService.GetAlbumCoverPresignURL(ctx, a[i].MusicCover)
			if err == nil {
				coverURL = url
			}
		}

		al = append(al, api.LikedTrack{
			MusicId: &a[i].MusicID,
			MusicName: &a[i].MusicName,
			UploaderId: &a[i].MusicUploaderID,
			UploaderUsername: &a[i].UserUsername,
			MusicLikes: &a[i].MusicLikes,
			MusicCover: &coverURL,
			SongUrl: &a[i].MusicSongURL,
			MusicDuration: &a[i].MusicDurationSeconds,
		})
	}

	return api.GetAlbumID200JSONResponse(al), nil
}