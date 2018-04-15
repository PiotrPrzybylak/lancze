package main

import (
	"crypto/subtle"
	"database/sql"
	"fmt"
	"github.com/PiotrPrzybylak/time"
	_ "github.com/lib/pq"
	"github.com/satori/go.uuid"
	"html"
	"html/template"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	gotime "time"
)

type Lunch struct {
	Place string
	Name  template.HTML
	Price float64
}

var sessionStore map[string]Client
var storageMutex sync.RWMutex

type Client struct {
	loggedIn bool
}

const loginPage = "<html><head><title>Login</title></head><body><form action=\"login\" method=\"post\"> <input type=\"password\" name=\"password\" /> <input type=\"submit\" value=\"login\" /> </form> </body> </html>"

func main() {

	sessionStore = make(map[string]Client)

	println("Hello!")

	db, err := sql.Open("postgres", os.Getenv("DATABASE_URL"))
	if err != nil {
		log.Fatalf("Error opening database: %q", err)
	}

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {

		t, err := template.ParseFiles("server/home.html")
		if err != nil {
			panic(err)
		}

		dateString := r.URL.Query().Get("date")
		var date time.LocalDate
		if dateString == "" {
			date = time.NewLocalDate(gotime.Now().Date())
		} else {
			date = time.MustParseLocalDate(dateString)
		}

		places := getPlaces(db)

		lunches := getLunches(db, places, date)

		t.Execute(w, lunches)
	})

	http.HandleFunc("/v2", func(w http.ResponseWriter, r *http.Request) {

		t, err := template.ParseFiles("server/templates/v2/index.html")
		if err != nil {
			panic(err)
		}

		dateString := r.URL.Query().Get("date")
		var date time.LocalDate
		if dateString == "" {
			date = time.NewLocalDate(gotime.Now().Date())
		} else {
			date = time.MustParseLocalDate(dateString)
		}

		places := getPlaces(db)

		lunches := getLunches(db, places, date)

		t.Execute(w, lunches)
	})

	http.Handle("/admin", authenticate(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		t, err := template.ParseFiles("server/admin.html")
		if err != nil {
			panic(err)
		}

		t.Execute(w, getPlaces(db))
	})))

	http.Handle("/admin/places", authenticate(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		t, err := template.ParseFiles("server/places.html")
		if err != nil {
			panic(err)
		}

		t.Execute(w, getPlaces(db))
	})))

	http.Handle("/admin/place", authenticate(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		t, err := template.ParseFiles("server/place.html")
		if err != nil {
			panic(err)
		}

		now := gotime.Now()
		today := time.NewLocalDate(now.Date())
		id := r.URL.Query().Get("id")

		values := map[string]interface{}{}
		values["id"] = id
		values["today"] = today
		i, err := strconv.Atoi(id)
		if err != nil {
			panic(err)
		}
		values["offer"] = getLunch(db, today, int64(i))

		print(values)

		t.Execute(w, values)
	})))

	http.Handle("/admin/add", authenticate(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()

		if err != nil {
			panic(err)
		}

		sqlStatement := `
UPDATE offers
SET offer = $1
WHERE place_id = $2 AND "date" = $3`
		res, err := db.Exec(sqlStatement, strings.Replace(html.EscapeString(r.Form.Get("menu")), "\n", "<br/>", -1), r.Form.Get("place_id"), r.Form.Get("date"))
		if err != nil {
			panic(err)
		}

		fmt.Printf("TEST 123")

		count, err := res.RowsAffected()
		if err != nil {
			panic(err)
		}
		fmt.Printf("counter: %d\n", int(count))
		if count == 0 {
			stmt, err := db.Prepare("INSERT INTO offers(offer, place_id, \"date\") VALUES($1, $2, $3)")
			if err != nil {
				panic(err)
			}
			menu := template.HTML(strings.Replace(html.EscapeString(r.Form.Get("menu")), "\n", "<br/>", -1))
			_, err = stmt.Exec(menu, r.Form.Get("place_id"), r.Form.Get("date"))
			if err != nil {
				panic(err)
			}
		}
		http.Redirect(w, r, "/", 301)
	})))

	http.HandleFunc("/login", handleLogin)
	http.HandleFunc("/admin/login", handleLogin)
	http.HandleFunc("/admin/logout", handleLogout)

	fs := http.FileServer(http.Dir("server/static"))
	http.Handle("/static/", http.StripPrefix("/static/", fs))

	port := os.Getenv("PORT")
	if port == "" {
		port = "1234"
	}


	println(http.ListenAndServe(":"+port, nil))
}
func getPlaces(db *sql.DB) map[int64]string {
	placesRows, err := db.Query("SELECT id, name FROM places")
	if err != nil {
		panic(err)
	}
	defer placesRows.Close()
	places := map[int64]string{}
	for placesRows.Next() {
		var name string
		var placeID int64
		if err := placesRows.Scan(&placeID, &name); err != nil {
			panic(err)
		}
		places[placeID] = name
	}
	return places
}
func getLunches(db *sql.DB, places map[int64]string, date time.LocalDate) []Lunch {
	rows, err := db.Query("SELECT offer, place_id FROM offers WHERE \"date\" = $1", date)
	if err != nil {
		panic(err)
	}
	defer rows.Close()
	lunches := []Lunch{}
	for rows.Next() {
		var name string
		var placeID int64
		if err := rows.Scan(&name, &placeID); err != nil {
			panic(err)
		}
		lunches = append(lunches, Lunch{Name: template.HTML(name), Place: places[placeID]})
	}
	return lunches
}

