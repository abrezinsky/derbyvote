package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/abrezinsky/derbyvote/internal/errors"
	"github.com/abrezinsky/derbyvote/internal/models"
)

// Repository provides data access methods
type Repository struct {
	db *sql.DB
}

// New creates a new Repository
func New(dbPath string) (*Repository, error) {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, err
	}

	// Enable foreign key constraints
	if _, err := db.Exec("PRAGMA foreign_keys = ON"); err != nil {
		return nil, err
	}

	// Set connection pool settings
	db.SetMaxOpenConns(1) // SQLite works best with single connection
	db.SetMaxIdleConns(1)

	repo := &Repository{db: db}

	// Run migrations
	if err := repo.migrate(); err != nil {
		return nil, err
	}

	return repo, nil
}

// DB returns the underlying database connection (for transactions)
func (r *Repository) DB() *sql.DB {
	return r.db
}

// Close closes the database connection
func (r *Repository) Close() error {
	if r.db != nil {
		return r.db.Close()
	}
	return nil
}

// Ping checks if the database connection is alive
func (r *Repository) Ping(ctx context.Context) error {
	return r.db.PingContext(ctx)
}

// migrate runs database migrations
func (r *Repository) migrate() error {
	migrations := []string{
		`CREATE TABLE IF NOT EXISTS voters (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			car_id INTEGER,
			name TEXT,
			email TEXT,
			voter_type TEXT DEFAULT 'general',
			qr_code TEXT UNIQUE NOT NULL,
			notes TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			last_voted_at DATETIME,
			FOREIGN KEY (car_id) REFERENCES cars(id) ON DELETE SET NULL
		)`,
		`CREATE TABLE IF NOT EXISTS cars (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			derbynet_racer_id INTEGER UNIQUE,
			car_number TEXT NOT NULL,
			racer_name TEXT,
			car_name TEXT,
			photo_url TEXT,
			rank TEXT,
			synced_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			active BOOLEAN DEFAULT 1
		)`,
		`CREATE TABLE IF NOT EXISTS category_groups (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL,
			description TEXT,
			exclusivity_pool_id INTEGER,
			display_order INTEGER NOT NULL,
			active BOOLEAN DEFAULT 1
		)`,
		`CREATE TABLE IF NOT EXISTS categories (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL,
			display_order INTEGER NOT NULL,
			group_id INTEGER,
			derbynet_award_id INTEGER,
			active BOOLEAN DEFAULT 1,
			override_winner_car_id INTEGER,
			override_reason TEXT,
			overridden_at DATETIME,
			FOREIGN KEY (group_id) REFERENCES category_groups(id) ON DELETE SET NULL,
			FOREIGN KEY (override_winner_car_id) REFERENCES cars(id) ON DELETE SET NULL
		)`,
		`CREATE TABLE IF NOT EXISTS votes (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			voter_id INTEGER NOT NULL,
			car_id INTEGER NOT NULL,
			category_id INTEGER NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (voter_id) REFERENCES voters(id),
			FOREIGN KEY (car_id) REFERENCES cars(id),
			FOREIGN KEY (category_id) REFERENCES categories(id),
			UNIQUE(voter_id, category_id)
		)`,
		`CREATE TABLE IF NOT EXISTS settings (
			key TEXT PRIMARY KEY,
			value TEXT NOT NULL
		)`,
		`CREATE INDEX IF NOT EXISTS idx_votes_voter ON votes(voter_id)`,
		`CREATE INDEX IF NOT EXISTS idx_votes_category ON votes(category_id)`,
		`CREATE INDEX IF NOT EXISTS idx_votes_car ON votes(car_id)`,
		`CREATE INDEX IF NOT EXISTS idx_voters_qr ON voters(qr_code)`,
		`CREATE INDEX IF NOT EXISTS idx_voters_car ON voters(car_id)`,
	}

	additionalMigrations := []string{
		`ALTER TABLE voters ADD COLUMN car_id INTEGER`,
		`ALTER TABLE voters ADD COLUMN name TEXT`,
		`ALTER TABLE voters ADD COLUMN email TEXT`,
		`ALTER TABLE voters ADD COLUMN voter_type TEXT DEFAULT 'general'`,
		`ALTER TABLE voters ADD COLUMN notes TEXT`,
		`ALTER TABLE categories ADD COLUMN group_id INTEGER`,
		`ALTER TABLE categories ADD COLUMN derbynet_award_id INTEGER`,
		`ALTER TABLE cars ADD COLUMN eligible BOOLEAN DEFAULT 1`,
		`ALTER TABLE categories ADD COLUMN override_winner_car_id INTEGER`,
		`ALTER TABLE categories ADD COLUMN override_reason TEXT`,
		`ALTER TABLE categories ADD COLUMN overridden_at DATETIME`,
		`ALTER TABLE category_groups ADD COLUMN max_wins_per_car INTEGER`,
		`ALTER TABLE categories ADD COLUMN allowed_voter_types TEXT`, // JSON array of voter types, NULL means all types allowed
		`ALTER TABLE cars ADD COLUMN rank TEXT`,
		`ALTER TABLE categories ADD COLUMN allowed_ranks TEXT`, // JSON array of ranks, NULL means all ranks allowed
	}

	for _, migration := range migrations {
		if _, err := r.db.Exec(migration); err != nil {
			return err
		}
	}

	for _, migration := range additionalMigrations {
		r.db.Exec(migration) // Ignore errors - columns may already exist
	}

	// Insert default settings if not exists
	// Note: base_url is intentionally not set here - it's set by app.go
	// with the detected LAN IP address on startup
	defaultSettings := map[string]string{
		"voting_open":  "true",
		"derbynet_url": "",
	}

	for key, value := range defaultSettings {
		_, err := r.db.Exec(`INSERT OR IGNORE INTO settings (key, value) VALUES (?, ?)`, key, value)
		if err != nil {
			return err
		}
	}

	return nil
}

