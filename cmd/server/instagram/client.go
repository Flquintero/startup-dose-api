package instagram

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	graphAPIBaseURL    = "https://graph.facebook.com"
	defaultAPIVersion  = "v23.0"
	maxCaptionLength   = 2200
	containerPollDelay = 2 * time.Second
	maxPollAttempts    = 15
)

// Client represents an Instagram Graph API client
type Client struct {
	userID      string
	accessToken string
	apiVersion  string
	httpClient  *http.Client
}

// PublishResult contains the result of a successful Instagram post
type PublishResult struct {
	MediaID   string `json:"media_id"`
	Posted    bool   `json:"posted"`
	Error     string `json:"error,omitempty"`
}

// containerResponse represents the response from creating a media container
type containerResponse struct {
	ID    string `json:"id"`
	Error *struct {
		Message string `json:"message"`
		Code    int    `json:"code"`
	} `json:"error,omitempty"`
}

// containerStatusResponse represents the response from checking container status
type containerStatusResponse struct {
	StatusCode string `json:"status_code"`
	Error      *struct {
		Message string `json:"message"`
		Code    int    `json:"code"`
	} `json:"error,omitempty"`
}

// publishResponse represents the response from publishing a container
type publishResponse struct {
	ID    string `json:"id"`
	Error *struct {
		Message string `json:"message"`
		Code    int    `json:"code"`
	} `json:"error,omitempty"`
}

// NewClient creates a new Instagram client
func NewClient(userID, accessToken, apiVersion string) *Client {
	if apiVersion == "" {
		apiVersion = defaultAPIVersion
	}
	return &Client{
		userID:      userID,
		accessToken: accessToken,
		apiVersion:  apiVersion,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// IsConfigured returns true if the client has required credentials
func (c *Client) IsConfigured() bool {
	return c.userID != "" && c.accessToken != ""
}

// PublishPost publishes an image to Instagram with a caption
func (c *Client) PublishPost(ctx context.Context, imageURL, caption string) (*PublishResult, error) {
	if !c.IsConfigured() {
		return &PublishResult{Posted: false, Error: "Instagram client not configured"}, nil
	}

	// Step 1: Create media container
	containerID, err := c.createMediaContainer(ctx, imageURL, caption)
	if err != nil {
		return &PublishResult{Posted: false, Error: fmt.Sprintf("failed to create media container: %v", err)}, nil
	}
	log.Printf("Instagram: Created media container: %s", containerID)

	// Step 2: Poll for container status until FINISHED
	if err := c.waitForContainerReady(ctx, containerID); err != nil {
		return &PublishResult{Posted: false, Error: fmt.Sprintf("container not ready: %v", err)}, nil
	}
	log.Printf("Instagram: Container ready for publishing")

	// Step 3: Publish the container
	mediaID, err := c.publishContainer(ctx, containerID)
	if err != nil {
		return &PublishResult{Posted: false, Error: fmt.Sprintf("failed to publish: %v", err)}, nil
	}
	log.Printf("Instagram: Successfully published post with media ID: %s", mediaID)

	return &PublishResult{
		MediaID: mediaID,
		Posted:  true,
	}, nil
}

// createMediaContainer creates a media container for the image
func (c *Client) createMediaContainer(ctx context.Context, imageURL, caption string) (string, error) {
	endpoint := fmt.Sprintf("%s/%s/%s/media", graphAPIBaseURL, c.apiVersion, c.userID)

	// Build form data
	data := url.Values{}
	data.Set("image_url", imageURL)
	data.Set("caption", truncateCaption(caption))
	data.Set("access_token", c.accessToken)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(data.Encode()))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	var result containerResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}

	if result.Error != nil {
		return "", fmt.Errorf("API error: %s (code: %d)", result.Error.Message, result.Error.Code)
	}

	if result.ID == "" {
		return "", fmt.Errorf("no container ID returned")
	}

	return result.ID, nil
}

// waitForContainerReady polls the container status until it's FINISHED
func (c *Client) waitForContainerReady(ctx context.Context, containerID string) error {
	endpoint := fmt.Sprintf("%s/%s/%s?fields=status_code&access_token=%s",
		graphAPIBaseURL, c.apiVersion, containerID, c.accessToken)

	for attempt := 0; attempt < maxPollAttempts; attempt++ {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
		if err != nil {
			return fmt.Errorf("failed to create request: %w", err)
		}

		resp, err := c.httpClient.Do(req)
		if err != nil {
			return fmt.Errorf("failed to send request: %w", err)
		}

		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			return fmt.Errorf("failed to read response: %w", err)
		}

		var result containerStatusResponse
		if err := json.Unmarshal(body, &result); err != nil {
			return fmt.Errorf("failed to parse response: %w", err)
		}

		if result.Error != nil {
			return fmt.Errorf("API error: %s (code: %d)", result.Error.Message, result.Error.Code)
		}

		switch result.StatusCode {
		case "FINISHED":
			return nil
		case "ERROR":
			return fmt.Errorf("container processing failed")
		case "EXPIRED":
			return fmt.Errorf("container expired")
		case "IN_PROGRESS":
			// Continue polling
			log.Printf("Instagram: Container status: IN_PROGRESS (attempt %d/%d)", attempt+1, maxPollAttempts)
		default:
			// Unknown status, continue polling
			log.Printf("Instagram: Container status: %s (attempt %d/%d)", result.StatusCode, attempt+1, maxPollAttempts)
		}

		time.Sleep(containerPollDelay)
	}

	return fmt.Errorf("container not ready after %d attempts", maxPollAttempts)
}

// publishContainer publishes the media container to the feed
func (c *Client) publishContainer(ctx context.Context, containerID string) (string, error) {
	endpoint := fmt.Sprintf("%s/%s/%s/media_publish", graphAPIBaseURL, c.apiVersion, c.userID)

	data := url.Values{}
	data.Set("creation_id", containerID)
	data.Set("access_token", c.accessToken)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(data.Encode()))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	var result publishResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}

	if result.Error != nil {
		return "", fmt.Errorf("API error: %s (code: %d)", result.Error.Message, result.Error.Code)
	}

	if result.ID == "" {
		return "", fmt.Errorf("no media ID returned")
	}

	return result.ID, nil
}

// truncateCaption ensures the caption doesn't exceed Instagram's limit
func truncateCaption(caption string) string {
	if len(caption) <= maxCaptionLength {
		return caption
	}
	// Truncate and add ellipsis
	return caption[:maxCaptionLength-3] + "..."
}

// BuildCaption creates a caption from company data in the required format
func BuildCaption(name, description, appeal, website string) string {
	// Convert HTML appeal to plain text bullet points
	plainAppeal := convertAppealToPlainText(appeal)

	// Build the caption
	caption := fmt.Sprintf("Today's Fix \xF0\x9F\x92\x8A\xE2\x9A\xA1\n\n%s\n\n%s\n\nWhy we like it:\n%s\n\nLearn more: %s\n\n#startupdose #startups #tech #innovation",
		name,
		description,
		plainAppeal,
		website,
	)

	return truncateCaption(caption)
}

// convertAppealToPlainText converts HTML <li> tags to plain text bullet points
func convertAppealToPlainText(appeal string) string {
	// Remove <li> and </li> tags, replace with bullet points
	appeal = strings.ReplaceAll(appeal, "<li>", "\xE2\x80\xA2 ")
	appeal = strings.ReplaceAll(appeal, "</li>", "\n")
	appeal = strings.TrimSpace(appeal)
	return appeal
}
