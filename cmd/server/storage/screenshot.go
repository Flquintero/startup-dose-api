package storage

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

// FetchScreenshotOneBytes captures a screenshot of a website using ScreenshotOne API
// Returns the raw screenshot image bytes (PNG format)
func FetchScreenshotOneBytes(ctx context.Context, websiteURL, apiKey string) ([]byte, error) {
	if apiKey == "" {
		return nil, fmt.Errorf("SCREENSHOTONE_API_KEY is not set")
	}

	// Build the ScreenshotOne API URL with query parameters
	baseURL := "https://api.screenshotone.com/take"
	params := url.Values{}
	params.Add("access_key", apiKey)
	params.Add("url", websiteURL)
	params.Add("full_page", "false")              // Only capture above the fold
	params.Add("viewport_width", "1280")
	params.Add("viewport_height", "720")
	params.Add("device_scale_factor", "2")        // Higher quality
	params.Add("format", "png")
	params.Add("block_cookie_banners", "true")    // Remove cookie banners

	fullURL := fmt.Sprintf("%s?%s", baseURL, params.Encode())

	// Create HTTP request with context
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fullURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create screenshot request: %w", err)
	}

	// Use HTTP client with timeout
	client := &http.Client{
		Timeout: 30 * time.Second, // ScreenshotOne can take a while
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to call ScreenshotOne API: %w", err)
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("ScreenshotOne API returned status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	// Read the screenshot bytes
	screenshotBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read screenshot data: %w", err)
	}

	return screenshotBytes, nil
}
