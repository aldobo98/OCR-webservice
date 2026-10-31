package job

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"slices"

	"github.com/aldobo98/OCR-backend/dto"
)

type CallbackResult int

const (
	CallbackProcessed CallbackResult = iota //200OK
	CallbackInvalid                         //BadRequest
	Callback404                             //Not found
	CallbackError                           //InternalServerError
	CallbackIgnored                         //idempotencia, 200OK
)

func convertToReturnCode(callres CallbackResult) int {
	switch callres {
	case CallbackProcessed:
		return http.StatusOK
	case CallbackInvalid:
		return http.StatusBadRequest
	case Callback404:
		return http.StatusNotFound
	case CallbackError:
		return http.StatusInternalServerError
	case CallbackIgnored:
		return http.StatusOK
	default:
		return http.StatusInternalServerError
	}
}

func getValidSteps() []string {
	return []string{"ocr_processing", "preprocessing", "layout_processing"}
}

func getValidStatuses() []string {
	return []string{"starting", "completed", "failed"}
}

func validate_callback_request(request dto.CallbackRequest, db *Database, str *Storage, ctx context.Context) (CallbackResult, error) {
	//jobID, step és status kötelező minden esetben
	if request.JobID == "" || request.Step == "" || request.Status == "" {
		return CallbackInvalid, errors.New("callback request should contain both jobID, step and status")
	}

	//A step és a status csak bizonyos érték lehet
	if slices.Index(getValidSteps(), request.Step) == -1 {
		return CallbackInvalid, fmt.Errorf("invalid step in callback request: %s, must be one of %s", request.Step, getValidSteps())
	}

	if slices.Index(getValidStatuses(), request.Status) == -1 {
		return CallbackInvalid, fmt.Errorf("invalid status in callback request: %s, must be one of %s", request.Status, getValidStatuses())
	}
	//Ha nem létezik a kért JobID, az is probléma
	existence, err := databaseRetry(ctx, func(_ctx context.Context) (bool, error) { return db.RecordExists(request.JobID, _ctx) })
	if err != nil {
		return CallbackError, fmt.Errorf("error while walidating record existence: %s", err)
	}
	if !existence {
		return Callback404, fmt.Errorf("the requested resource with id %s doesn't exist", request.JobID)
	}

	//ha failed a status akkor kell errormessage
	if request.Status == "failed" && request.ErrorMessage == "" {
		return CallbackInvalid, errors.New("error message is mandatory when status is failed")
	}
	//a preprocess nem fog completed-et visszaküldeni
	if request.Status == "completed" {
		//preprocessing nem küld vissza completed statust
		if request.Step == "preprocessing" {
			return CallbackInvalid, errors.New("preprocessing is not supposed to send back results")
		}
		if len(request.Data) == 0 || string(request.Data) == "null" {
			return CallbackInvalid, errors.New("when OCR processing or Layout processing is completed, result data is mandatory")
		}
		//Egyébként meg kell nézni, hogy az írni kivánt data objectben már van-e valami, mert ha igen, akkor asszumálhatjuk az idempotenciát
		var key string
		switch request.Step {
		case "ocr_request":
			key = "ocr_results"
		case "layout_processing":
			key = "layout_results"
		}
		existingData, err := databaseRetry(ctx, func(_ctx context.Context) (string, error) { return db.GetObjectAtKey(request.JobID, key, _ctx) })
		if err != nil {
			return CallbackError, fmt.Errorf("failed to validate existing data for record id %s, error %s", request.JobID, err)
		}
		if existingData != "" {
			//Ez lehet idempotencia, elfogadjuk azt csokolom
			return CallbackIgnored, nil
		}
	}
	//ha van objectRequest
	if request.Requests != (dto.ObjectRequest{}) {
		//ha kérünk olvasási hozzáférést, akkor az objektumnak léteznie kell
		re := request.Requests.Read
		if re != "" {
			if re != "original" && re != "preprocess" {
				return CallbackInvalid, errors.New("read requests should be either for original or for preprocess object")
			}
			objectKey := fmt.Sprintf("%s_object_key", re)
			ret, err := databaseRetry(ctx, func(_ctx context.Context) (string, error) {
				return db.GetObjectAtKey(request.JobID, objectKey, _ctx)
			})
			if err != nil {
				return CallbackError, errors.New("failed to query database for requested read object")
			}
			if ret == "" {
				return CallbackError, errors.New("the requested read object doesn't exist in database")
			}
			//Ha van az adatbázisban entry, akkor léteznie is kell az object storage-ban
			existence, err := str.ObjectExists(ctx, ret)
			if err != nil {
				return CallbackError, errors.New("failed to validate read object existence in object storage")
			}
			if !existence {
				return CallbackError, errors.New("the requested object doesn't exist in object storage")
			}
		}
		wr := request.Requests.Write
		if wr != "" {
			//Egyenlőre csak a preprocess írását engedem
			if wr != "preprocess" {
				return CallbackError, errors.New("the requested object cannot be written")
			}
			//Ha létezik az adatbázisban, akkor már valaki írta -> nem írjuk felül
			ret, err := databaseRetry(ctx, func(_ctx context.Context) (string, error) {
				return db.GetObjectAtKey(request.JobID, "preprocess_object_key", _ctx)
			})
			if err != nil {
				return CallbackError, errors.New("failed to query database for requested write object")
			}
			if ret != "" {
				//Ha már valaki írta, akkor idempotencia lehet, nem módosítjuk, csak elfogadjuk
				return CallbackIgnored, nil
			}
		}
	}
	return CallbackProcessed, nil
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
