package job

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
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
	return []string{"completed", "failed"}
}

func validate_callback_request(request dto.CallbackRequest, db *Database, str *Storage, ctx context.Context) (CallbackResult, error) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
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
	//A következő ellenőrzésekhez lekérünk egy sort az adatbázisból
	row, err := databaseRetry(ctx, func(_ctx context.Context) (dto.DatabaseRow, error) { return db.GetRow(request.JobID, _ctx) })
	logger.Info("DATA OF THE ROW OF THE CALLBACK", "data", row)
	//Ha nem létezik a kért JobID, az is probléma
	if errors.Is(err, sql.ErrNoRows) {
		return CallbackInvalid, fmt.Errorf("no job found for jobId %s", request.JobID)
	}
	//ha failed a status akkor kell errormessage
	if request.Status == "failed" && request.ErrorMessage == "" {
		return CallbackInvalid, errors.New("error message is mandatory when status is failed")
	}
	if request.Status == "completed" {
		if len(request.Data) == 0 || string(request.Data) == "null" {
			return CallbackInvalid, errors.New("when OCR processing or Layout processing is completed, result data is mandatory")
		}
		//Egyébként meg kell nézni, hogy az írni kivánt data objectben már van-e valami, mert ha igen, akkor asszumálhatjuk az idempotenciát
		switch request.Step {
		case "ocr_processing":
			if string(row.OcrResult) != "{}" {
				return CallbackIgnored, nil
			}
		case "layout_processing":
			if string(row.LayoutResult) != "{}" {
				return CallbackIgnored, nil
			}
		}
	}
	return CallbackProcessed, nil
}
