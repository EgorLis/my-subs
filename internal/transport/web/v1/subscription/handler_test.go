package subscription

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/EgorLis/my-subs/internal/domain"
	mockrepo "github.com/EgorLis/my-subs/internal/infra/database/mock"
	"github.com/google/uuid"
)

// ---------- helpers ----------

func newHandler(repo domain.SubscriptionRepository) *Handler {
	return &Handler{
		Log:  log.New(io.Discard, "", 0),
		Repo: repo,
	}
}

func mustJSON(v any) *bytes.Reader {
	b, _ := json.Marshal(v)
	return bytes.NewReader(b)
}

func ym(mm, yyyy int) YearMonth {
	return YearMonth(time.Date(yyyy, time.Month(mm), 1, 0, 0, 0, 0, time.UTC))
}

func readErrorStr(t *testing.T, body []byte) string {
	t.Helper()
	var m map[string]string
	_ = json.Unmarshal(body, &m)
	return m["error"]
}

// ---------- repos for failure simulation ----------

type timeoutRepo struct{ domain.SubscriptionRepository }

func (timeoutRepo) AddSub(ctx context.Context, sub domain.Subscription) (domain.Subscription, error) {
	return domain.Subscription{}, context.DeadlineExceeded
}
func (timeoutRepo) GetSub(ctx context.Context, id string) (domain.Subscription, error) {
	return domain.Subscription{}, context.DeadlineExceeded
}
func (timeoutRepo) UpdateSub(ctx context.Context, sub domain.Subscription) error {
	return context.DeadlineExceeded
}
func (timeoutRepo) DeleteSub(ctx context.Context, id string) error {
	return context.DeadlineExceeded
}
func (timeoutRepo) ListSubs(ctx context.Context) ([]domain.Subscription, error) {
	return nil, context.DeadlineExceeded
}
func (timeoutRepo) TotalCost(ctx context.Context, _ string, _ string, _, _ time.Time) (int, error) {
	return 0, context.DeadlineExceeded
}

type internalErrRepo struct{ domain.SubscriptionRepository }

var errInternal = errors.New("boom")

func (internalErrRepo) AddSub(ctx context.Context, sub domain.Subscription) (domain.Subscription, error) {
	return domain.Subscription{}, errInternal
}
func (internalErrRepo) GetSub(ctx context.Context, id string) (domain.Subscription, error) {
	return domain.Subscription{}, errInternal
}
func (internalErrRepo) UpdateSub(ctx context.Context, sub domain.Subscription) error {
	return errInternal
}
func (internalErrRepo) DeleteSub(ctx context.Context, id string) error {
	return errInternal
}
func (internalErrRepo) ListSubs(ctx context.Context) ([]domain.Subscription, error) {
	return nil, errInternal
}
func (internalErrRepo) TotalCost(ctx context.Context, _ string, _ string, _, _ time.Time) (int, error) {
	return 0, errInternal
}

// ---------- CREATE ----------

