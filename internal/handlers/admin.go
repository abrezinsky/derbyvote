package handlers

import (
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/abrezinsky/derbyvote/internal/logger"
	"github.com/abrezinsky/derbyvote/internal/services"
	"github.com/abrezinsky/derbyvote/pkg/derbynet"
)

// ==================== Public Pages ====================

func (h *Handlers) handleIndex(w http.ResponseWriter, r *http.Request) {
	h.templates.Index.Execute(w, nil)
}

// ==================== Admin Pages ====================

func (h *Handlers) handleAdminDashboard(w http.ResponseWriter, r *http.Request) {
	data := AdminPageData{
		Title:     "Admin Dashboard",
		PageTitle: "Admin Dashboard",
		ActiveNav: "dashboard",
	}
	h.templates.AdminDashboard.ExecuteTemplate(w, "admin", data)
}

func (h *Handlers) handleAdminCategories(w http.ResponseWriter, r *http.Request) {
	data := AdminPageData{
		Title:     "Manage Categories",
		PageTitle: "Manage Categories",
		ActiveNav: "categories",
	}
	h.templates.AdminCategories.ExecuteTemplate(w, "admin", data)
}

func (h *Handlers) handleAdminResults(w http.ResponseWriter, r *http.Request) {
	data := AdminPageData{
		Title:     "Voting Results",
		PageTitle: "Voting Results",
		ActiveNav: "results",
	}
	h.templates.AdminResults.ExecuteTemplate(w, "admin", data)
}

func (h *Handlers) handleAdminVoters(w http.ResponseWriter, r *http.Request) {
	data := AdminPageData{
		Title:     "Manage Voters",
		PageTitle: "Manage Voters",
		ActiveNav: "voters",
	}
	h.templates.AdminVoters.ExecuteTemplate(w, "admin", data)
}

func (h *Handlers) handleAdminSettings(w http.ResponseWriter, r *http.Request) {
	data := AdminPageData{
		Title:     "Admin Settings",
		PageTitle: "Admin Settings",
		ActiveNav: "settings",
	}
	h.templates.AdminSettings.ExecuteTemplate(w, "admin", data)
}

// ==================== Categories ====================

func (h *Handlers) handleGetCategories(w http.ResponseWriter, r *http.Request) {
	categories, err := h.Category.ListAllCategories(r.Context())
	if err != nil {
		respondError(w, err)
		return
	}
	respondOK(w, categories)
}

func (h *Handlers) handleCreateCategory(w http.ResponseWriter, r *http.Request) {
	var req CategoryCreateRequest
	if err := decodeJSON(r, &req); err != nil {
		respondError(w, err)
		return
	}

	cat := services.Category{
		Name:              req.Name,
		DisplayOrder:      req.DisplayOrder,
		GroupID:           req.GroupID,
		Active:            req.Active,
		AllowedVoterTypes: req.AllowedVoterTypes,
		AllowedRanks:      req.AllowedRanks,
	}
	id, err := h.Category.CreateCategory(r.Context(), cat)
	if err != nil {
		respondError(w, err)
		return
	}

	respondCreated(w, CategoryResponse{
		ID:                id,
		Name:              cat.Name,
		DisplayOrder:      cat.DisplayOrder,
		GroupID:           cat.GroupID,
		Active:            true,
		AllowedVoterTypes: cat.AllowedVoterTypes,
		AllowedRanks:      cat.AllowedRanks,
	})
}

func (h *Handlers) handleUpdateCategory(w http.ResponseWriter, r *http.Request) {
	id, err := parseIntParam(r, "id")
	if err != nil {
		respondError(w, err)
		return
	}

	var req CategoryUpdateRequest
	if err := decodeJSON(r, &req); err != nil {
		respondError(w, err)
		return
	}

	cat := services.Category{
		Name:              req.Name,
		DisplayOrder:      req.DisplayOrder,
		GroupID:           req.GroupID,
		Active:            req.Active,
		AllowedVoterTypes: req.AllowedVoterTypes,
		AllowedRanks:      req.AllowedRanks,
	}
	if err := h.Category.UpdateCategory(r.Context(), id, cat); err != nil {
		respondError(w, err)
		return
	}

	respondOK(w, CategoryResponse{
		ID:                int64(id),
		Name:              cat.Name,
		DisplayOrder:      cat.DisplayOrder,
		GroupID:           cat.GroupID,
		Active:            cat.Active,
		AllowedVoterTypes: cat.AllowedVoterTypes,
		AllowedRanks:      cat.AllowedRanks,
	})
}

