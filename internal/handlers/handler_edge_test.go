package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
)

// This file contains tests for edge cases that can't be reached through the router
// These tests live in the handlers package (not handlers_test) to access unexported methods

func TestHandleVotePage_EmptyQRCode(t *testing.T) {
	// Create a minimal handlers instance with mock templates
	h := &Handlers{
		templates: &Templates{
			Vote: nil, // We won't reach template execution
		},
	}

	req := httptest.NewRequest(http.MethodGet, "/vote/", nil)
	rec := httptest.NewRecorder()

	// Create route context with empty qrCode parameter
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("qrCode", "")
	ctx := context.WithValue(req.Context(), chi.RouteCtxKey, rctx)
	req = req.WithContext(ctx)

	h.handleVotePage(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}
}

func TestHandleDeleteCategoryGroup_EmptyID(t *testing.T) {
	h := &Handlers{
		Category: nil, // We won't reach service calls
	}

	req := httptest.NewRequest(http.MethodDelete, "/api/admin/category-groups/", nil)
	rec := httptest.NewRecorder()

	// Create route context with empty id parameter
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "")
	ctx := context.WithValue(req.Context(), chi.RouteCtxKey, rctx)
	req = req.WithContext(ctx)

	h.handleDeleteCategoryGroup(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}
}

func TestHandleUpdateCategoryGroup_EmptyID(t *testing.T) {
	h := &Handlers{
		Category: nil, // We won't reach service calls
	}

	payload := map[string]interface{}{
		"name":          "Updated",
		"description":   "Desc",
		"display_order": 1,
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest(http.MethodPut, "/api/admin/category-groups/", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	// Create route context with empty id parameter
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "")
	ctx := context.WithValue(req.Context(), chi.RouteCtxKey, rctx)
	req = req.WithContext(ctx)

	h.handleUpdateCategoryGroup(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}
}

func TestHandleGetVoteData_EmptyQRCode(t *testing.T) {
	h := &Handlers{
		Voting: nil, // We won't reach service calls
	}

	req := httptest.NewRequest(http.MethodGet, "/api/vote/", nil)
	rec := httptest.NewRecorder()

	// Create route context with empty qrCode parameter
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("qrCode", "")
	ctx := context.WithValue(req.Context(), chi.RouteCtxKey, rctx)
	req = req.WithContext(ctx)

	h.handleGetVoteData(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}
}

func TestHandleGetCategoryGroup_EmptyID(t *testing.T) {
	h := &Handlers{
		Category: nil, // We won't reach service calls
	}

	req := httptest.NewRequest(http.MethodGet, "/api/admin/category-groups/", nil)
	rec := httptest.NewRecorder()

	// Create route context with empty id parameter
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "")
	ctx := context.WithValue(req.Context(), chi.RouteCtxKey, rctx)
	req = req.WithContext(ctx)

	h.handleGetCategoryGroup(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}
}

func TestParseIntParam_EmptyParam(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/test/", nil)

	// Create route context with empty id parameter
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "")
	ctx := context.WithValue(req.Context(), chi.RouteCtxKey, rctx)
	req = req.WithContext(ctx)

	_, err := parseIntParam(req, "id")
	if err == nil {
		t.Error("expected error for empty parameter, got nil")
	}
}
