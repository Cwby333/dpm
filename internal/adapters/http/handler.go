package http

import (
	"bufio"
	"bytes"
	"os"
	"slices"
	"strings"

	"context"
	"dpm/internal/models"
	errs "dpm/internal/errors"
	"dpm/internal/services"
	"dpm/pkg/api/v1"
	"encoding/json"
	"strconv"
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
	aService    *services.AlbumsService
	pService    *services.PlaylistService
}

func NewHandler(uService *services.UserService, mService *services.MusicService, lhService *services.ListeningHistoryService, fService *services.FavorService, likeService *services.LikeService, aService *services.AlbumsService, pService *services.PlaylistService) Handler {
	return Handler{
		Mux:         http.NewServeMux(),
		uServices:   uService,
		mService:    mService,
		lhService:   lhService,
		fService:    fService,
		likeService: likeService,
		aService:    aService,
		pService:    pService,
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
	h.Mux.Handle("POST /refresh", corsMiddleware(http.HandlerFunc(h.PostRefresh)))
	h.Mux.Handle("OPTIONS /refresh", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		slog.Info(r.Header.Get("Origin"))
		w.Header().Set("Access-Control-Allow-Origin", r.Header.Get("Origin"))
		w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		w.Header().Set("Access-Control-Allow-Credentials", "true")
		w.WriteHeader(http.StatusOK)
	}))
	h.Mux.Handle("POST /register", corsMiddleware(http.HandlerFunc(h.Register)))
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
	h.Mux.Handle("GET /users", corsMiddleware(wrapGetUsers(strict)))
	h.Mux.Handle("OPTIONS /users", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		slog.Info(r.Header.Get("Origin"))
		w.Header().Set("Access-Control-Allow-Origin", r.Header.Get("Origin"))
		w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		w.Header().Set("Access-Control-Allow-Credentials", "true")
		w.WriteHeader(http.StatusOK)
	}))
	h.Mux.Handle("GET /users/{userID}", corsMiddleware(wrapGetUserProfile(strict)))
	h.Mux.Handle("OPTIONS /users/{userID}", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		slog.Info(r.Header.Get("Origin"))
		w.Header().Set("Access-Control-Allow-Origin", r.Header.Get("Origin"))
		w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		w.Header().Set("Access-Control-Allow-Credentials", "true")
		w.WriteHeader(http.StatusOK)
	}))
	h.Mux.Handle("GET /users/{userID}/albums", corsMiddleware(wrapGetUserAlbums(strict)))
	h.Mux.Handle("OPTIONS /users/{userID}/albums", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		slog.Info(r.Header.Get("Origin"))
		w.Header().Set("Access-Control-Allow-Origin", r.Header.Get("Origin"))
		w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		w.Header().Set("Access-Control-Allow-Credentials", "true")
		w.WriteHeader(http.StatusOK)
	}))
	h.Mux.Handle("GET /users/{userID}/playlists", corsMiddleware(wrapGetUserPlaylists(strict)))
	h.Mux.Handle("OPTIONS /users/{userID}/playlists", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
	h.Mux.Handle("GET /album", corsMiddleware(wrapGetAlbums(strict)))
	h.Mux.Handle("POST /album", corsMiddleware(http.HandlerFunc(h.UploadAlbum)))
	h.Mux.Handle("GET /album/my", corsMiddleware(wrapGetMyAlbums(strict)))
	h.Mux.Handle("OPTIONS /album", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		slog.Info(r.Header.Get("Origin"))
		w.Header().Set("Access-Control-Allow-Origin", r.Header.Get("Origin"))
		w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		w.Header().Set("Access-Control-Allow-Credentials", "true")
		w.WriteHeader(http.StatusOK)
	}))
	h.Mux.Handle("GET /album/{albumID}", corsMiddleware(wrapGetAlbum(strict)))
	h.Mux.Handle("DELETE /album/{albumID}", corsMiddleware(wrapDeleteAlbum(strict)))
	h.Mux.Handle("PATCH /album/{albumID}", corsMiddleware(http.HandlerFunc(h.UpdateAlbum)))
	h.Mux.Handle("OPTIONS /album/{albumID}", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		slog.Info(r.Header.Get("Origin"))
		w.Header().Set("Access-Control-Allow-Origin", r.Header.Get("Origin"))
		w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE, PATCH")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		w.Header().Set("Access-Control-Allow-Credentials", "true")
		w.WriteHeader(http.StatusOK)
	}))
	h.Mux.Handle("POST /playlist", corsMiddleware(http.HandlerFunc(h.UploadPlaylist)))
	h.Mux.Handle("OPTIONS /playlist", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		slog.Info(r.Header.Get("Origin"))
		w.Header().Set("Access-Control-Allow-Origin", r.Header.Get("Origin"))
		w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		w.Header().Set("Access-Control-Allow-Credentials", "true")
		w.WriteHeader(http.StatusOK)
	}))
	h.Mux.Handle("GET /playlist/my/likes", corsMiddleware(wrapGetLikedPlaylists(strict)))
	h.Mux.Handle("OPTIONS /playlist/my/likes", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		slog.Info(r.Header.Get("Origin"))
		w.Header().Set("Access-Control-Allow-Origin", r.Header.Get("Origin"))
		w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		w.Header().Set("Access-Control-Allow-Credentials", "true")
		w.WriteHeader(http.StatusOK)
	}))
	h.Mux.Handle("POST /playlist/like", corsMiddleware(wrapPostLikePlaylist(strict)))
	h.Mux.Handle("DELETE /playlist/like", corsMiddleware(wrapDeleteLikePlaylist(strict)))
	h.Mux.Handle("OPTIONS /playlist/like", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		slog.Info(r.Header.Get("Origin"))
		w.Header().Set("Access-Control-Allow-Origin", r.Header.Get("Origin"))
		w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		w.Header().Set("Access-Control-Allow-Credentials", "true")
		w.WriteHeader(http.StatusOK)
	}))
	h.Mux.Handle("GET /album/my/likes", corsMiddleware(wrapGetLikedAlbums(strict)))
	h.Mux.Handle("OPTIONS /album/my/likes", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		slog.Info(r.Header.Get("Origin"))
		w.Header().Set("Access-Control-Allow-Origin", r.Header.Get("Origin"))
		w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		w.Header().Set("Access-Control-Allow-Credentials", "true")
		w.WriteHeader(http.StatusOK)
	}))
	h.Mux.Handle("POST /album/like", corsMiddleware(wrapPostLikeAlbum(strict)))
	h.Mux.Handle("DELETE /album/like", corsMiddleware(wrapDeleteLikeAlbum(strict)))
	h.Mux.Handle("OPTIONS /album/like", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		slog.Info(r.Header.Get("Origin"))
		w.Header().Set("Access-Control-Allow-Origin", r.Header.Get("Origin"))
		w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		w.Header().Set("Access-Control-Allow-Credentials", "true")
		w.WriteHeader(http.StatusOK)
	}))
	h.Mux.Handle("GET /playlist/my", corsMiddleware(wrapGetMyPlaylists(strict)))
	h.Mux.Handle("OPTIONS /playlist/my", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		slog.Info(r.Header.Get("Origin"))
		w.Header().Set("Access-Control-Allow-Origin", r.Header.Get("Origin"))
		w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		w.Header().Set("Access-Control-Allow-Credentials", "true")
		w.WriteHeader(http.StatusOK)
	}))
	h.Mux.Handle("GET /playlist/public", corsMiddleware(wrapGetPublicPlaylists(strict)))
	h.Mux.Handle("OPTIONS /playlist/public", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		slog.Info(r.Header.Get("Origin"))
		w.Header().Set("Access-Control-Allow-Origin", r.Header.Get("Origin"))
		w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		w.Header().Set("Access-Control-Allow-Credentials", "true")
		w.WriteHeader(http.StatusOK)
	}))
	h.Mux.Handle("GET /playlist/{playlistID}", corsMiddleware(wrapGetPlaylistTracks(strict)))
	h.Mux.Handle("DELETE /playlist/{playlistID}", corsMiddleware(wrapDeletePlaylist(strict)))
	h.Mux.Handle("PATCH /playlist/{playlistID}", corsMiddleware(http.HandlerFunc(h.UpdatePlaylist)))
	h.Mux.Handle("OPTIONS /playlist/{playlistID}", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		slog.Info(r.Header.Get("Origin"))
		w.Header().Set("Access-Control-Allow-Origin", r.Header.Get("Origin"))
		w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		w.Header().Set("Access-Control-Allow-Credentials", "true")
		w.WriteHeader(http.StatusOK)
	}))
	h.Mux.Handle("POST /playlist/{playlistID}/add", corsMiddleware(wrapAddMusicToPlaylist(strict)))
	h.Mux.Handle("OPTIONS /playlist/{playlistID}/add", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		slog.Info(r.Header.Get("Origin"))
		w.Header().Set("Access-Control-Allow-Origin", r.Header.Get("Origin"))
		w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		w.Header().Set("Access-Control-Allow-Credentials", "true")
		w.WriteHeader(http.StatusOK)
	}))
	h.Mux.Handle("GET /music/my", corsMiddleware(wrapGetMusicMy(strict)))
	h.Mux.Handle("OPTIONS /music/my", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		slog.Info(r.Header.Get("Origin"))
		w.Header().Set("Access-Control-Allow-Origin", r.Header.Get("Origin"))
		w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		w.Header().Set("Access-Control-Allow-Credentials", "true")
		w.WriteHeader(http.StatusOK)
	}))
	h.Mux.Handle("POST /music/inc-lis-count", corsMiddleware(http.HandlerFunc(strict.PostMusicIncLisCount)))
	h.Mux.Handle("OPTIONS /music/inc-lis-count", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		slog.Info(r.Header.Get("Origin"))
		w.Header().Set("Access-Control-Allow-Origin", r.Header.Get("Origin"))
		w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		w.Header().Set("Access-Control-Allow-Credentials", "true")
		w.WriteHeader(http.StatusOK)
	}))
}

