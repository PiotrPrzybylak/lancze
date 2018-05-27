package auth

import (
	"context"
	"crypto/subtle"
	"database/sql"
	"fmt"
	"github.com/satori/go.uuid"
	"net/http"
	"sync"
	gotime "time"
)

type Auth struct {
	db                      *sql.DB
	sessions                map[string]User
	cookieName              string
	loginFormURL            string
	loginSuccessRedirectURL string
	findUserSQL             string
}

func NewAuth(
	db *sql.DB,
	cookieName string,
	loginFormURL string,
	loginSuccessRedirectURL string,
	findUserSQL string,
) Auth {
	return Auth{
		db:                      db,
		cookieName:              cookieName,
		loginFormURL:            loginFormURL,
		loginSuccessRedirectURL: loginSuccessRedirectURL,
		findUserSQL:             findUserSQL,
		sessions:                make(map[string]User),
	}
}

var storageMutex sync.RWMutex

type User struct {
	PlaceID int64
}

func setNoCache(w http.ResponseWriter) {
	var epoch = gotime.Unix(0, 0).Format(gotime.RFC1123)
	var noCacheHeaders = map[string]string{
		"Expires":         epoch,
		"Cache-Control":   "no-cache, private, max-age=0",
		"Pragma":          "no-cache",
		"X-Accel-Expires": "0",
	}
	for k, v := range noCacheHeaders {
		w.Header().Set(k, v)
	}
}

type restaurantAuthenticationMiddleware struct {
	wrappedHandler http.Handler
	auth           Auth
}

func (h restaurantAuthenticationMiddleware) ServeHTTP(w http.ResponseWriter, r *http.Request) {

	setNoCache(w)

	cookie, err := r.Cookie(h.auth.cookieName)
	if err != nil {
		if err != http.ErrNoCookie {
			fmt.Fprint(w, err)
		} else {
			http.Redirect(w, r, h.auth.loginFormURL, 301)
		}
		return
	}

	user, present := h.auth.sessions[cookie.Value]

	if !present {
		http.Redirect(w, r, h.auth.loginFormURL, 301)
		return
	}

	r = r.WithContext(context.WithValue(r.Context(), "user", user))
	h.wrappedHandler.ServeHTTP(w, r)

}

func (auth Auth) Authenticate(h http.Handler) http.Handler {
	return restaurantAuthenticationMiddleware{h, auth}
}

func (auth Auth) HandleLogin() http.HandlerFunc {

	return func(w http.ResponseWriter, r *http.Request) {

		setNoCache(w)

		err := r.ParseForm()
		if err != nil {
			fmt.Fprint(w, err)
			return
		}

		username := r.FormValue("username")

		var placeID int64
		var password sql.NullString

		err = auth.db.QueryRow(auth.findUserSQL, username).Scan(&placeID, &password)
		if err == sql.ErrNoRows {
			http.Redirect(w, r, auth.loginFormURL+"?error=bad_credentials", 301)
			return
		}
		if err != nil {
			panic(err)
		}

		if !password.Valid {
			http.Redirect(w, r, auth.loginFormURL+"?error=bad_credentials", 301)
			return
		}

		if subtle.ConstantTimeCompare([]byte(r.FormValue("password")), []byte(password.String)) != 1 {
			http.Redirect(w, r, auth.loginFormURL+"?error=bad_credentials", 301)
			return
		}

		cookie := &http.Cookie{
			Name:  auth.cookieName,
			Value: uuid.NewV4().String(),
		}

		auth.sessions[cookie.Value] = User{PlaceID: placeID}
		http.SetCookie(w, cookie)
		http.Redirect(w, r, auth.loginSuccessRedirectURL, 301)

	}
}

func (auth Auth) HandleLogout() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		setNoCache(w)

		cookie, err := r.Cookie(auth.cookieName)
		if err != nil {
			if err != http.ErrNoCookie {
				fmt.Fprint(w, err)
				return
			} else {
				err = nil
			}
		} else {
			delete(auth.sessions, cookie.Value)
		}

		http.Redirect(w, r, auth.loginFormURL, 301)
	}
}

func CurrentUser(r *http.Request) User {
	return r.Context().Value("user").(User)
}
