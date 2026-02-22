package middleware

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
)

// ErrorResponse represents a standardized error response
type ErrorResponse struct {
	Status  int         `json:"status"`
	Message string      `json:"message"`
	Errors  interface{} `json:"errors,omitempty"`
}

// ErrorHandler middleware handles errors globally
func ErrorHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		// Check if there are any errors
		if len(c.Errors) > 0 {
			err := c.Errors.Last()

			// Log the error
			log.Printf("Error: %v\n", err.Err)

			// Handle different types of errors
			switch e := err.Err.(type) {
			case validator.ValidationErrors:
				handleValidationError(c, e)
			default:
				handleGenericError(c, err.Err)
			}
		}
	}
}

// handleValidationError handles validation errors
func handleValidationError(c *gin.Context, errs validator.ValidationErrors) {
	var errorMessages []string
	for _, err := range errs {
		errorMessages = append(errorMessages, formatValidationError(err))
	}

	c.JSON(http.StatusBadRequest, ErrorResponse{
		Status:  http.StatusBadRequest,
		Message: "Validation failed",
		Errors:  errorMessages,
	})
}

// handleGenericError handles generic errors
func handleGenericError(c *gin.Context, err error) {
	status := http.StatusInternalServerError
	message := "Internal server error"

	// You can add custom error type checking here
	// switch err.(type) {
	// case *customerrors.NotFoundError:
	//     status = http.StatusNotFound
	//     message = err.Error()
	// }

	c.JSON(status, ErrorResponse{
		Status:  status,
		Message: message,
		Errors:  err.Error(),
	})
}

// formatValidationError formats a single validation error
func formatValidationError(err validator.FieldError) string {
	field := err.Field()
	tag := err.Tag()
	param := err.Param()

	switch tag {
	case "required":
		return field + " is required"
	case "email":
		return field + " must be a valid email address"
	case "min":
		return field + " must be at least " + param + " characters"
	case "max":
		return field + " must be at most " + param + " characters"
	case "oneof":
		return field + " must be one of: " + param
	default:
		return field + " failed validation on " + tag
	}
}

// Recovery middleware recovers from panics
func Recovery() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("Panic recovered: %v\n", r)
				c.JSON(http.StatusInternalServerError, ErrorResponse{
					Status:  http.StatusInternalServerError,
					Message: "Internal server error",
				})
				c.Abort()
			}
		}()
		c.Next()
	}
}