// ==================== Voter Methods ====================

// GetVoterByQR retrieves a voter by QR code
func (r *Repository) GetVoterByQR(ctx context.Context, qrCode string) (int, error) {
	var voterID int
	err := r.db.QueryRowContext(ctx, `SELECT id FROM voters WHERE qr_code = ?`, qrCode).Scan(&voterID)
	if err == sql.ErrNoRows {
		return 0, ErrNotFound
	}
	return voterID, err
}

// GetVoterType returns the voter type for a given voter ID
func (r *Repository) GetVoterType(ctx context.Context, voterID int) (string, error) {
	var voterType sql.NullString
	err := r.db.QueryRowContext(ctx, `SELECT voter_type FROM voters WHERE id = ?`, voterID).Scan(&voterType)
	if err == sql.ErrNoRows {
		return "", ErrNotFound
	}
	if err != nil {
		return "", err
	}
	if voterType.Valid {
		return voterType.String, nil
	}
	return "general", nil // Default to general if NULL
}

// CreateVoter creates a new voter
func (r *Repository) CreateVoter(ctx context.Context, qrCode string) (int, error) {
	result, err := r.db.ExecContext(ctx, `INSERT INTO voters (qr_code) VALUES (?)`, qrCode)
	if err != nil {
		return 0, err
	}
	id, err := result.LastInsertId()
	return int(id), err
}

// CreateVoterFull creates a voter with all fields
func (r *Repository) CreateVoterFull(ctx context.Context, carID *int, name, email, voterType, qrCode, notes string) (int64, error) {
	result, err := r.db.ExecContext(ctx, `
		INSERT INTO voters (car_id, name, email, voter_type, qr_code, notes)
		VALUES (?, ?, ?, ?, ?, ?)
	`, carID, name, email, voterType, qrCode, notes)
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}

