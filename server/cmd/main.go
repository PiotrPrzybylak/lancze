package main

import (
	"database/sql"
	_ "github.com/lib/pq"
	"html"
	"html/template"
	"log"
	"net/http"
	"os"
	"strings"
	"strconv"
)

func main() {
	println("Hello!")

	db, err := sql.Open("postgres", os.Getenv("DATABASE_URL"))
	if err != nil {
		log.Fatalf("Error opening database: %q", err)
	}

	type Lunch struct {
		Place string
		Name  template.HTML
		Price float64
	}

	lunches := []Lunch{{"Restauracja Kolorowa", "✔️ consome wołowe z makaronem ryżowym ✔️ pilaw warzywny z kaszy pęczak z dynią", 11},
		{"Sznycelek", "Schabowy", 12.99},
		{"Sznycelek", "Schabowy", 12.99},
		{"Sznycelek", "Schabowy", 12.99},
		{"Sznycelek", "✔️ zupa pieczarkowa <br/> ✔️ spaghetti di mare", 19.00},
		{"Sznycelek", "✔️ zupa pieczarkowa ✔️ spaghetti di mare", 12.99}}

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {

		t, err := template.ParseFiles("server/home.html")
		if err != nil {
			panic( err)
		}

		rows, err := db.Query("SELECT offer, place_id FROM offers")
		if err != nil {
			panic(err)
		}

		lunches := []Lunch{}
		defer rows.Close()
		for rows.Next() {
			var name string
			var placeID int64;
			if err := rows.Scan(&name, &placeID); err != nil {
				panic(err)
			}
			lunches = append(lunches, Lunch{Name: template.HTML(name), Place: strconv.Itoa(int(placeID))})
		}


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
			panic( err)
		}

		t.Execute(w, names)
	})

	http.HandleFunc("/add", func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()
		println(r.Form)
		println(r.Form.Get("restaurant"))

		price, err := strconv.ParseFloat(r.Form.Get("price"), 64);
		if err != nil {
			panic(err);
		}
		lunches = append(lunches,
			Lunch{
				Place: r.Form.Get("restaurant"),
				Name: template.HTML(strings.Replace(html.EscapeString(r.Form.Get("menu")), "\n", "<br/>", -1)),
				Price: price})
		http.Redirect(w, r, "/", 301)
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "1234"
	}

	println(http.ListenAndServe(":"+port, nil))
}
