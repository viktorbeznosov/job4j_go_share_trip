package api

import (
	"github.com/gofiber/fiber/v2"
)

type ReadyResponse struct {
	Status string `json:"status"`
	Ready  bool   `json:"ready"`
}

func (s *Server) Ready(c *fiber.Ctx) error {
	return c.Status(fiber.StatusOK).JSON(ReadyResponse{
		Status: "Success",
		Ready:  true,
	})
}