func (h *Handlers) handleDeleteCategory(w http.ResponseWriter, r *http.Request) {
	id, err := parseIntParam(r, "id")
	if err != nil {
		respondError(w, err)
		return
	}

	// Check for force parameter
	force := r.URL.Query().Get("force") == "true"

	// Check if category has received votes
	if !force {
		voteCount, err := h.Category.CountVotesForCategory(r.Context(), id)
		if err != nil {
			respondError(w, err)
			return
		}
		if voteCount > 0 {
			// Return confirmation needed response
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusConflict)
			w.Write([]byte(fmt.Sprintf(`{"error":"This category has received %d vote(s). Are you sure you want to delete it?","confirmation_required":true,"vote_count":%d}`, voteCount, voteCount)))
			return
		}
	}

	if err := h.Category.DeleteCategory(r.Context(), id); err != nil {
		respondError(w, err)
		return
	}

	respondDeleted(w)
}

// ==================== Category Groups ====================

func (h *Handlers) handleGetCategoryGroups(w http.ResponseWriter, r *http.Request) {
	groups, err := h.Category.ListGroups(r.Context())
	if err != nil {
		respondError(w, err)
		return
	}
	respondOK(w, groups)
}

func (h *Handlers) handleCreateCategoryGroup(w http.ResponseWriter, r *http.Request) {
	var req CategoryGroupCreateRequest
	if err := decodeJSON(r, &req); err != nil {
		respondError(w, err)
		return
	}

	group := services.CategoryGroup{
		Name:              req.Name,
		Description:       req.Description,
		ExclusivityPoolID: req.ExclusivityPoolID,
		MaxWinsPerCar:     req.MaxWinsPerCar,
		DisplayOrder:      req.DisplayOrder,
	}
	id, err := h.Category.CreateGroup(r.Context(), group)
	if err != nil {
		respondError(w, err)
		return
	}

	respondCreated(w, CategoryGroupResponse{ID: id})
}

func (h *Handlers) handleGetCategoryGroup(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		respondError(w, BadRequest("Invalid category group ID"))
		return
	}

	group, err := h.Category.GetGroup(r.Context(), id)
	if err != nil {
		respondError(w, err)
		return
	}

	respondOK(w, group)
}

func (h *Handlers) handleUpdateCategoryGroup(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		respondError(w, BadRequest("Invalid category group ID"))
		return
	}

	// Check if group exists first
	_, err := h.Category.GetGroup(r.Context(), id)
	if err != nil {
		respondError(w, err)
		return
	}

	var req CategoryGroupUpdateRequest
	if err := decodeJSON(r, &req); err != nil {
		respondError(w, err)
		return
	}

	group := services.CategoryGroup{
		Name:              req.Name,
		Description:       req.Description,
		ExclusivityPoolID: req.ExclusivityPoolID,
		MaxWinsPerCar:     req.MaxWinsPerCar,
		DisplayOrder:      req.DisplayOrder,
	}
	if err := h.Category.UpdateGroup(r.Context(), id, group); err != nil {
		respondError(w, err)
		return
	}

	respondSuccess(w, "Category group updated")
}

func (h *Handlers) handleDeleteCategoryGroup(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		respondError(w, BadRequest("Invalid category group ID"))
		return
	}

	// Check if group exists first
	_, err := h.Category.GetGroup(r.Context(), id)
	if err != nil {
		respondError(w, err)
		return
	}

	if err := h.Category.DeleteGroup(r.Context(), id); err != nil {
		respondError(w, err)
		return
	}

	respondDeleted(w)
}

// ==================== Voting Control ====================

func (h *Handlers) handleSetVotingStatus(w http.ResponseWriter, r *http.Request) {
	var req VotingStatusRequest
	if err := decodeJSON(r, &req); err != nil {
		respondError(w, err)
		return
	}

	ctx := r.Context()
	var err error
	if req.Open {
		err = h.Settings.OpenVoting(ctx)
	} else {
		err = h.Settings.CloseVoting(ctx)
	}
	if err != nil {
		respondError(w, err)
		return
	}

	respondOK(w, VotingStatusResponse{Open: req.Open})
}

