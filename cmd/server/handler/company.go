// Added: CompanyLatestHandler to handle GET /companies/latest endpoint
// Returns the most recently created company from the database
package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"

	"startupdose.com/cmd/server/database"
	"startupdose.com/cmd/server/models"
	"startupdose.com/cmd/server/storage"
)

// OpenAI API constants
const (
	openAIEndpoint = "https://api.openai.com/v1/chat/completions"
	openAIModel    = "gpt-4o-mini"
)

// startupDosePrompt is the prompt sent to OpenAI to generate a startup
const startupDosePrompt = `You are the content curator for Startup Dose, a site that spotlights one promising tech startup per day.

Your task:

* Pick ONE tech startup that is:
  * In the technology space (software, hardware, SaaS, AI, dev tools, fintech, etc.).
  * NOT big or famous (avoid any company that is a household name or widely covered like Stripe, Airbnb, Dropbox, OpenAI, Meta, Google, etc.).
  * STILL ACTIVE (based on your knowledge; avoid companies that are clearly shut down, defunct, or discontinued).
  * Interesting enough to feature in a daily startup spotlight.

Use your knowledge of tech startups from news, blogs, databases, and other media in your training data. Prefer a startup where you know:

* The company website.
* At least one good image URL.

Return your answer as a SINGLE JSON object with the following fields:

* "name": string

  The name of the startup (company name), e.g. "Acme AI".
* "website": string

  The main website URL for the startup, e.g. "https://example.com".
* "cover_image": string

  A URL to a good image that we can use later in a social media post. **CRITICAL: Only provide image URLs that you are highly confident actually exist and are publicly accessible.** Do not guess or construct URLs that seem plausible but may not exist.

  Prefer, in this order:
  1. A photo featuring one or more founders, OR
  2. A strong, descriptive product/brand image related to what the company does.
     If you cannot confidently provide (1) or (2), then:
  3. Use a clear logo image from the company's own website (e.g. a logo or brand asset image), and if that is not available,
  4. Use a suitable profile or header image from one of the company's social media accounts (LinkedIn, Instagram, Facebook, or Twitter/X).

  Use a direct image URL (ending in .jpg, .jpeg, .png, .webp, or similar) if possible. **If you cannot provide a verified, working image URL, use the company's website homepage URL as a fallback.**
* "description": string

  A single short paragraph (2–4 sentences) describing:
  * What the company does,
  * Who it is for,
  * Why it's interesting.

    This paragraph should be written so it can be reused almost directly as social media caption text.
* "appeal": string

  EXACTLY five HTML list items (<li>...</li>) explaining why we like this startup. DO NOT wrap them in a <ul> tag.

  Example shape (just for structure, NOT content):

  "<li>Reason 1…</li>
<li>Reason 2…</li>
<li>Reason 3…</li>
<li>Reason 4…</li>
<li>Reason 5…</li>"

* Each bullet should be specific and compelling (traction, innovation, niche, team, product quality, etc.), written in a tone suitable for social media.
* "linkedin": string

  The company's LinkedIn page URL IF you are reasonably confident it exists and you know it.

  If you are not reasonably sure, set this to an empty string "".
* "instagram": string

  The company's Instagram profile URL IF you are reasonably confident it exists and you know it.

  Otherwise, "".
* "facebook": string

  The company's Facebook page URL IF you are reasonably confident it exists and you know it.

  Otherwise, "".
* "twitter": string

  The company's Twitter/X profile URL IF you are reasonably confident it exists and you know it.

  Otherwise, "".

Important formatting rules:

* Output MUST be valid JSON.
* Do NOT wrap the JSON in backticks or any other formatting.
* Do NOT add any extra commentary or explanation outside of the JSON.
* Exactly one startup per response.
* Make sure the "appeal" field is a single string containing exactly five <li> items WITHOUT any <ul> wrapper.

Now select an appropriate, lesser-known, still-active tech startup and return the JSON object.`

// OpenAI request/response types
type OpenAIMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type OpenAIChatRequest struct {
	Model    string          `json:"model"`
	Messages []OpenAIMessage `json:"messages"`
}