// UpdateVoter updates a voter
func (r *Repository) UpdateVoter(ctx context.Context, id int, carID *int, name, email, voterType, notes string) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE voters SET car_id = ?, name = ?, email = ?, voter_type = ?, notes = ?
		WHERE id = ?
	`, carID, name, email, voterType, notes, id)
	return err
}

// DeleteVoter deletes a voter
func (r *Repository) DeleteVoter(ctx context.Context, id int) error {
	// Delete voter's votes first (foreign key constraint)
	_, err := r.db.ExecContext(ctx, `DELETE FROM votes WHERE voter_id = ?`, id)
	if err != nil {
		return err
	}

	// Now delete the voter
	_, err = r.db.ExecContext(ctx, `DELETE FROM voters WHERE id = ?`, id)
	return err
}

// GetVoterQRCode returns the QR code for a voter by ID
func (r *Repository) GetVoterQRCode(ctx context.Context, id int) (string, error) {
	var qrCode string
	err := r.db.QueryRowContext(ctx, `SELECT qr_code FROM voters WHERE id = ?`, id).Scan(&qrCode)
	return qrCode, err
}

// ListVoters returns all voters with car info
func (r *Repository) ListVoters(ctx context.Context) ([]map[string]interface{}, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT v.id, v.car_id, v.name, v.email, v.voter_type, v.qr_code, v.notes,
		       v.created_at, v.last_voted_at, c.car_number, c.racer_name
		FROM voters v
		LEFT JOIN cars c ON v.car_id = c.id
		ORDER BY v.created_at DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var voters []map[string]interface{}
	for rows.Next() {
		var id, carID sql.NullInt64
		var name, email, voterType, qrCode, notes, createdAt, lastVotedAt sql.NullString
		var carNumber, racerName sql.NullString

		if err := rows.Scan(&id, &carID, &name, &email, &voterType, &qrCode, &notes,
			&createdAt, &lastVotedAt, &carNumber, &racerName); err != nil {
			continue
		}

		voter := map[string]interface{}{
			"id":         id.Int64,
			"qr_code":    qrCode.String,
			"voter_type": voterType.String,
			"created_at": createdAt.String,
		}

		if carID.Valid {
			voter["car_id"] = carID.Int64
			voter["car_number"] = carNumber.String
			voter["racer_name"] = racerName.String
		}
		if name.Valid {
			voter["name"] = name.String
		}
		if email.Valid {
			voter["email"] = email.String
		}
		if notes.Valid {
			voter["notes"] = notes.String
		}
		if lastVotedAt.Valid {
			voter["last_voted_at"] = lastVotedAt.String
			voter["has_voted"] = true
		} else {
			voter["has_voted"] = false
		}

		voters = append(voters, voter)
	}
	return voters, nil
}

// ==================== Category Methods ====================

// ListCategories returns all active categories with group info
func (r *Repository) ListCategories(ctx context.Context) ([]models.Category, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT c.id, c.name, c.display_order, c.group_id, c.derbynet_award_id, cg.name, cg.exclusivity_pool_id,
		       c.override_winner_car_id, c.override_reason, c.overridden_at, c.allowed_voter_types, c.allowed_ranks
		FROM categories c
		LEFT JOIN category_groups cg ON c.group_id = cg.id
		WHERE c.active = 1
		ORDER BY c.display_order
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var categories []models.Category
	for rows.Next() {
		var cat models.Category
		var groupID, derbynetAwardID, exclusivityPoolID, overrideWinnerCarID sql.NullInt64
		var groupName, overrideReason, overriddenAt, allowedVoterTypesJSON, allowedRanksJSON sql.NullString
		if err := rows.Scan(&cat.ID, &cat.Name, &cat.DisplayOrder, &groupID, &derbynetAwardID, &groupName, &exclusivityPoolID,
			&overrideWinnerCarID, &overrideReason, &overriddenAt, &allowedVoterTypesJSON, &allowedRanksJSON); err != nil {
			return nil, err
		}
		if groupID.Valid {
			id := int(groupID.Int64)
			cat.GroupID = &id
			cat.GroupName = groupName.String
		}
		if derbynetAwardID.Valid {
			awardID := int(derbynetAwardID.Int64)
			cat.DerbyNetAwardID = &awardID
		}
		if exclusivityPoolID.Valid {
			poolID := int(exclusivityPoolID.Int64)
			cat.ExclusivityPoolID = &poolID
		}
		if overrideWinnerCarID.Valid {
			carID := int(overrideWinnerCarID.Int64)
			cat.OverrideWinnerCarID = &carID
		}
		if overrideReason.Valid {
			cat.OverrideReason = overrideReason.String
		}
		if overriddenAt.Valid {
			cat.OverriddenAt = overriddenAt.String
		}
		if allowedVoterTypesJSON.Valid && allowedVoterTypesJSON.String != "" {
			if err := json.Unmarshal([]byte(allowedVoterTypesJSON.String), &cat.AllowedVoterTypes); err != nil {
				return nil, err
			}
		}
		if allowedRanksJSON.Valid && allowedRanksJSON.String != "" {
			if err := json.Unmarshal([]byte(allowedRanksJSON.String), &cat.AllowedRanks); err != nil {
				return nil, err
			}
		}
		categories = append(categories, cat)
	}
	return categories, nil
}

// ListAllCategories returns all categories (including inactive) with group info
func (r *Repository) ListAllCategories(ctx context.Context) ([]map[string]interface{}, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT c.id, c.name, c.display_order, c.group_id, c.derbynet_award_id, c.active, cg.name as group_name,
		       c.override_winner_car_id, c.override_reason, c.overridden_at, c.allowed_voter_types, c.allowed_ranks
		FROM categories c
		LEFT JOIN category_groups cg ON c.group_id = cg.id
		ORDER BY c.display_order
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var categories []map[string]interface{}
	for rows.Next() {
		var id, displayOrder int
		var groupID, derbynetAwardID, overrideWinnerCarID sql.NullInt64
		var name string
		var groupName, overrideReason, overriddenAt, allowedVoterTypesJSON, allowedRanksJSON sql.NullString
		var active bool
		if err := rows.Scan(&id, &name, &displayOrder, &groupID, &derbynetAwardID, &active, &groupName,
			&overrideWinnerCarID, &overrideReason, &overriddenAt, &allowedVoterTypesJSON, &allowedRanksJSON); err != nil {
			return nil, err
		}
		cat := map[string]interface{}{
			"id":            id,
			"name":          name,
			"display_order": displayOrder,
			"active":        active,
		}
		if groupID.Valid {
			cat["group_id"] = int(groupID.Int64)
			cat["group_name"] = groupName.String
		}
		if derbynetAwardID.Valid {
			cat["derbynet_award_id"] = int(derbynetAwardID.Int64)
		}
		if overrideWinnerCarID.Valid {
			cat["override_winner_car_id"] = int(overrideWinnerCarID.Int64)
		}
		if overrideReason.Valid {
			cat["override_reason"] = overrideReason.String
		}
		if overriddenAt.Valid {
			cat["overridden_at"] = overriddenAt.String
		}
		// Parse allowed_voter_types JSON
		if allowedVoterTypesJSON.Valid && allowedVoterTypesJSON.String != "" {
			var allowedTypes []string
			if err := json.Unmarshal([]byte(allowedVoterTypesJSON.String), &allowedTypes); err == nil {
				cat["allowed_voter_types"] = allowedTypes
			}
		}
		// Parse allowed_ranks JSON
		if allowedRanksJSON.Valid && allowedRanksJSON.String != "" {
			var allowedRanks []string
			if err := json.Unmarshal([]byte(allowedRanksJSON.String), &allowedRanks); err == nil {
				cat["allowed_ranks"] = allowedRanks
			}
		}
		categories = append(categories, cat)
	}
	return categories, nil
}

// CreateCategory creates a new category
func (r *Repository) CreateCategory(ctx context.Context, name string, displayOrder int, groupID *int, allowedVoterTypes []string, allowedRanks []string) (int64, error) {
	var voterTypesJSON, ranksJSON sql.NullString
	if len(allowedVoterTypes) > 0 {
		jsonData, _ := json.Marshal(allowedVoterTypes) // Marshal on []string never fails
		voterTypesJSON = sql.NullString{String: string(jsonData), Valid: true}
	}
	if len(allowedRanks) > 0 {
		jsonData, _ := json.Marshal(allowedRanks) // Marshal on []string never fails
		ranksJSON = sql.NullString{String: string(jsonData), Valid: true}
	}

	result, err := r.db.ExecContext(ctx,
		`INSERT INTO categories (name, display_order, group_id, allowed_voter_types, allowed_ranks, active) VALUES (?, ?, ?, ?, ?, 1)`,
		name, displayOrder, groupID, voterTypesJSON, ranksJSON)
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}

// UpdateCategory updates a category including allowed voter types and allowed ranks
func (r *Repository) UpdateCategory(ctx context.Context, id int, name string, displayOrder int, groupID *int, allowedVoterTypes []string, allowedRanks []string, active bool) error {
	var voterTypesJSON, ranksJSON sql.NullString
	if len(allowedVoterTypes) > 0 {
		jsonData, _ := json.Marshal(allowedVoterTypes) // Marshal on []string never fails
		voterTypesJSON = sql.NullString{String: string(jsonData), Valid: true}
	}
	if len(allowedRanks) > 0 {
		jsonData, _ := json.Marshal(allowedRanks) // Marshal on []string never fails
		ranksJSON = sql.NullString{String: string(jsonData), Valid: true}
	}

	_, err := r.db.ExecContext(ctx,
		`UPDATE categories SET name = ?, display_order = ?, group_id = ?, allowed_voter_types = ?, allowed_ranks = ?, active = ? WHERE id = ?`,
		name, displayOrder, groupID, voterTypesJSON, ranksJSON, active, id)
	return err
}

// DeleteCategory soft-deletes a category
func (r *Repository) DeleteCategory(ctx context.Context, id int) error {
	_, err := r.db.ExecContext(ctx, `UPDATE categories SET active = 0 WHERE id = ?`, id)
	return err
}

// CategoryExists checks if a category with the given name exists
func (r *Repository) CategoryExists(ctx context.Context, name string) (bool, error) {
	var exists bool
	err := r.db.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM categories WHERE name = ?)`, name).Scan(&exists)
	return exists, err
}

