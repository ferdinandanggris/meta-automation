package main

import (
	"context"
	"log"
	"meta-automation/internal/infrastructure/browser"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	if err := run(); err != nil {
		log.Printf("browser gagal: %v", err)
		os.Exit(1)
	}
}

func run() error {
	session, err := browser.NewSession(
		context.Background(),
		os.Getenv("META_BROWSER_PATH"),
		".local/browser-profile",
	)
	if err != nil {
		return err
	}

	defer func() {
		if err := session.Close(); err != nil {
			log.Printf("tutup browser gagal: %v", err)
		}
	}()

	stopCtx, stopSignals := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
	)
	defer stopSignals()

	if err := session.Start(); err != nil {
		return err
	}

	if err := session.PrepareWABA(
		stopCtx,
		"manual-test-001",
		os.Getenv("META_BUSINESS_ID"),
		os.Getenv("META_WABA_NAME"),
	); err != nil {
		return err
	}

	log.Println("Form siap: nama terisi dan Lanjutkan ditemukan.")
	log.Println("Lanjutkan tidak diklik. Tekan Ctrl+C untuk menutup.")

	<-stopCtx.Done()
	return nil
}
