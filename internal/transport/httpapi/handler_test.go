package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"meta-automation/internal/application/onboarding"
	"meta-automation/internal/domain/job"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type fakeJobReader struct {
	snapshot job.Snapshot
	found    bool
}

func (f fakeJobReader) Get(id string) (job.Snapshot, bool) {
	if !f.found || id != f.snapshot.ID {
		return job.Snapshot{}, false
	}

	return f.snapshot, true
}

type fakeJobSubmitter struct {
	snapshot job.Snapshot
	err      error
}

func (f fakeJobSubmitter) Submit(
	ctx context.Context,
	key, businessID, name string,
) (job.Snapshot, error) {
	return f.snapshot, f.err
}

func TestGetJobReturnsSnapshot(t *testing.T) {
	handler := NewHandler(fakeJobReader{
		found: true,
		snapshot: job.Snapshot{
			ID:         "job-001",
			BusinessID: "123456",
			Name:       "WABA Percobaan",
			Status:     job.StatusQueued,
		},
	}, fakeJobSubmitter{}, "test-key")

	request := httptest.NewRequest(http.MethodGet, "/jobs/job-001", nil)
	request.Header.Set("Authorization", "Bearer test-key")
	request.Header.Set("Idempotency-Key", "request-test-001")
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status HTTP = %d, ingin 200", response.Code)
	}

	var body struct {
		ID     string     `json:"id"`
		Status job.Status `json:"status"`
	}

	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}

	if body.ID != "job-001" || body.Status != job.StatusQueued {
		t.Errorf("respons tidak sesuai: %+v", body)
	}
}

func TestGetJobReturnsNotFound(t *testing.T) {
	handler := NewHandler(fakeJobReader{}, fakeJobSubmitter{}, "test-key")

	request := httptest.NewRequest(http.MethodGet, "/jobs/tidak-ada", nil)
	request.Header.Set("Authorization", "Bearer test-key")
	request.Header.Set("Idempotency-Key", "request-test-001")

	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if response.Code != http.StatusNotFound {
		t.Errorf("status HTTP = %d, ingin 404", response.Code)
	}
}

func TestHealthReturnsOK(t *testing.T) {
	handler := NewHandler(fakeJobReader{}, fakeJobSubmitter{}, "test-key")

	request := httptest.NewRequest(http.MethodGet, "/health", nil)
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf(
			"status HTTP = %d, ingin %d",
			response.Code,
			http.StatusOK,
		)
	}

	if got := response.Header().Get("Content-Type"); got != "application/json" {
		t.Errorf("Content-Type = %q, ingin application/json", got)
	}

	var body struct {
		Status string `json:"status"`
	}

	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatalf("respons bukan JSON valid: %v", err)
	}

	if body.Status != "ok" {
		t.Errorf("status = %q, ingin ok", body.Status)
	}
}

func TestHealthRejectsPost(t *testing.T) {
	handler := NewHandler(fakeJobReader{}, fakeJobSubmitter{}, "test-key")

	request := httptest.NewRequest(http.MethodPost, "/health", nil)
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if response.Code != http.StatusMethodNotAllowed {
		t.Errorf(
			"status HTTP = %d, ingin %d",
			response.Code,
			http.StatusMethodNotAllowed,
		)
	}

	if got := response.Header().Get("Allow"); got != http.MethodGet {
		t.Errorf("Allow = %q, ingin GET", got)
	}
}

func TestGetJobRequiresAPIKey(t *testing.T) {
	tests := []struct {
		name          string
		authorization string
	}{
		{name: "tanpa key"},
		{
			name:          "key salah",
			authorization: "Bearer wrong-key",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := NewHandler(fakeJobReader{}, fakeJobSubmitter{}, "test-key")
			request := httptest.NewRequest(
				http.MethodGet,
				"/jobs/job-001",
				nil,
			)
			request.Header.Set("Idempotency-Key", "request-test-001")
			request.Header.Set("Authorization", tt.authorization)

			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)

			if response.Code != http.StatusUnauthorized {
				t.Errorf(
					"status HTTP = %d, ingin 401",
					response.Code,
				)
			}
		})
	}
}

