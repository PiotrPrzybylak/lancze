package main

import (
	"database/sql"
	"fmt"
	"github.com/PiotrPrzybylak/lancze/server/app/auth"
	"github.com/PiotrPrzybylak/lancze/server/domain"
	"github.com/PiotrPrzybylak/time"
	_ "github.com/lib/pq"
	"html"
	"html/template"
	"log"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"
	gotime "time"
)

type Lunch struct {
	Date          time.LocalDate
	Place         string
	Name          template.HTML
	Price         float64
	PlaceWithZone Place
}

func (l Lunch) PriceFormatted() string {
	precision := 2
	if isIntegral(l.Price) {
		precision = 0
	}
	return strconv.FormatFloat(l.Price, 'f', precision, 64)
}

func isIntegral(val float64) bool {
	return val == float64(int(val))
}

type Place struct {
	Name    string
	Zone    string
	Address string
	Phone   string
}

var db *sql.DB

func main() {

	println("Hello!")

	var err error
	db, err = sql.Open("postgres", os.Getenv("DATABASE_URL"))
	if err != nil {
		log.Fatalf("Error opening database: %q", err)
	}

	users := auth.NewAuth(db, "user_session", "/restaurant/login_form", "/restaurant/edit", "SELECT id, password FROM places WHERE username = $1")
	admins := auth.NewAuth(db, "admin_session", "/admin/login_form", "/admin/places", "SELECT 0, password FROM admins WHERE username = $1")

	http.HandleFunc("/dev", func(w http.ResponseWriter, r *http.Request) {
		renderHome("server/home-dev.html", r, db, w)
	})

	http.HandleFunc("/skriny", func(w http.ResponseWriter, r *http.Request) {
		renderHome("server/skriny.html", r, db, w)
	})

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		renderHome("server/home.html", r, db, w)
	})

	http.Handle("/admin", admins.Authenticate(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		t, err := template.ParseFiles("server/admin.html")
		if err != nil {
			panic(err)
		}

		t.Execute(w, getPlaces(db))
	})))

	http.Handle("/admin/places", admins.Authenticate(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		t, err := template.ParseFiles("server/places.html")
		if err != nil {
			panic(err)
		}

		t.Execute(w, getPlaces(db))
	})))

	http.Handle("/admin/place", admins.Authenticate(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

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

	http.Handle("/admin/add", admins.Authenticate(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()

		price, err := strconv.ParseFloat(r.Form.Get("price"), 64)
		if err != nil {
			if err != nil {
				print("ERROR: Błędna cena!")
				print(err)
				http.Redirect(w, r, "/restaurant/edit?error=Błędna cena", 301)
				return
			}
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
	http.Handle("/admin/day", admins.Authenticate(http.HandlerFunc(renderAdminHome)))

	http.HandleFunc("/admin/login_form", handleLoginForm("/admin/login"))
	http.HandleFunc("/admin/login", admins.HandleLogin())
	http.HandleFunc("/admin/logout", admins.HandleLogout())

	http.HandleFunc("/restaurant/login_form", handleLoginForm("/restaurant/login"))
	http.HandleFunc("/restaurant/login", users.HandleLogin())
	http.HandleFunc("/restaurant/logout", users.HandleLogout())
	http.Handle("/restaurant/edit", users.Authenticate(http.HandlerFunc(handleRestaurantEdit)))
	http.Handle("/restaurant/delete", users.Authenticate(http.HandlerFunc(handleRestaurantDelete)))
	http.Handle("/restaurant/add", users.Authenticate(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		user := auth.CurrentUser(r)

		r.ParseForm()

		priceString := r.Form.Get("price")
		priceString = strings.Replace(priceString, ",", ".", -1)
		price, err := strconv.ParseFloat(priceString, 64)
		if err != nil {
			if err != nil {
				print("ERROR: Błędna cena!")
				print(err)
				http.Redirect(w, r, "/restaurant/edit?error=Błędna cena", 301)
				return
			}
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
			_, err = stmt.Exec(menu, user.PlaceID, r.Form.Get("date"), price)
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

	user := auth.CurrentUser(r)

	sqlStatement := `
		DELETE FROM offers
		WHERE place_id = $1 AND "date" = $2`
	res, err := db.Exec(sqlStatement, user.PlaceID, r.URL.Query().Get("date"))
	if err != nil {
		panic(err)
	}

	count, err := res.RowsAffected()
	if err != nil {
		panic(err)
	}

	print("Deleted rows: ", count, user.PlaceID, r.URL.Query().Get("date"))
	http.Redirect(w, r, "/restaurant/edit", 301)

}

func handleRestaurantEdit(w http.ResponseWriter, r *http.Request) {

	dateString := r.URL.Query().Get("date")
	var chosenDate time.LocalDate
	var err error
	if dateString == "" {
		chosenDate = time.NewLocalDate(gotime.Now().Date())
	} else {
		chosenDate, err = time.ParseLocalDate(dateString)
		if err != nil {
			print(err)
			http.Redirect(w, r, "/restaurant/edit?error=Błędna data", 301)
			return
		}
	}

	t, err := template.ParseFiles("server/restaurant.html")
	if err != nil {
		panic(err)
	}

	now := gotime.Now()
	today := time.NewLocalDate(now.Date())
	user := auth.CurrentUser(r)
	placeID := user.PlaceID

	places := getPlaces(db)

	values := map[string]interface{}{}
	values["id"] = placeID
	values["today"] = today
	values["lunch"] = getLunch(db, today, placeID)
	values["restaurant"] = places[placeID]
	values["error"] = r.URL.Query().Get("error")

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

	err = t.Execute(w, values)
	if err != nil {
		panic(err)
	}
}

func handleLoginForm(loginURL string) http.HandlerFunc {

	return func(w http.ResponseWriter, r *http.Request) {

		t, err := template.ParseFiles("server/login.html")
		if err != nil {
			panic(err)
		}

		t.Execute(w, loginURL)
	}
}

func renderHome(home_template string, r *http.Request, db *sql.DB, w http.ResponseWriter) {
	t, err := template.ParseFiles(home_template)
	if err != nil {
		panic(err)
	}
	date := getSelectedDate(r)
	places := getPlacesWithZone(db)
	lunches := getLunches(db, places, date)

	sort.Slice(lunches, func(i, j int) bool { return lunches[i].Place < lunches[j].Place })

	lunchesByZone := map[string][]Lunch{}
	for _, lunch := range lunches {
		lunchesByZone[lunch.PlaceWithZone.Zone] = append(lunchesByZone[lunch.PlaceWithZone.Zone], lunch)
	}

	type Zone struct {
		Name   string
		Offers []Lunch
	}

	values := map[string]interface{}{}
	values["zones"] = []Zone{
		{"OFF Piotrkowska", lunchesByZone["off"]},
		{"Centrum", lunchesByZone["centrum"]},
		{"Piotrkowska 217", lunchesByZone["off2"]},
		{"Łódź", lunchesByZone["lodz"]},
	}
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

	for placeID, placeName := range places {
		lunch := getLunch(db, date, placeID)
		lunch.Place = placeName
		lunches = append(lunches, LunchForEdition{Lunch: lunch, PlaceID: placeID})

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
	placesRows, err := db.Query("SELECT id, name, zone, address, phone FROM places")
	if err != nil {
		panic(err)
	}
	defer placesRows.Close()
	places := map[int64]Place{}
	for placesRows.Next() {
		var place Place
		var placeID int64
		var phone sql.NullString
		if err := placesRows.Scan(&placeID, &place.Name, &place.Zone, &place.Address, &phone); err != nil {
			panic(err)
		}
		if phone.Valid {
			place.Phone = phone.String
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

type LunchForEdition struct {
	Lunch   Lunch
	I       int
	Edit    bool
	Weekend bool
	Weekday string
	PlaceID int64
}
