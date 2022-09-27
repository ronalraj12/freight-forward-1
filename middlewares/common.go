package middlewares

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/RemoteState/yourdaily-server/dbHelpers"
	"github.com/RemoteState/yourdaily-server/firebase"
	"github.com/RemoteState/yourdaily-server/models"
	"github.com/RemoteState/yourdaily-server/utils"
	"github.com/dgrijalva/jwt-go"
	"github.com/go-chi/chi"
	"github.com/rs/cors"
	"github.com/sirupsen/logrus"
	"net/http"
	"os"
)

type contextString string

const userContext contextString = "__userContext"
const storeManagerContext contextString = "__smContext"
const jwtSigningMethod contextString = "HS256"

//corsOptions setting up routes for cors
func corsOptions() *cors.Cors {
	return cors.New(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS", "PATCH"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token", "Access-Token", "importDate", "X-Client-Version", "Cache-Control", "Pragma", "x-started-at", "x-api-key"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: true,
		MaxAge:           300, // Maximum value not ignored by any of major browsers
	})
}

//CommonMiddlewares middleware common for all routes
func CommonMiddlewares() chi.Middlewares {
	return chi.Chain(
		func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Add("Content-Type", "application/json")
				next.ServeHTTP(w, r)
			})
		},
		corsOptions().Handler,
		func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

				defer func() {
					err := recover()
					if err != nil {
						logrus.Errorf("Request Panic err: %v", err)
						jsonBody, _ := json.Marshal(map[string]string{
							"error": "There was an internal server error",
						})
						w.Header().Set("Content-Type", "application/json")
						w.WriteHeader(http.StatusInternalServerError)
						_, err := w.Write(jsonBody)
						if err != nil {
							logrus.Errorf("Failed to send response from middleware with error: %+v", err)
						}
					}
				}()

				next.ServeHTTP(w, r)

			})
		},
	)
}

func AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		jwt, err := firebase.FireAuthInstance.VerifyToken(r.Header.Get("Authorization"))
		if err != nil {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		authID, err := firebase.GetAuthId(jwt)
		if err != nil {
			logrus.Errorf("AuthMiddleware: failed to get auth id from token error: %v", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		user, err := dbHelpers.GetUserByAuthId(authID)
		if err != nil {
			logrus.Errorf("AuthMiddleware: failed to get user info from authID %s error: %v", authID, err)
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		if user == nil {
			logrus.Error(errors.New("failed to get user context in middleware"))
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		permissions, err := dbHelpers.UserPermissionById(user.ID)
		if err != nil {
			logrus.Errorf("Failed to find user permission with error: %+v", err)
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		user.Permissions = permissions

		imageInfo, err := dbHelpers.GetImageInfoByUserID(user.ID)
		if err != nil {
			logrus.Errorf("Failed to get user profile image info with error: %+v", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		if imageInfo != nil {
			imageURL, err := firebase.GetURL(imageInfo)
			if err != nil {
				logrus.Errorf("Failed to get image url with error: %+v", err)
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			user.ProfileImageLink = imageURL
		}

		ctx := context.WithValue(r.Context(), userContext, user)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func UserContext(r *http.Request) *models.User {
	if user, ok := r.Context().Value(userContext).(*models.User); ok && user != nil {
		return user
	}
	return nil
}

func AuthMiddlewareForSm(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		jwtToken := r.Header.Get("Authorization")
		if jwtToken == "" {
			utils.RespondError(w, http.StatusBadRequest, errors.New("empty authorization token"), "Authorization token can't be empty!")
			return
		}

		claims := &models.JWTClaims{}
		token, err := jwt.ParseWithClaims(jwtToken, claims, func(t *jwt.Token) (interface{}, error) {
			if t.Method.Alg() != string(jwtSigningMethod) {
				return nil, fmt.Errorf("unexpected jwt signing method=%v", t.Header["alg"])
			}
			return []byte(os.Getenv("SECRET_KEY")), nil
		})
		if err != nil {
			utils.RespondError(w, http.StatusUnauthorized, err, "Authentication failed!")
			return
		}
		if !token.Valid {
			utils.RespondError(w, http.StatusUnauthorized, errors.New("invalid token"), "Authentication failed!")
			return
		}

		storeManager, err := dbHelpers.GetStoreManagerByEmail(claims.Email)
		if err != nil {
			utils.RespondError(w, http.StatusInternalServerError, err, "Authentication failed!")
			return
		}

		if storeManager.Permission != models.StoreManager {
			utils.RespondError(w, http.StatusUnauthorized, errors.New("given email does not belong to a store-manager"), "Authentication failed!")
			return
		}

		//ctx := context.WithValue(r.Context(), storeManagerContext, storeManager)
		ctx := context.WithValue(r.Context(), userContext, storeManager)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func StaffPermission(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := UserContext(r)
		if ctx == nil {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		found := false
		for _, p := range ctx.Permissions {
			if p == models.DeliveryBoy {
				ctx.AllowedMode = models.DeliveryMode
				found = true
				break
			} else if p == models.CartBoy {
				ctx.AllowedMode = models.CartMode
				found = true
				break
			}
		}
		if !found {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		next.ServeHTTP(w, r)
	})
}
