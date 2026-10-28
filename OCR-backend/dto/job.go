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

// A knative függvények ezen a requesten keresztül fognak dolgokat kérni a backendtől
type CallbackRequest struct {
	JobID        string        `json:"jobId"`
	Step         string        `json:"step"`
	Status       string        `json:"status"`
	Data         string        `json:"data"`
	Requests     ObjectRequest `json:"requests"`
	ErrorMessage string        `json:"error"`
}

type ObjectRequest struct {
	Read  string `json:"read"`
	Write string `json:"write"`
}

// Válasz a knative függvényeknek, hogyha kell visszaküldeni vmit (presigned URL-eket) a statuscode-n kívül
type CallbackResponse struct {
	Read    ObjectDownloadResponse `json:"read"`
	Write   ObjectUploadResponse   `json:"write"`
	Message string                 `json:"message"`
}
