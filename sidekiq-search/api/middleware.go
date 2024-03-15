package api

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"runtime/debug"
	"strings"
	"time"

	"github.com/TestingSDK2/sidekiq-backend/sidekiq-people/util"
	"github.com/TestingSDK2/sidekiq-backend/sidekiq-search/api/common"
	"github.com/TestingSDK2/sidekiq-backend/sidekiq-search/app"

	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
)

func (a *API) handler(f common.HandlerFuncWithCTX, auth ...bool) http.HandlerFunc {
	checkAuth := auth[0]
	onlyUserAuth := false
	if len(auth) > 1 {
		onlyUserAuth = auth[1]
	}
	return func(w http.ResponseWriter, r *http.Request) {
		fmt.Println("API -", r.URL.Path)
		r.Body = http.MaxBytesReader(w, r.Body, a.Config.MaxContentSize*1024*1024)
		beginTime := time.Now()
		hijacker, _ := w.(http.Hijacker)
		ctx := a.App.NewContext().WithRemoteAddress(a.IPAddressForRequest(r))
		ctx = ctx.WithLogger(ctx.Logger.WithField("request_id", base64.RawURLEncoding.EncodeToString(util.NewID())))
		ctx.Vars = mux.Vars(r)

		w = &common.StatusCodeRecorder{
			ResponseWriter: w,
			Hijacker:       hijacker,
		}

		if checkAuth {
			authResp := validateUser(a.Config, ctx, r, a.App, true)
			if authResp.Error != nil {
				ctx.Logger.WithError(authResp.Error).Error(authResp.ErrMsg)
				if authResp.ErrCode == http.StatusUnauthorized {
					http.Error(w, authResp.ErrMsg, authResp.ErrCode)
				} else {
					http.Error(w, "error from checkForUserAuth", http.StatusForbidden)
				}
				return
			}
			ctx = ctx.WithUserProfile(authResp.User, authResp.Profile)
		}

		if len(auth) > 1 && onlyUserAuth {
			authResp := validateUser(a.Config, ctx, r, a.App, false)
			if authResp.Error != nil {
				ctx.Logger.WithError(authResp.Error).Error(authResp.ErrMsg)
				if authResp.ErrCode == http.StatusUnauthorized {
					// http.Error(w, "Token has expired!", errCode)
					http.Error(w, authResp.ErrMsg, authResp.ErrCode)
				} else {
					http.Error(w, "error from checkForUserAuth", http.StatusForbidden)
				}
				return
			}
			ctx = ctx.WithUser(authResp.User)
		}

		defer func() {
			statusCode := w.(*common.StatusCodeRecorder).StatusCode
			if statusCode == 0 {
				statusCode = 200
			}
			duration := time.Since(beginTime)

			logger := ctx.Logger.WithFields(logrus.Fields{
				"duration":    duration,
				"status_code": statusCode,
				"remote":      ctx.RemoteAddress,
			})
			logger.Info(r.Method + " " + r.URL.RequestURI())
		}()

		defer func() {
			if localRecover := recover(); localRecover != nil {
				ctx.Logger.Error(fmt.Errorf("recovered from panic\n %v: %s", localRecover, debug.Stack()))
				json.NewEncoder(w).Encode(util.SetResponse(nil, 0, "server failed to process request"))
			}
		}()

		w.Header().Set("Content-Type", "application/json")

		if err := f(ctx, w, r); err != nil {
			if verr, ok := err.(*app.ValidationError); ok {
				data, err := json.Marshal(verr)
				if err == nil {
					w.WriteHeader(http.StatusBadRequest)
					_, err = w.Write(data)
				}

				if err != nil {
					ctx.Logger.Error(err)
					w.WriteHeader(http.StatusBadRequest)
					json.NewEncoder(w).Encode(util.SetResponse(nil, 0, err.Error()))
					// http.Error(w, "internal server error", http.StatusInternalServerError)
				}
			} else if uerr, ok := err.(*app.UserError); ok {
				data, err := json.Marshal(uerr)
				if err == nil {
					w.WriteHeader(uerr.StatusCode)
					_, err = w.Write(data)
				}

				if err != nil {
					ctx.Logger.Error(err)
					w.WriteHeader(http.StatusBadRequest)
					json.NewEncoder(w).Encode(util.SetResponse(nil, 0, err.Error()))
					// http.Error(w, "internal server error", http.StatusInternalServerError)
				}
			} else {
				ctx.Logger.Error(err)
				w.WriteHeader(http.StatusBadRequest)
				json.NewEncoder(w).Encode(util.SetResponse(nil, 0, err.Error()))
				// http.Error(w, "internal server error", http.StatusInternalServerError)
			}
		}
	}
}

// IPAddressForRequest determines IP address for request
func (a *API) IPAddressForRequest(r *http.Request) string {
	addr := r.RemoteAddr
	if a.Config.ProxyCount > 0 {
		h := r.Header.Get("X-Forwarded-For")
		if h != "" {
			clients := strings.Split(h, ",")
			if a.Config.ProxyCount > len(clients) {
				addr = clients[0]
			} else {
				addr = clients[len(clients)-a.Config.ProxyCount]
			}
		}
	}
	return strings.Split(strings.TrimSpace(addr), ":")[0]
}