func TestCreateJobReturnsAccepted(t *testing.T) {
	handler := NewHandler(
		fakeJobReader{},
		fakeJobSubmitter{
			snapshot: job.Snapshot{
				ID:     "job-001",
				Status: job.StatusQueued,
			},
		},
		"test-key",
	)

	request := httptest.NewRequest(
		http.MethodPost,
		"/jobs/waba",
		strings.NewReader(`{
			"business_id": "123456",
			"name": "WABA Percobaan"
		}`),
	)
	request.Header.Set("Idempotency-Key", "request-test-001")
	request.Header.Set("Authorization", "Bearer test-key")
	request.Header.Set("Content-Type", "application/json")

	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)

	if response.Code != http.StatusAccepted {
		t.Fatalf(
			"status HTTP = %d, ingin 202; body = %s",
			response.Code,
			response.Body.String(),
		)
	}

	if got := response.Header().Get("Location"); got != "/jobs/job-001" {
		t.Errorf("Location = %q, ingin /jobs/job-001", got)
	}

	var body struct {
		JobID  string     `json:"job_id"`
		Status job.Status `json:"status"`
	}

	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}

	if body.JobID != "job-001" || body.Status != job.StatusQueued {
		t.Errorf("respons tidak sesuai: %+v", body)
	}
}

