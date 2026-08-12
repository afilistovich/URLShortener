package save_test

import (
	"URLShortener/internal/http-server/handlers/save"
	"URLShortener/internal/http-server/handlers/save/mocks"
	"URLShortener/internal/storage"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"URLShortener/internal/lib/logger/handlers/slogdiscard"
)

func TestSaveHandler_Enhanced(t *testing.T) {
	type testCase struct {
		name           string
		alias          string
		url            string
		respError      string
		mockError      error
		expectSaveCall bool
		expectedStatus int
	}

	cases := []testCase{
		{
			name:           "Success with alias",
			alias:          "test_alias",
			url:            "https://google.com",
			expectSaveCall: true,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Success without alias (auto-generate)",
			alias:          "",
			url:            "https://google.com",
			expectSaveCall: true,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Empty URL",
			alias:          "some_alias",
			url:            "",
			respError:      "field URL is a required field",
			expectSaveCall: false,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Invalid URL",
			alias:          "some_alias",
			url:            "invalid-url",
			respError:      "field URL is not a valid URL",
			expectSaveCall: false,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "SaveURL error",
			alias:          "test_alias",
			url:            "https://google.com",
			respError:      "failed to add url",
			mockError:      errors.New("database error"),
			expectSaveCall: true,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "URL already exists",
			alias:          "existing",
			url:            "https://example.com",
			respError:      "url already exists",
			mockError:      storage.ErrURLExists,
			expectSaveCall: true,
			expectedStatus: http.StatusOK,
		},
	}

	for _, tc := range cases {

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			urlSaverMock := mocks.NewURLSaver(t)

			if tc.expectSaveCall {
				if tc.alias == "" {
					urlSaverMock.On("SaveURL", tc.url, mock.AnythingOfType("string")).
						Return(int64(1), tc.mockError).
						Once()
				} else {
					urlSaverMock.On("SaveURL", tc.url, tc.alias).
						Return(int64(1), tc.mockError).
						Once()
				}
			}

			handler := save.New(slogdiscard.NewDiscardLogger(), urlSaverMock)

			reqBody := fmt.Sprintf(`{"url": "%s", "alias": "%s"}`, tc.url, tc.alias)
			req := httptest.NewRequest(http.MethodPost, "/save", bytes.NewReader([]byte(reqBody)))
			rr := httptest.NewRecorder()

			handler.ServeHTTP(rr, req)

			require.Equal(t, tc.expectedStatus, rr.Code)

			var resp save.Response
			require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &resp))

			require.Equal(t, tc.respError, resp.Error)

			if tc.respError == "" {
				require.Equal(t, "OK", resp.Status)
				require.NotEmpty(t, resp.Alias)

				if tc.alias != "" {
					require.Equal(t, tc.alias, resp.Alias)
				} else {
					require.Len(t, resp.Alias, 6) // aliasLength в save.go
				}
			} else {
				require.Equal(t, "Error", resp.Status)
				require.Empty(t, resp.Alias)
			}

			urlSaverMock.AssertExpectations(t)
		})
	}
}

func TestSaveHandler_RequiresAuth(t *testing.T) {
	urlSaverMock := mocks.NewURLSaver(t)
	urlSaverMock.On("SaveURL", mock.Anything, mock.Anything).
		Return(int64(1), nil).
		Maybe() // не должен вызваться при 401

	r := chi.NewRouter()
	r.Route("/url", func(r chi.Router) {
		r.Use(middleware.BasicAuth("url-shortener", map[string]string{
			"admin": "secret",
		}))
		r.Post("/", save.New(slogdiscard.NewDiscardLogger(), urlSaverMock))
	})

	ts := httptest.NewServer(r)
	defer ts.Close()

	body := `{"url": "https://google.com"}`

	t.Run("без креденшлов -> 401", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodPost, ts.URL+"/url", strings.NewReader(body))
		res, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer res.Body.Close()
		require.Equal(t, http.StatusUnauthorized, res.StatusCode)
	})

	t.Run("с верными креденшлами -> 200", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodPost, ts.URL+"/url", strings.NewReader(body))
		req.SetBasicAuth("admin", "secret")
		res, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer res.Body.Close()
		require.Equal(t, http.StatusOK, res.StatusCode)
	})
}
