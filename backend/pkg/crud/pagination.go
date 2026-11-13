package crud

// Page menyimpan metadata pagination.
type Page struct {
	Page       int
	Limit      int
	Total      int
	TotalPages int
}

// NormalizePage membatasi page/limit ke nilai yang aman.
func NormalizePage(page, limit, maxLimit int) (int, int) {
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 20
	}
	if maxLimit > 0 && limit > maxLimit {
		limit = maxLimit
	}
	return page, limit
}

// Paginate memotong items sesuai page dan limit.
func Paginate[T any](items []T, page, limit, maxLimit int) ([]T, Page) {
	page, limit = NormalizePage(page, limit, maxLimit)
	total := len(items)

	start := (page - 1) * limit
	if start > total {
		start = total
	}
	end := start + limit
	if end > total {
		end = total
	}

	totalPages := 0
	if limit > 0 {
		totalPages = (total + limit - 1) / limit
	}

	return items[start:end], Page{
		Page:       page,
		Limit:      limit,
		Total:      total,
		TotalPages: totalPages,
	}
}
