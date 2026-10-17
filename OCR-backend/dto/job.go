package dto

// API formátumok gyűjtőhelye

type CreateJobRequest struct {
	ContentType string `json:"contentType"`
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
