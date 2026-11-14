package api

import (
	"unicode/utf8"

	"github.com/gofiber/fiber/v2"

	tripErrors "job4j_go_share_trip/internal/api/errors"
)

// maxTermRunes — максимум символов в параметре term: длинный запрос
// бессмысленен для подсказки, но раздувает время поиска.
const maxTermRunes = 256

// SuggestPoints обрабатывает GET /points?term=... и отдаёт до 10
// уникальных пунктов поездок и заявок, содержащих term как подстроку
// без учёта регистра. Доступ — все четыре роли (см. route.go).
func (h *TripHandler) SuggestPoints(c *fiber.Ctx) error {
	term := c.Query("term")

	if utf8.RuneCountInString(term) > maxTermRunes {
		return h.errorMapper.MapValidationError(c, tripErrors.ErrTermTooLong)
	}

	points, err := h.TripService.SuggestPoints(c.UserContext(), term)
	if err != nil {
		return h.errorMapper.MapError(c, err)
	}

	return c.Status(fiber.StatusOK).JSON(SuggestPointsResponseWrapper{
		Status: "Success",
		Data:   SuggestPointsResponse{Items: points},
	})
}