func TestCreate_Various(t *testing.T) {
	okUser := uuid.NewString()

	cases := []struct {
		name       string
		repo       domain.SubscriptionRepository
		body       any
		wantCode   int
		wantInBody string // contains
	}{
		{
			name:     "OK",
			repo:     mockrepo.NewMockRepo(),
			body:     CreateRequest{ServiceName: "Yandex Plus", Price: 400, UserID: okUser, StartDate: ym(7, 2025), EndDate: ym(7, 2026)},
			wantCode: http.StatusOK,
		},
		{
			name:       "BadJSON",
			repo:       mockrepo.NewMockRepo(),
			body:       rawJSON("{bad"),
			wantCode:   http.StatusBadRequest,
			wantInBody: "invalid JSON",
		},
		{
			name:       "Validation_MissingServiceName",
			repo:       mockrepo.NewMockRepo(),
			body:       CreateRequest{ServiceName: "", Price: 1, UserID: okUser, StartDate: ym(7, 2025), EndDate: ym(8, 2025)},
			wantCode:   http.StatusBadRequest,
			wantInBody: "service_name",
		},
		{
			name:       "Validation_NegativePrice",
			repo:       mockrepo.NewMockRepo(),
			body:       CreateRequest{ServiceName: "A", Price: -1, UserID: okUser, StartDate: ym(7, 2025), EndDate: ym(8, 2025)},
			wantCode:   http.StatusBadRequest,
			wantInBody: "price",
		},
		{
			name:       "Validation_BadGUID",
			repo:       mockrepo.NewMockRepo(),
			body:       CreateRequest{ServiceName: "A", Price: 1, UserID: "not-a-guid", StartDate: ym(7, 2025), EndDate: ym(8, 2025)},
			wantCode:   http.StatusBadRequest,
			wantInBody: "user_id",
		},
		{
			name:       "Validation_StartAfterEnd",
			repo:       mockrepo.NewMockRepo(),
			body:       CreateRequest{ServiceName: "A", Price: 1, UserID: okUser, StartDate: ym(9, 2025), EndDate: ym(8, 2025)},
			wantCode:   http.StatusBadRequest,
			wantInBody: "date range",
		},
		{
			name:       "Timeout",
			repo:       timeoutRepo{},
			body:       CreateRequest{ServiceName: "A", Price: 1, UserID: okUser, StartDate: ym(7, 2025), EndDate: ym(8, 2025)},
			wantCode:   http.StatusGatewayTimeout,
			wantInBody: "timed out",
		},
		{
			name:     "InternalError",
			repo:     internalErrRepo{},
			body:     CreateRequest{ServiceName: "A", Price: 1, UserID: okUser, StartDate: ym(7, 2025), EndDate: ym(8, 2025)},
			wantCode: http.StatusInternalServerError,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			h := newHandler(tc.repo)
			var r *http.Request
			switch b := tc.body.(type) {
			case rawJSON:
				r = httptest.NewRequest(http.MethodPost, "/v1/subscriptions", bytes.NewBufferString(string(b)))
			default:
				r = httptest.NewRequest(http.MethodPost, "/v1/subscriptions", mustJSON(tc.body))
			}
			w := httptest.NewRecorder()
			h.Create(w, r)

			if w.Code != tc.wantCode {
				t.Fatalf("want %d, got %d. body=%s", tc.wantCode, w.Code, w.Body.String())
			}
			if tc.wantInBody != "" && !strings.Contains(w.Body.String(), tc.wantInBody) {
				t.Fatalf("body should contain %q, got %s", tc.wantInBody, w.Body.String())
			}
		})
	}
}

type rawJSON string

// ---------- GET ----------

func TestGet_Various(t *testing.T) {
	repo := mockrepo.NewMockRepo()

	// prepare one
	sub, _ := repo.AddSub(context.Background(), domain.Subscription{
		ServiceName: "Netflix", Price: 500, UserID: uuid.NewString(),
		StartDate: time.Date(2025, 7, 1, 0, 0, 0, 0, time.UTC),
		EndDate:   time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC),
	})

	cases := []struct {
		name     string
		repo     domain.SubscriptionRepository
		id       string
		wantCode int
	}{
		{"OK", repo, sub.ID, http.StatusOK},
		{"BadGUID", repo, "bad-guid", http.StatusBadRequest},
		{"NotFound", repo, uuid.NewString(), http.StatusNotFound},
		{"Timeout", timeoutRepo{}, uuid.NewString(), http.StatusGatewayTimeout},
		{"Internal", internalErrRepo{}, uuid.NewString(), http.StatusInternalServerError},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			h := newHandler(tc.repo)

			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodGet, "/v1/subscriptions/"+tc.id, nil)
			r.SetPathValue("id", tc.id)

			h.Get(w, r)

			if w.Code != tc.wantCode {
				t.Fatalf("want %d, got %d. body=%s", tc.wantCode, w.Code, w.Body.String())
			}
		})
	}
}

// ---------- UPDATE ----------

