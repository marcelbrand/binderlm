package drive

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"google.golang.org/api/drive/v3"
	"google.golang.org/api/googleapi"
)

// DefaultMimeType is the standard MIME type used for raw Markdown uploads to Google Drive.
const DefaultMimeType = "text/markdown"

// SyncResult holds the metadata and status of an upload or upsert operation.
type SyncResult struct {
	FileID     string    `json:"file_id"`
	Action     string    `json:"action"` // "created" | "updated" | "dry_run (create)" | "dry_run (update)"
	Filename   string    `json:"filename"`
	FolderID   string    `json:"folder_id"`
	WebLink    string    `json:"web_link"`
	Size       int64     `json:"size"`
	ModifiedAt time.Time `json:"modified_at"`
}

// Uploader defines the interface for synchronizing markdown content to Google Drive.
type Uploader interface {
	Sync(ctx context.Context, folderID string, filename string, content io.Reader) (*SyncResult, error)
}

// UploaderOption configures the DriveUploader.
type UploaderOption func(*DriveUploader)

// WithMimeType configures the upload MIME type.
func WithMimeType(mimeType string) UploaderOption {
	return func(u *DriveUploader) {
		if mimeType != "" {
			u.mimeType = mimeType
		}
	}
}

// WithDryRun enables or disables dry-run mode.
func WithDryRun(dryRun bool) UploaderOption {
	return func(u *DriveUploader) {
		u.dryRun = dryRun
	}
}

// DriveUploader implements Uploader using the Google Drive v3 API.
type DriveUploader struct {
	service  *drive.Service
	mimeType string
	dryRun   bool
}

// NewUploader constructs a new DriveUploader.
func NewUploader(service *drive.Service, opts ...UploaderOption) *DriveUploader {
	u := &DriveUploader{
		service:  service,
		mimeType: DefaultMimeType,
		dryRun:   false,
	}

	for _, opt := range opts {
		opt(u)
	}

	return u
}

// Sync searches for an existing file in folderID by filename.
// If found, it updates the file content; otherwise, it creates a new file.
// In dry-run mode, it reports what action would be taken without mutating remote state.
func (u *DriveUploader) Sync(ctx context.Context, folderID string, filename string, content io.Reader) (*SyncResult, error) {
	if strings.TrimSpace(folderID) == "" {
		return nil, fmt.Errorf("target Google Drive folder_id must not be empty")
	}
	if strings.TrimSpace(filename) == "" {
		return nil, fmt.Errorf("target filename must not be empty")
	}

	escapedFolderID := escapeDriveQuery(folderID)
	escapedFilename := escapeDriveQuery(filename)
	query := fmt.Sprintf("'%s' in parents and name = '%s' and trashed = false", escapedFolderID, escapedFilename)

	fileList, err := u.service.Files.List().
		Q(query).
		SupportsAllDrives(true).
		IncludeItemsFromAllDrives(true).
		Fields("files(id, name, webViewLink, size, modifiedTime, trashed)").
		Context(ctx).
		Do()
	if err != nil {
		return nil, fmt.Errorf("failed to query Google Drive folder: %w", err)
	}

	// 1. Dry run handling
	if u.dryRun {
		if len(fileList.Files) > 0 {
			existing := fileList.Files[0]
			return &SyncResult{
				FileID:   existing.Id,
				Action:   "dry_run (update)",
				Filename: filename,
				FolderID: folderID,
				WebLink:  existing.WebViewLink,
				Size:     existing.Size,
			}, nil
		}
		return &SyncResult{
			Action:   "dry_run (create)",
			Filename: filename,
			FolderID: folderID,
		}, nil
	}

	// 2. Existing file found -> Update
	if len(fileList.Files) > 0 {
		existing := fileList.Files[0]
		fileUpdate := &drive.File{}

		res, err := u.service.Files.Update(existing.Id, fileUpdate).
			SupportsAllDrives(true).
			Media(content, googleapi.ContentType(u.mimeType)).
			Fields("id, name, webViewLink, size, modifiedTime").
			Context(ctx).
			Do()
		if err != nil {
			return nil, fmt.Errorf("failed to update file %q (ID: %s) in Google Drive: %w", filename, existing.Id, err)
		}

		modTime, _ := time.Parse(time.RFC3339, res.ModifiedTime)
		return &SyncResult{
			FileID:     res.Id,
			Action:     "updated",
			Filename:   res.Name,
			FolderID:   folderID,
			WebLink:    res.WebViewLink,
			Size:       res.Size,
			ModifiedAt: modTime,
		}, nil
	}

	// 3. File not found -> Create
	newFile := &drive.File{
		Name:     filename,
		Parents:  []string{folderID},
		MimeType: u.mimeType,
	}

	res, err := u.service.Files.Create(newFile).
		SupportsAllDrives(true).
		Media(content, googleapi.ContentType(u.mimeType)).
		Fields("id, name, webViewLink, size, modifiedTime").
		Context(ctx).
		Do()
	if err != nil {
		return nil, formatDriveCreateError(err, filename, folderID)
	}

	modTime, _ := time.Parse(time.RFC3339, res.ModifiedTime)
	return &SyncResult{
		FileID:     res.Id,
		Action:     "created",
		Filename:   res.Name,
		FolderID:   folderID,
		WebLink:    res.WebViewLink,
		Size:       res.Size,
		ModifiedAt: modTime,
	}, nil
}

// formatDriveCreateError converts common Drive API errors (like quota exhaustion) into actionable messages.
func formatDriveCreateError(err error, filename string, folderID string) error {
	var gErr *googleapi.Error
	if errors.As(err, &gErr) {
		for _, item := range gErr.Errors {
			if item.Reason == "storageQuotaExceeded" || strings.Contains(item.Message, "storage quota") {
				return fmt.Errorf("Google Drive blocked file creation due to Service Account quota limits on personal 'My Drive' folders.\n\n"+
					"💡 Quick Solutions:\n"+
					"  1. Create a blank file named %q once in your target folder:\n"+
					"     👉 https://drive.google.com/drive/folders/%s\n"+
					"     Then run 'binderlm sync' again (binderlm will update it seamlessly going forward).\n"+
					"  2. Or, move the target folder to a Google Workspace 'Shared Drive' (Team Drive).\n",
					filename, folderID)
			}
		}
		if gErr.Code == 404 || strings.Contains(gErr.Message, "File not found") {
			return fmt.Errorf("Google Drive folder %q not found or not accessible. Ensure the folder exists and is shared with your Service Account as Editor", folderID)
		}
	} else if strings.Contains(err.Error(), "storageQuotaExceeded") || strings.Contains(err.Error(), "storage quota") {
		return fmt.Errorf("Google Drive blocked file creation due to Service Account quota limits on personal 'My Drive' folders.\n\n"+
			"💡 Quick Solutions:\n"+
			"  1. Create a blank file named %q once in your target folder:\n"+
			"     👉 https://drive.google.com/drive/folders/%s\n"+
			"     Then run 'binderlm sync' again (binderlm will update it seamlessly going forward).\n"+
			"  2. Or, move the target folder to a Google Workspace 'Shared Drive' (Team Drive).\n",
			filename, folderID)
	}

	return fmt.Errorf("failed to create file %q in Google Drive folder %s: %w", filename, folderID, err)
}

// escapeDriveQuery escapes single quotes and backslashes in Drive query terms.
func escapeDriveQuery(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	return strings.ReplaceAll(s, `'`, `\'`)
}

