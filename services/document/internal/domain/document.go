package domain

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"time"
)

var ErrDocumentNotFound = errors.New("document not found")

const (
	StatusUploadURLGenerated = "upload_url_generated"
	StatusUploaded           = "uploaded"
)

type Document struct {
	ID          string     `json:"document_id"`
	ProposalID  string     `json:"proposal_id"`
	Type        string     `json:"document_type"`
	FileName    string     `json:"file_name"`
	ContentType string     `json:"content_type"`
	FileKey     string     `json:"file_key"`
	Status      string     `json:"status"`
	UploadURL   string     `json:"upload_url"`
	UploadedAt  *time.Time `json:"uploaded_at,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

type UploadRequest struct {
	DocumentType string
	FileName     string
	ContentType  string
}

func (r UploadRequest) Validate() map[string]any {
	var fields []string

	if strings.TrimSpace(r.DocumentType) == "" {
		fields = append(fields, "document_type")
	}
	if strings.TrimSpace(r.FileName) == "" {
		fields = append(fields, "file_name")
	}
	if strings.TrimSpace(r.ContentType) == "" {
		fields = append(fields, "content_type")
	}

	if len(fields) == 0 {
		return nil
	}

	return map[string]any{"invalid_fields": fields}
}

func NewDocument(proposalID string, request UploadRequest, uploadBaseURL string, now time.Time) Document {
	fileName := filepath.Base(strings.TrimSpace(request.FileName))
	fileKey := fmt.Sprintf("%s/%s/%s", proposalID, strings.TrimSpace(request.DocumentType), fileName)

	return Document{
		ID:          "doc_" + randomToken(12),
		ProposalID:  proposalID,
		Type:        strings.TrimSpace(request.DocumentType),
		FileName:    fileName,
		ContentType: strings.TrimSpace(request.ContentType),
		FileKey:     fileKey,
		Status:      StatusUploadURLGenerated,
		UploadURL:   strings.TrimRight(uploadBaseURL, "/") + "/" + fileKey,
		CreatedAt:   now.UTC(),
		UpdatedAt:   now.UTC(),
	}
}

func (d Document) MarkUploaded(now time.Time) Document {
	d.Status = StatusUploaded
	d.UpdatedAt = now.UTC()
	uploadedAt := now.UTC()
	d.UploadedAt = &uploadedAt
	return d
}

func randomToken(size int) string {
	bytes := make([]byte, size)
	if _, err := rand.Read(bytes); err != nil {
		return time.Now().UTC().Format("20060102150405")
	}

	return hex.EncodeToString(bytes)[:size]
}

func containsKeyword(value, keyword string) bool {
	return strings.Contains(strings.ToLower(value), strings.ToLower(keyword))
}
