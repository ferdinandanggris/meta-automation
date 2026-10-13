package random

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
)

type IDGenerator struct{}

func NewIDGenerator() *IDGenerator {
	return &IDGenerator{}
}

func (g *IDGenerator) Generate() (string, error) {
	var data [16]byte

	if _, err := rand.Read(data[:]); err != nil {
		return "", fmt.Errorf("buat ID acak: %w", err)
	}

	return "job-" + hex.EncodeToString(data[:]), nil
}
