package main

import (
	"fmt"
	"log"
	"meta-automation/internal/domain/job"
)

func main() {
	task, err := job.New("job-001", "5475458029853092", "Waba Percobaan")

	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Sebelum: ", task.Status())

	fmt.Printf("Job: %s\n", task.ID)
	fmt.Printf("Nama: %s\n", task.Name)
	fmt.Printf("Status: %s\n", task.Status())

	if err := task.Start(); err != nil {
		log.Fatal(err)
	}

	fmt.Println("Sesudah: ", task.Status())

	if err := task.Start(); err != nil {
		log.Fatal("mulai ulang ditolak:", err)
	}
}
