package api

import (
	"github.com/gofiber/fiber/v2"

	tripErrors "job4j_go_share_trip/internal/api/errors"
)

type ErrorMapper struct{}

func NewErrorMapper() *ErrorMapper {
	return &ErrorMapper{}
}

func (m *ErrorMapper) MapError(c *fiber.Ctx, err error) error {
	if err == nil {
		return c.Status(fiber.StatusInternalServerError).JSON(Response{
			Status:  "Error",
			Message: "Unknown error",
		})
	}

	status := tripErrors.GetHTTPStatus(err)
	message := tripErrors.GetErrorMessage(err)

	return c.Status(status).JSON(Response{
		Status:  "Error",
		Message: message,
		Error:   err.Error(),
	})
}

func (m *ErrorMapper) MapValidationError(c *fiber.Ctx, err error) error {
	return c.Status(fiber.StatusBadRequest).JSON(Response{
		Status:  "Error",
		Message: err.Error(),
	})
}

func (m *ErrorMapper) MapParseError(c *fiber.Ctx, err error, message string) error {
	return c.Status(fiber.StatusBadRequest).JSON(Response{
		Status:  "Error",
		Message: err.Error(),
	})
}

func (m *ErrorMapper) MapNotFound(c *fiber.Ctx, err error) error {
    return c.Status(fiber.StatusNotFound).JSON(Response{
        Status:  "Error",
        Message: err.Error(),
    })
}

func (m *ErrorMapper) MapConflict(c *fiber.Ctx, err error) error {
    return c.Status(fiber.StatusConflict).JSON(Response{
        Status:  "Error",
        Message: err.Error(),
    })
}

func (m *ErrorMapper) MapForbidden(c *fiber.Ctx, err error) error {
    return c.Status(fiber.StatusForbidden).JSON(Response{
        Status:  "Error",
        Message: err.Error(),
    })
}

func (m *ErrorMapper) MapInternalError(c *fiber.Ctx, err error, message string) error {
	return c.Status(fiber.StatusInternalServerError).JSON(Response{
		Status:  "Error",
		Message: err.Error(),
	})
}