// UpsertCategory creates or updates a category by name, returns whether it was created
// Also links to DerbyNet award ID if provided (for bi-directional sync)
func (r *Repository) UpsertCategory(ctx context.Context, name string, displayOrder int, derbynetAwardID *int) (bool, error) {
	// Check if category exists (by name)
	exists, err := r.CategoryExists(ctx, name)
	if err != nil {
		return false, err
	}

	if exists {
		// Update existing category - also update derbynet_award_id if provided
		// Note: We don't change the active state - user may have intentionally deactivated
		_, err := r.db.ExecContext(ctx,
			`UPDATE categories SET display_order = ?, derbynet_award_id = COALESCE(?, derbynet_award_id) WHERE name = ?`,
			displayOrder, derbynetAwardID, name)
		return false, err
	}

	// Create new category
	_, err = r.db.ExecContext(ctx,
		`INSERT INTO categories (name, display_order, derbynet_award_id) VALUES (?, ?, ?)`,
		name, displayOrder, derbynetAwardID)
	return true, err
}

// SetManualWinner sets the manual winner override for a category
func (r *Repository) SetManualWinner(ctx context.Context, categoryID, carID int, reason string) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE categories
		 SET override_winner_car_id = ?, override_reason = ?, overridden_at = CURRENT_TIMESTAMP
		 WHERE id = ?`,
		carID, reason, categoryID)
	return err
}

// ClearManualWinner clears the manual winner override for a category
func (r *Repository) ClearManualWinner(ctx context.Context, categoryID int) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE categories
		 SET override_winner_car_id = NULL, override_reason = NULL, overridden_at = NULL
		 WHERE id = ?`,
		categoryID)
	return err
}

// ==================== Category Group Methods ====================

