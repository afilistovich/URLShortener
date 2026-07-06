package main

import (
	"net/http"
	"testing"
)

func TestShortenerHandler(t *testing.T) {

	cases := []struct {
		name       string
		body       string
		wantStatus int
		wantBody   bool
	}{
		{
			name:       "valid https url",
			body:       "https://practicum.yandex.ru/",
			wantStatus: http.StatusCreated,
			wantBody:   true,
		},
		{
			name:       "valid http url",
			body:       "http://example.com/path?q=1",
			wantStatus: http.StatusCreated,
			wantBody:   true,
		},
		{
			name:       "url with trailing newline is trimmed",
			body:       "https://ya.ru\n",
			wantStatus: http.StatusCreated,
			wantBody:   true,
		},
	}

}
