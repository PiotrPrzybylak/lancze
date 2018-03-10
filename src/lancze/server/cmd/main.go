package main

import (
	"net/http"
	"html/template"
	"fmt"
)

func main() {
	println("Hello!")



	type Lunch struct {
	Place string
		Name template.HTML
		Price float64
	}


	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {

		t, err := template.ParseFiles("home.html")
		if err != nil {
			fmt.Errorf("Error parsing string", err)
		}

		t.Execute(w, []Lunch{{"Restauracja Kolorowa","✔️ consome wołowe z makaronem ryżowym ✔️ pilaw warzywny z kaszy pęczak z dynią", 11},
		{ "Sznycelek", "Schabowy", 12.99},
		{ "Sznycelek", "Schabowy", 12.99},
		{ "Sznycelek", "Schabowy", 12.99},
		{ "Sznycelek", "✔️ zupa pieczarkowa <br/> ✔️ spaghetti di mare", 19.00},
		{ "Sznycelek", "✔️ zupa pieczarkowa ✔️ spaghetti di mare", 12.99}})
	})
	println(http.ListenAndServe(":2233", nil))
}
