package logger

import (
	"fmt"
	"log"
	"time"
)

// Info log pesan informasi dengan timestamp.
func Info(format string, args ...any) {
	log.Printf("[INFO] "+format, args...)
}

// Error log pesan error dengan timestamp.
func Error(domain string, err error) {
	log.Printf("[ERROR] %s: %v", domain, err)
}

// Timed log durasi eksekusi sebuah operasi.
func Timed(domain string, fn func() error) error {
	start := time.Now()
	err := fn()
	log.Printf("[%s] %s — %s", status(err), domain, fmt.Sprintf("%dms", time.Since(start).Milliseconds()))
	return err
}

func status(err error) string {
	if err != nil {
		return "ERROR"
	}
	return "OK"
}
