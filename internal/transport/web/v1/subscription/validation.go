package subscription

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

// ---- GUID helper ----

func ValidateGUID(id string) error {
	if _, err := uuid.Parse(id); err != nil {
		return fmt.Errorf("must be a valid GUID: %q", id)
	}
	return nil
}

// ---- общие хелперы ----

func isZeroYM(ym YearMonth) bool {
	return time.Time(ym).IsZero()
}

func isStartLessEnd(a, b YearMonth) bool {
	ta, tb := time.Time(a), time.Time(b)
	return tb.After(ta)
}

// аккумулируем ошибки в один error
func joinErrs(errs []string) error {
	if len(errs) == 0 {
		return nil
	}
	return errors.New(strings.Join(errs, "; "))
}

// ---- ВАЛИДАТОРЫ ЗАПРОСОВ ----

func ValidateCreateRequest(req CreateRequest) error {
	var errs []string

	if strings.TrimSpace(req.ServiceName) == "" {
		errs = append(errs, "service_name: required")
	}
	if req.Price < 0 {
		errs = append(errs, "price: must be >= 0")
	}
	if err := ValidateGUID(req.UserID); err != nil {
		errs = append(errs, "user_id: "+err.Error())
	}
	if isZeroYM(req.StartDate) {
		errs = append(errs, "start_date: required (MM-YYYY)")
	}
	if isZeroYM(req.EndDate) {
		errs = append(errs, "end_date: required (MM-YYYY)")
	}

	if !isZeroYM(req.StartDate) && !isZeroYM(req.EndDate) && !isStartLessEnd(req.StartDate, req.EndDate) {
		errs = append(errs, "date range: start_date must be <= end_date")
	}

	return joinErrs(errs)
}

func ValidateUpdateRequest(req UpdateRequest) error {
	var errs []string

	if err := ValidateGUID(req.ID); err != nil {
		errs = append(errs, "id: "+err.Error())
	}
	if strings.TrimSpace(req.ServiceName) == "" {
		errs = append(errs, "service_name: required")
	}
	if req.Price < 0 {
		errs = append(errs, "price: must be >= 0")
	}
	if err := ValidateGUID(req.UserID); err != nil {
		errs = append(errs, "user_id: "+err.Error())
	}
	if isZeroYM(req.StartDate) {
		errs = append(errs, "start_date: required (MM-YYYY)")
	}
	if isZeroYM(req.EndDate) {
		errs = append(errs, "end_date: required (MM-YYYY)")
	}
	if !isZeroYM(req.StartDate) && !isZeroYM(req.EndDate) && !isStartLessEnd(req.StartDate, req.EndDate) {
		errs = append(errs, "date range: start_date must be <= end_date")
	}

	return joinErrs(errs)
}
