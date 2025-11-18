package storage

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
)

// S3Uploader handles uploading files to S3
type S3Uploader struct {
	client     *s3.S3
	bucketName string
	region     string
}

// NewS3Uploader creates a new S3 uploader instance
func NewS3Uploader(region, accessKeyID, secretAccessKey, bucketName string) (*S3Uploader, error) {
	sess, err := session.NewSession(&aws.Config{
		Region:      aws.String(region),
		Credentials: credentials.NewStaticCredentials(accessKeyID, secretAccessKey, ""),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create AWS session: %w", err)
	}

	return &S3Uploader{
		client:     s3.New(sess),
		bucketName: bucketName,
		region:     region,
	}, nil
}

// UploadImageFromURL downloads an image from a URL and uploads it to S3
// Returns the S3 URL of the uploaded image
func (u *S3Uploader) UploadImageFromURL(imageURL, companySlug string) (string, error) {
	// Download the image
	resp, err := http.Get(imageURL)
	if err != nil {
		return "", fmt.Errorf("failed to download image: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to download image: status code %d", resp.StatusCode)
	}

	// Read the image data
	imageData, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read image data: %w", err)
	}

	// Get content type from response header
	contentType := resp.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	// Generate a unique filename
	ext := getExtensionFromURL(imageURL)
	if ext == "" {
		ext = getExtensionFromContentType(contentType)
	}
	timestamp := time.Now().Unix()
	key := fmt.Sprintf("companies/%s/%d%s", companySlug, timestamp, ext)

	// Upload to S3
	_, err = u.client.PutObject(&s3.PutObjectInput{
		Bucket:      aws.String(u.bucketName),
		Key:         aws.String(key),
		Body:        bytes.NewReader(imageData),
		ContentType: aws.String(contentType),
		ACL:         aws.String("public-read"), // Make the image publicly accessible
	})
	if err != nil {
		return "", fmt.Errorf("failed to upload to S3: %w", err)
	}

	// Construct the S3 URL
	s3URL := fmt.Sprintf("https://%s.s3.%s.amazonaws.com/%s", u.bucketName, u.region, key)
	return s3URL, nil
}

// UploadScreenshot uploads screenshot bytes directly to S3
// Returns the S3 URL of the uploaded screenshot
func (u *S3Uploader) UploadScreenshot(screenshotData []byte, companySlug string) (string, error) {
	// Generate a unique filename for the screenshot
	timestamp := time.Now().Unix()
	key := fmt.Sprintf("startup-screenshots/%s-%d.png", companySlug, timestamp)

	// Upload to S3
	_, err := u.client.PutObject(&s3.PutObjectInput{
		Bucket:      aws.String(u.bucketName),
		Key:         aws.String(key),
		Body:        bytes.NewReader(screenshotData),
		ContentType: aws.String("image/png"),
		ACL:         aws.String("public-read"), // Make the screenshot publicly accessible
	})
	if err != nil {
		return "", fmt.Errorf("failed to upload screenshot to S3: %w", err)
	}

	// Construct the S3 URL
	s3URL := fmt.Sprintf("https://%s.s3.%s.amazonaws.com/%s", u.bucketName, u.region, key)
	return s3URL, nil
}

// getExtensionFromURL extracts the file extension from a URL
func getExtensionFromURL(url string) string {
	// Remove query parameters
	if idx := strings.Index(url, "?"); idx != -1 {
		url = url[:idx]
	}

	ext := filepath.Ext(url)
	if ext != "" {
		return ext
	}
	return ""
}

// getExtensionFromContentType returns a file extension based on content type
func getExtensionFromContentType(contentType string) string {
	switch contentType {
	case "image/jpeg", "image/jpg":
		return ".jpg"
	case "image/png":
		return ".png"
	case "image/gif":
		return ".gif"
	case "image/webp":
		return ".webp"
	default:
		return ".jpg" // Default to .jpg
	}
}
