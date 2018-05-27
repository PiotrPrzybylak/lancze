package auth

import (
	"net/http"
	"fmt"
	"github.com/satori/go.uuid"
	"sync"
	gotime "time"
	"context"
	"database/sql"
	"crypto/subtle"
)

var SessionStore map[string]Client
var SessionStore2 map[string]PlaceAdmin
var storageMutex sync.RWMutex

var DB *sql.DB

type Client struct {
	loggedIn bool
}

type PlaceAdmin struct {
	PlaceID int64
}

type authenticationMiddleware struct {
	wrappedHandler http.Handler
}

func (h authenticationMiddleware) ServeHTTP(w http.ResponseWriter, r *http.Request) {

	SetNoCache(w)

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


func SetNoCache(w http.ResponseWriter) {
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

func Authenticate(h http.Handler) authenticationMiddleware {
	return authenticationMiddleware{h}
}

type restaurantAuthenticationMiddleware struct {
	wrappedHandler http.Handler
}

func (h restaurantAuthenticationMiddleware) ServeHTTP(w http.ResponseWriter, r *http.Request) {

	SetNoCache(w)

	cookie, err := r.Cookie("rsession")
	if err != nil {
		if err != http.ErrNoCookie {
			fmt.Fprint(w, err)
		} else {
			http.Redirect(w, r, "/restaurant/login_form", 301)
		}
		return
	}

	user, present := SessionStore2[cookie.Value]

	if !present {
		http.Redirect(w, r, "/restaurant/login_form", 301)
		return
	}

	r = r.WithContext(context.WithValue(r.Context(), "user", user))
	h.wrappedHandler.ServeHTTP(w, r)

}

func AuthenticateRestaurant(h http.Handler) http.Handler {
	return restaurantAuthenticationMiddleware{h}
}

const loginPage = "<html><head><title>Login</title></head><body><form action=\"login\" method=\"post\"> <input type=\"password\" name=\"password\" /> <input type=\"submit\" value=\"login\" /> </form> </body> </html>"

func HandleRestaurantLogin(w http.ResponseWriter, r *http.Request) {

	SetNoCache(w)

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
	}

	if subtle.ConstantTimeCompare([]byte(r.FormValue("password")), []byte(password.String)) != 1 {
		http.Redirect(w, r, "/restaurant/login_form?error=bad_credentials", 301)
	}

	cookie := &http.Cookie{
		Name:  "rsession",
		Value: uuid.NewV4().String(),
	}

	SessionStore2[cookie.Value] = PlaceAdmin{PlaceID: placeID}
	http.SetCookie(w, cookie)
	http.Redirect(w, r, "/restaurant/edit", 301)
}

func HandleLogin(w http.ResponseWriter, r *http.Request) {

	SetNoCache(w)

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

func HandleRestaurantLogout(w http.ResponseWriter, r *http.Request) {

	SetNoCache(w)

	cookie, err := r.Cookie("rsession")
	if err != nil {
		if err != http.ErrNoCookie {
			fmt.Fprint(w, err)
			return
		} else {
			err = nil
		}
	} else {
		delete(SessionStore2, cookie.Value)
	}

	http.Redirect(w, r, "/restaurant/login_form", 301)

}