type OpenAIChatResponse struct {
	Choices []struct {
		Message OpenAIMessage `json:"message"`
	} `json:"choices"`
}

// CompanyFromAI represents the company data structure returned by OpenAI
type CompanyFromAI struct {
	Name        string `json:"name"`
	Website     string `json:"website"`
	CoverImage  string `json:"cover_image"`
	Description string `json:"description"`
	Appeal      string `json:"appeal"`
	LinkedIn    string `json:"linkedin"`
	Instagram   string `json:"instagram"`
	Facebook    string `json:"facebook"`
	Twitter     string `json:"twitter"`
}

// CompanyLatestHandler handles GET /companies/latest
// Returns the most recently created company from the database
func CompanyLatestHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Create repository instance
	repo := database.NewCompanyRepository()

	// Get the latest company
	company, err := repo.GetLatest()
	if err != nil {
		// Check if it's a "no companies found" error
		if strings.Contains(err.Error(), "no companies found") {
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(ErrorResponse{
				Error:   "not_found",
				Message: "no companies found",
			})
			return
		}

		// Database or other error
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(ErrorResponse{
			Error:   "internal_server_error",
			Message: "Failed to retrieve company",
		})
		return
	}

	// Success - return the company
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(company)
}

// GenerateCompaniesHandler handles POST /companies/generate
// Calls OpenAI to generate a startup, saves it to the database, and returns it
func GenerateCompaniesHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get OpenAI API key from environment
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		log.Println("ERROR: OPENAI_API_KEY not set")
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(ErrorResponse{
			Error:   "internal_server_error",
			Message: "OpenAI API key not configured",
		})
		return
	}

	// Generate company from OpenAI
	companyData, err := generateCompanyFromAI(apiKey)
	if err != nil {
		log.Printf("ERROR: Failed to generate company from AI: %v\n", err)
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusBadGateway)
		json.NewEncoder(w).Encode(ErrorResponse{
			Error:   "bad_gateway",
			Message: "Failed to generate company from AI",
		})
		return
	}

	// Generate slug from company name
	slug := generateSlug(companyData.Name)

	// Capture screenshot of company website and upload to S3
	awsRegion := os.Getenv("AWS_REGION")
	awsAccessKey := os.Getenv("AWS_ACCESS_KEY_ID")
	awsSecretKey := os.Getenv("AWS_SECRET_ACCESS_KEY")
	s3Bucket := os.Getenv("S3_BUCKET_NAME")
	screenshotAPIKey := os.Getenv("SCREENSHOTONE_API_KEY")

	var coverImageURL string
	if awsRegion != "" && awsAccessKey != "" && awsSecretKey != "" && s3Bucket != "" && screenshotAPIKey != "" && companyData.Website != "" {
		// Create S3 uploader
		uploader, err := storage.NewS3Uploader(awsRegion, awsAccessKey, awsSecretKey, s3Bucket)
		if err != nil {
			log.Printf("ERROR: Failed to create S3 uploader: %v\n", err)
			// Fall back to using the original URL from OpenAI
			coverImageURL = companyData.CoverImage
		} else {
			// Capture screenshot of the website
			ctx := r.Context()
			screenshotBytes, err := storage.FetchScreenshotOneBytes(ctx, companyData.Website, screenshotAPIKey)
			if err != nil {
				log.Printf("ERROR: Failed to capture screenshot: %v\n", err)
				// Fall back to using the original URL from OpenAI
				coverImageURL = companyData.CoverImage
			} else {
				// Upload screenshot to S3
				s3URL, err := uploader.UploadScreenshot(screenshotBytes, slug)
				if err != nil {
					log.Printf("ERROR: Failed to upload screenshot to S3: %v\n", err)
					// Fall back to using the original URL from OpenAI
					coverImageURL = companyData.CoverImage
				} else {
					log.Printf("Successfully captured and uploaded screenshot to S3: %s\n", s3URL)
					coverImageURL = s3URL
				}
			}
		}
	} else {
		log.Println("WARNING: Screenshot or S3 not fully configured, using original image URL from OpenAI")
		coverImageURL = companyData.CoverImage
	}

	// Convert to a map for database insertion (only include fields we want to set)
	// This avoids sending empty strings for auto-generated fields like ID
	companyMap := map[string]interface{}{
		"name":        companyData.Name,
		"slug":        slug,
		"website":     stripProtocol(companyData.Website),
		"cover_image": coverImageURL,
		"description": companyData.Description,
		"appeal":      companyData.Appeal,
	}

	// Add social media fields only if they're not empty
	if companyData.Twitter != "" {
		companyMap["twitter"] = companyData.Twitter
	}
	if companyData.LinkedIn != "" {
		companyMap["linkedin"] = companyData.LinkedIn
	}
	if companyData.Facebook != "" {
		companyMap["facebook"] = companyData.Facebook
	}
	if companyData.Instagram != "" {
		companyMap["instagram"] = companyData.Instagram
	}

	// Insert into database using the map
	repo := database.NewCompanyRepository()
	var createdCompany *models.Company
	createdCompany, err = repo.InsertMap(companyMap)
	if err != nil {
		log.Printf("ERROR: Failed to insert company into database: %v\n", err)
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(ErrorResponse{
			Error:   "internal_server_error",
			Message: "Failed to save company to database",
		})
		return
	}

	// Success - return the created company
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(createdCompany)
}

