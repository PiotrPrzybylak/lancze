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
	"context"
	"github.com/PiotrPrzybylak/lancze/server/domain"
)

type Lunch struct {
	Date          time.LocalDate
	Place         string
	Name          template.HTML
	Price         float64
	PlaceWithZone Place
}

type Place struct {
	Name string
	Zone string
}

var sessionStore map[string]Client
var sessionStore2 map[string]PlaceAdmin
var storageMutex sync.RWMutex

type Client struct {
	loggedIn bool
}

type PlaceAdmin struct {
	placeID int64
}

var db *sql.DB

const loginPage = "<html><head><title>Login</title></head><body><form action=\"login\" method=\"post\"> <input type=\"password\" name=\"password\" /> <input type=\"submit\" value=\"login\" /> </form> </body> </html>"

func main() {

	sessionStore = make(map[string]Client)
	sessionStore2 = make(map[string]PlaceAdmin)

	println("Hello!")

	var err error
	db, err = sql.Open("postgres", os.Getenv("DATABASE_URL"))
	if err != nil {
		log.Fatalf("Error opening database: %q", err)
	}

	http.HandleFunc("/v3", func(w http.ResponseWriter, r *http.Request) {
		renderHome("server/home.html", r, db, w)
	})

	http.HandleFunc("/v4", func(w http.ResponseWriter, r *http.Request) {
		renderHome("server/homev4.html", r, db, w)
	})

	http.HandleFunc("/v2", func(w http.ResponseWriter, r *http.Request) {
		renderHome("server/templates/v2/index.html", r, db, w)
	})

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		renderHome("server/homev3.html", r, db, w)
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
		values["lunch"] = getLunch(db, today, int64(i))

		print(values)

		t.Execute(w, values)
	})))

	http.Handle("/admin/add", authenticate(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()

		price, err := strconv.ParseFloat(r.Form.Get("price"), 64);
		if err != nil {
			panic(err);
		}

		sqlStatement := `
		UPDATE offers
		SET offer = $1, price = $4
		WHERE place_id = $2 AND "date" = $3`
		res, err := db.Exec(sqlStatement, strings.Replace(html.EscapeString(r.Form.Get("menu")), "\n", "<br/>", -1), r.Form.Get("place_id"), r.Form.Get("date"), price)
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
			stmt, err := db.Prepare("INSERT INTO offers(offer, place_id, \"date\", price) VALUES($1, $2, $3, $4)")
			if err != nil {
				panic(err)
			}
			menu := template.HTML(strings.Replace(html.EscapeString(r.Form.Get("menu")), "\n", "<br/>", -1))
			_, err = stmt.Exec(menu, r.Form.Get("place_id"), r.Form.Get("date"), price)
			if err != nil {
				panic(err)
			}
		}
		http.Redirect(w, r, "/", 301)
	})))
	http.Handle("/admin/day", authenticate(http.HandlerFunc(renderAdminHome)))

	http.HandleFunc("/login", handleLogin)
	http.HandleFunc("/admin/login", handleLogin)
	http.HandleFunc("/admin/logout", handleLogout)

	http.HandleFunc("/restaurant/login_form", handleLoginForm)
	http.HandleFunc("/restaurant/login", handleRestaurantLogin)
	http.HandleFunc("/restaurant/logout", handleRestaurantLogout)
	http.Handle("/restaurant/edit", authenticateRestaurant(http.HandlerFunc(handleRestaurantEdit)))
	http.Handle("/restaurant/delete", authenticateRestaurant(http.HandlerFunc(handleRestaurantDelete)))
	http.Handle("/restaurant/add", authenticateRestaurant(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		user := r.Context().Value("user").(PlaceAdmin)

		r.ParseForm()

		priceString := r.Form.Get("price")
		priceString = strings.Replace(priceString, ",", ".", -1)
		price, err := strconv.ParseFloat(priceString, 64);
		if err != nil {
			panic(err);
		}

		sqlStatement := `
		UPDATE offers
		SET offer = $1, price = $4
		WHERE place_id = $2 AND "date" = $3`
		res, err := db.Exec(sqlStatement, strings.Replace(html.EscapeString(r.Form.Get("menu")), "\n", "<br/>", -1), r.Form.Get("place_id"), r.Form.Get("date"), price)
		if err != nil {
			panic(err)
		}

		count, err := res.RowsAffected()
		if err != nil {
			panic(err)
		}
		fmt.Printf("counter: %d\n", int(count))
		if count == 0 {
			stmt, err := db.Prepare("INSERT INTO offers(offer, place_id, \"date\", price) VALUES($1, $2, $3, $4)")
			if err != nil {
				panic(err)
			}
			menu := template.HTML(strings.Replace(html.EscapeString(r.Form.Get("menu")), "\n", "<br/>", -1))
			_, err = stmt.Exec(menu, user.placeID, r.Form.Get("date"), price)
			if err != nil {
				panic(err)
			}
		}
		http.Redirect(w, r, "/restaurant/edit", 301)
	})))


	fs := http.FileServer(http.Dir("server/static"))
	http.Handle("/static/", http.StripPrefix("/static/", fs))

	port := os.Getenv("PORT")
	if port == "" {
		port = "1234"
	}

	println(http.ListenAndServe(":"+port, nil))
}
func handleRestaurantDelete(w http.ResponseWriter, r *http.Request) {

	user := r.Context().Value("user").(PlaceAdmin)


		sqlStatement := `
		DELETE FROM offers
		WHERE place_id = $1 AND "date" = $2`
		res, err := db.Exec(sqlStatement, user.placeID, r.URL.Query().Get("date"))
		if err != nil {
			panic(err)
		}

		count, err := res.RowsAffected()
		if err != nil {
			panic(err)
		}

		print("Deleted rows: ", count, user.placeID,r.URL.Query().Get("date"))
	http.Redirect(w, r, "/restaurant/edit", 301)


}


