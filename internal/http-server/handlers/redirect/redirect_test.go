package redirect_test

import (
	"URLShortener/internal/http-server/handlers/redirect"
	"URLShortener/internal/http-server/handlers/redirect/mocks"
	"URLShortener/internal/lib/api"
	resp "URLShortener/internal/lib/api/response"
	"URLShortener/internal/lib/logger/handlers/slogdiscard"
	"URLShortener/internal/storage"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/require"
)

// Успешный кейс - сквозной тест через реальный роутер и HTTP-клиент.
// Клиент останавливается ровно на 302 (см. api.GetRedirect), поэтому
// реального перехода на resURL не происходит.
func TestRedirectHandler_Success(t *testing.T) {
	urlGetterMock := mocks.NewURLGetter(t)
	urlGetterMock.On("GetURL", "test_alias").
		Return("https://www.google.com/", nil).
		Once()

	r := chi.NewRouter()
	r.Get("/{alias}", redirect.New(slogdiscard.NewDiscardLogger(), urlGetterMock))

	ts := httptest.NewServer(r)
	defer ts.Close()

	redirectedToURL, err := api.GetRedirect(ts.URL + "/test_alias")
	require.NoError(t, err)
	require.Equal(t, "https://www.google.com/", redirectedToURL)
}

// Ошибочные ветки - прямой вызов хендлера, без реального роутинга.
// "Пустой alias" в реальном chi-роутинге до хендлера вообще не дойдёт
// (паттерн /{alias} не матчит пустой сегмент), поэтому единственный
// способ проверить эту защитную ветку - вызвать хендлер напрямую.
func TestRedirectHandler_Errors(t *testing.T) {
	type testCase struct {
		name              string
		alias             string
		mockError         error
		expectGetCall     bool
		expectedRespError string
	}

	cases := []testCase{
		{
			name:              "Empty alias",
			alias:             "",
			expectGetCall:     false,
			expectedRespError: "invalid request",
		},
		{
			name:              "URL not found",
			alias:             "missing_alias",
			mockError:         storage.ErrURLNotFound,
			expectGetCall:     true,
			expectedRespError: "not found",
		},
		{
			name:              "GetURL internal error",
			alias:             "test_alias",
			mockError:         errors.New("database error"),
			expectGetCall:     true,
			expectedRespError: "internal error",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			urlGetterMock := mocks.NewURLGetter(t)
			if tc.expectGetCall {
				urlGetterMock.On("GetURL", tc.alias).
					Return("", tc.mockError).
					Once()
			}

			handler := redirect.New(slogdiscard.NewDiscardLogger(), urlGetterMock)

			req := httptest.NewRequest(http.MethodGet, "/"+tc.alias, nil)
			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("alias", tc.alias)
			req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

			rr := httptest.NewRecorder()
			handler.ServeHTTP(rr, req)

			require.Equal(t, http.StatusOK, rr.Code)

			var response resp.Response
			require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &response))
			require.Equal(t, "Error", response.Status)
			require.Equal(t, tc.expectedRespError, response.Error)
		})
	}
}