func (h *Handlers) handleSetVotingTimer(w http.ResponseWriter, r *http.Request) {
	var req VotingTimerRequest
	if err := decodeJSON(r, &req); err != nil {
		respondError(w, err)
		return
	}

	closeTimeStr, err := h.Settings.StartVotingTimer(r.Context(), req.Minutes)
	if err != nil {
		respondError(w, err)
		return
	}

	respondOK(w, VotingTimerResponse{
		CloseTime: closeTimeStr,
		Minutes:   req.Minutes,
	})
}

// ==================== Stats & Results ====================

func (h *Handlers) handleGetStats(w http.ResponseWriter, r *http.Request) {
	stats, err := h.Results.GetStats(r.Context())
	if err != nil {
		respondError(w, err)
		return
	}

	respondOK(w, stats)
}

func (h *Handlers) handleGetResults(w http.ResponseWriter, r *http.Request) {
	results, err := h.Results.GetResults(r.Context())
	if err != nil {
		respondError(w, err)
		return
	}

	// Ensure we return an empty array, not null
	categories := results.Categories
	if categories == nil {
		categories = []services.CategoryResult{}
	}
	respondOK(w, categories)
}

// ==================== DerbyNet Sync ====================

func (h *Handlers) handleSyncDerbyNet(w http.ResponseWriter, r *http.Request) {
	var req DerbyNetSyncRequest
	if err := decodeJSON(r, &req); err != nil {
		respondError(w, err)
		return
	}

	if req.DerbyNetURL == "" {
		respondError(w, BadRequest("derbynet_url is required"))
		return
	}

	result, err := h.Car.SyncFromDerbyNet(r.Context(), req.DerbyNetURL)
	if err != nil {
		respondError(w, err)
		return
	}

	respondOK(w, result)
}

func (h *Handlers) handleSyncCategoriesDerbyNet(w http.ResponseWriter, r *http.Request) {
	var req DerbyNetSyncRequest
	if err := decodeJSON(r, &req); err != nil {
		respondError(w, err)
		return
	}

	if req.DerbyNetURL == "" {
		respondError(w, BadRequest("derbynet_url is required"))
		return
	}

	result, err := h.Category.SyncFromDerbyNet(r.Context(), req.DerbyNetURL)
	if err != nil {
		respondError(w, err)
		return
	}

	respondOK(w, result)
}

func (h *Handlers) handleTestDerbyNet(w http.ResponseWriter, r *http.Request) {
	var req DerbyNetSyncRequest
	if err := decodeJSON(r, &req); err != nil {
		respondError(w, err)
		return
	}

	if req.DerbyNetURL == "" {
		respondError(w, BadRequest("derbynet_url is required"))
		return
	}

	// Create a temporary DerbyNet client
	client := derbynet.NewHTTPClient(req.DerbyNetURL, logger.New())

	// Try to fetch racers to test basic connectivity
	racers, err := client.FetchRacers(r.Context())
	if err != nil {
		respondError(w, BadRequest("Failed to connect to DerbyNet: "+err.Error()))
		return
	}

	// Try to fetch awards
	awards, err := client.FetchAwards(r.Context())
	if err != nil {
		// Awards might fail if not authenticated, but connection works
		awards = []derbynet.Award{}
	}

	// Check if we have credentials to test authentication
	role, _ := h.Settings.GetSetting(r.Context(), "derbynet_role")
	password, _ := h.Settings.GetSetting(r.Context(), "derbynet_password")
	authenticated := false
	if role != "" && password != "" {
		client.SetCredentials(role, password)
		// Try to fetch award types which requires some level of access
		_, authErr := client.FetchAwardTypes(r.Context())
		authenticated = authErr == nil
	}

	respondOK(w, map[string]interface{}{
		"status":        "success",
		"total_racers":  len(racers),
		"total_awards":  len(awards),
		"authenticated": authenticated,
		"role":          role,
	})
}

