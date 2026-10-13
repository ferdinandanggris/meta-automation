package httpapi

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"io"
	"meta-automation/internal/application/onboarding"
	"meta-automation/internal/domain/job"
	"net/http"
	"strings"
)

type JobReader interface {
	Get(id string) (job.Snapshot, bool)
}

func NewHandler(
	reader JobReader,
	submitter JobSubmitter,
	apiKey string,
) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/health", health)

	mux.Handle(
		"/jobs/waba",
		requireAPIKey(apiKey, createJob(submitter)),
	)

	mux.Handle(
		"/jobs/",
		requireAPIKey(apiKey, getJob(reader)),
	)

	return mux
}

func health(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		http.Error(w, "method tidak diizinkan", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	json.NewEncoder(w).Encode(struct {
		Status string `json:"status"`
	}{
		Status: "ok",
	})
}

func requireAPIKey(apiKey string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		expected := "Bearer " + apiKey
		actual := r.Header.Get("Authorization")

		if apiKey == "" ||
			subtle.ConstantTimeCompare(
				[]byte(actual),
				[]byte(expected),
			) != 1 {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func getJob(reader JobReader) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", http.MethodGet)
			http.Error(w, "method tidak diizinkan", http.StatusMethodNotAllowed)
			return
		}

		id := strings.TrimPrefix(r.URL.Path, "/jobs/")
		if id == "" || strings.Contains(id, "/") {
			http.NotFound(w, r)
			return
		}

		snapshot, found := reader.Get(id)
		if !found {
			http.NotFound(w, r)
			return
		}

		body := struct {
			ID         string     `json:"id"`
			BusinessID string     `json:"business_id"`
			Name       string     `json:"name"`
			Status     job.Status `json:"status"`
		}{
			ID:         snapshot.ID,
			BusinessID: snapshot.BusinessID,
			Name:       snapshot.Name,
			Status:     snapshot.Status,
		}

		w.Header().Set("Content-Type", "application/json")

		if err := json.NewEncoder(w).Encode(body); err != nil {
			return
		}
	}
}

func createJob(submitter JobSubmitter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.Header().Set("Allow", http.MethodPost)
			http.Error(w, "method tidak diizinkan", http.StatusMethodNotAllowed)
			return
		}

		r.Body = http.MaxBytesReader(w, r.Body, 64*1024)

		var input struct {
			BusinessID string `json:"business_id"`
			Name       string `json:"name"`
		}

		decoder := json.NewDecoder(r.Body)
		decoder.DisallowUnknownFields()

		if err := decoder.Decode(&input); err != nil {
			http.Error(w, "body JSON tidak valid", http.StatusBadRequest)
			return
		}

		// Tolak data tambahan setelah objek JSON pertama.
		if err := decoder.Decode(new(any)); err != io.EOF {
			http.Error(w, "body harus berisi satu objek JSON", http.StatusBadRequest)
			return
		}

		snapshot, err := submitter.Submit(
			r.Context(),
			r.Header.Get("Idempotency-Key"),
			input.BusinessID,
			input.Name,
		)
		if err != nil {
			var validationErr *job.ValidationError

			switch {
			case errors.As(err, &validationErr):
				http.Error(w, validationErr.Error(), http.StatusBadRequest)

			case errors.Is(err, onboarding.ErrQueueFull),
				errors.Is(err, onboarding.ErrQueueClosed):
				http.Error(w, "service belum dapat menerima job", http.StatusServiceUnavailable)

			case errors.Is(err, context.Canceled):
				http.Error(w, "permintaan dibatalkan", http.StatusRequestTimeout)

			case errors.Is(err, context.DeadlineExceeded):
				http.Error(w, "permintaan melewati batas waktu", http.StatusRequestTimeout)
			case errors.Is(err, onboarding.ErrInvalidIdempotencyKey):
				http.Error(w, "Idempotency-Key wajib diisi dan maksimal 128 byte", http.StatusBadRequest)

			case errors.Is(err, onboarding.ErrIdempotencyConflict):
				http.Error(w, "Idempotency-Key sudah digunakan dengan input berbeda", http.StatusConflict)

			default:
				http.Error(w, "gagal membuat job", http.StatusInternalServerError)
			}

			return
		}

		body := struct {
			JobID  string     `json:"job_id"`
			Status job.Status `json:"status"`
		}{
			JobID:  snapshot.ID,
			Status: snapshot.Status,
		}

		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Location", "/jobs/"+snapshot.ID)
		w.WriteHeader(http.StatusAccepted)

		if err := json.NewEncoder(w).Encode(body); err != nil {
			return
		}
	}
}

type JobSubmitter interface {
	Submit(
		ctx context.Context,
		key, businessID, name string,
	) (job.Snapshot, error)
}