// generateCompanyFromAI calls the OpenAI API to generate a startup company
func generateCompanyFromAI(apiKey string) (*CompanyFromAI, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	// Build the OpenAI request
	reqBody := OpenAIChatRequest{
		Model: openAIModel,
		Messages: []OpenAIMessage{
			{
				Role:    "user",
				Content: startupDosePrompt,
			},
		},
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, openAIEndpoint, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	// Send the request
	client := &http.Client{
		Timeout: 20 * time.Second,
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to call OpenAI API: %w", err)
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("OpenAI API returned status %d", resp.StatusCode)
	}

	// Parse the OpenAI response
	var chatResp OpenAIChatResponse
	if err := json.NewDecoder(resp.Body).Decode(&chatResp); err != nil {
		return nil, fmt.Errorf("failed to decode OpenAI response: %w", err)
	}

	// Validate response structure
	if len(chatResp.Choices) == 0 {
		return nil, fmt.Errorf("OpenAI returned no choices")
	}

	// Get the content (which should be JSON)
	content := chatResp.Choices[0].Message.Content

	// Log the raw content for debugging
	log.Printf("OpenAI response content: %s\n", content)

	// Parse the content as JSON into CompanyFromAI
	var company CompanyFromAI
	if err := json.Unmarshal([]byte(content), &company); err != nil {
		log.Printf("ERROR: Failed to unmarshal company JSON. Raw content: %s\n", content)
		return nil, fmt.Errorf("failed to parse company JSON from OpenAI: %w", err)
	}

	// Post-process: Remove any <ul> or </ul> tags from the appeal field
	// This ensures we only have <li> tags as required
	company.Appeal = strings.ReplaceAll(company.Appeal, "<ul>", "")
	company.Appeal = strings.ReplaceAll(company.Appeal, "</ul>", "")
	company.Appeal = strings.TrimSpace(company.Appeal)

	return &company, nil
}

// generateSlug creates a URL-friendly slug from a company name
// Converts to lowercase, replaces spaces/special chars with hyphens
func generateSlug(name string) string {
	// Convert to lowercase
	slug := strings.ToLower(name)

	// Replace spaces and special characters with hyphens
	reg := regexp.MustCompile(`[^a-z0-9]+`)
	slug = reg.ReplaceAllString(slug, "-")

	// Remove leading/trailing hyphens
	slug = strings.Trim(slug, "-")

	return slug
}

// stripProtocol removes http:// or https:// from a URL
func stripProtocol(url string) string {
	url = strings.TrimPrefix(url, "https://")
	url = strings.TrimPrefix(url, "http://")
	return url
}