func wrapGetAlbums(strict api.ServerInterface) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		params := api.GetAlbumsParams{}

		if v := r.URL.Query().Get("likes_min"); v != "" {
            if n, parseErr := strconv.Atoi(v); parseErr == nil {
                params.LikesMin = &n
            }
        }
        if v := r.URL.Query().Get("likes_max"); v != "" {
            if n, parseErr := strconv.Atoi(v); parseErr == nil {
                params.LikesMax = &n
            }
        }

		strict.GetAlbums(w, r, params)
	}
}

func wrapGetPublicPlaylists(strict api.ServerInterface) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		params := api.GetPublicPlaylistsParams{}

		if v := r.URL.Query().Get("likes_min"); v != "" {
            if n, parseErr := strconv.Atoi(v); parseErr == nil {
                params.LikesMin = &n
            }
        }
        if v := r.URL.Query().Get("likes_max"); v != "" {
            if n, parseErr := strconv.Atoi(v); parseErr == nil {
                params.LikesMax = &n
            }
        }

		slog.Info(fmt.Sprintf("%+v", params))

		strict.GetPublicPlaylists(w, r, params)
	}
}

func wrapGetLikedPlaylists(strict api.ServerInterface) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		params := api.GetPlaylistMyLikesParams{}
		
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

		params.AccessToken = c.Value

		if v := r.URL.Query().Get("likes_min"); v != "" {
            if n, parseErr := strconv.Atoi(v); parseErr == nil {
                params.LikesMin = &n
            }
        }
        if v := r.URL.Query().Get("likes_max"); v != "" {
            if n, parseErr := strconv.Atoi(v); parseErr == nil {
                params.LikesMax = &n
            }
        }

		strict.GetPlaylistMyLikes(w, r, params)
	}
}

func wrapPostLikePlaylist(strict api.ServerInterface) http.HandlerFunc {
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

		strict.PostPlaylistLike(w, r, api.PostPlaylistLikeParams{
			AccessToken: c.Value,
		})
	}
}

func wrapDeleteLikePlaylist(strict api.ServerInterface) http.HandlerFunc {
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

		strict.DeletePlaylistLike(w, r, api.DeletePlaylistLikeParams{
			AccessToken: c.Value,
		})
	}
}

func wrapGetLikedAlbums(strict api.ServerInterface) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		params := api.GetAlbumMyLikesParams{}
		
		c, err := r.Cookie("Access-Token")
		if err != nil {
			slog.Info("wrapGetLikedAlbums")
			slog.Error(err.Error())
			c = &http.Cookie{Value: ""}
		}

		params.AccessToken = c.Value

		if v := r.URL.Query().Get("likes_min"); v != "" {
            if n, parseErr := strconv.Atoi(v); parseErr == nil {
                params.LikesMin = &n
            }
        }
        if v := r.URL.Query().Get("likes_max"); v != "" {
            if n, parseErr := strconv.Atoi(v); parseErr == nil {
                params.LikesMax = &n
            }
        }

		strict.GetAlbumMyLikes(w, r, params)
	}
}

func wrapPostLikeAlbum(strict api.ServerInterface) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		c, err := r.Cookie("Access-Token")
		if err != nil {
			slog.Info("wrapPostLikeAlbum")
			slog.Error(err.Error())
			c = &http.Cookie{Value: ""}
		}
		strict.PostAlbumLike(w, r, api.PostAlbumLikeParams{
			AccessToken: c.Value,
		})
	}
}

func wrapDeleteLikeAlbum(strict api.ServerInterface) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		c, err := r.Cookie("Access-Token")
		if err != nil {
			slog.Info("wrapDeleteLikeAlbum")
			slog.Error(err.Error())
			c = &http.Cookie{Value: ""}
		}
		strict.DeleteAlbumLike(w, r, api.DeleteAlbumLikeParams{
			AccessToken: c.Value,
		})
	}
}

func wrapDeleteAlbum(strict api.ServerInterface) http.HandlerFunc {
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

		strict.DeleteAlbumAlbumID(w, r, r.PathValue("albumID"), api.DeleteAlbumAlbumIDParams{AccessToken: c.Value})
	}
}

func wrapGetMusicMy(strict api.ServerInterface) http.HandlerFunc {
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

		strict.GetMusicMy(w, r, api.GetMusicMyParams{AccessToken: c.Value})
	}
}

func wrapGetMyAlbums(strict api.ServerInterface) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		params := api.GetAlbumMyParams{}
		
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

		params.AccessToken = c.Value

		if v := r.URL.Query().Get("likes_min"); v != "" {
            if n, parseErr := strconv.Atoi(v); parseErr == nil {
                params.LikesMin = &n
            }
        }
        if v := r.URL.Query().Get("likes_max"); v != "" {
            if n, parseErr := strconv.Atoi(v); parseErr == nil {
                params.LikesMax = &n
            }
        }

		strict.GetAlbumMy(w, r, params)
	}
}

func wrapGetAlbum(strict api.ServerInterface) http.HandlerFunc {
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

		strict.GetAlbumID(w, r, r.PathValue("albumID"), api.GetAlbumIDParams{AccessToken: &c.Value})
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
		
		var params api.GetAllMusicParams
		params.AccessToken = &c.Value

		if v := r.URL.Query().Get("likes_min"); v != "" {
            if n, parseErr := strconv.Atoi(v); parseErr == nil {
                params.LikesMin = &n
            }
        }
        if v := r.URL.Query().Get("likes_max"); v != "" {
            if n, parseErr := strconv.Atoi(v); parseErr == nil {
                params.LikesMax = &n
            }
        }
        if v := r.URL.Query().Get("dur_min"); v != "" {
            if n, parseErr := strconv.Atoi(v); parseErr == nil {
                params.DurMin = &n
            }
        }
        if v := r.URL.Query().Get("dur_max"); v != "" {
            if n, parseErr := strconv.Atoi(v); parseErr == nil {
                params.DurMax = &n
            }
        }
		if v := r.URL.Query().Get("lis_count_min"); v != "" {
            if n, parseErr := strconv.Atoi(v); parseErr == nil {
                params.LisCountMin = &n
            }
        }
		if v := r.URL.Query().Get("lis_count_max"); v != "" {
            if n, parseErr := strconv.Atoi(v); parseErr == nil {
                params.LisCountMax = &n
            }
        }

		strict.GetAllMusic(w, r, params)
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

func wrapGetUserProfile(strict api.ServerInterface) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		strict.GetUserProfile(w, r, r.PathValue("userID"))
	}
}

func wrapGetUserAlbums(strict api.ServerInterface) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		strict.GetUserAlbums(w, r, r.PathValue("userID"))
	}
}

