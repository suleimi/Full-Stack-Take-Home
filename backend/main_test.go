package main

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/HW-Emissions/Full-Stack-Take-Home/eiae/internal/config"
	v1 "github.com/HW-Emissions/Full-Stack-Take-Home/eiae/internal/handlers/v1"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCorsMiddleware_OptionsShortCircuit(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(corsMiddleware())
	r.OPTIONS("/x", func(c *gin.Context) { c.Status(http.StatusOK) })

	req := httptest.NewRequest(http.MethodOptions, "/x", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusNoContent, rr.Code)
	assert.Equal(t, "*", rr.Header().Get("Access-Control-Allow-Origin"))
}

func TestNewServer_HealthzAndMetrics(t *testing.T) {
	a := &v1.AppV1{Config: &config.Config{Port: "8080"}}
	srv, err := newServer(a)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rr := httptest.NewRecorder()
	srv.Handler.ServeHTTP(rr, req)
	require.Equal(t, http.StatusOK, rr.Code)
	assert.Contains(t, rr.Body.String(), "up")

	req = httptest.NewRequest(http.MethodGet, "/metric", nil)
	rr = httptest.NewRecorder()
	srv.Handler.ServeHTTP(rr, req)
	require.Equal(t, http.StatusOK, rr.Code)
}
