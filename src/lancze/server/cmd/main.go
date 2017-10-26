package main

import (
	"net/http"
	"html/template"
	"fmt"
)

func main() {
	println("Hello!")

	t, err := template.ParseFiles("home.html")
	if err != nil {
		fmt.Errorf("Error parsing string", err)
	}

	type Lunch struct {
	Place string
		Name string
		Price float64
	}


	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		t.Execute(w, []Lunch{{"Restauracja Kolorowa","✔️ consome wołowe z makaronem ryżowym ✔️ pilaw warzywny z kaszy pęczak z dynią", 11}, { "Sznycelek", "Schabowy", 12.99}})
	})
	println(http.ListenAndServe(":2233", nil))
}
