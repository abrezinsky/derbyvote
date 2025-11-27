package handlers

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/abrezinsky/derbyvote/internal/models"
)

// Stock placeholder image (simple gray SVG)
var stockPhotoSVG = []byte(`<svg xmlns="http://www.w3.org/2000/svg" width="200" height="150" viewBox="0 0 200 150">
  <rect width="200" height="150" fill="#e5e7eb"/>
  <text x="100" y="75" font-family="Arial, sans-serif" font-size="14" fill="#9ca3af" text-anchor="middle" dominant-baseline="middle">No Photo</text>
</svg>`)

// handleVotePage serves the voting page
func (h *Handlers) handleVotePage(w http.ResponseWriter, r *http.Request) {
	qrCode := chi.URLParam(r, "qrCode")
	if qrCode == "" {
		respondError(w, BadRequest("Invalid QR code"))
		return
	}

	data := map[string]string{
		"QRCode": qrCode,
	}
	h.templates.Vote.Execute(w, data)
}

// handleGenerateVoteCode generates a unique random code and redirects to the voting page
func (h *Handlers) handleGenerateVoteCode(w http.ResponseWriter, r *http.Request) {
	// Generate a unique code using the voter service
	code, err := h.Voter.GenerateUniqueCode(r.Context())
	if err != nil {
		respondError(w, err)
		return
	}

	// Redirect to the voting page with the generated code
	http.Redirect(w, r, "/vote/"+code, http.StatusFound)
}

// handleGetVoteData returns vote data for a voter
func (h *Handlers) handleGetVoteData(w http.ResponseWriter, r *http.Request) {
	qrCode := chi.URLParam(r, "qrCode")
	if qrCode == "" {
		respondError(w, BadRequest("Invalid QR code"))
		return
	}

	voteData, err := h.Voting.GetVoteData(r.Context(), qrCode)
	if err != nil {
		respondError(w, err)
		return
	}

	respondOK(w, voteData)
}

// handleSubmitVote handles vote submissions
func (h *Handlers) handleSubmitVote(w http.ResponseWriter, r *http.Request) {
	var req VoteSubmitRequest
	if err := decodeJSON(r, &req); err != nil {
		respondError(w, err)
		return
	}

	vote := models.Vote{
		VoterQR:    req.VoterQR,
		CategoryID: req.CategoryID,
		CarID:      req.CarID,
	}
	result, err := h.Voting.SubmitVote(r.Context(), vote)
	if err != nil {
		respondError(w, err)
		return
	}

	respondOK(w, result)
}

// handleCarPhoto proxies car photos from DerbyNet or returns a stock image
func (h *Handlers) handleCarPhoto(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		serveStockPhoto(w)
		return
	}

	// Get photo from service
	photo, err := h.Car.GetCarPhoto(r.Context(), id)
	if err != nil {
		serveStockPhoto(w)
		return
	}

	// Set response headers
	w.Header().Set("Content-Type", photo.ContentType)
	w.Header().Set("Cache-Control", "public, max-age=3600")

	// Write the photo data
	w.Write(photo.Data)
}

// serveStockPhoto returns the placeholder image
func serveStockPhoto(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "image/svg+xml")
	w.Header().Set("Cache-Control", "public, max-age=86400") // Cache for 24 hours
	w.Write(stockPhotoSVG)
}
