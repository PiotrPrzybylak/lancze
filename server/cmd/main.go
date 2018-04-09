package main

import (
	"database/sql"
	_ "github.com/lib/pq"
	"html/template"
	"log"
	"net/http"
	"os"
	"github.com/PiotrPrzybylak/time"
	gotime "time"
	"strings"
	"html"
	"sync"
	fmt "fmt"
	"crypto/subtle"
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

		print(values)

		t.Execute(w, values);
	})))

	http.Handle("/add", authenticate(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()

		if err != nil {
			panic(err);
		}

		stmt, err := db.Prepare("INSERT INTO offers(offer, place_id, \"date\") VALUES($1, $2, $3)")
		if err != nil {
			panic(err)
		}
		menu := template.HTML(strings.Replace(html.EscapeString(r.Form.Get("menu")), "\n", "<br/>", -1))
		_, err = stmt.Exec(menu, r.Form.Get("place_id"), r.Form.Get("date"))
		if err != nil {
			panic(err)
		}
		http.Redirect(w, r, "/", 301)
	})))


	http.HandleFunc("/login", handleLogin)

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
		var placeID int64;
		if err := placesRows.Scan(&placeID, &name); err != nil {
			panic(err)
		}
		places[placeID] = name;
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
		var placeID int64;
		if err := rows.Scan(&name, &placeID); err != nil {
			panic(err)
		}
		lunches = append(lunches, Lunch{Name: template.HTML(name), Place: places[placeID]})
	}
	return lunches
}



type authenticationMiddleware struct {
	wrappedHandler http.Handler
}

func (h authenticationMiddleware) ServeHTTP(w http.ResponseWriter, r *http.Request) {
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
			Value: "DUUUUUPAAA",
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
			Value: "aaaaaaaaaaaaaaaaaaaaaaa",
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
		fmt.Fprintln(w, "Thank you for logging in.")
		storageMutex.Lock()
		sessionStore[cookie.Value] = client
		storageMutex.Unlock()
	} else {
		fmt.Fprintln(w, "Wrong password.")
	}

}