func (h *Handlers) handlePushResultsDerbyNet(w http.ResponseWriter, r *http.Request) {
	var req DerbyNetSyncRequest
	if err := decodeJSON(r, &req); err != nil {
		respondError(w, err)
		return
	}

	if req.DerbyNetURL == "" {
		respondError(w, BadRequest("derbynet_url is required"))
		return
	}

	ctx := r.Context()

	// Check for conflicts before pushing
	ties, err := h.Results.DetectTies(ctx)
	if err != nil {
		respondError(w, err)
		return
	}

	multiWins, err := h.Results.DetectMultipleWins(ctx)
	if err != nil {
		respondError(w, err)
		return
	}

	if len(ties) > 0 || len(multiWins) > 0 {
		respondError(w, Conflict("Cannot push results: conflicts exist (ties or multiple wins). Please resolve all conflicts first."))
		return
	}

	result, err := h.Results.PushResultsToDerbyNet(ctx, req.DerbyNetURL)
	if err != nil {
		respondError(w, err)
		return
	}

	respondOK(w, result)
}

// handleGetConflicts returns all detected ties and multiple-win conflicts
func (h *Handlers) handleGetConflicts(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Detect ties
	ties, err := h.Results.DetectTies(ctx)
	if err != nil {
		respondError(w, err)
		return
	}

	// Detect multiple wins
	multiWins, err := h.Results.DetectMultipleWins(ctx)
	if err != nil {
		respondError(w, err)
		return
	}

	// Build response
	tieResponses := []TieConflictResponse{}
	for _, tie := range ties {
		var tiedCars []TiedCarResponse
		for _, car := range tie.TiedCars {
			tiedCars = append(tiedCars, TiedCarResponse{
				CarID:     car.CarID,
				CarNumber: car.CarNumber,
				CarName:   car.CarName,
				RacerName: car.RacerName,
				VoteCount: car.VoteCount,
			})
		}
		tieResponses = append(tieResponses, TieConflictResponse{
			CategoryID:   tie.CategoryID,
			CategoryName: tie.CategoryName,
			TiedCars:     tiedCars,
		})
	}

	multiWinResponses := []MultiWinConflictResponse{}
	for _, mw := range multiWins {
		multiWinResponses = append(multiWinResponses, MultiWinConflictResponse{
			CarID:         mw.CarID,
			CarNumber:     mw.CarNumber,
			RacerName:     mw.RacerName,
			AwardsWon:     mw.AwardsWon,
			CategoryIDs:   mw.CategoryIDs,
			GroupID:       mw.GroupID,
			GroupName:     mw.GroupName,
			MaxWinsPerCar: mw.MaxWinsPerCar,
		})
	}

	respondOK(w, ConflictsResponse{
		Ties:      tieResponses,
		MultiWins: multiWinResponses,
	})
}

// handleOverrideWinner sets a manual winner for a category
func (h *Handlers) handleOverrideWinner(w http.ResponseWriter, r *http.Request) {
	var req OverrideWinnerRequest
	if err := decodeJSON(r, &req); err != nil {
		respondError(w, err)
		return
	}

	if req.CategoryID == 0 {
		respondError(w, BadRequest("category_id is required"))
		return
	}
	if req.CarID == 0 {
		respondError(w, BadRequest("car_id is required"))
		return
	}
	if req.Reason == "" {
		respondError(w, BadRequest("reason is required"))
		return
	}

	// Check if voting is still open
	votingOpen, err := h.Settings.IsVotingOpen(r.Context())
	if err != nil {
		respondError(w, err)
		return
	}
	if votingOpen {
		respondError(w, BadRequest("Cannot resolve conflicts while voting is still open"))
		return
	}

	err = h.Results.SetManualWinner(r.Context(), req.CategoryID, req.CarID, req.Reason)
	if err != nil {
		respondError(w, err)
		return
	}

	respondOK(w, map[string]interface{}{
		"message": "Manual winner set successfully",
	})
}

// handleClearOverride clears the manual winner override for a category
func (h *Handlers) handleClearOverride(w http.ResponseWriter, r *http.Request) {
	categoryID, err := parseIntParam(r, "categoryID")
	if err != nil {
		respondError(w, err)
		return
	}

	// Check if voting is still open
	votingOpen, err := h.Settings.IsVotingOpen(r.Context())
	if err != nil {
		respondError(w, err)
		return
	}
	if votingOpen {
		respondError(w, BadRequest("Cannot clear conflict resolution while voting is still open"))
		return
	}

	err = h.Results.ClearManualWinner(r.Context(), categoryID)
	if err != nil {
		respondError(w, err)
		return
	}

	respondOK(w, map[string]interface{}{
		"message": "Manual winner override cleared",
	})
}

