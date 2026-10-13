package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"meta-automation/internal/application/onboarding"
	"meta-automation/internal/infrastructure/memory"
	"meta-automation/internal/infrastructure/random"
	"meta-automation/internal/transport/httpapi"
)

type demoBrowser struct{}

func (demoBrowser) PrepareWABA(
	ctx context.Context,
	businessID, name string,
) error {
	return nil
}

func main() {
	apiKey := os.Getenv("META_AUTOMATION_API_KEY")
	if apiKey == "" {
		log.Fatal("META_AUTOMATION_API_KEY wajib diisi")
	}

	store := memory.NewJobStore()
	ids := random.NewIDGenerator()
	service := onboarding.NewService(demoBrowser{}, store, ids)
	queue := onboarding.NewQueue(service)

	task, err := service.Create("123456", "WABA Percobaan")
	if err != nil {
		log.Fatal(err)
	}

	if err := queue.Enqueue(context.Background(), task); err != nil {
		log.Fatal(err)
	}

	log.Printf("Job demo dibuat: %s", task.ID)

	workerCtx, cancelWorker := context.WithCancel(context.Background())
	defer cancelWorker()

	workerResult := make(chan error, 1)

	go func() {
		workerResult <- queue.Run(workerCtx, func(err error) {
			log.Printf("job gagal: %v", err)
		})
	}()
	submitter := onboarding.NewSubmitter(service, queue)
	idempotentSubmitter := onboarding.NewIdempotentSubmitter(submitter)
	server := &http.Server{
		Addr:              "127.0.0.1:8091",
		Handler:           httpapi.NewHandler(service, idempotentSubmitter, apiKey),
		ReadHeaderTimeout: 5 * time.Second,
	}

	serverResult := make(chan error, 1)

	go func() {
		log.Printf("service berjalan di http://%s", server.Addr)
		serverResult <- server.ListenAndServe()
	}()

	stopCtx, stopSignals := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
	)
	defer stopSignals()

	select {
	case <-stopCtx.Done():
		log.Println("sinyal shutdown diterima")
	case err := <-serverResult:
		if !errors.Is(err, http.ErrServerClosed) {
			log.Printf("HTTP server berhenti: %v", err)
		}
	}

	// Jangan menerima Job baru selama shutdown.
	queue.Close()

	shutdownCtx, cancelShutdown := context.WithTimeout(
		context.Background(),
		5*time.Second,
	)
	defer cancelShutdown()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Printf("shutdown HTTP gagal: %v", err)

		if closeErr := server.Close(); closeErr != nil {
			log.Printf("tutup HTTP gagal: %v", closeErr)
		}
	}

	// Hentikan Job aktif melalui context worker.
	cancelWorker()

	select {
	case err := <-workerResult:
		if err != nil && !errors.Is(err, context.Canceled) {
			log.Printf("worker berhenti dengan error: %v", err)
		}
	case <-time.After(10 * time.Second):
		log.Println("worker belum berhenti; shutdown tidak selesai")
		return
	}

	// Aman mengambil sisa antrean setelah worker berhenti.
	pending, err := queue.Drain()
	if err != nil {
		log.Printf("kosongkan antrean gagal: %v", err)
		return
	}

	for _, task := range pending {
		if err := service.CancelQueued(task); err != nil {
			log.Printf("batalkan job %s gagal: %v", task.ID, err)
		}
	}

	log.Println("service berhenti")
}
