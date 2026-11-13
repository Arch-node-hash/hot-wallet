package response

import (
	"errors"
	"fmt"
	"math"
	"strings"
	"time"

	"hotwallet/backend/pkg/crud"

	"github.com/gofiber/fiber/v3"
)

// SendSuccess kirim response sukses dengan data.
func SendSuccess(c fiber.Ctx, status int, data any) error {
	return c.Status(status).JSON(data)
}

// SendError kirim response error dengan pesan.
func SendError(c fiber.Ctx, err error) error {
	switch {
	case errors.Is(err, crud.ErrNotFound):
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": err.Error()})
	case errors.Is(err, crud.ErrInvalid):
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	case errors.Is(err, crud.ErrConflict):
		return c.Status(fiber.StatusConflict).JSON(fiber.Map{"error": err.Error()})
	case errors.Is(err, crud.ErrUnauthorized):
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": err.Error()})
	case errors.Is(err, crud.ErrForbidden):
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": err.Error()})
	default:
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "internal server error"})
	}
}

// SendMessage kirim response dengan pesan sederhana.
func SendMessage(c fiber.Ctx, status int, message string) error {
	return c.Status(status).JSON(fiber.Map{"message": message})
}

// SendPlaceholder adalah response standar untuk endpoint skeleton.
// Ganti isi handler dengan logika nyata saat mulai diimplementasikan.
func SendPlaceholder(c fiber.Ctx, status int, resource string) error {
	payload := fiber.Map{
		"resource": resource,
		"message":  "placeholder",
	}
	if id := c.Params("id"); id != "" {
		payload["id"] = id
	}
	return c.Status(status).JSON(payload)
}

// Meta describes pagination for list responses.
type Meta struct {
	Page       int `json:"page"`
	Limit      int `json:"limit"`
	Total      int `json:"total"`
	TotalPages int `json:"total_pages"`
}

// Paginated is the standard shape for paginated data.
type Paginated struct {
	Data any  `json:"data"`
	Meta Meta `json:"meta"`
}

// SendPaginated sends a list response with pagination metadata.
func SendPaginated(c fiber.Ctx, status int, data any, meta Meta) error {
	return c.Status(status).JSON(Paginated{Data: data, Meta: meta})
}

// FormatUSDT memformat angka menjadi string USDT dengan dua desimal.
func FormatUSDT(amount float64) string {
	return fmt.Sprintf("%.2f USDT", amount)
}

// FormatDate memformat time.Time menjadi tanggal lokal.
func FormatDate(t time.Time) string {
	return t.Format("2006-01-02")
}

// FormatDateTime memformat time.Time menjadi tanggal dan waktu lokal.
func FormatDateTime(t time.Time) string {
	return t.Format("2006-01-02 15:04:05")
}

// FormatUSDTThousands memformat angka dengan pemisah ribuan.
func FormatUSDTThousands(amount float64) string {
	negative := amount < 0
	amount = math.Abs(amount)

	raw := fmt.Sprintf("%.2f", amount)
	whole, decimals, _ := strings.Cut(raw, ".")
	var grouped []string
	for len(whole) > 3 {
		grouped = append([]string{whole[len(whole)-3:]}, grouped...)
		whole = whole[:len(whole)-3]
	}
	grouped = append([]string{whole}, grouped...)

	result := strings.Join(grouped, ",") + "." + decimals + " USDT"
	if negative {
		return "-" + result
	}
	return result
}

// TruncateMiddle memendekkan hash/alamat menjadi bentuk 0x71C4...9F2A.
func TruncateMiddle(value string, keep int) string {
	if len(value) <= keep*2+3 {
		return value
	}
	return value[:keep] + "..." + value[len(value)-keep:]
}