// ListCategoryGroups returns all active category groups
func (r *Repository) ListCategoryGroups(ctx context.Context) ([]models.CategoryGroup, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, name, description, exclusivity_pool_id, max_wins_per_car, display_order, active
		FROM category_groups WHERE active = 1 ORDER BY display_order
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var groups []models.CategoryGroup
	for rows.Next() {
		var group models.CategoryGroup
		var description sql.NullString
		var exclusivityPoolID sql.NullInt64
		var maxWinsPerCar sql.NullInt64
		if err := rows.Scan(&group.ID, &group.Name, &description, &exclusivityPoolID, &maxWinsPerCar, &group.DisplayOrder, &group.Active); err != nil {
			return nil, err
		}
		group.Description = description.String
		if exclusivityPoolID.Valid {
			poolID := int(exclusivityPoolID.Int64)
			group.ExclusivityPoolID = &poolID
		}
		if maxWinsPerCar.Valid {
			maxWins := int(maxWinsPerCar.Int64)
			group.MaxWinsPerCar = &maxWins
		}
		groups = append(groups, group)
	}
	return groups, nil
}

// GetCategoryGroup retrieves a category group by ID
func (r *Repository) GetCategoryGroup(ctx context.Context, id string) (*models.CategoryGroup, error) {
	var group models.CategoryGroup
	var description sql.NullString
	var exclusivityPoolID sql.NullInt64
	var maxWinsPerCar sql.NullInt64
	err := r.db.QueryRowContext(ctx,
		`SELECT id, name, description, exclusivity_pool_id, max_wins_per_car, display_order, active FROM category_groups WHERE id = ?`,
		id).Scan(&group.ID, &group.Name, &description, &exclusivityPoolID, &maxWinsPerCar, &group.DisplayOrder, &group.Active)

	if err == sql.ErrNoRows {
		return nil, errors.NotFound("category group not found")
	}
	if err != nil {
		return nil, err
	}

	group.Description = description.String
	if exclusivityPoolID.Valid {
		poolID := int(exclusivityPoolID.Int64)
		group.ExclusivityPoolID = &poolID
	}
	if maxWinsPerCar.Valid {
		maxWins := int(maxWinsPerCar.Int64)
		group.MaxWinsPerCar = &maxWins
	}
	return &group, nil
}

// CreateCategoryGroup creates a new category group
func (r *Repository) CreateCategoryGroup(ctx context.Context, name, description string, exclusivityPoolID *int, maxWinsPerCar *int, displayOrder int) (int64, error) {
	result, err := r.db.ExecContext(ctx,
		`INSERT INTO category_groups (name, description, exclusivity_pool_id, max_wins_per_car, display_order, active) VALUES (?, ?, ?, ?, ?, 1)`,
		name, description, exclusivityPoolID, maxWinsPerCar, displayOrder)
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}

// UpdateCategoryGroup updates a category group
func (r *Repository) UpdateCategoryGroup(ctx context.Context, id string, name, description string, exclusivityPoolID *int, maxWinsPerCar *int, displayOrder int) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE category_groups SET name = ?, description = ?, exclusivity_pool_id = ?, max_wins_per_car = ?, display_order = ? WHERE id = ?`,
		name, description, exclusivityPoolID, maxWinsPerCar, displayOrder, id)
	return err
}

// DeleteCategoryGroup deletes a category group
func (r *Repository) DeleteCategoryGroup(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM category_groups WHERE id = ?`, id)
	return err
}

// ==================== Car Methods ====================

// ListCars returns all active cars (including ineligible ones, for admin views)
func (r *Repository) ListCars(ctx context.Context) ([]models.Car, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, car_number, racer_name, car_name, photo_url, rank, COALESCE(eligible, 1) as eligible
		FROM cars WHERE active = 1
		ORDER BY CAST(car_number AS INTEGER)
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var cars []models.Car
	for rows.Next() {
		var car models.Car
		var racerName, carName, photoURL, rank sql.NullString
		if err := rows.Scan(&car.ID, &car.CarNumber, &racerName, &carName, &photoURL, &rank, &car.Eligible); err != nil {
			return nil, err
		}
		car.RacerName = racerName.String
		car.CarName = carName.String
		car.PhotoURL = photoURL.String
		car.Rank = rank.String
		cars = append(cars, car)
	}
	return cars, nil
}

// ListEligibleCars returns all active and eligible cars (for voting)
func (r *Repository) ListEligibleCars(ctx context.Context) ([]models.Car, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, car_number, racer_name, car_name, photo_url, rank, COALESCE(eligible, 1) as eligible
		FROM cars WHERE active = 1 AND COALESCE(eligible, 1) = 1
		ORDER BY CAST(car_number AS INTEGER)
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var cars []models.Car
	for rows.Next() {
		var car models.Car
		var racerName, carName, photoURL, rank sql.NullString
		if err := rows.Scan(&car.ID, &car.CarNumber, &racerName, &carName, &photoURL, &rank, &car.Eligible); err != nil {
			return nil, err
		}
		car.RacerName = racerName.String
		car.CarName = carName.String
		car.PhotoURL = photoURL.String
		car.Rank = rank.String
		cars = append(cars, car)
	}
	return cars, nil
}

