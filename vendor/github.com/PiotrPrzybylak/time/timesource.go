package time

import (
	"context"
	"encoding/json"
	"github.com/pkg/errors"
	"net/http"
	"time"
)

type Source interface {
	Now(ctx context.Context) (*time.Time, error)
}

func NewSystemSource() Source {
	return systemSource{}
}

type systemSource struct{}

func (ss systemSource) Now(ctx context.Context) (*time.Time, error) {
	now := time.Now()
	return &now, nil
}

func NewRESTSource(url string) Source {
	return restSource{url}
}

type restSource struct {
	url string
}

func (rs restSource) Now(ctx context.Context) (*time.Time, error) {

	// Build the request
	req, err := http.NewRequest("GET", rs.url, nil)
	if err != nil {
		return nil, errors.Wrap(err, "NewRequest: ")
	}

	// For control over HTTP client headers,
	// redirect policy, and other settings,
	// create a Client
	// A Client is an HTTP client
	client := &http.Client{}

	// Send the request via a client
	// Do sends an HTTP request and
	// returns an HTTP response
	resp, err := client.Do(req)
	if err != nil {
		return nil, errors.Wrap(err, "Do: ")
	}

	// Callers should close resp.Body
	// when done reading from it
	// Defer the closing of the body
	defer resp.Body.Close()

	type TimeResponse struct {
		Time string
	}

	// Fill the record with the data from the JSON
	var record TimeResponse

	// Use json.Decode for reading streams of JSON data
	if err := json.NewDecoder(resp.Body).Decode(&record); err != nil {
		return nil, errors.Wrap(err, "Decode: ")
	}

	time, err := time.Parse(time.RFC3339, record.Time)
	return &time, err
}
