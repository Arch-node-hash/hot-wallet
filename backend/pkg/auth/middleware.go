package auth

import (
	"strings"

	"github.com/gofiber/fiber/v3"
)

const claimsKey = "auth.claims"

// Protect adalah middleware penjaga rute API. Saat ini belum dipasang di
// router supaya perilaku lama tetap sama; tinggal panggil app.Use(...) kalau
// lock wallet ingin diaktifkan.
func Protect(secret []byte, publicPaths ...string) fiber.Handler {
	public := make(map[string]struct{}, len(publicPaths))
	for _, path := range publicPaths {
		public[path] = struct{}{}
	}

	return func(c fiber.Ctx) error {
		requestPath := c.Path()
		if !strings.HasPrefix(requestPath, "/api/") {
			return c.Next()
		}
		if _, ok := public[requestPath]; ok {
			return c.Next()
		}

		token := Bearer(c.Get(fiber.HeaderAuthorization))
		if token == "" {
			token = c.Cookies("hotwallet_token")
		}

		claims, err := Verify(secret, token)
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "wallet terkunci, kirim token unlock terlebih dahulu",
			})
		}

		c.Locals(claimsKey, claims)
		return c.Next()
	}
}

// ClaimsFrom mengembalikan claims yang disimpan oleh Protect.
func ClaimsFrom(c fiber.Ctx) (*Claims, bool) {
	claims, ok := c.Locals(claimsKey).(*Claims)
	return claims, ok
}