// GetCarByDerbyNetID checks if a car exists by DerbyNet racer ID
func (r *Repository) GetCarByDerbyNetID(ctx context.Context, racerID int) (int64, bool, error) {
	var id sql.NullInt64
	err := r.db.QueryRowContext(ctx, `SELECT id FROM cars WHERE derbynet_racer_id = ?`, racerID).Scan(&id)
	if err == sql.ErrNoRows {
		return 0, false, nil
	}
	if err != nil {
		return 0, false, err
	}
	return id.Int64, id.Valid, nil
}

// UpsertCar creates or updates a car
func (r *Repository) UpsertCar(ctx context.Context, derbynetRacerID int, carNumber, racerName, carName, photoURL, rank string) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO cars (derbynet_racer_id, car_number, racer_name, car_name, photo_url, rank, active)
		VALUES (?, ?, ?, ?, ?, ?, 1)
		ON CONFLICT(derbynet_racer_id) DO UPDATE SET
			car_number = excluded.car_number,
			racer_name = excluded.racer_name,
			car_name = excluded.car_name,
			photo_url = excluded.photo_url,
			rank = excluded.rank,
			synced_at = CURRENT_TIMESTAMP
	`, derbynetRacerID, carNumber, racerName, carName, photoURL, rank)
	return err
}

// UpsertVoterForCar creates or updates a voter for a car
func (r *Repository) UpsertVoterForCar(ctx context.Context, carID int64, name, qrCode string) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO voters (car_id, name, voter_type, qr_code)
		VALUES (?, ?, 'racer', ?)
		ON CONFLICT(qr_code) DO UPDATE SET
			car_id = excluded.car_id,
			name = excluded.name
	`, carID, name, qrCode)
	return err
}

// GetVoterByQRCode checks if a voter exists by QR code
func (r *Repository) GetVoterByQRCode(ctx context.Context, qrCode string) (int64, bool, error) {
	var id sql.NullInt64
	err := r.db.QueryRowContext(ctx, `SELECT id FROM voters WHERE qr_code = ?`, qrCode).Scan(&id)
	if err == sql.ErrNoRows {
		return 0, false, nil
	}
	if err != nil {
		return 0, false, err
	}
	return id.Int64, id.Valid, nil
}

// CarExists checks if a car with the given number exists
func (r *Repository) CarExists(ctx context.Context, carNumber string) (bool, error) {
	var exists bool
	err := r.db.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM cars WHERE car_number = ?)`, carNumber).Scan(&exists)
	return exists, err
}

// CreateCar creates a new car
func (r *Repository) CreateCar(ctx context.Context, carNumber, racerName, carName, photoURL string) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO cars (car_number, racer_name, car_name, photo_url, rank, active) VALUES (?, ?, ?, ?, '', 1)`,
		carNumber, racerName, carName, photoURL)
	return err
}

// GetCar returns a car by ID
func (r *Repository) GetCar(ctx context.Context, id int) (*models.Car, error) {
	var car models.Car
	var racerName, carName, photoURL, rank sql.NullString
	err := r.db.QueryRowContext(ctx, `
		SELECT id, car_number, racer_name, car_name, photo_url, rank, COALESCE(eligible, 1) as eligible
		FROM cars WHERE id = ? AND active = 1
	`, id).Scan(&car.ID, &car.CarNumber, &racerName, &carName, &photoURL, &rank, &car.Eligible)
	if err == sql.ErrNoRows {
		return nil, errors.NotFound("car not found")
	}
	if err != nil {
		return nil, err
	}
	car.RacerName = racerName.String
	car.CarName = carName.String
	car.PhotoURL = photoURL.String
	car.Rank = rank.String
	return &car, nil
}

// UpdateCar updates a car
func (r *Repository) UpdateCar(ctx context.Context, id int, carNumber, racerName, carName, photoURL, rank string) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE cars SET car_number = ?, racer_name = ?, car_name = ?, photo_url = ?, rank = ? WHERE id = ?`,
		carNumber, racerName, carName, photoURL, rank, id)
	return err
}

// SetCarEligibility updates a car's eligibility for voting
func (r *Repository) SetCarEligibility(ctx context.Context, id int, eligible bool) error {
	_, err := r.db.ExecContext(ctx, `UPDATE cars SET eligible = ? WHERE id = ?`, eligible, id)
	return err
}

// DeleteCar soft deletes a car
func (r *Repository) DeleteCar(ctx context.Context, id int) error {
	_, err := r.db.ExecContext(ctx, `UPDATE cars SET active = 0 WHERE id = ?`, id)
	return err
}

// ==================== Vote Methods ====================

// GetVoterVotes returns all votes for a voter
func (r *Repository) GetVoterVotes(ctx context.Context, voterID int) (map[int]int, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT category_id, car_id FROM votes WHERE voter_id = ?`, voterID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	votes := make(map[int]int)
	for rows.Next() {
		var categoryID, carID int
		if err := rows.Scan(&categoryID, &carID); err != nil {
			return nil, err
		}
		votes[categoryID] = carID
	}
	return votes, nil
}

