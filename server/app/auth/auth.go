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
	db           *sql.DB
	sessions     map[string]User
	cookieName   string
	loginFormURL string
}

func NewAuth(db *sql.DB, cookieName string, loginFormURL string) Auth {
	return Auth{db: db, cookieName: cookieName, loginFormURL: loginFormURL, sessions: make(map[string]User)}
}

var SessionStore map[string]Client
var storageMutex sync.RWMutex

var DB *sql.DB

type User struct {
	PlaceID int64
}

type Client struct {
	loggedIn bool
}

type authenticationMiddleware struct {
	wrappedHandler http.Handler
}

func (h authenticationMiddleware) ServeHTTP(w http.ResponseWriter, r *http.Request) {

	setNoCache(w)

	cookie, err := r.Cookie("session")
	if err != nil {
		if err != http.ErrNoCookie {
			fmt.Fprint(w, err)
			return
		} else {
			err = nil
		}
	}

	var present bool
	var client Client
	if cookie != nil {
		storageMutex.RLock()
		client, present = SessionStore[cookie.Value]
		storageMutex.RUnlock()
	} else {
		present = false
	}

	if present == false {
		cookie = &http.Cookie{
			Name:  "session",
			Value: uuid.NewV4().String(),
		}
		client = Client{loggedIn: false}
		storageMutex.Lock()
		SessionStore[cookie.Value] = client
		storageMutex.Unlock()
	}

	http.SetCookie(w, cookie)
	if client.loggedIn == false {
		fmt.Fprint(w, loginPage)
		return
	}
	if client.loggedIn == true {
		h.wrappedHandler.ServeHTTP(w, r)
		return
	}

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

func AuthenticateAdmin(h http.Handler) authenticationMiddleware {
	return authenticationMiddleware{h}
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
			http.Redirect(w, r, "/restaurant/login_form", 301)
		}
		return
	}

	user, present := h.auth.sessions[cookie.Value]

	if !present {
		http.Redirect(w, r, "/restaurant/login_form", 301)
		return
	}

	r = r.WithContext(context.WithValue(r.Context(), "user", user))
	h.wrappedHandler.ServeHTTP(w, r)

}

func (auth Auth) Authenticate(h http.Handler) http.Handler {
	return restaurantAuthenticationMiddleware{h, auth}
}

const loginPage = "<html><head><title>Login</title></head><body><form action=\"login\" method=\"post\"> <input type=\"password\" name=\"password\" /> <input type=\"submit\" value=\"login\" /> </form> </body> </html>"

func (auth Auth) HandleRestaurantLogin() http.HandlerFunc {

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

		err = DB.QueryRow("SELECT id, password FROM places WHERE username = $1", username).Scan(&placeID, &password)
		if err == sql.ErrNoRows {
			http.Redirect(w, r, "/restaurant/login_form?error=bad_credentials", 301)
			return
		}
		if err != nil {
			panic(err)
		}

		if !password.Valid {
			http.Redirect(w, r, "/restaurant/login_form?error=bad_credentials", 301)
			return
		}

		if subtle.ConstantTimeCompare([]byte(r.FormValue("password")), []byte(password.String)) != 1 {
			http.Redirect(w, r, "/restaurant/login_form?error=bad_credentials", 301)
			return
		}

		cookie := &http.Cookie{
			Name:  auth.cookieName,
			Value: uuid.NewV4().String(),
		}

		auth.sessions[cookie.Value] = User{PlaceID: placeID}
		http.SetCookie(w, cookie)
		http.Redirect(w, r, "/restaurant/edit", 301)

	}
}

func HandleLogin(w http.ResponseWriter, r *http.Request) {

	setNoCache(w)

	cookie, err := r.Cookie("session")
	if err != nil {
		if err != http.ErrNoCookie {
			fmt.Fprint(w, err)
			return
		} else {
			err = nil
		}
	}
	var present bool
	var client Client
	if cookie != nil {
		storageMutex.RLock()
		client, present = SessionStore[cookie.Value]
		storageMutex.RUnlock()
	} else {
		present = false
	}

	if present == false {
		cookie = &http.Cookie{
			Name:  "session",
			Value: uuid.NewV4().String(),
		}
		client = Client{loggedIn: false}
		storageMutex.Lock()
		SessionStore[cookie.Value] = client
		storageMutex.Unlock()
	}
	http.SetCookie(w, cookie)
	err = r.ParseForm()
	if err != nil {
		fmt.Fprint(w, err)
		return
	}

	if subtle.ConstantTimeCompare([]byte(r.FormValue("password")), []byte("password123")) == 1 {
		client.loggedIn = true
		storageMutex.Lock()
		SessionStore[cookie.Value] = client
		storageMutex.Unlock()
		http.Redirect(w, r, "/admin/places", 301)
	} else {
		fmt.Fprintln(w, "Wrong password.")
	}

}

func HandleLogout(w http.ResponseWriter, r *http.Request) {

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

	cookie, err := r.Cookie("session")
	if err != nil {
		if err != http.ErrNoCookie {
			fmt.Fprint(w, err)
			return
		} else {
			err = nil
		}
	}

	var present bool
	if cookie != nil {
		storageMutex.RLock()
		_, present = SessionStore[cookie.Value]
		storageMutex.RUnlock()
	} else {
		present = false
	}

	if present == true {
		client := Client{loggedIn: false}
		storageMutex.Lock()
		SessionStore[cookie.Value] = client
		storageMutex.Unlock()
	}
	http.Redirect(w, r, "/", 301)

}

func (auth Auth) HandleRestaurantLogout() http.HandlerFunc {
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

		http.Redirect(w, r, "/restaurant/login_form", 301)
	}
}

func CurrentUser(r *http.Request) User {
	return r.Context().Value("user").(User)
}
