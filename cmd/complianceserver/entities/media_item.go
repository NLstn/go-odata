package entities

import "time"

// MediaItem represents a media entity (media link entry) for compliance testing
// Media entities have a binary stream as their primary content
type MediaItem struct {
	ID          uint      `json:"ID" gorm:"primaryKey" odata:"key"`
	Name        string    `json:"Name" gorm:"not null" odata:"required,maxlength=100"`
	ContentType string    `json:"ContentType" gorm:"not null" odata:"required,maxlength=100"` // MIME type of the media
	Size        *int64    `json:"Size" odata:"nullable"`                                      // Size in bytes
	Content     []byte    `json:"-" gorm:"type:blob"`                                         // Binary content (excluded from JSON)
	CreatedAt   time.Time `json:"CreatedAt" gorm:"not null"`
	ModifiedAt  time.Time `json:"ModifiedAt" gorm:"not null"`
}

// HasStream returns true indicating this is a media entity
func (MediaItem) HasStream() bool {
	return true
}

// GetMediaContent returns the binary content of the media entity
func (m *MediaItem) GetMediaContent() []byte {
	return m.Content
}

// SetMediaContent sets the binary content of the media entity
func (m *MediaItem) SetMediaContent(content []byte) {
	m.Content = content
	size := int64(len(content))
	m.Size = &size
	m.ModifiedAt = time.Now()
}

// GetMediaContentType returns the MIME type of the media content
func (m *MediaItem) GetMediaContentType() string {
	return m.ContentType
}

// SetMediaContentType sets the MIME type of the media content
func (m *MediaItem) SetMediaContentType(contentType string) {
	m.ContentType = contentType
}
