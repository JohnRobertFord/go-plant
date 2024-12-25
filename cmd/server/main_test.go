package main

import (
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/JohnRobertFord/go-plant/internal/config"
	"github.com/JohnRobertFord/go-plant/internal/server"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testRequest(t *testing.T, ts *httptest.Server, method, path string) (*http.Response, string) {

	req, err := http.NewRequest(method, ts.URL+path, nil)
	require.NoError(t, err)

	resp, err := ts.Client().Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	return resp, string(respBody)
}

func TestMetricRouter(t *testing.T) {
	cfg, err := config.InitConfig()
	if err != nil {
		log.Fatalf("cant start server: %e", err)
	}

	m := server.NewMemStorage(cfg)
	ts := httptest.NewServer(MetricRouter(m))
	defer ts.Close()

	var tests = []struct {
		name   string
		url    string
		method string
		want   string
		status int
	}{
		{
			name:   "First test",
			url:    "/",
			method: "GET",
			want:   "",
			status: http.StatusOK,
		},
		{
			name:   "Second test",
			url:    "/update/guage/test/17",
			method: "PUT",
			want:   "Only POST or GET requests are allowed!\n",
			status: http.StatusMethodNotAllowed,
		},
		{
			name:   "Third test",
			url:    "/notvalidpath",
			method: "GET",
			want:   "",
			status: http.StatusNotFound,
		},
		{
			name:   "Fourth test",
			url:    "/value/counter/test89",
			method: "GET",
			want:   "",
			status: http.StatusNotFound,
		},
		{
			name:   "Fifth test",
			url:    "/update/gauge/Alloc/188893",
			method: "POST",
			want:   "",
			status: http.StatusOK,
		},
		{
			name:   "Sixth test",
			url:    "/update/counter/counter/1",
			method: "POST",
			want:   "",
			status: http.StatusOK,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			resp, _ := testRequest(t, ts, test.method, test.url)
			assert.Equal(t, test.status, resp.StatusCode)
			// assert.Equal(t, test.want, get)
			defer resp.Body.Close()

		})

	}
}
