package dto

import "encoding/json"

// API formátumok gyűjtőhelye

type CreateJobRequest struct {
	ContentType string `json:"contentType"`
	DocType     string `json:"documentType"`
}

type CreateJobResponse struct {
	JobID  string               `json:"jobId"`
	Upload ObjectUploadResponse `json:"upload"`
}

type ObjectUploadRequest struct {
	Path string `json:"objectPath"`
}

type ObjectUploadResponse struct {
	URL    string            `json:"url"`
	Fields map[string]string `json:"fields"`
}

type ObjectDownloadResponse struct {
	URL    string              `json:"url"`
	Method string              `json:"method"`
	Fields map[string][]string `json:"fields"`
}

type StartCharacterRecognition struct {
	JobID       string                 `json:"jobId"`
	Download    ObjectDownloadResponse `json:"download"`
	CallBackUrl string                 `json:"callbackUrl"`
}

// A knative függvények ezen a requesten keresztül kommunikálnak a backenddel
type CallbackRequest struct {
	JobID        string          `json:"jobId"`
	Step         string          `json:"step"`
	Status       string          `json:"status"`
	Data         json.RawMessage `json:"data"`
	ErrorMessage string          `json:"error"`
}