// handleGetOverrides returns all categories with manual overrides
func (h *Handlers) handleGetOverrides(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Get results which includes override info
	results, err := h.Results.GetResults(ctx)
	if err != nil {
		respondError(w, err)
		return
	}

	overrides := []OverrideResponse{}
	for _, cat := range results.Categories {
		if cat.HasOverride && cat.OverrideCarID != nil {
			// Find the car details from the votes
			var carNumber, racerName string
			for _, vote := range cat.Votes {
				if vote.CarID == *cat.OverrideCarID {
					carNumber = vote.CarNumber
					racerName = vote.RacerName
					break
				}
			}

			overrides = append(overrides, OverrideResponse{
				CategoryID:        cat.CategoryID,
				CategoryName:      cat.CategoryName,
				OverrideCarID:     cat.OverrideCarID,
				OverrideCarNumber: carNumber,
				OverrideRacerName: racerName,
				OverrideReason:    cat.OverrideReason,
				OverriddenAt:      cat.OverriddenAt,
			})
		}
	}

	respondOK(w, overrides)
}

// ==================== QR Codes ====================

func (h *Handlers) handleGenerateQRCodes(w http.ResponseWriter, r *http.Request) {
	var req QRCodeGenerateRequest
	if err := decodeJSON(r, &req); err != nil {
		respondError(w, err)
		return
	}

	qrCodes, err := h.Voter.GenerateQRCodes(r.Context(), req.Count)
	if err != nil {
		respondError(w, err)
		return
	}

	respondOK(w, QRCodesResponse{QRCodes: qrCodes})
}

func (h *Handlers) handleGetQRImage(w http.ResponseWriter, r *http.Request) {
	id, err := parseIntParam(r, "id")
	if err != nil {
		respondError(w, err)
		return
	}

	png, err := h.Voter.GenerateQRImage(r.Context(), id)
	if err != nil {
		respondError(w, err)
		return
	}

	w.Header().Set("Content-Type", "image/png")
	w.Header().Set("Cache-Control", "no-cache")
	w.Write(png)
}

func (h *Handlers) handleGetOpenVotingQR(w http.ResponseWriter, r *http.Request) {
	png, err := h.Voter.GenerateDynamicQRImage(r.Context())
	if err != nil {
		respondError(w, err)
		return
	}

	w.Header().Set("Content-Type", "image/png")
	w.Header().Set("Cache-Control", "no-cache")
	w.Write(png)
}

// ==================== Settings ====================

func (h *Handlers) handleGetSettings(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	derbynetURL, _ := h.Settings.GetDerbyNetURL(ctx)
	baseURL, _ := h.Settings.GetBaseURL(ctx)
	derbynetRole, _ := h.Settings.GetSetting(ctx, "derbynet_role")
	requireRegisteredQR, _ := h.Settings.RequireRegisteredQR(ctx)
	votingInstructions, _ := h.Settings.GetSetting(ctx, "voting_instructions")
	voterTypes, _ := h.Settings.GetVoterTypes(ctx)

	respondOK(w, SettingsResponse{
		DerbyNetURL:         derbynetURL,
		BaseURL:             baseURL,
		DerbyNetRole:        derbynetRole,
		RequireRegisteredQR: requireRegisteredQR,
		VotingInstructions:  votingInstructions,
		VoterTypes:          voterTypes,
	})
}

func (h *Handlers) handleUpdateSettings(w http.ResponseWriter, r *http.Request) {
	var req SettingsUpdateRequest
	if err := decodeJSON(r, &req); err != nil {
		respondError(w, err)
		return
	}

	settings := services.Settings{
		DerbyNetURL:         req.DerbyNetURL,
		BaseURL:             req.BaseURL,
		DerbyNetRole:        req.DerbyNetRole,
		DerbyNetPassword:    req.DerbyNetPassword,
		RequireRegisteredQR: req.RequireRegisteredQR,
		VotingInstructions:  req.VotingInstructions,
		VoterTypes:          req.VoterTypes,
	}
	if err := h.Settings.UpdateSettings(r.Context(), settings); err != nil {
		respondError(w, err)
		return
	}

	respondSuccess(w, "Settings updated")
}

func (h *Handlers) handleGetVoterTypes(w http.ResponseWriter, r *http.Request) {
	voterTypes, err := h.Settings.GetVoterTypes(r.Context())
	if err != nil {
		respondError(w, err)
		return
	}

	respondOK(w, map[string]interface{}{
		"voter_types": voterTypes,
	})
}

