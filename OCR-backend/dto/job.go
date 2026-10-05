package dto

//API formátumok gyűjtőhelye

type CreateJobRequest struct {
	ContentType string `json:"contentType"`
}

type CreateJobUpload struct {
	URL    string            `json:"url"`
	Fields map[string]string `json:"fields"`
}

type CreateJobResponse struct {
	JobID  string          `json:"jobId"`
	Upload CreateJobUpload `json:"upload"`
}