func TestUpdate_Various(t *testing.T) {
	baseRepo := mockrepo.NewMockRepo()
	base, _ := baseRepo.AddSub(context.Background(), domain.Subscription{
		ServiceName: "Spotify", Price: 300, UserID: uuid.NewString(),
		StartDate: time.Date(2025, 7, 1, 0, 0, 0, 0, time.UTC),
		EndDate:   time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC),
	})

	okReq := UpdateRequest{
		ID: base.ID, ServiceName: "Spotify", Price: 450,
		UserID: base.UserID, StartDate: YearMonth(base.StartDate), EndDate: YearMonth(base.EndDate),
	}

	cases := []struct {
		name       string
		repo       domain.SubscriptionRepository
		body       any
		wantCode   int
		wantInBody string
	}{
		{"OK", baseRepo, okReq, http.StatusOK, ""},
		{"BadJSON", baseRepo, rawJSON("{bad"), http.StatusBadRequest, "invalid JSON"},
		{"Validation_BadID", baseRepo, func() UpdateRequest { x := okReq; x.ID = "bad"; return x }(), http.StatusBadRequest, "id"},
		{"Validation_BadUser", baseRepo, func() UpdateRequest { x := okReq; x.UserID = "bad"; return x }(), http.StatusBadRequest, "user_id"},
		{"Validation_NegativePrice", baseRepo, func() UpdateRequest { x := okReq; x.Price = -1; return x }(), http.StatusBadRequest, "price"},
		{"Validation_StartAfterEnd", baseRepo, func() UpdateRequest { x := okReq; x.StartDate = ym(9, 2025); x.EndDate = ym(8, 2025); return x }(), http.StatusBadRequest, "date range"},
		{"NotFound", baseRepo, func() UpdateRequest { x := okReq; x.ID = uuid.NewString(); return x }(), http.StatusNotFound, ""},
		{"Timeout", timeoutRepo{}, okReq, http.StatusGatewayTimeout, ""},
		{"Internal", internalErrRepo{}, okReq, http.StatusInternalServerError, ""},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			h := newHandler(tc.repo)

			var r *http.Request
			switch b := tc.body.(type) {
			case rawJSON:
				r = httptest.NewRequest(http.MethodPut, "/v1/subscriptions/"+base.ID, bytes.NewBufferString(string(b)))
			default:
				r = httptest.NewRequest(http.MethodPut, "/v1/subscriptions/"+base.ID, mustJSON(tc.body))
			}
			r.SetPathValue("id", base.ID)

			w := httptest.NewRecorder()
			h.Update(w, r)

			if w.Code != tc.wantCode {
				t.Fatalf("want %d, got %d. body=%s", tc.wantCode, w.Code, w.Body.String())
			}
			if tc.wantInBody != "" && !strings.Contains(w.Body.String(), tc.wantInBody) {
				t.Fatalf("body should contain %q, got %s", tc.wantInBody, w.Body.String())
			}
		})
	}
}

// ---------- DELETE ----------

func TestDelete_Various(t *testing.T) {
	repo := mockrepo.NewMockRepo()
	sub, _ := repo.AddSub(context.Background(), domain.Subscription{
		ServiceName: "YouTube", Price: 199, UserID: uuid.NewString(),
		StartDate: time.Date(2025, 7, 1, 0, 0, 0, 0, time.UTC),
		EndDate:   time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC),
	})

	cases := []struct {
		name     string
		repo     domain.SubscriptionRepository
		id       string
		wantCode int
	}{
		{"OK", repo, sub.ID, http.StatusOK},
		{"BadGUID", repo, "bad-guid", http.StatusBadRequest},
		{"NotFound", repo, uuid.NewString(), http.StatusNotFound},
		{"Timeout", timeoutRepo{}, uuid.NewString(), http.StatusGatewayTimeout},
		{"Internal", internalErrRepo{}, uuid.NewString(), http.StatusInternalServerError},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			h := newHandler(tc.repo)
			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodDelete, "/v1/subscriptions/"+tc.id, nil)
			r.SetPathValue("id", tc.id)

			h.Delete(w, r)

			if w.Code != tc.wantCode {
				t.Fatalf("want %d, got %d. body=%s", tc.wantCode, w.Code, w.Body.String())
			}
		})
	}
}

// ---------- LIST ----------