func handleRestaurantEdit(w http.ResponseWriter, r *http.Request) {

	dateString := r.URL.Query().Get("date")
	var chosenDate time.LocalDate
	if dateString == "" {
		chosenDate = time.NewLocalDate(gotime.Now().Date())
	} else {
		chosenDate = time.MustParseLocalDate(dateString)
	}


	t, err := template.ParseFiles("server/restaurant.html")
	if err != nil {
		panic(err)
	}

	now := gotime.Now()
	today := time.NewLocalDate(now.Date())
	user := r.Context().Value("user").(PlaceAdmin)
	placeID := user.placeID

	places := getPlaces(db)

	values := map[string]interface{}{}
	values["id"] = placeID
	values["today"] = today
	values["lunch"] = getLunch(db, today, placeID)
	values["restaurant"] = places[placeID]





	date := today
	dates := []LunchForEdition{}
	for i := 1; i <= 20; i++ {
		lunch := getLunch(db, date, placeID)
		lunch.Date = date
		lunchForEdition := LunchForEdition{Lunch: lunch, I: i}
		if date == chosenDate {
				lunchForEdition.Edit = true
			}

		if date.Weekday() == time.Saturday || date.Weekday() == time.Sunday {
			lunchForEdition.Weekend = true
		}

		lunchForEdition.Weekday = domain.Weekdays[date.Weekday()]

		dates = append(dates, lunchForEdition)
		date = date.Next()

	}

	values["dates"] = dates
	values["chosenDate"] = chosenDate

	//print(values)

	err = t.Execute(w, values)
	if err != nil {
		panic(err)
	}
	}

func handleRestaurantLogin(w http.ResponseWriter, r *http.Request) {

	setNoCache(w)

	err := r.ParseForm()
	if err != nil {
		fmt.Fprint(w, err)
		return
	}

	username := r.FormValue("username")

		var placeID int64
		var password sql.NullString


	err = db.QueryRow("SELECT id, password FROM places WHERE username = $1", username).Scan(&placeID, &password)
	if err == sql.ErrNoRows {
		http.Redirect(w, r, "/restaurant/login_form?error=bad_credentials", 301)
		return
	}
	if err != nil {
		panic(err)
	}


	if ! password.Valid {
		http.Redirect(w, r, "/restaurant/login_form?error=bad_credentials", 301)
	}

	if subtle.ConstantTimeCompare([]byte(r.FormValue("password")), []byte(password.String)) != 1 {
		http.Redirect(w, r, "/restaurant/login_form?error=bad_credentials", 301)
	}

	cookie := &http.Cookie{
		Name:  "rsession",
		Value: uuid.NewV4().String(),
	}

	sessionStore2[cookie.Value] = PlaceAdmin{placeID: placeID}
	http.SetCookie(w, cookie)
	http.Redirect(w, r, "/restaurant/edit", 301)
}


func handleLoginForm(w http.ResponseWriter, r *http.Request) {

	t, err := template.ParseFiles("server/login.html")
	if err != nil {
		panic(err)
	}

	t.Execute(w, nil)
}

func renderHome(home_template string, r *http.Request, db *sql.DB, w http.ResponseWriter) {
	t, err := template.ParseFiles(home_template)
	if err != nil {
		panic(err)
	}
	date := getSelectedDate(r)
	places := getPlacesWithZone(db)
	lunches := getLunches(db, places, date)

	lunchesByZone :=  map[string][]Lunch{}
	for _, lunch := range lunches {
		lunchesByZone[lunch.PlaceWithZone.Zone] =append(lunchesByZone[lunch.PlaceWithZone.Zone], lunch)
	}

	lunches = lunchesByZone["off"]
	lunches = append(lunches, lunchesByZone["centrum"]...)
	lunches = append(lunches, lunchesByZone[""]...)

	values := map[string]interface{}{}
	values["offers"] = lunches
	values["date"] = date
	t.Execute(w, values)
}

