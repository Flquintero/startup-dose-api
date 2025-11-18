// Added: CompanyRepository with GetLatest() method to query the latest company from the database
package database

import (
	"fmt"

	"github.com/supabase-community/postgrest-go"
	"startupdose.com/cmd/server/models"
)

// CompanyRepository handles company-related database operations
type CompanyRepository struct{}

// NewCompanyRepository creates a new CompanyRepository instance
func NewCompanyRepository() *CompanyRepository {
	return &CompanyRepository{}
}

// GetLatest retrieves the most recently created company from the database
// Returns an error if no companies are found or if a database error occurs
func (r *CompanyRepository) GetLatest() (*models.Company, error) {
	client := GetClient()
	if client == nil {
		return nil, fmt.Errorf("database client not initialized")
	}

	var companies []models.Company

	// Query: SELECT * FROM companies ORDER BY created_at DESC LIMIT 1
	_, err := client.
		From("companies").
		Select("*", "", false).
		Order("created_at", &postgrest.OrderOpts{Ascending: false}).
		Limit(1, "").
		ExecuteTo(&companies)

	if err != nil {
		return nil, fmt.Errorf("failed to query database: %w", err)
	}

	if len(companies) == 0 {
		return nil, fmt.Errorf("no companies found")
	}

	return &companies[0], nil
}

// Insert creates a new company in the database
// Returns the created company or an error if the operation fails
func (r *CompanyRepository) Insert(company *models.Company) (*models.Company, error) {
	client := GetClient()
	if client == nil {
		return nil, fmt.Errorf("database client not initialized")
	}

	var result []models.Company

	// Insert the company and return the created record
	_, err := client.
		From("companies").
		Insert(company, false, "", "", "").
		ExecuteTo(&result)

	if err != nil {
		return nil, fmt.Errorf("failed to insert company: %w", err)
	}

	if len(result) == 0 {
		return nil, fmt.Errorf("insert succeeded but no company was returned")
	}

	return &result[0], nil
}

// InsertMap creates a new company in the database using a map
// This allows inserting only specific fields without sending empty values for auto-generated fields
func (r *CompanyRepository) InsertMap(companyData map[string]interface{}) (*models.Company, error) {
	client := GetClient()
	if client == nil {
		return nil, fmt.Errorf("database client not initialized")
	}

	var result []models.Company

	// Insert the company and return the created record
	_, err := client.
		From("companies").
		Insert(companyData, false, "", "", "").
		ExecuteTo(&result)

	if err != nil {
		return nil, fmt.Errorf("failed to insert company: %w", err)
	}

	if len(result) == 0 {
		return nil, fmt.Errorf("insert succeeded but no company was returned")
	}

	return &result[0], nil
}