func getLunch(db *sql.DB, date time.LocalDate, placeID int64) string {
	var offer string
	err := db.QueryRow("SELECT offer FROM offers WHERE \"date\" = $1 AND place_id = $2", date, placeID).Scan(&offer)
	if err == sql.ErrNoRows {
		return ""
	}
	if err != nil {
		panic(err)
	}
	offer = strings.Replace(offer, "<br/>", "\n", -1)
	return offer
}

type authenticationMiddleware struct {
	wrappedHandler http.Handler
}

func (h authenticationMiddleware) ServeHTTP(w http.ResponseWriter, r *http.Request) {

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
	var client Client
	if cookie != nil {
		storageMutex.RLock()
		client, present = sessionStore[cookie.Value]
		storageMutex.RUnlock()
	} else {
		present = false
	}

	if present == false {
		cookie = &http.Cookie{
			Name:  "session",
			Value: uuid.NewV4().String(),
		}
		client = Client{false}
		storageMutex.Lock()
		sessionStore[cookie.Value] = client
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

func authenticate(h http.Handler) authenticationMiddleware {
	return authenticationMiddleware{h}
}

func handleLogin(w http.ResponseWriter, r *http.Request) {
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
		client, present = sessionStore[cookie.Value]
		storageMutex.RUnlock()
	} else {
		present = false
	}

	if present == false {
		cookie = &http.Cookie{
			Name:  "session",
			Value: uuid.NewV4().String(),
		}
		client = Client{false}
		storageMutex.Lock()
		sessionStore[cookie.Value] = client
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
		sessionStore[cookie.Value] = client
		storageMutex.Unlock()
		http.Redirect(w, r, "/admin/places", 301)
	} else {
		fmt.Fprintln(w, "Wrong password.")
	}

}

func handleLogout(w http.ResponseWriter, r *http.Request) {

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
		_, present = sessionStore[cookie.Value]
		storageMutex.RUnlock()
	} else {
		present = false
	}

	if present == true {
		client := Client{false}
		storageMutex.Lock()
		sessionStore[cookie.Value] = client
		storageMutex.Unlock()
	}
	http.Redirect(w, r, "/", 301)

}