func renderAdminHome(w http.ResponseWriter, r *http.Request) {
	t, err := template.ParseFiles("server/admin_day_review.html")
	if err != nil {
		panic(err)
	}
	date := getSelectedDate(r)
	places := getPlaces(db)
	lunches := []LunchForEdition{}

	for placeID, placeName := range  places {
		lunch := getLunch(db, date, placeID)
		lunch.Place = placeName
		lunches = append( lunches, LunchForEdition{ Lunch : lunch, PlaceID: placeID})

	}
	values := map[string]interface{}{}

	values["dates"] = lunches
	values["chosenDate"] = date
	err = t.Execute(w, values)
	if err != nil {
		panic(err)
	}
}

func getSelectedDate(r *http.Request) time.LocalDate {
	dateString := r.URL.Query().Get("date")
	var date time.LocalDate
	if dateString == "" {
		date = time.NewLocalDate(gotime.Now().Date())
	} else {
		date = time.MustParseLocalDate(dateString)
	}
	return date
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

func getPlacesWithZone(db *sql.DB) map[int64]Place {
	placesRows, err := db.Query("SELECT id, name, zone FROM places")
	if err != nil {
		panic(err)
	}
	defer placesRows.Close()
	places := map[int64]Place{}
	for placesRows.Next() {
		var place Place
		var placeID int64
		if err := placesRows.Scan(&placeID, &place.Name, &place.Zone); err != nil {
			panic(err)
		}
		places[placeID] = place
	}
	return places
}

func getLunches(db *sql.DB, places map[int64]Place, date time.LocalDate) []Lunch {
	rows, err := db.Query("SELECT offer, place_id, price FROM offers WHERE \"date\" = $1", date)
	if err != nil {
		panic(err)
	}
	defer rows.Close()
	lunches := []Lunch{}
	for rows.Next() {
		var name string
		var placeID int64
		var price float64
		if err := rows.Scan(&name, &placeID, &price); err != nil {
			panic(err)
		}
		lunches = append(lunches, Lunch{Name: template.HTML(name), Place: places[placeID].Name, Price: price, PlaceWithZone: places[placeID]})
	}
	return lunches
}

func getLunch(db *sql.DB, date time.LocalDate, placeID int64) Lunch {
	lunch := Lunch{}

	err := db.QueryRow("SELECT offer, price FROM offers WHERE \"date\" = $1 AND place_id = $2", date, placeID).Scan(&lunch.Name, &lunch.Price)
	if err == sql.ErrNoRows {
		return lunch
	}
	if err != nil {
		panic(err)
	}
	lunch.Name = template.HTML(strings.Replace(string(lunch.Name), "<br/>", "\n", -1))
	return lunch
}

type authenticationMiddleware struct {
	wrappedHandler http.Handler
}

type restaurantAuthenticationMiddleware struct {
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
		client = Client{loggedIn: false}
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



func authenticate(h http.Handler) authenticationMiddleware {
	return authenticationMiddleware{h}
}

func authenticateRestaurant(h http.Handler) http.Handler {
	return restaurantAuthenticationMiddleware{h}
}


func (h restaurantAuthenticationMiddleware) ServeHTTP(w http.ResponseWriter, r *http.Request) {

	setNoCache(w)

	cookie, err := r.Cookie("rsession")
	if err != nil {
		if err != http.ErrNoCookie {
			fmt.Fprint(w, err)
		} else {
			http.Redirect(w, r, "/restaurant/login_form", 301)
		}
		return
	}

	user, present := sessionStore2[cookie.Value]

	if !present {
		http.Redirect(w, r, "/restaurant/login_form", 301)
		return
	}

	r = r.WithContext(context.WithValue(r.Context(), "user", user))
	h.wrappedHandler.ServeHTTP(w, r)

}


func handleLogin(w http.ResponseWriter, r *http.Request) {

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
		client = Client{loggedIn: false}
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
		client := Client{ loggedIn: false}
		storageMutex.Lock()
		sessionStore[cookie.Value] = client
		storageMutex.Unlock()
	}
	http.Redirect(w, r, "/", 301)

}

func handleRestaurantLogout(w http.ResponseWriter, r *http.Request) {


	setNoCache(w)

	cookie, err := r.Cookie("rsession")
	if err != nil {
		if err != http.ErrNoCookie {
			fmt.Fprint(w, err)
			return
		} else {
			err = nil
		}
	} else {
		delete(sessionStore2, cookie.Value)
	}

	http.Redirect(w, r, "/restaurant/login_form", 301)

}


type LunchForEdition struct {
	Lunch Lunch
	I int
	Edit bool
	Weekend bool
	Weekday string
	PlaceID int64
}