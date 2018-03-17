package main

import (
	"database/sql"
	"fmt"
	_ "github.com/lib/pq"
	"html"
	"html/template"
	"log"
	"net/http"
	"os"
	"strings"
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

		t, err := template.ParseFiles("src/lancze/server/home.html")
		if err != nil {
			panic( err)
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

		t, err := template.ParseFiles("src/lancze/server/admin.html")
		if err != nil {
			panic( err)
		}

		t.Execute(w, names)
	})

	http.HandleFunc("/add", func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()
		println(r.Form)
		println(r.Form.Get("restaurant"))
		lunches = append(lunches, Lunch{Place: r.Form.Get("restaurant"), Name: template.HTML(strings.Replace(html.EscapeString(r.Form.Get("menu")), "\n", "<br/>", -1)), Price: 123})
		http.Redirect(w, r, "/", 301)
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "1234"
	}

	println(http.ListenAndServe(":"+port, nil))
}