func TestCreateJobRejectsInvalidJSON(t *testing.T) {
	tests := []struct {
		name string
		body string
	}{
		{
			name: "JSON rusak",
			body: `{"business_id":`,
		},
		{
			name: "field tidak dikenal",
			body: `{"business_id":"123456","name":"WABA","extra":true}`,
		},
		{
			name: "dua objek JSON",
			body: `{"business_id":"123456","name":"WABA"} {}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := NewHandler(
				fakeJobReader{},
				fakeJobSubmitter{},
				"test-key",
			)

			request := httptest.NewRequest(
				http.MethodPost,
				"/jobs/waba",
				strings.NewReader(tt.body),
			)
			request.Header.Set("Idempotency-Key", "request-test-001")
			request.Header.Set("Authorization", "Bearer test-key")

			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)

			if response.Code != http.StatusBadRequest {
				t.Errorf(
					"status HTTP = %d, ingin 400",
					response.Code,
				)
			}
		})
	}
}

func TestCreateJobMapsApplicationErrors(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		wantStatus int
	}{
		{
			name: "validasi gagal",
			err: &job.ValidationError{
				Field:   "name",
				Message: "wajib diisi",
			},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "antrean penuh",
			err:        onboarding.ErrQueueFull,
			wantStatus: http.StatusServiceUnavailable,
		},
		{
			name:       "antrean ditutup",
			err:        onboarding.ErrQueueClosed,
			wantStatus: http.StatusServiceUnavailable,
		},
		{
			name:       "error internal",
			err:        errors.New("detail internal rahasia"),
			wantStatus: http.StatusInternalServerError,
		},
		{
			name:       "idempotency key invalid",
			err:        onboarding.ErrInvalidIdempotencyKey,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "idempotency key konflik",
			err:        onboarding.ErrIdempotencyConflict,
			wantStatus: http.StatusConflict,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := NewHandler(
				fakeJobReader{},
				fakeJobSubmitter{err: tt.err},
				"test-key",
			)

			request := httptest.NewRequest(
				http.MethodPost,
				"/jobs/waba",
				strings.NewReader(
					`{"business_id":"123456","name":"WABA"}`,
				),
			)
			request.Header.Set("Idempotency-Key", "request-test-001")
			request.Header.Set("Authorization", "Bearer test-key")

			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)

			if response.Code != tt.wantStatus {
				t.Errorf(
					"status HTTP = %d, ingin %d",
					response.Code,
					tt.wantStatus,
				)
			}

			if strings.Contains(
				response.Body.String(),
				"detail internal rahasia",
			) {
				t.Error("detail error internal bocor ke respons")
			}
		})
	}
}

type countingSubmission struct {
	calls int
}

func (s *countingSubmission) Submit(
	ctx context.Context,
	businessID, name string,
) (job.Snapshot, error) {
	s.calls++

	return job.Snapshot{
		ID:         "job-001",
		BusinessID: businessID,
		Name:       name,
		Status:     job.StatusQueued,
	}, nil
}

type countingJobSubmitter struct {
	calls int
}

func (s *countingJobSubmitter) Submit(
	ctx context.Context,
	key, businessID, name string,
) (job.Snapshot, error) {
	s.calls++

	return job.Snapshot{
		ID:     "job-001",
		Status: job.StatusQueued,
	}, nil
}

func TestCreateJobWithoutAPIKeyDoesNotSubmit(t *testing.T) {
	submitter := &countingJobSubmitter{}

	handler := NewHandler(
		fakeJobReader{},
		submitter,
		"test-key",
	)

	request := httptest.NewRequest(
		http.MethodPost,
		"/jobs/waba",
		strings.NewReader(
			`{"business_id":"123456","name":"WABA"}`,
		),
	)

	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)

	if response.Code != http.StatusUnauthorized {
		t.Errorf("status HTTP = %d, ingin 401", response.Code)
	}

	if submitter.calls != 0 {
		t.Error("request tanpa API key tidak boleh membuat Job")
	}
}

func TestCreateJobReusesIDAndRejectsConflictingInput(t *testing.T) {
	next := &countingSubmission{}
	submitter := onboarding.NewIdempotentSubmitter(next)

	handler := NewHandler(
		fakeJobReader{},
		submitter,
		"test-key",
	)

	send := func(body string) *httptest.ResponseRecorder {
		request := httptest.NewRequest(
			http.MethodPost,
			"/jobs/waba",
			strings.NewReader(body),
		)
		request.Header.Set("Authorization", "Bearer test-key")
		request.Header.Set("Idempotency-Key", "request-001")
		request.Header.Set("Content-Type", "application/json")

		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)

		return response
	}

	body := `{"business_id":"123456","name":"WABA Pertama"}`

	first := send(body)
	second := send(body)

	if first.Code != http.StatusAccepted {
		t.Fatalf("request pertama = %d, ingin 202", first.Code)
	}

	if second.Code != http.StatusAccepted {
		t.Fatalf("retry = %d, ingin 202", second.Code)
	}

	var firstBody, secondBody struct {
		JobID string `json:"job_id"`
	}

	if err := json.Unmarshal(first.Body.Bytes(), &firstBody); err != nil {
		t.Fatal(err)
	}

	if err := json.Unmarshal(second.Body.Bytes(), &secondBody); err != nil {
		t.Fatal(err)
	}

	if firstBody.JobID == "" || firstBody.JobID != secondBody.JobID {
		t.Error("retry harus mengembalikan ID yang sama dan tidak kosong")
	}

	conflict := send(
		`{"business_id":"123456","name":"WABA Berbeda"}`,
	)

	if conflict.Code != http.StatusConflict {
		t.Errorf("request konflik = %d, ingin 409", conflict.Code)
	}

	if next.calls != 1 {
		t.Errorf("application dipanggil %d kali, ingin 1", next.calls)
	}
}

func TestCreateJobRequiresIdempotencyKey(t *testing.T) {
	next := &countingSubmission{}
	submitter := onboarding.NewIdempotentSubmitter(next)

	handler := NewHandler(
		fakeJobReader{},
		submitter,
		"test-key",
	)

	request := httptest.NewRequest(
		http.MethodPost,
		"/jobs/waba",
		strings.NewReader(
			`{"business_id":"123456","name":"WABA Percobaan"}`,
		),
	)
	request.Header.Set("Authorization", "Bearer test-key")

	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)

	if response.Code != http.StatusBadRequest {
		t.Errorf("status HTTP = %d, ingin 400", response.Code)
	}

	if next.calls != 0 {
		t.Error("request tanpa Idempotency-Key tidak boleh membuat Job")
	}
}