// ==================== Database Management ====================

func (h *Handlers) handleResetDatabase(w http.ResponseWriter, r *http.Request) {
	var req DatabaseResetRequest
	if err := decodeJSON(r, &req); err != nil {
		respondError(w, err)
		return
	}

	result, err := h.Settings.ResetTables(r.Context(), req.Tables)
	if err != nil {
		respondError(w, err)
		return
	}

	respondSuccess(w, result.Message)
}

func (h *Handlers) handleSeedMockData(w http.ResponseWriter, r *http.Request) {
	var req SeedMockDataRequest
	if err := decodeJSON(r, &req); err != nil {
		respondError(w, err)
		return
	}

	ctx := r.Context()

	var message string
	var addedCount int

	switch req.SeedType {
	case "categories":
		count, err := h.Category.SeedMockCategories(ctx)
		if err != nil {
			respondError(w, err)
			return
		}
		addedCount = count
		if addedCount == 0 {
			message = "All default categories already exist"
		} else {
			message = fmt.Sprintf("Added %d new categories", addedCount)
		}

	case "cars":
		count, err := h.Car.SeedMockCars(ctx)
		if err != nil {
			respondError(w, err)
			return
		}
		addedCount = count
		if addedCount == 0 {
			message = "All mock cars already exist"
		} else {
			message = fmt.Sprintf("Added %d new cars", addedCount)
		}

	default:
		respondError(w, BadRequest("Invalid seed type"))
		return
	}

	respondSuccess(w, message)
}

// ==================== Voters ====================

func (h *Handlers) handleGetVoters(w http.ResponseWriter, r *http.Request) {
	voters, err := h.Voter.ListVoters(r.Context())
	if err != nil {
		respondError(w, err)
		return
	}
	respondOK(w, voters)
}

func (h *Handlers) handleCreateVoter(w http.ResponseWriter, r *http.Request) {
	var req VoterCreateRequest
	if err := decodeJSON(r, &req); err != nil {
		respondError(w, err)
		return
	}

	voter := services.Voter{
		CarID:     req.CarID,
		Name:      req.Name,
		Email:     req.Email,
		VoterType: req.VoterType,
		QRCode:    req.QRCode,
		Notes:     req.Notes,
	}
	id, qrCode, err := h.Voter.CreateVoter(r.Context(), voter)
	if err != nil {
		respondError(w, err)
		return
	}

	respondCreated(w, VoterResponse{
		ID:        id,
		CarID:     req.CarID,
		Name:      req.Name,
		Email:     req.Email,
		VoterType: req.VoterType,
		QRCode:    qrCode,
		Notes:     req.Notes,
	})
}

func (h *Handlers) handleUpdateVoter(w http.ResponseWriter, r *http.Request) {
	var req VoterUpdateRequest
	if err := decodeJSON(r, &req); err != nil {
		respondError(w, err)
		return
	}

	voter := services.Voter{
		ID:        req.ID,
		CarID:     req.CarID,
		Name:      req.Name,
		Email:     req.Email,
		VoterType: req.VoterType,
		Notes:     req.Notes,
	}
	if err := h.Voter.UpdateVoter(r.Context(), voter); err != nil {
		respondError(w, err)
		return
	}

	respondSuccess(w, "Voter updated")
}

func (h *Handlers) handleDeleteVoter(w http.ResponseWriter, r *http.Request) {
	id, err := parseIntParam(r, "id")
	if err != nil {
		respondError(w, err)
		return
	}

	if err := h.Voter.DeleteVoter(r.Context(), id); err != nil {
		respondError(w, err)
		return
	}

	respondDeleted(w)
}

// ==================== Cars ====================

func (h *Handlers) handleAdminCars(w http.ResponseWriter, r *http.Request) {
	data := AdminPageData{
		Title:     "Manage Cars",
		PageTitle: "Manage Cars",
		ActiveNav: "cars",
	}
	h.templates.AdminCars.ExecuteTemplate(w, "admin", data)
}

func (h *Handlers) handleGetCars(w http.ResponseWriter, r *http.Request) {
	cars, err := h.Car.ListCars(r.Context())
	if err != nil {
		respondError(w, err)
		return
	}
	respondOK(w, cars)
}

