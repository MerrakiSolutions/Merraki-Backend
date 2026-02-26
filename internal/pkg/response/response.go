package response

import (
	"github.com/gofiber/fiber/v2"
	apperrors "github.com/merraki/merraki-backend/internal/pkg/errors"
)

type Response struct {
	Success bool        `json:"success"`
	Message string      `json:"message,omitempty"`
	Code    string      `json:"code,omitempty"`
	Data    interface{} `json:"data,omitempty"`
	Errors  interface{} `json:"errors,omitempty"`
}

type PaginatedResponse struct {
	Success    bool        `json:"success"`
	Data       interface{} `json:"data"`
	Pagination Pagination  `json:"pagination"`
}

type Pagination struct {
	Total   int  `json:"total"`
	Page    int  `json:"page"`
	Pages   int  `json:"pages"`
	Limit   int  `json:"limit"`
	HasNext bool `json:"has_next"`
	HasPrev bool `json:"has_prev"`
}

func Success(c *fiber.Ctx, message string, data interface{}) error {
	return c.JSON(Response{
		Success: true,
		Message: message,
		Data:    data,
	})
}

func SuccessData(c *fiber.Ctx, data interface{}) error {
	return c.JSON(Response{
		Success: true,
		Data:    data,
	})
}

func Created(c *fiber.Ctx, message string, data interface{}) error {
	return c.Status(fiber.StatusCreated).JSON(Response{
		Success: true,
		Message: message,
		Data:    data,
	})
}

func Error(c *fiber.Ctx, err error) error {
	if appErr, ok := err.(*apperrors.AppError); ok {
		return c.Status(appErr.Status).JSON(Response{
			Success: false,
			Message: appErr.Message,
			Code:    appErr.Code,
		})
	}

	return c.Status(fiber.StatusInternalServerError).JSON(Response{
		Success: false,
		Message: "Internal server error",
		Code:    "INTERNAL_SERVER_ERROR",
	})
}

func ValidationError(c *fiber.Ctx, errors interface{}) error {
	return c.Status(fiber.StatusUnprocessableEntity).JSON(Response{
		Success: false,
		Message: "Validation failed",
		Code:    "VALIDATION_ERROR",
		Errors:  errors,
	})
}

func Paginated(c *fiber.Ctx, data interface{}, total, page, limit int) error {
	pages := total / limit
	if total%limit > 0 {
		pages++
	}

	return c.JSON(PaginatedResponse{
		Success: true,
		Data:    data,
		Pagination: Pagination{
			Total:   total,
			Page:    page,
			Pages:   pages,
			Limit:   limit,
			HasNext: page < pages,
			HasPrev: page > 1,
		},
	})
}