func wrapGetUserPlaylists(strict api.ServerInterface) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		strict.GetUserPlaylists(w, r, r.PathValue("userID"))
	}
}

func wrapGetUsers(strict api.ServerInterface) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		params := api.GetUsersParams{}

		if v := r.URL.Query().Get("lis_count_min"); v != "" {
			if n, err := strconv.Atoi(v); err == nil {
				params.LisCountMin = &n
			}
		}
		if v := r.URL.Query().Get("lis_count_max"); v != "" {
			if n, err := strconv.Atoi(v); err == nil {
				params.LisCountMax = &n
			}
		}
		if v := r.URL.Query().Get("reg_date_min"); v != "" {
			if t, err := time.Parse(time.RFC3339, v); err == nil {
				params.RegisterAtMin = &t
			}
		}
		if v := r.URL.Query().Get("reg_date_max"); v != "" {
			if t, err := time.Parse(time.RFC3339, v); err == nil {
				params.RegisterAtMax = &t
			}
		}
		if v := r.URL.Query().Get("likes_count_min"); v != "" {
			if t, err := strconv.Atoi(v); err == nil {
				params.LikesCountMin = &t
			}
		}
		if v := r.URL.Query().Get("likes_count_max"); v != "" {
			if t, err := strconv.Atoi(v); err == nil {
				params.LikesCountMax = &t
			}
		}
		if v := r.URL.Query().Get("favor_count_min"); v != "" {
			if t, err := strconv.Atoi(v); err == nil {
				params.FavorCountMin = &t
			}
		}
		if v := r.URL.Query().Get("favor_count_max"); v != "" {
			if t, err := strconv.Atoi(v); err == nil {
				params.FavorCountMax = &t	
			}
		}

		strict.GetUsers(w, r, params)
	}
}

func wrapGetMyPlaylists(strict api.ServerInterface) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		params := api.GetMyPlaylistsParams{}

		c, err := r.Cookie("Access-Token")
		if err != nil {
			slog.Error(err.Error())
			c = &http.Cookie{}
		}

		params.AccessToken = c.Value

		if v := r.URL.Query().Get("likes_min"); v != "" {
            if n, parseErr := strconv.Atoi(v); parseErr == nil {
                params.LikesMin = &n
            }
        }
        if v := r.URL.Query().Get("likes_max"); v != "" {
            if n, parseErr := strconv.Atoi(v); parseErr == nil {
                params.LikesMax = &n
            }
        }

		strict.GetMyPlaylists(w, r, params)
	}
}

func wrapGetPlaylistTracks(strict api.ServerInterface) http.HandlerFunc {
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

		strict.GetPlaylistTracks(w, r, r.PathValue("playlistID"), api.GetPlaylistTracksParams{AccessToken: &c.Value})
	}
}

func wrapAddMusicToPlaylist(strict api.ServerInterface) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		c, err := r.Cookie("Access-Token")
		if err != nil {
			slog.Error(err.Error())
			c = &http.Cookie{}
		}
		strict.AddMusicToPlaylist(w, r, r.PathValue("playlistID"), api.AddMusicToPlaylistParams{AccessToken: c.Value})
	}
}

func wrapDeletePlaylist(strict api.ServerInterface) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		c, err := r.Cookie("Access-Token")
		if err != nil {
			slog.Error(err.Error())
			c = &http.Cookie{}
		}
		strict.DeletePlaylist(w, r, r.PathValue("playlistID"), api.DeletePlaylistParams{AccessToken: c.Value})
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
		if errors.Is(err, errs.ErrBadUsernameOrPassword) {
			slog.Debug("Bad username or password")
			http.Error(w, "Bad username or password", http.StatusBadRequest)
		}
		slog.Error(fmt.Errorf("%s: %w", op, err).Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return		
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "Access-Token",
		Value:    access.Sign,
		Expires:  time.Now().Add(time.Second * 24 * 3),
		Secure:   true,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteNoneMode,
		Domain:   "",
	})
	http.SetCookie(w, &http.Cookie{
		Name:     "Refresh-Token",
		Value:    refresh.Sign,
		Expires:  time.Now().Add(time.Hour * 24 * 7),
		Secure:   true,
		HttpOnly: true,
		Domain:   "",
		Path:     "/refresh",
		SameSite: http.SameSiteNoneMode,
	})
	http.SetCookie(w, &http.Cookie{
		Name:     "Refresh-Token-Logout",
		Value:    refresh.Sign,
		Expires:  time.Now().Add(time.Hour * 24 * 7),
		Secure:   true,
		HttpOnly: true,
		Domain:   "",
		Path:     "/logout",
		SameSite: http.SameSiteNoneMode,
	})

	w.WriteHeader(200)
}