// CountVotesForCar returns the number of votes a car has received
func (r *Repository) CountVotesForCar(ctx context.Context, carID int) (int, error) {
	var count int
	err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM votes WHERE car_id = ?`, carID).Scan(&count)
	return count, err
}

// CountVotesForCategory returns the number of votes in a category
func (r *Repository) CountVotesForCategory(ctx context.Context, categoryID int) (int, error) {
	var count int
	err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM votes WHERE category_id = ?`, categoryID).Scan(&count)
	return count, err
}

// SaveVote saves or updates a vote
func (r *Repository) SaveVote(ctx context.Context, voterID, categoryID, carID int) error {
	now := time.Now()

	if carID == 0 {
		_, err := r.db.ExecContext(ctx, `DELETE FROM votes WHERE voter_id = ? AND category_id = ?`, voterID, categoryID)
		return err
	}

	_, err := r.db.ExecContext(ctx, `
		INSERT INTO votes (voter_id, category_id, car_id, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?)
		ON CONFLICT(voter_id, category_id) DO UPDATE SET
			car_id = excluded.car_id,
			updated_at = excluded.updated_at
	`, voterID, categoryID, carID, now, now)

	if err != nil {
		return err
	}

	_, err = r.db.ExecContext(ctx, `UPDATE voters SET last_voted_at = ? WHERE id = ?`, now, voterID)
	return err
}

// GetExclusivityPoolID returns the exclusivity pool ID for a category
func (r *Repository) GetExclusivityPoolID(ctx context.Context, categoryID int) (int64, bool, error) {
	var exclusivityPoolID sql.NullInt64
	err := r.db.QueryRowContext(ctx, `
		SELECT cg.exclusivity_pool_id
		FROM categories c
		LEFT JOIN category_groups cg ON c.group_id = cg.id
		WHERE c.id = ?
	`, categoryID).Scan(&exclusivityPoolID)
	if err != nil {
		return 0, false, err
	}
	return exclusivityPoolID.Int64, exclusivityPoolID.Valid, nil
}

// FindConflictingVote finds a conflicting vote in the same exclusivity pool
func (r *Repository) FindConflictingVote(ctx context.Context, voterID, carID, categoryID int, poolID int64) (int, string, bool, error) {
	var conflictCategoryID int
	var conflictCategoryName string
	err := r.db.QueryRowContext(ctx, `
		SELECT v.category_id, c.name
		FROM votes v
		JOIN categories c ON v.category_id = c.id
		JOIN category_groups cg ON c.group_id = cg.id
		WHERE v.voter_id = ? AND v.car_id = ? AND v.category_id != ? AND cg.exclusivity_pool_id = ?
		LIMIT 1
	`, voterID, carID, categoryID, poolID).Scan(&conflictCategoryID, &conflictCategoryName)

	if err == sql.ErrNoRows {
		return 0, "", false, nil
	}
	if err != nil {
		return 0, "", false, err
	}
	return conflictCategoryID, conflictCategoryName, true, nil
}

