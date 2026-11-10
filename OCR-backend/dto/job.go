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

type GetResultsResponse struct {
	JobID    string                 `json:"jobId"`
	Download ObjectDownloadResponse `json:"download"`
	Results  string                 `json:"text"`
	//Majd ha lesz layout eredmény, akkor kitalálom, mit küldök vissza
}

// A knative függvények ezen a requesten keresztül kommunikálnak a backenddel
type CallbackRequest struct {
	JobID        string          `json:"jobId"`
	Step         string          `json:"step"`
	Status       string          `json:"status"`
	Data         json.RawMessage `json:"data"`
	ErrorMessage string          `json:"error"`
}

type DatabaseRow struct {
	Status       string
	OrigFileExt  string
	DocumentType string
	PrepAttempt  int
	OrigKey      string
	PrepKey      string
	OcrResult    json.RawMessage
	LayoutResult json.RawMessage
	ErrorMessage string
}

type ServerSentEvent struct {
	Step   string `json:"step"`
	Status string `json:"status"`
}

type ServerSentDone struct {
	JobID  string `json:"jobId"`
	Status string `json:"status"`
}

type OCRRow struct {
	BoundingBox [4][2]int `json:"bounding_box"`
	Text        string    `json:"text"`
	Confidence  float64   `json:"confidence"`
}

type OCRResponse struct {
	Results []OCRRow `json:"results"`
}
