package main

import "net/http"

func main() {
	println("Hello!")
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Hejka"))
	})
	println(http.ListenAndServe(":2233", nil))
}
