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
)

type Lunch struct {
	Place string
	Name  template.HTML
	Price float64
}

func main() {
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

	http.HandleFunc("/admin", func(w http.ResponseWriter, r *http.Request) {

		rows, err := db.Query("SELECT name FROM places")
		if err != nil {
			panic(err)
		}

		names := []string{}
		defer rows.Close()
		for rows.Next() {
			var name string
			if err := rows.Scan(&name); err != nil {
				panic(err)
			}
			names = append(names, name)
		}

		t, err := template.ParseFiles("server/admin.html")
		if err != nil {
			panic(err)
		}

		t.Execute(w, names)
	})

	http.HandleFunc("/add", func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()
		println(r.Form)
		println(r.Form.Get("restaurant"))

		if err != nil {
			panic(err);
		}
		//lunches = append(lunches,
		//	Lunch{
		//		Place: r.Form.Get("restaurant"),
		//		Name:  template.HTML(strings.Replace(html.EscapeString(r.Form.Get("menu")), "\n", "<br/>", -1))})
		http.Redirect(w, r, "/", 301)
	})

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
func getLunches( db *sql.DB, places map[int64]string, date time.LocalDate) []Lunch {
	rows, err := db.Query("SELECT offer, place_id FROM offers WHERE \"date\" = $1"	, date )
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