// ClearConflictingVote removes a vote
func (r *Repository) ClearConflictingVote(ctx context.Context, voterID, categoryID, carID int) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM votes WHERE voter_id = ? AND category_id = ? AND car_id = ?`, voterID, categoryID, carID)
	return err
}

// GetVoteResults returns vote counts per category and car
func (r *Repository) GetVoteResults(ctx context.Context) (map[int]map[int]int, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT category_id, car_id, COUNT(*) as vote_count
		FROM votes GROUP BY category_id, car_id ORDER BY category_id, vote_count DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	results := make(map[int]map[int]int)
	for rows.Next() {
		var categoryID, carID, count int
		if err := rows.Scan(&categoryID, &carID, &count); err != nil {
			return nil, err
		}
		if results[categoryID] == nil {
			results[categoryID] = make(map[int]int)
		}
		results[categoryID][carID] = count
	}
	return results, nil
}

// VoteResultRow represents a single vote result with car details
type VoteResultRow struct {
	CategoryID int
	CarID      int
	CarNumber  string
	CarName    string
	RacerName  string
	PhotoURL   string
	VoteCount  int
}

// GetVoteResultsWithCars returns vote results with car details (only cars with votes)
func (r *Repository) GetVoteResultsWithCars(ctx context.Context) ([]VoteResultRow, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT v.category_id, v.car_id, c.car_number, c.car_name, c.racer_name, c.photo_url, COUNT(*) as vote_count
		FROM votes v
		JOIN cars c ON v.car_id = c.id
		GROUP BY v.category_id, v.car_id
		ORDER BY v.category_id, vote_count DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []VoteResultRow
	for rows.Next() {
		var row VoteResultRow
		var carName, racerName, photoURL sql.NullString
		if err := rows.Scan(&row.CategoryID, &row.CarID, &row.CarNumber, &carName, &racerName, &photoURL, &row.VoteCount); err != nil {
			return nil, err
		}
		row.CarName = carName.String
		row.RacerName = racerName.String
		row.PhotoURL = photoURL.String
		results = append(results, row)
	}
	return results, nil
}

// WinnerForDerbyNet represents a winner with DerbyNet IDs for syncing
type WinnerForDerbyNet struct {
	CategoryID      int
	CategoryName    string
	DerbyNetAwardID *int
	CarID           int
	DerbyNetRacerID *int
	VoteCount       int
}

// GetWinnersForDerbyNet returns the winner per category with DerbyNet IDs, respecting manual overrides
func (r *Repository) GetWinnersForDerbyNet(ctx context.Context) ([]WinnerForDerbyNet, error) {
	// Get top vote for each category with DerbyNet IDs, respecting manual overrides
	rows, err := r.db.QueryContext(ctx, `
		WITH ranked_votes AS (
			SELECT
				v.category_id,
				v.car_id,
				COUNT(*) as vote_count,
				ROW_NUMBER() OVER (PARTITION BY v.category_id ORDER BY COUNT(*) DESC) as rn
			FROM votes v
			GROUP BY v.category_id, v.car_id
		)
		SELECT
			c.id,
			c.name,
			c.derbynet_award_id,
			COALESCE(c.override_winner_car_id, rv.car_id) as winner_car_id,
			cars.derbynet_racer_id,
			COALESCE(rv.vote_count, 0) as vote_count
		FROM categories c
		LEFT JOIN ranked_votes rv ON rv.category_id = c.id AND rv.rn = 1
		LEFT JOIN cars ON cars.id = COALESCE(c.override_winner_car_id, rv.car_id)
		WHERE c.active = 1
		  AND (c.override_winner_car_id IS NOT NULL OR (rv.car_id IS NOT NULL AND rv.vote_count > 0))
		ORDER BY c.display_order
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var winners []WinnerForDerbyNet
	for rows.Next() {
		var w WinnerForDerbyNet
		var derbynetAwardID, derbynetRacerID sql.NullInt64
		if err := rows.Scan(&w.CategoryID, &w.CategoryName, &derbynetAwardID, &w.CarID, &derbynetRacerID, &w.VoteCount); err != nil {
			return nil, err
		}
		if derbynetAwardID.Valid {
			id := int(derbynetAwardID.Int64)
			w.DerbyNetAwardID = &id
		}
		if derbynetRacerID.Valid {
			id := int(derbynetRacerID.Int64)
			w.DerbyNetRacerID = &id
		}
		winners = append(winners, w)
	}
	return winners, nil
}

// ==================== Settings Methods ====================

// GetSetting retrieves a setting value
func (r *Repository) GetSetting(ctx context.Context, key string) (string, error) {
	var value string
	err := r.db.QueryRowContext(ctx, `SELECT value FROM settings WHERE key = ?`, key).Scan(&value)
	if err == sql.ErrNoRows {
		return "", ErrNotFound
	}
	return value, err
}

// SetSetting updates a setting value
func (r *Repository) SetSetting(ctx context.Context, key, value string) error {
	_, err := r.db.ExecContext(ctx, `INSERT OR REPLACE INTO settings (key, value) VALUES (?, ?)`, key, value)
	return err
}

// ==================== Stats Methods ====================

// GetVotingStats returns overall voting statistics
func (r *Repository) GetVotingStats(ctx context.Context) (map[string]interface{}, error) {
	stats := make(map[string]interface{})

	var totalVoters int
	if err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM voters`).Scan(&totalVoters); err != nil {
		return nil, err
	}
	stats["total_voters"] = totalVoters

	var votersWhoVoted int
	if err := r.db.QueryRowContext(ctx, `SELECT COUNT(DISTINCT voter_id) FROM votes`).Scan(&votersWhoVoted); err != nil {
		return nil, err
	}
	stats["voters_who_voted"] = votersWhoVoted

	var totalVotes int
	if err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM votes`).Scan(&totalVotes); err != nil {
		return nil, err
	}
	stats["total_votes"] = totalVotes

	var totalCategories int
	if err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM categories WHERE active = 1`).Scan(&totalCategories); err != nil {
		return nil, err
	}
	stats["total_categories"] = totalCategories

	var totalCars int
	if err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM cars WHERE active = 1`).Scan(&totalCars); err != nil {
		return nil, err
	}
	stats["total_cars"] = totalCars

	return stats, nil
}

// ==================== Database Management Methods ====================

// validTables defines which tables can be safely cleared
var validTables = map[string]bool{
	"votes": true, "voters": true, "cars": true, "categories": true, "settings": true,
}

// ClearTable clears all data from a table
// Only allows clearing whitelisted tables to prevent SQL injection
func (r *Repository) ClearTable(ctx context.Context, table string) error {
	// Validate table name against whitelist
	if !validTables[table] {
		return ErrInvalidTable
	}

	// Safe to use string concatenation now that we've validated the table name
	_, err := r.db.ExecContext(ctx, "DELETE FROM "+table)
	return err
}

// InsertVoterIgnore inserts a voter, ignoring conflicts
func (r *Repository) InsertVoterIgnore(ctx context.Context, qrCode string) error {
	_, err := r.db.ExecContext(ctx, `INSERT OR IGNORE INTO voters (qr_code) VALUES (?)`, qrCode)
	return err
}
