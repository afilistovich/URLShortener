package remove_test

import (
	"URLShortener/internal/http-server/handlers/remove"
	"URLShortener/internal/http-server/handlers/remove/mocks"
	"URLShortener/internal/storage"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/require"

	"URLShortener/internal/lib/logger/handlers/slogdiscard"
)

func TestRemoveHandler_Enhanced(t *testing.T) {
	type testCase struct {
		name             string
		alias            string
		respError        string
		mockError        error
		expectRemoveCall bool
		expectedStatus   int
	}

	cases := []testCase{
		{
			name:             "Success",
			alias:            "test_alias",
			expectRemoveCall: true,
			expectedStatus:   http.StatusOK,
		},
		{
			name:             "Empty alias",
			alias:            "",
			respError:        "invalid request",
			expectRemoveCall: false,
			expectedStatus:   http.StatusOK,
		},
		{
			name:             "URL not found",
			alias:            "missing_alias",
			respError:        "url not found",
			mockError:        storage.ErrURLNotFound,
			expectRemoveCall: true,
			expectedStatus:   http.StatusOK,
		},
		{
			name:             "DeleteURL internal error",
			alias:            "test_alias",
			respError:        "internal error",
			mockError:        errors.New("database error"),
			expectRemoveCall: true,
			expectedStatus:   http.StatusOK,
		},
	}

	for _, tc := range cases {

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			urlRemoverMock := mocks.NewURLRemover(t)

			if tc.expectRemoveCall {
				urlRemoverMock.On("DeleteURL", tc.alias).
					Return(tc.mockError).
					Once()
			}

			handler := remove.New(slogdiscard.NewDiscardLogger(), urlRemoverMock)

			req := httptest.NewRequest(http.MethodDelete, "/url/"+tc.alias, nil)

			// chi.URLParam читает alias из RouteContext, поэтому его нужно
			// положить в контекст запроса вручную, минуя реальный роутер.
			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("alias", tc.alias)
			req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

			rr := httptest.NewRecorder()

			handler.ServeHTTP(rr, req)

			require.Equal(t, tc.expectedStatus, rr.Code)

			var resp remove.Response
			require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &resp))

			require.Equal(t, tc.respError, resp.Error)

			if tc.respError == "" {
				require.Equal(t, "OK", resp.Status)
			} else {
				require.Equal(t, "Error", resp.Status)
			}

			urlRemoverMock.AssertExpectations(t)
		})
	}
}