func (h Handler) PostRefresh(w http.ResponseWriter, r *http.Request) {
	const op = "./internal/adapters/http/handler.go.PostRefresh()"

	slog.Debug("PostRefresh request")
	
	refreshT, err := r.Cookie("Refresh-Token")
	if err != nil {
		slog.Error(err.Error())
		http.Error(w, "Missing refresh token", http.StatusBadRequest)
		return
	}

	if refreshT.Value == "" {
		slog.Debug("PostRefresh refresh empty")
		http.Error(w, "Refresh token empty", http.StatusUnauthorized)
		return
	}

	claims, err := h.uServices.CheckRefreshToken(r.Context(), refreshT.Value)
	if err != nil {
		slog.Error(err.Error())
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	access, refresh, err := h.uServices.RefreshTokens(r.Context(), models.User{ID: claims["sub"].(string)})
	if err != nil {
		slog.Error(err.Error())
		http.Error(w, "Server error", http.StatusInternalServerError)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "Access-Token",
		Value:    access.Sign,
		Expires:  time.Now().Add(time.Hour * 24 * 3),
		Secure:   true,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteNoneMode,
		Domain:   "",
	})
	http.SetCookie(w, &http.Cookie{
		Name:     "Refresh-Token",
		Value:    refresh.Sign,
		Expires:  time.Now().Add(time.Hour * 24 * 7),
		Secure:   true,
		HttpOnly: true,
		Domain:   "",
		Path:     "/refresh",
		SameSite: http.SameSiteNoneMode,
	})
	http.SetCookie(w, &http.Cookie{
		Name:     "Refresh-Token-Logout",
		Value:    refresh.Sign,
		Expires:  time.Now().Add(time.Hour * 24 * 7),
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

// func (h Handler) Register(ctx context.Context, request api.RegisterRequestObject) (api.RegisterResponseObject, error) {
// 	const op = "./internal/adapters/http/handlers.go.Login()"

// 	u := models.User{
// 		Username:       *request.Body.Username,
// 		HashPsw:        *request.Body.Password,
// 		PrivateProfile: request.Body.PrivateProfile != nil && *request.Body.PrivateProfile,
// 	}

// 	err := h.uServices.RegisterUser(ctx, u)
// 	if err != nil {
// 		slog.Error(fmt.Errorf("%s: %w", op, err).Error())
// 		msg := err.Error()
// 		return api.Register500JSONResponse{
// 			Message: &msg,
// 		}, err
// 	}

// 	msg := "Success register"
// 	return api.Register200JSONResponse{
// 		Message: &msg,
// 	}, nil
// }

func (h Handler) Register(w http.ResponseWriter, r *http.Request) {
	const op = "./internal/adapters/http/handler.go.Register()"

	r.Body = http.MaxBytesReader(w, r.Body, 50<<20)

	if err := r.ParseMultipartForm(32<<20); err != nil {
		slog.Error(fmt.Sprint(op, err.Error()))
		http.Error(w, "Failed to parse form: "+err.Error(), http.StatusBadRequest)
		return
	}

	username := r.FormValue("username")
	if username == "" {
		slog.Warn("Register: missing username")
		http.Error(w, "Missing username", http.StatusBadRequest)
		return
	}

	psw := r.FormValue("password")
	if psw == "" {
		slog.Warn("Register: missing password")
		http.Error(w, "Missing password", http.StatusBadRequest)
		return
	}

	p := false
	private := r.FormValue("private_profile")
	if private == "" {
		slog.Warn("Register: missing private profile")
	}
	if private == "true" {
		p = true
	}

	file, header, err := r.FormFile("image")
	if err != nil {
		slog.Error(fmt.Sprint(op, err.Error()))
	}
	if file != nil {
		defer file.Close()
	}

	avatarData := []byte{}
	if file != nil {
		slog.Info("File:", slog.String("filename", header.Filename), slog.Int64("size", header.Size), slog.String("CT", header.Header.Get("Content-Type")))
	
		avatarData, err = io.ReadAll(file)
		if err != nil {
			slog.Error(fmt.Sprint(op, err.Error()))
			http.Error(w, "Failed to read song file", http.StatusInternalServerError)
			return
		}
	}

	if len(avatarData) == 0 {
		slog.Warn("Register avatar empty")
	}

	ct := "image/jpg"
	if header != nil {
		ct = header.Header.Get("Content-Type")
	}

	err = h.uServices.RegisterUser(r.Context(), models.User{
		Username: username,
		HashPsw: psw,
		PrivateProfile: p,
	}, avatarData, ct)
	if err != nil {
		slog.Error(err.Error())
		http.Error(w, "Server error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(200)	
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

	mf := models.MusicFilterQuery{
		LikeMin: request.Params.LikesMin,
		LikeMax: request.Params.LikesMax,
		DurMin: request.Params.DurMin,
		DurMax: request.Params.DurMax,
		LisCountMin: request.Params.LisCountMin,
		LisCountMax: request.Params.LisCountMax,
	}
	slog.Info(fmt.Sprintf("%v", mf))

	p, l, err := h.mService.GetMusicSQ(ctx, mf, u.ID)
	if err != nil {
		slog.Error(err.Error())
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
			UploaderUsername: p[i].UploaderUsername,
			Likes:           p[i].Likes,
			DurationSeconds: p[i].DurationSec,
			MusicCover:      &urlCover,
			SongUrl:         p[i].SongURL,
			ListeningCount:  &p[i].ListeningCount,
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
					UploaderUsername: product.UploaderUsername,
					Name:            product.Name,
					Likes:           product.Likes,
					DurationSeconds: product.DurationSec,
					MusicCover:      &urlCover,
					SongUrl:         product.SongURL,
					ListeningCount:  &product.ListeningCount,
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
		slog.Info("AddListeningHistory empty")
		return api.AddListeningToLH401Response{}, nil
	}

	claims, err := h.uServices.CheckAccessToken(ctx, t)
	if err != nil {
		slog.Error(err.Error())
		return api.AddListeningToLH401Response{}, nil
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
		return api.GetLH401Response{}, nil
	}

	claims, err := h.uServices.CheckAccessToken(ctx, t)
	if err != nil {
		slog.Error(err.Error())
		return api.GetLH401Response{}, nil
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
		if lh[i].MusicCover != "" {
			url, err := h.mService.GetPresignURLSong(ctx, lh[i].MusicCover)
			if err != nil {
				slog.Error(err.Error())
				continue
			}

			lh[i].MusicCover = url
		}

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
			ListeningCount:  &lh[i].MusicListeningCount,
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
		slog.Info("DeleteListeningFromLH token empty")
		return api.DeleteListeningFromLH401Response{}, nil
	}

	claims, err := h.uServices.CheckAccessToken(ctx, t)
	if err != nil {
		slog.Error(err.Error())
		return api.DeleteListeningFromLH401Response{}, nil
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
		slog.Info("AddFavor token empty")
		return api.AddFavor401Response{}, nil
	}

	claims, err := h.uServices.CheckAccessToken(ctx, t)
	if err != nil {
		slog.Error(err.Error())
		return api.AddFavor401Response{}, nil
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
		slog.Info("getfavor token empty")
		return api.GetFavor401Response{}, nil
	}

	claims, err := h.uServices.CheckAccessToken(ctx, t)
	if err != nil {
		slog.Error(err.Error())
		return api.GetFavor401Response{}, nil
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
		if favor[i].MusicCover != "" {
			url, err := h.mService.GetPresignURLSong(ctx, favor[i].MusicCover)
			if err != nil {
				slog.Error(err.Error())
				continue
			}

			favor[i].MusicCover = url
		}

		fAPI = append(fAPI, api.Favor{
			Id:              favor[i].MusicID,
			Name:            favor[i].MusicName,
			MusicCover:      &favor[i].MusicCover,
			SongUrl:         favor[i].MusicSongURL,
			UploaderId:      favor[i].MusicUploaderID,
			Likes:           favor[i].MusicLikes,
			ListeningCount:  &favor[i].MusicListeningCount,
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
		return api.DeleteFavor401Response{}, nil
	}

	claims, err := h.uServices.CheckAccessToken(ctx, t)
	if err != nil {
		slog.Error(err.Error())
		return api.DeleteFavor401Response{}, nil
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

func (h Handler) GetUsers(ctx context.Context, request api.GetUsersRequestObject) (api.GetUsersResponseObject, error) {
	const op = "./internal/adapters/http/handler.go.GetUsers()"

	uf := models.UserFilter{
		MinRegisterAt: request.Params.RegisterAtMin,
		MinLikes: request.Params.LikesCountMin,
		MinFavorCount: request.Params.FavorCountMin,
		MinLisCount: request.Params.LisCountMin,
		MaxRegisterAt: request.Params.RegisterAtMax,
		MaxLikes: request.Params.LikesCountMax,
		MaxFavorCount: request.Params.FavorCountMax,
		MaxLisCount: request.Params.LisCountMax,
	}

	users, err := h.uServices.GetPublicUsers(ctx, uf)
	if err != nil {
		slog.Error(fmt.Errorf("%s: %w", op, err).Error())
		return api.GetUsers500Response{}, err
	}

	resp := make(api.GetUsers200JSONResponse, 0, len(users))
	for i := range users {
		id := users[i].ID
		uname := users[i].Username
		sTime := fmt.Sprint(users[i].RegisterAt)
		likes := users[i].Likes
		lCount := users[i].ListeningCount
		fCount := users[i].FavorCount
		resp = append(resp, api.UserPublic{
			Id:             &id,
			Username:       &uname,
			Image: &users[i].Image,
			RegisterAt:     &sTime,
			Likes:          &likes,
			ListeningCount: &lCount,
			FavorCount:     &fCount,
		})
	}

	return resp, nil
}

func (h Handler) GetUserProfile(ctx context.Context, request api.GetUserProfileRequestObject) (api.GetUserProfileResponseObject, error) {
	const op = "./internal/adapters/http/handler.go.GetUserProfile()"

	slog.Info("GetUserProfile")

	user, tracks, err := h.uServices.GetPublicUserProfile(ctx, request.UserID)
	if err != nil {
		slog.Error(fmt.Errorf("%s: %w", op, err).Error())
		return api.GetUserProfile404Response{}, nil
	}

	sTime := fmt.Sprint(user.RegisterAt)
	apiUser := api.UserPublic{
		Id:             &user.ID,
		Username:       &user.Username,
		RegisterAt:     &sTime,
		Image: &user.Image,
		Likes:          &user.Likes,
		ListeningCount: &user.ListeningCount,
		FavorCount:     &user.FavorCount,
	}

	apiTracks := make([]api.LikedTrack, 0, len(tracks))
	for i := range tracks {
		mid := tracks[i].MusicID
		mname := tracks[i].MusicName
		mcover := tracks[i].MusicCover
		mlikes := tracks[i].MusicLikes
		mdur := tracks[i].MusicDurationSeconds
		surl := tracks[i].MusicSongURL
		uid := tracks[i].MusicUploaderID
		uuname := tracks[i].UserUsername
		slog.Info(fmt.Sprintf("%s: %v", "listening_count", tracks[i].MusicListeningCount))
		apiTracks = append(apiTracks, api.LikedTrack{
				MusicId:          &mid,
				MusicName:        &mname,
				MusicCover:       &mcover,
				MusicLikes:       &mlikes,
				MusicDuration:    &mdur,
				SongUrl:          &surl,
				UploaderId:       &uid,
				UploaderUsername: &uuname,
				ListeningCount:  &tracks[i].MusicListeningCount,
			})
	}

	return api.GetUserProfile200JSONResponse{
		User:   &apiUser,
		Tracks: &apiTracks,
	}, nil
}

func (h Handler) GetUserAlbums(ctx context.Context, request api.GetUserAlbumsRequestObject) (api.GetUserAlbumsResponseObject, error) {
	const op = "./internal/adapters/http/handler.go.GetUserAlbums()"

	slog.Info("GetUserAlbums", "userID", request.UserID)

	u, err := h.uServices.Pg.ReadUser(ctx, models.User{ID: request.UserID})
	if err != nil {
		slog.Error(fmt.Errorf("%s: %w", op, err).Error())
		return api.GetUserAlbums404Response{}, nil
	}
	if u.PrivateProfile {
		return api.GetUserAlbums404Response{}, nil
	}

	albums, err := h.aService.GetUserAlbums(ctx, request.UserID)
	if err != nil {
		slog.Error(fmt.Errorf("%s: %w", op, err).Error())
		return api.GetUserAlbums500Response{}, nil
	}

	apiAlbums := make([]api.Album, 0, len(albums))
	for i := range albums {
		name := albums[i].Name
		id := albums[i].ID
		uid := albums[i].UploaderID
		uname := albums[i].Username
		cover := albums[i].Cover
		apiAlbums = append(apiAlbums, api.Album{
			Id:               &id,
			Name:             &name,
			UploaderId:       &uid,
			UploaderUsername: &uname,
			Cover:            &cover,
		})
	}

	return api.GetUserAlbums200JSONResponse(apiAlbums), nil
}

func (h Handler) GetUserPlaylists(ctx context.Context, request api.GetUserPlaylistsRequestObject) (api.GetUserPlaylistsResponseObject, error) {
	const op = "./internal/adapters/http/handler.go.GetUserPlaylists()"

	u, err := h.uServices.Pg.ReadUser(ctx, models.User{ID: request.UserID})
	if err != nil {
		return api.GetUserPlaylists404Response{}, nil
	}
	if u.PrivateProfile {
		return api.GetUserPlaylists404Response{}, nil
	}

	pl, err := h.pService.GetPublicPlaylistsByID(ctx, request.UserID)
	if err != nil {
		return api.GetUserPlaylists500Response{}, nil
	}

	resp := make(api.GetUserPlaylists200JSONResponse, 0, len(pl))
	for i := range pl {
		id := pl[i].ID
		name := pl[i].Name
		uploaderID := pl[i].UploaderID
		cover := pl[i].Cover
		private := pl[i].Private
		username := pl[i].Username
		resp = append(resp, api.PlaylistInfo{
			Id:         &id,
			Name:       &name,
			UploaderId: &uploaderID,
			Cover:      &cover,
			Private:    &private,
			Username:   &username,
		})
	}
	return resp, nil
}

func (h Handler) GetProfile(ctx context.Context, request api.GetProfileRequestObject) (api.GetProfileResponseObject, error) {
	const op = "./internal/adapters/http/handler.go.GetProfile()"

	t := request.Params.AccessToken

	if t == "" {
		slog.Info("GET /profile token empty")
		return api.GetProfile401Response{}, nil
	}

	claims, err := h.uServices.CheckAccessToken(ctx, t)
	if err != nil {
		slog.Error(err.Error())
		return api.GetProfile401Response{}, nil
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
			Id: &us.ID,
			Username:       &us.Username,
			RegisterAt:     &sTime,
			Image: &us.Image,
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
		return api.PostMusicLike401Response{}, nil
	}

	claims, err := h.uServices.CheckAccessToken(ctx, t)
	if err != nil {
		slog.Error(err.Error())
		return api.PostMusicLike401Response{}, nil
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
		return api.DeleteMusicLike401Response{}, nil
	}

	claims, err := h.uServices.CheckAccessToken(ctx, t)
	if err != nil {
		slog.Error(err.Error())
		return api.DeleteMusicLike401Response{}, nil
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
		return api.GetLikes401Response{}, nil
	}

	claims, err := h.uServices.CheckAccessToken(ctx, t)
	if err != nil {
		slog.Error(err.Error())
		return api.GetLikes401Response{}, nil
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
		if l[i].MusicCover != "" {
			url, err := h.mService.GetPresignURLSong(ctx, l[i].MusicCover)
			if err != nil {
				slog.Error(err.Error())
				continue
			}

			l[i].MusicCover = url
		}

		lR = append(lR, api.LikedTrack{
				MusicId:          &l[i].MusicID,
				UploaderId:       &l[i].MusicUploaderID,
				UploaderUsername: &l[i].UserUsername,
				MusicName:        &l[i].MusicName,
				MusicDuration:    &l[i].MusicDurationSeconds,
				MusicLikes:       &l[i].MusicLikes,
				MusicCover:       &l[i].MusicCover,
				SongUrl:          &l[i].MusicSongURL,
				ListeningCount:  &l[i].MusicListeningCount,
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
		http.Error(w, "Not authenticate", 401)
		return
	}

	if t.Value == "" {
		slog.Info("Token empty")
		http.Error(w, "Not authenticate", 401)
		return
	}

	claims, err := h.uServices.CheckAccessToken(r.Context(), t.Value)
	if err != nil {
		slog.Error(err.Error())
		http.Error(w, err.Error(), 401)
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

	err = h.mService.UploadHLSMusic(r.Context(), m, music)
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
		http.Error(w, "Unauthorized", 401)
		return
	}

	if t.Value == "" {
		slog.Info("Token empty")
		http.Error(w, "Unauthorized", 401)
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

func (h Handler) GetAlbumMy(ctx context.Context, request api.GetAlbumMyRequestObject) (api.GetAlbumMyResponseObject, error) {
	const op = "./internal/adapters/http/handler.go.GetAlbumMy"

	t := request.Params.AccessToken

	if t == "" {
		slog.Warn("GetAlbumMy token missing")
		return api.GetAlbumMy401Response{}, nil
	}

	claims, err := h.uServices.CheckAccessToken(ctx, t)
	if err != nil {
		slog.Warn(fmt.Sprintf("%s: %s", "GetAlbumsMy check token", err.Error()))
		return api.GetAlbumMy401Response{}, nil
	}

	id := claims["sub"].(string)

	af := models.AlbumFilter{
		LikesMin: request.Params.LikesMin,
		LikesMax: request.Params.LikesMax,
	}

	a, err := h.aService.GetUploadedByUserAlbums(ctx, id, af)
	if err != nil {
		slog.Error(fmt.Sprintf("%s: %s", "GetAlbumsMy get", err.Error()))
		return api.GetAlbumMy500Response{}, err
	}

	ar := make(api.GetAlbumJSONResponse, 0, len(a))
	for i := range a {
		ar = append(ar, struct{Cover *string "json:\"cover,omitempty\""; Id *string "json:\"id,omitempty\""; Name *string "json:\"name,omitempty\""}{&a[i].Cover, &a[i].ID, &a[i].Name})
	}

	return api.GetAlbumMy200JSONResponse{
		GetAlbumJSONResponse: ar,
	}, nil
}

func (h Handler) PostMusicPlay(ctx context.Context, request api.PostMusicPlayRequestObject) (api.PostMusicPlayResponseObject, error) {
	const op = "./internal/adapters/http/handler.go.PlayMusic()"

	// url, err := h.mService.GetPresignURLSong(ctx, *request.Body.MusicId+"-song")
	// if err != nil {
	// 	slog.Error(err.Error())
	// 	return api.PostMusicPlay500JSONResponse(err.Error()), fmt.Errorf("%s: %w", op, err)
	// }

	m, _, err := h.mService.GetMusic(ctx, *request.Body.MusicId, "")
	if err != nil {
		slog.Error(fmt.Sprintf("%s: %s", op, err.Error()))
		return api.PostMusicPlay500JSONResponse(err.Error()), nil
	}	

	o, err := h.mService.GetObject(ctx, m.SongURL)
	if err != nil {
		slog.Error(err.Error())
		return api.PostMusicPlay500JSONResponse(err.Error()), err
	}

	buf, err := io.ReadAll(o)
	if err != nil {
		slog.Error(err.Error())
		return api.PostMusicPlay500JSONResponse(err.Error()), err
	}

	bufReader := bufio.NewReader(bytes.NewReader(buf))

	m3u8 := ""

	for range 4 {
		line, err := bufReader.ReadString('\n')
		if err != nil {
			return api.PostMusicPlay500JSONResponse(err.Error()), err
		}

		m3u8 += line
	}

	extinfs := make([]string, 0)

	for {
		line, err := bufReader.ReadString('\n')
		if err != nil && errors.Is(err, io.EOF) {
			break
		}
		
		if err != nil {
			slog.Error(err.Error())
			return api.PostMusicPlay500JSONResponse(err.Error()), err
		}

		if strings.Contains(line, "EXTINF") {
			extinfs = append(extinfs, line)
		}
	}

	slog.Info(m.SongURL)
	slog.Info(strings.TrimSuffix(m.SongURL, "playlist.m3u8"))
	segKeys, err := h.mService.ListObjects(ctx, strings.TrimSuffix(m.SongURL, "playlist.m3u8"), ".ts")
	if err != nil {
		slog.Error(err.Error())
		return api.PostMusicPlay500JSONResponse(err.Error()), err
	}

	slices.Sort(segKeys)
	
	for i, key := range segKeys {
		url, err := h.mService.GetPresignURLSong(ctx, key)
		if err != nil {
			slog.Error(err.Error())
			return api.PostMusicPlay500JSONResponse(err.Error()), err
		}

		m3u8 += fmt.Sprintf("%s%s\n", extinfs[i], url)
	}

	m3u8 += "#EXT-X-ENDLIST\n"

	f, err := os.Create("m3u8")
	if err != nil {
		slog.Error(err.Error())
	}else {
		slog.Info("Success create file")
		slog.Info(f.Name())
	}

	f.Write([]byte(m3u8))

	return api.PostMusicPlay200JSONResponse{
		PresignUrl: &m3u8,
	}, nil
}

func (h Handler) UploadPlaylist(w http.ResponseWriter, r *http.Request) {
	const op = "./internal/adapters/http/handler.go.UploadPlaylist()"

	t, err := r.Cookie("Access-Token")
	if err != nil {
		slog.Error(err.Error())
		http.Error(w, "Unauthorized", 401)
		return
	}

	if t.Value == "" {
		slog.Info("Token empty")
		http.Error(w, "Unauthorized", 401)
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

	playlistName := r.FormValue("playlist_name")
	if playlistName == "" {
		slog.Warn("playlist_name field empty")
		http.Error(w, "playlist_name field is required", http.StatusBadRequest)
		return
	}

	private := r.FormValue("private") == "true"

	uploaderID := claims["sub"].(string)

	var coverData []byte
	coverContentType := ""
	coverFile, _, err := r.FormFile("playlist_cover")
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

	playlistID, err := h.pService.Create(r.Context(), playlistName, uploaderID, coverData, coverContentType, private)
	if err != nil {
		slog.Error(fmt.Sprint(op, err.Error()))
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(200)
	json.NewEncoder(w).Encode(map[string]string{"playlist_id": playlistID})
}

func (h Handler) GetMyPlaylists(ctx context.Context, request api.GetMyPlaylistsRequestObject) (api.GetMyPlaylistsResponseObject, error) {
	const op = "./internal/adapters/http/handler.go.GetMyPlaylists()"

	claims, err := h.uServices.CheckAccessToken(ctx, request.Params.AccessToken)
	if err != nil {
		slog.Error(fmt.Errorf("%s: %w", op, err).Error())
		return api.GetMyPlaylists401Response{}, nil
	}

	userID := claims["sub"].(string)

	plf := models.PlaylistFilter{
		LikesMin: request.Params.LikesMin,
		LikesMax: request.Params.LikesMax,
	}

	pl, err := h.pService.GetMyPlaylists(ctx, userID, plf)
	if err != nil {
		slog.Error(fmt.Errorf("%s: %w", op, err).Error())
		return api.GetMyPlaylists500JSONResponse(err.Error()), err
	}

	resp := make(api.GetMyPlaylists200JSONResponse, 0, len(pl))
	for i := range pl {
		id := pl[i].ID
		name := pl[i].Name
		uploaderID := pl[i].UploaderID
		cover := pl[i].Cover
		private := pl[i].Private
		likesCount := pl[i].LikesCount
		username := pl[i].Username
		resp = append(resp, api.PlaylistInfo{
			Id:         &id,
			Name:       &name,
			UploaderId: &uploaderID,
			Cover:      &cover,
			Private:    &private,
			Username:   &username,
			LikesCount: &likesCount,
		})
	}

	return resp, nil
}

func (h Handler) GetPublicPlaylists(ctx context.Context, request api.GetPublicPlaylistsRequestObject) (api.GetPublicPlaylistsResponseObject, error) {
	const op = "./internal/adapters/http/handler.go.GetPublicPlaylists()"

	plf := models.PlaylistFilter{
		LikesMin: request.Params.LikesMin,
		LikesMax: request.Params.LikesMax,
	}

	pl, err := h.pService.GetPublicPlaylists(ctx, plf)
	if err != nil {
		slog.Error(fmt.Errorf("%s: %w", op, err).Error())
		return api.GetPublicPlaylists500JSONResponse(err.Error()), err
	}

	resp := make(api.GetPublicPlaylists200JSONResponse, 0, len(pl))
	for i := range pl {
		id := pl[i].ID
		name := pl[i].Name
		uploaderID := pl[i].UploaderID
		cover := pl[i].Cover
		private := pl[i].Private
		username := pl[i].Username
		likesCount := pl[i].LikesCount
		resp = append(resp, api.PlaylistInfo{
			Id:         &id,
			Name:       &name,
			UploaderId: &uploaderID,
			Cover:      &cover,
			Private:    &private,
			Username:   &username,
			LikesCount: &likesCount,
		})
	}

	return resp, nil
}

func (h Handler) DeletePlaylist(ctx context.Context, request api.DeletePlaylistRequestObject) (api.DeletePlaylistResponseObject, error) {
	const op = "./internal/adapters/http/handler.go.DeletePlaylist()"

	slog.Info("DeletePlaylist")

	claims, err := h.uServices.CheckAccessToken(ctx, request.Params.AccessToken)
	if err != nil {
		return api.DeletePlaylist403Response{}, nil
	}

	userID := claims["sub"].(string)

	pl, err := h.pService.GetPlaylist(ctx, request.PlaylistID)
	if err != nil {
		return api.DeletePlaylist404Response{}, nil
	}

	if pl.UploaderID != userID {
		return api.DeletePlaylist403Response{}, nil
	}

	err = h.pService.DeletePlaylist(ctx, request.PlaylistID)
	if err != nil {
		slog.Error(err.Error())
		return api.DeletePlaylist500Response{}, nil
	}

	status := "ok"
	return api.DeletePlaylist200JSONResponse{Status: &status}, nil
}

func (h Handler) UpdatePlaylist(w http.ResponseWriter, r *http.Request) {
	const op = "./internal/adapters/http/handler.go.UpdatePlaylist()"

	c, err := r.Cookie("Access-Token")
	if err != nil {
		slog.Error(err.Error())
		http.Error(w, "Unauthorized", 401)
		return
	}

	if c.Value == "" {
		slog.Info("UpdatePlaylist token empty")
		http.Error(w, "Unauthorized", 401)
		return
	}

	claims, err := h.uServices.CheckAccessToken(r.Context(), c.Value)
	if err != nil {
		slog.Error(err.Error())
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	userID := claims["sub"]

	p, err := h.pService.GetPlaylist(r.Context(), r.PathValue("playlistID"))
	if err != nil {
		slog.Error(err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if userID != p.UploaderID {
		slog.Info("UpdatePlaylist userID not match with p.UploaderID")
		http.Error(w, "UserID not match with uploaderID", http.StatusForbidden)
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, 50<<20)

	if err := r.ParseMultipartForm(32 << 20); err != nil {
		slog.Error(fmt.Sprint(op, err.Error()))
		http.Error(w, "Failed to parse form: "+err.Error(), http.StatusBadRequest)
		return
	}

	playlistName := r.FormValue("name")
	if playlistName == "" {
		slog.Warn("playlist_name field empty")
	}

	privateInt := r.FormValue("private")
	var private bool
	n, err := strconv.Atoi(privateInt)
	if err != nil {
		slog.Error(fmt.Sprintf("%s: %w", op, err))
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if n == 0 {
		private = false
	}
	if n == 1 {
		private = true
	}
	if n != 0 && n != 1 {
		slog.Warn("UpdatePlaylist private not in (0,1)")
		http.Error(w, errors.New("private not in (0,1)").Error(), http.StatusBadRequest)
		return
	}

	var coverData []byte
	coverContentType := ""
	coverFile, _, err := r.FormFile("cover")
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

	plu := models.PlaylistUpdate{
		ID: r.PathValue("playlistID"),
		Name: &playlistName,
		Private: &private,
	}

	err = h.pService.UpdatePlaylist(r.Context(), plu, coverData, coverContentType)
	if err != nil {
		slog.Error(err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Write([]byte("Success"))
	w.WriteHeader(200)
}

func (h Handler) GetPlaylistTracks(ctx context.Context, request api.GetPlaylistTracksRequestObject) (api.GetPlaylistTracksResponseObject, error) {
	const op = "./internal/adapters/http/handler.go.GetPlaylistTracks()"

	slog.Info("GetPlaylistTracks req")

	t := request.Params.AccessToken
	userID := ""
	availableUpdate := false
	if t != nil{
		slog.Info("GetPlaylistTracks token not nill")

		claims, err := h.uServices.CheckAccessToken(ctx, *t)
		
		switch (err) {
		case nil:
			userID = claims["sub"].(string)
		default:
			slog.Error(err.Error())
		}

		switch (userID) {
		case "":
			slog.Info("userID empty")
		default:
			p, err := h.pService.GetPlaylist(ctx, request.PlaylistID)
			
			switch (err) {
			case nil:
				if userID == p.UploaderID {
					availableUpdate = true
				}
			default:
				slog.Error(err.Error())
			}
		}
	}else {
		slog.Info("GetPlaylistTracks token nil")
	}

	tracks, err := h.pService.GetPlaylistMusic(ctx, request.PlaylistID)
	if err != nil {
		slog.Error(fmt.Errorf("%s: %w", op, err).Error())
		return api.GetPlaylistTracks500JSONResponse(err.Error()), err
	}

	resp := api.GetPlaylistTracks200JSONResponse{}
	respArr := make([]api.LikedTrack, 0, len(tracks))
	for i := range tracks {
		mid := tracks[i].MusicID
		mname := tracks[i].MusicName
		mcover := tracks[i].MusicCover
		mlikes := tracks[i].MusicLikes
		mdur := tracks[i].MusicDurationSeconds
		surl := tracks[i].MusicSongURL
		uid := tracks[i].MusicUploaderID
		uuname := tracks[i].UserUsername
		respArr = append(respArr, api.LikedTrack{
				MusicId:          &mid,
				MusicName:        &mname,
				MusicCover:       &mcover,
				MusicLikes:       &mlikes,
				MusicDuration:    &mdur,
				SongUrl:          &surl,
				UploaderId:       &uid,
				UploaderUsername: &uuname,
				ListeningCount:  &tracks[i].MusicListeningCount,
			})
	}

	resp.Tracks = &respArr
	resp.AvailableForUpdate = &availableUpdate

	return resp, nil
}

func (h Handler) AddMusicToPlaylist(ctx context.Context, request api.AddMusicToPlaylistRequestObject) (api.AddMusicToPlaylistResponseObject, error) {
	const op = "./internal/adapters/http/handler.go.AddMusicToPlaylist()"

	claims, err := h.uServices.CheckAccessToken(ctx, request.Params.AccessToken)
	if err != nil {
		slog.Error(fmt.Errorf("%s: %w", op, err).Error())
		return api.AddMusicToPlaylist401Response{}, nil
	}

	_ = claims

	err = h.pService.AddMusic(ctx, request.PlaylistID, request.Body.MusicId)
	if err != nil {
		slog.Error(fmt.Errorf("%s: %w", op, err).Error())
		return api.AddMusicToPlaylist500JSONResponse(err.Error()), err
	}

	status := "ok"
	return api.AddMusicToPlaylist200JSONResponse{Status: &status}, nil
}

func (h Handler) GetAlbums(ctx context.Context, request api.GetAlbumsRequestObject) (api.GetAlbumsResponseObject, error) {
	const op = "./internal/adapters/http/handler.go.GetAlbums()"

	af := models.AlbumFilter{
		LikesMin: request.Params.LikesMin,
		LikesMax: request.Params.LikesMax,
	}

	a, err := h.aService.GetAlbumsInfo(ctx, af)
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
			LikesCount: &a[i].LikesCount,
		})
	}

	return api.GetAlbums200JSONResponse(al), nil
}

func (h Handler) GetAlbumID(ctx context.Context, request api.GetAlbumIDRequestObject) (api.GetAlbumIDResponseObject, error) {
	const op = "./internal/adapters/http/handler.go.GetAlbumID()"

	availableUpdate := false

	if t := request.Params.AccessToken; t != nil && *t != "" {
		claims, err := h.uServices.CheckAccessToken(ctx, *t)
		if err != nil {
			slog.Error(err.Error())
			return api.GetAlbumID500JSONResponse(err.Error()), err
		}

		userID := claims["sub"].(string)

		a, err := h.aService.GetAlbum(ctx, request.AlbumID)
		if err != nil {
			slog.Error(err.Error())
			return api.GetAlbumID500JSONResponse(err.Error()), err
		}

		if userID == a.UploaderID {
			availableUpdate = true
		}else {
			slog.Info("GetAlbumID different id's")
		}
	}

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
				MusicId:          &a[i].MusicID,
				MusicName:        &a[i].MusicName,
				UploaderId:       &a[i].MusicUploaderID,
				UploaderUsername: &a[i].UserUsername,
				MusicLikes:       &a[i].MusicLikes,
				MusicCover:       &coverURL,
				SongUrl:          &a[i].MusicSongURL,
				MusicDuration:    &a[i].MusicDurationSeconds,
				ListeningCount:  &a[i].MusicListeningCount,
			})
	}

	return api.GetAlbumID200JSONResponse{
		Tracks: &al,
		AvailableUpdate: &availableUpdate,
	}, nil
}

func (h Handler) GetAlbumMyLikes(ctx context.Context, request api.GetAlbumMyLikesRequestObject) (api.GetAlbumMyLikesResponseObject, error) {
	const op = "./internal/adapters/http/handler.go.GetAlbumMyLikes()"
	slog.Debug(op)

	t := request.Params.AccessToken
	if t == "" {
		return api.GetAlbumMyLikes401Response{}, nil
	}

	claims, err := h.uServices.CheckAccessToken(ctx, t)
	if err != nil {
		slog.Error(err.Error())
		return api.GetAlbumMyLikes401Response{}, nil
	}

	userID := claims["sub"].(string)

	af := models.AlbumFilter{
		LikesMin: request.Params.LikesMin,
		LikesMax: request.Params.LikesMax,
	}

	slog.Info(fmt.Sprintf("%+v", af))

	al, err := h.aService.GetLikedAlbums(ctx, userID, af)
	if err != nil {
		slog.Error(fmt.Sprintf("%s: %s", "GetLikedAlbums", err.Error()))
		return api.GetAlbumMyLikes500Response{}, err
	}

	alr := make([]api.Album, 0, len(al))
	for i := range al {
		alr = append(alr, api.Album{
			Id:               &al[i].ID,
			Name:             &al[i].Name,
			UploaderId:       &al[i].UploaderID,
			UploaderUsername: &al[i].Username,
			Cover:            &al[i].Cover,
			LikesCount:       &al[i].LikesCount,
		})
	}

	return api.GetAlbumMyLikes200JSONResponse(alr), nil
}

func (h Handler) PostAlbumLike(ctx context.Context, request api.PostAlbumLikeRequestObject) (api.PostAlbumLikeResponseObject, error) {
	const op = "./internal/adapters/http/handler.go.PostAlbumLike()"
	slog.Debug(fmt.Sprintf("%s: %s", "LikeAlbum", op))

	t := request.Params.AccessToken
	if t == "" {
		return api.PostAlbumLike401Response{}, nil
	}

	claims, err := h.uServices.CheckAccessToken(ctx, t)
	if err != nil {
		return api.PostAlbumLike401Response{}, nil
	}

	userID := claims["sub"].(string)
	err = h.aService.LikeAlbum(ctx, *request.Body.AlbumId, userID)
	if err != nil {
		slog.Error(fmt.Sprintf("%s: %s", "LikeAlbum", err.Error()))
		return api.PostAlbumLike500Response{}, err
	}

	return api.PostAlbumLike200Response{}, nil
}

func (h Handler) DeleteAlbumLike(ctx context.Context, request api.DeleteAlbumLikeRequestObject) (api.DeleteAlbumLikeResponseObject, error) {
	const op = "./internal/adapters/http/handler.go.DeleteAlbumLike()"
	slog.Debug(op)

	t := request.Params.AccessToken
	if t == "" {
		return api.DeleteAlbumLike401Response{}, nil
	}

	claims, err := h.uServices.CheckAccessToken(ctx, t)
	if err != nil {
		return api.DeleteAlbumLike401Response{}, nil
	}

	userID := claims["sub"].(string)
	err = h.aService.DeleteLikeAlbum(ctx, *request.Body.AlbumId, userID)
	if err != nil {
		return api.DeleteAlbumLike500Response{}, err
	}

	return api.DeleteAlbumLike200Response{}, nil
}

func (h Handler) UpdateAlbum(w http.ResponseWriter, r *http.Request) {
	const op = "./internal/adapters/http/handler.go.UpdateAlbum()"

	t, err := r.Cookie("Access-Token")
	if err != nil {
		slog.Error(err.Error())
		http.Error(w, "Unauthorized", 401)
		return
	}

	if t.Value == "" {
		slog.Info("Token empty")
		http.Error(w, "Unauthorized", 401)
		return
	}

	claims, err := h.uServices.CheckAccessToken(r.Context(), t.Value)
	if err != nil {
		slog.Error(err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	userID := claims["sub"].(string)

	r.Body = http.MaxBytesReader(w, r.Body, 50<<20)

	if err := r.ParseMultipartForm(32 << 20); err != nil {
		slog.Error(fmt.Sprint(op, err.Error()))
		http.Error(w, "Failed to parse form: "+err.Error(), http.StatusBadRequest)
		return
	}
	
	name := r.FormValue("name")
	
	var coverData []byte
	coverContentType := ""
	coverFile, _, err := r.FormFile("cover")
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

	album := models.Album{
		ID: r.PathValue("albumID"),
		Name: name,
	}

	err = h.aService.UpdateAlbum(r.Context(), album, coverData, coverContentType, userID)
	if err != nil {
		slog.Error(err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Write([]byte("Success"))
	w.WriteHeader(200)
}

func (h Handler) DeleteAlbumAlbumID(ctx context.Context, request api.DeleteAlbumAlbumIDRequestObject) (api.DeleteAlbumAlbumIDResponseObject, error) {
	const op = "./internal/adapters/http/handler.go.DeleteAlbumAlbumID"

	t := request.Params.AccessToken

	if t == "" {
		slog.Warn("DeleteAlbumID token empty")
		return api.DeleteAlbumAlbumID401Response{}, nil
	}

	claims, err := h.uServices.CheckAccessToken(ctx, t)
	if err != nil {
		slog.Error(err.Error())
		return api.DeleteAlbumAlbumID401Response{}, nil
	}

	userID := claims["sub"].(string)

	err = h.aService.DeleteAlbum(ctx, request.AlbumID, userID)
	if err != nil {
		slog.Error(err.Error())
		return api.DeleteAlbumAlbumID500Response{}, err
	}

	return api.DeleteAlbumAlbumID200Response{}, nil
}

func (h Handler) GetMusicMy(ctx context.Context, request api.GetMusicMyRequestObject) (api.GetMusicMyResponseObject, error) {
	const op = "./internal/adapters/http/handler.go.GetMusicMy"

	t := request.Params.AccessToken

	if t == "" {
		slog.Warn("GetMusicMy token empty")
		return api.GetMusicMy401Response{}, nil
	}

	claims, err := h.uServices.CheckAccessToken(ctx, t)
	if err != nil {
		slog.Error("GetMusicMy error " + err.Error())
		return api.GetMusicMy401Response{}, nil
	}

	userID := claims["sub"].(string)

	p, err := h.mService.GetMusicByUploaderID(ctx, userID)
	if err != nil {
		slog.Error(err.Error())
		return api.GetMusicMy500Response{}, err
	}

	pResp := make([]api.Music, 0, len(p))
	for i := range p {
		urlCover := p[i].CoverURL
		if p[i].CoverURL != "" {
			urlCover, err = h.mService.GetPresignURLSong(ctx, p[i].CoverURL)
			if err != nil {
				slog.Error(err.Error())
				return api.GetMusicMy500Response{}, fmt.Errorf("%s: %w", op, err)
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
				ListeningCount:  &p[i].ListeningCount,
			})
	}

	return api.GetMusicMy200JSONResponse{
		api.GetMusicJSONResponse{
			Music: pResp,
		},
	}, nil
}

func (h Handler) GetPlaylistMyLikes(ctx context.Context, request api.GetPlaylistMyLikesRequestObject) (api.GetPlaylistMyLikesResponseObject, error) {
	const op = "./internal/adapters/http/handler.go.GetPlaylistMyLikes()"
	
	t := request.Params.AccessToken

	if t == "" {
		slog.Warn("GetPlaylistMyLikes token empty")
		return api.GetPlaylistMyLikes401Response{}, nil
	}

	claims, err := h.uServices.CheckAccessToken(ctx, t)
	if err != nil {
		slog.Error("GetPlaylistMyLikes error " + err.Error())
		return api.GetPlaylistMyLikes401Response{}, nil
	}

	userID := claims["sub"].(string)

	plf := models.PlaylistFilter{
		LikesMin: request.Params.LikesMin,
		LikesMax: request.Params.LikesMax,
	}

	slog.Info(fmt.Sprintf("%s: %+v", "GetPlaylistMyLikes: ", plf))

	pl, err := h.pService.GetLikedPlaylists(ctx, userID, plf)
	if err != nil {
		slog.Error(err.Error())
		return api.GetPlaylistMyLikes500Response{}, err
	}

	plr := make([]api.PlaylistInfo, 0, len(pl))
	for i := range pl {
		plr = append(plr, api.PlaylistInfo{
			Id: &pl[i].ID,
			Name: &pl[i].Name,
			Cover: &pl[i].Cover,
			Username: &pl[i].Username,
			UploaderId: &pl[i].UploaderID,
			Private: &pl[i].Private,
			LikesCount: &pl[i].LikesCount,
		})
	}

	return api.GetPlaylistMyLikes200JSONResponse(plr), nil
}

func (h Handler) PostPlaylistLike(ctx context.Context, request api.PostPlaylistLikeRequestObject) (api.PostPlaylistLikeResponseObject, error) {
	const op = "./internal/adapters/http/handler.go.PlaylistLike"

	t := request.Params.AccessToken

	if t == "" {
		slog.Warn("PostPlaylistLike token empty")
		return api.PostPlaylistLike401Response{}, nil
	}

	claims, err := h.uServices.CheckAccessToken(ctx, t)
	if err != nil {
		slog.Error("PostPlaylistLike error " + err.Error())
		return api.PostPlaylistLike401Response{}, nil
	}

	userID := claims["sub"].(string)

	err = h.pService.LikePlaylist(ctx, *request.Body.PlaylistId, userID)
	if err != nil {
		slog.Error(err.Error())
		return api.PostPlaylistLike500Response{}, err
	}

	return api.PostPlaylistLike200Response{}, nil
}

func (h Handler) DeletePlaylistLike(ctx context.Context, request api.DeletePlaylistLikeRequestObject) (api.DeletePlaylistLikeResponseObject, error) {
	const op = "./internal/adapters/http/handler.go.DeletePlaylistLike"

	t := request.Params.AccessToken

	if t == "" {
		slog.Warn("PostPlaylistLike token empty")
		return api.DeletePlaylistLike401Response{}, nil
	}

	claims, err := h.uServices.CheckAccessToken(ctx, t)
	if err != nil {
		slog.Error("PostPlaylistLike error " + err.Error())
		return api.DeletePlaylistLike401Response{}, nil
	}

	userID := claims["sub"].(string)

	err = h.pService.DeleteLikePlaylist(ctx, *request.Body.PlaylistId, userID)
	if err != nil {
		slog.Error(err.Error())
		return api.DeletePlaylistLike500Response{}, err
	}

	return api.DeletePlaylistLike200Response{}, nil
}

func (h Handler) PostMusicIncLisCount(ctx context.Context, request api.PostMusicIncLisCountRequestObject) (api.PostMusicIncLisCountResponseObject, error) {
	const op = "./internal/adapters/http/handler.go.IncLisCount"

	slog.Info("IncMusicLisCount get req")

	err := h.mService.AddListening(ctx, *request.Body.MusicId)
	if err != nil {
		slog.Error(err.Error())
		return api.PostMusicIncLisCount500Response{}, err
	}

	return api.PostMusicIncLisCount200Response{}, nil
}