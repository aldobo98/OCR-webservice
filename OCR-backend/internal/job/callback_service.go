package job

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/aldobo98/OCR-backend/dto"
)

func validate_callback_request(request dto.CallbackRequest, db *Database, str *Storage, ctx context.Context) error {
	//jobID, step és status kötelező minden esetben
	if request.JobID == "" || request.Step == "" || request.Status == "" {
		return errors.New("callback request should contain both jobID, step and status")
	}
	//ha failed a status akkor kell errormessage
	if request.Status == "failed" && request.ErrorMessage == "" {
		return errors.New("error message is mandatory when status is failed")
	}
	//a preprocess nem fog completed-et visszaküldeni
	if request.Status == "completed" {
		if request.Data == "" {
			return errors.New("when OCR processing or Layout processing is completed, result data is mandatory")
		}
	}
	//ha van objectRequest
	if request.Requests != (dto.ObjectRequest{}) {
		//ha kérünk olvasási hozzáférést, akkor az objektumnak léteznie kell
		re := request.Requests.Read
		if re != "" {
			if re != "original" && re != "preprocess" {
				return errors.New("read requests should be either for original or for preprocess object")
			}
			objectKey := fmt.Sprintf("%s_object_key", re)
			ret, err := queryJobValue(ctx, func(_ctx context.Context) (string, error) {
				return db.GetObjectAtKey(request.JobID, objectKey, _ctx)
			})
			if err != nil {
				return errors.New("failed to query database for requested read object")
			}
			if ret == "" {
				return errors.New("the requested read object doesn't exist in database")
			}
			//Ha van az adatbázisban entry, akkor léteznie is kell az object storage-ban
			existence, err := str.ObjectExists(ctx, ret)
			if err != nil {
				return errors.New("failed to validate read object existence in object storage")
			}
			if !existence {
				return errors.New("the requested object doesn't exist in object storage")
			}
		}
		wr := request.Requests.Write
		if wr != "" {
			//Egyenlőre csak a preprocess írását engedem
			if wr != "preprocess" {
				return errors.New("the requested object cannot be written")
			}
			//Ha létezik az adatbázisban, akkor már valaki írta -> nem írjuk felül
			ret, err := queryJobValue(ctx, func(_ctx context.Context) (string, error) {
				return db.GetObjectAtKey(request.JobID, "preprocess_object_key", _ctx)
			})
			if err != nil {
				return errors.New("failed to query database for requested write object")
			}
			if ret != "" {
				return errors.New("the object requested to write already exists, writing it again is forbidden")
			}
		}
	}
	return nil
}

func write_callback_response(response dto.CallbackResponse, w *http.ResponseWriter, logger *slog.Logger) {
	data, err := json.Marshal(response)
	if err != nil {
		logger.Error("Failed to marshal response", "error", err)
		return
	}
	_, err = (*w).Write(data)
	if err != nil {
		logger.Error("Failed to write response data", "error", err)
		(*w).WriteHeader(http.StatusInternalServerError)
	}
}
