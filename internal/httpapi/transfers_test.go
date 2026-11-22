package httpapi

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"skillswap/internal/transfer"
)

type transferStub struct{ called bool }