func TestList_Various(t *testing.T) {
	okRepo := mockrepo.NewMockRepo()
	_, _ = okRepo.AddSub(context.Background(), domain.Subscription{
		ServiceName: "A", Price: 1, UserID: uuid.NewString(),
		StartDate: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
		EndDate:   time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC),
	})
	_, _ = okRepo.AddSub(context.Background(), domain.Subscription{
		ServiceName: "B", Price: 2, UserID: uuid.NewString(),
		StartDate: time.Date(2025, 2, 1, 0, 0, 0, 0, time.UTC),
		EndDate:   time.Date(2025, 7, 1, 0, 0, 0, 0, time.UTC),
	})

	cases := []struct {
		name     string
		repo     domain.SubscriptionRepository
		wantCode int
		wantLen  int // expected number of items (if 200)
	}{
		{"OK", okRepo, http.StatusOK, 2},
		{"Timeout", timeoutRepo{}, http.StatusGatewayTimeout, 0},
		{"Internal", internalErrRepo{}, http.StatusInternalServerError, 0},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			h := newHandler(tc.repo)
			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodGet, "/v1/subscriptions", nil)

			h.List(w, r)

			if w.Code != tc.wantCode {
				t.Fatalf("want %d, got %d. body=%s", tc.wantCode, w.Code, w.Body.String())
			}
			if tc.wantCode == http.StatusOK {
				var resp ListResponse
				_ = json.Unmarshal(w.Body.Bytes(), &resp)
				if len(resp.Subs) != tc.wantLen {
					t.Fatalf("want %d items, got %d", tc.wantLen, len(resp.Subs))
				}
			}
		})
	}
}

// ---------- TOTALCOST ----------

func TestTotalCost_Various(t *testing.T) {
	userID := uuid.NewString()

	okRepo := mockrepo.NewMockRepo()
	// подходящие
	_, _ = okRepo.AddSub(context.Background(), domain.Subscription{
		ServiceName: "Yandex Plus", Price: 400, UserID: userID,
		StartDate: time.Date(2025, 7, 1, 0, 0, 0, 0, time.UTC),
		EndDate:   time.Date(2025, 8, 1, 0, 0, 0, 0, time.UTC),
	})
	_, _ = okRepo.AddSub(context.Background(), domain.Subscription{
		ServiceName: "Yandex Plus", Price: 300, UserID: userID,
		StartDate: time.Date(2025, 7, 15, 0, 0, 0, 0, time.UTC),
		EndDate:   time.Date(2025, 8, 15, 0, 0, 0, 0, time.UTC),
	})

	cases := []struct {
		name       string
		repo       domain.SubscriptionRepository
		query      string
		wantCode   int
		wantInBody string
	}{
		{
			name:     "OK",
			repo:     okRepo,
			query:    "user_id=" + userID + "&service_name=Yandex%20Plus&from=07-2025&to=08-2025",
			wantCode: http.StatusOK,
		},
		{
			name:       "BadUserID",
			repo:       okRepo,
			query:      "user_id=bad-guid&service_name=Yandex%20Plus&from=07-2025&to=08-2025",
			wantCode:   http.StatusBadRequest,
			wantInBody: "user_id",
		},
		{
			name:       "BadFromFormat",
			repo:       okRepo,
			query:      "user_id=" + userID + "&service_name=Yandex%20Plus&from=bad&to=08-2025",
			wantCode:   http.StatusBadRequest,
			wantInBody: "from",
		},
		{
			name:       "BadToFormat",
			repo:       okRepo,
			query:      "user_id=" + userID + "&service_name=Yandex%20Plus&from=07-2025&to=bad",
			wantCode:   http.StatusBadRequest,
			wantInBody: "to",
		},
		{
			name:       "StartAfterEnd",
			repo:       okRepo,
			query:      "user_id=" + userID + "&service_name=Yandex%20Plus&from=09-2025&to=08-2025",
			wantCode:   http.StatusBadRequest,
			wantInBody: "date range",
		},
		{
			name:       "Timeout",
			repo:       timeoutRepo{},
			query:      "user_id=" + userID + "&service_name=Yandex%20Plus&from=07-2025&to=08-2025",
			wantCode:   http.StatusGatewayTimeout,
			wantInBody: "timed out",
		},
		{
			name:     "Internal",
			repo:     internalErrRepo{},
			query:    "user_id=" + userID + "&service_name=Yandex%20Plus&from=07-2025&to=08-2025",
			wantCode: http.StatusInternalServerError,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			h := newHandler(tc.repo)

			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodGet, "/v1/subscriptions/totalcost?"+tc.query, nil)

			h.TotalCost(w, r)

			if w.Code != tc.wantCode {
				t.Fatalf("want %d, got %d. body=%s", tc.wantCode, w.Code, w.Body.String())
			}
			if tc.wantInBody != "" && !strings.Contains(w.Body.String(), tc.wantInBody) {
				t.Fatalf("body should contain %q, got %s", tc.wantInBody, w.Body.String())
			}
		})
	}
}
