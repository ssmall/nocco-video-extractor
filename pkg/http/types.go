package http

// ExtractionRequest represents the body of a request to the ClipExtractionHandler
type ExtractionRequest struct {
	SourceFileID        string `json:"sourceFileId"`
	ClipStartTime       string `json:"clipStartTime"`
	ClipEndTime         string `json:"clipEndTime"`
	DestinationFolderID string `json:"destinationFolderId"`
}

// ExtractionResponse represents the success response for the ClipExtractionHandler
type ExtractionResponse struct {
	FileURL string `json:"fileUrl"`
}