func (h *Handlers) handleGetCar(w http.ResponseWriter, r *http.Request) {
	id, err := parseIntParam(r, "id")
	if err != nil {
		respondError(w, err)
		return
	}

	car, err := h.Car.GetCar(r.Context(), id)
	if err != nil {
		respondError(w, err)
		return
	}

	respondOK(w, car)
}

func (h *Handlers) handleCreateCar(w http.ResponseWriter, r *http.Request) {
	var req CarCreateRequest
	if err := decodeJSON(r, &req); err != nil {
		respondError(w, err)
		return
	}

	if req.CarNumber == "" {
		respondError(w, BadRequest("car_number is required"))
		return
	}

	if err := h.Car.CreateCar(r.Context(), req.CarNumber, req.RacerName, req.CarName, req.PhotoURL); err != nil {
		respondError(w, err)
		return
	}

	respondCreated(w, CarResponse{
		CarNumber: req.CarNumber,
		RacerName: req.RacerName,
		CarName:   req.CarName,
		PhotoURL:  req.PhotoURL,
		Rank:      req.Rank,
	})
}

func (h *Handlers) handleUpdateCar(w http.ResponseWriter, r *http.Request) {
	id, err := parseIntParam(r, "id")
	if err != nil {
		respondError(w, err)
		return
	}

	// Check if car exists first
	_, err = h.Car.GetCar(r.Context(), id)
	if err != nil {
		respondError(w, err)
		return
	}

	var req CarUpdateRequest
	if err := decodeJSON(r, &req); err != nil {
		respondError(w, err)
		return
	}

	if err := h.Car.UpdateCar(r.Context(), id, req.CarNumber, req.RacerName, req.CarName, req.PhotoURL, req.Rank); err != nil {
		respondError(w, err)
		return
	}

	respondOK(w, CarResponse{
		ID:        id,
		CarNumber: req.CarNumber,
		RacerName: req.RacerName,
		CarName:   req.CarName,
		PhotoURL:  req.PhotoURL,
		Rank:      req.Rank,
	})
}

func (h *Handlers) handleDeleteCar(w http.ResponseWriter, r *http.Request) {
	id, err := parseIntParam(r, "id")
	if err != nil {
		respondError(w, err)
		return
	}

	// Check if car exists first
	_, err = h.Car.GetCar(r.Context(), id)
	if err != nil {
		respondError(w, err)
		return
	}

	// Check for force parameter
	force := r.URL.Query().Get("force") == "true"

	// Check if car has received votes
	if !force {
		voteCount, err := h.Car.CountVotesForCar(r.Context(), id)
		if err != nil {
			respondError(w, err)
			return
		}
		if voteCount > 0 {
			// Return confirmation needed response
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusConflict)
			w.Write([]byte(fmt.Sprintf(`{"error":"This car has received %d vote(s). Are you sure you want to delete it?","confirmation_required":true,"vote_count":%d}`, voteCount, voteCount)))
			return
		}
	}

	if err := h.Car.DeleteCar(r.Context(), id); err != nil {
		respondError(w, err)
		return
	}

	respondDeleted(w)
}

func (h *Handlers) handleSetCarEligibility(w http.ResponseWriter, r *http.Request) {
	id, err := parseIntParam(r, "id")
	if err != nil {
		respondError(w, err)
		return
	}

	// Check if car exists first
	car, err := h.Car.GetCar(r.Context(), id)
	if err != nil {
		respondError(w, err)
		return
	}

	var req CarEligibilityRequest
	if err := decodeJSON(r, &req); err != nil {
		respondError(w, err)
		return
	}

	// Check if car has received votes when marking as ineligible
	if !req.Eligible && car.Eligible && !req.Force {
		voteCount, err := h.Car.CountVotesForCar(r.Context(), id)
		if err != nil {
			respondError(w, err)
			return
		}
		if voteCount > 0 {
			// Return confirmation needed response
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusConflict)
			w.Write([]byte(fmt.Sprintf(`{"error":"This car has received %d vote(s). Are you sure you want to mark it as ineligible?","confirmation_required":true,"vote_count":%d}`, voteCount, voteCount)))
			return
		}
	}

	if err := h.Car.SetCarEligibility(r.Context(), id, req.Eligible); err != nil {
		respondError(w, err)
		return
	}

	respondOK(w, map[string]interface{}{
		"id":       id,
		"eligible": req.Eligible,
	})
}

