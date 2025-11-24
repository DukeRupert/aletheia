package validation

import (
	"errors"
	"fmt"
	"mime/multipart"
	"net/http"
	"strings"
	"unicode"

	"github.com/go-playground/validator/v10"
)

// Validator provides input validation using go-playground/validator.
//
// Purpose:
// - Centralize input validation logic
// - Use struct tags for declarative validation
// - Provide consistent error messages
// - Integrate with Echo's validation interface
//
// Benefits over manual validation:
// - Less boilerplate in handlers
// - Consistent validation rules across application
// - Easy to add custom validators
// - Automatic error message generation
//
// Library: github.com/go-playground/validator/v10
type Validator struct {
	validate *validator.Validate
}

// NewValidator creates a new validator instance.
//
// Purpose:
// - Initialize the validator
// - Register custom validation functions
// - Configure error message formatting
//
// Usage in main.go:
//
//	e.Validator = validation.NewValidator()
func NewValidator() *Validator {
	// Create new validator instance
	v := validator.New()

	// Note: Custom validators (safecode, uniqueemail, etc.) would be registered here
	// but require database access. For now, we use standard validators.
	// Custom validators can be added later via RegisterCustomValidators()

	return &Validator{
		validate: v,
	}
}

// Validate validates a struct using its validation tags.
//
// Purpose:
// - Implement echo.Validator interface
// - Called automatically by Echo when c.Validate() is used
// - Return formatted validation errors
//
// Flow:
// 1. Call validator.Struct(i)
// 2. If no errors, return nil
// 3. If validation errors, format them into user-friendly messages
// 4. Return error
//
// Usage in handlers:
//
//	var req RegisterRequest
//	if err := c.Bind(&req); err != nil {
//	    return err
//	}
//	if err := c.Validate(&req); err != nil {
//	    return err
//	}
func (v *Validator) Validate(i interface{}) error {
	// Validate the struct using validator tags
	if err := v.validate.Struct(i); err != nil {
		// Check if it's validation errors
		if validationErrors, ok := err.(validator.ValidationErrors); ok {
			// Format the validation errors into user-friendly messages
			formattedErrors := FormatValidationErrors(validationErrors)

			// Return a single error message listing all validation failures
			var errorMessages []string
			for field, message := range formattedErrors {
				errorMessages = append(errorMessages, fmt.Sprintf("%s: %s", field, message))
			}
			return errors.New(strings.Join(errorMessages, "; "))
		}
		// Return the original error if it's not validation errors
		return err
	}
	return nil
}

// Example request structs with validation tags

// RegisterRequest represents user registration input.
//
// Validation rules:
// - Email: required, valid email format, max 255 chars
// - Username: required, 3-50 chars, alphanumeric + underscore
// - Password: required, min 12 chars (complexity checked separately)
// - FirstName: required, max 100 chars
// - LastName: required, max 100 chars
type RegisterRequest struct {
	Email     string `json:"email" validate:"required,email,max=255"`
	Username  string `json:"username" validate:"required,min=3,max=50,alphanum"`
	Password  string `json:"password" validate:"required,min=12"`
	FirstName string `json:"first_name" validate:"required,max=100"`
	LastName  string `json:"last_name" validate:"required,max=100"`
}

// LoginRequest represents login input.
type LoginRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required"`
}

// CreateInspectionRequest represents inspection creation input.
type CreateInspectionRequest struct {
	ProjectID   string `json:"project_id" validate:"required,uuid"`
	Title       string `json:"title" validate:"required,min=3,max=200"`
	Description string `json:"description" validate:"max=2000"`
	// Add other fields with appropriate validation tags
}

// Custom validators

// RegisterCustomValidators registers application-specific validation rules.
//
// Purpose:
// - Add custom validation beyond built-in validators
// - Examples: valid safety code, unique email, valid organization ID
//
// Custom validators to implement:
// - safecode: validates against safety_codes table
// - uniqueemail: checks if email already exists
// - orgaccess: validates user has access to organization
// - validimage: checks file type and size for uploads
//
// Usage in NewValidator():
//
//	v.RegisterValidation("safecode", validateSafetyCode)
func RegisterCustomValidators(v interface{}) {
	// TODO: Get validator instance
	// TODO: Register "safecode" validator
	// TODO: Register "uniqueemail" validator
	// TODO: Register "orgaccess" validator
	// TODO: Register "validimage" validator
}

// validateSafetyCode validates that a code exists in safety_codes table.
//
// Purpose:
// - Ensure violations reference valid safety codes
// - Prevent invalid foreign key insertions
//
// Implementation:
// - Query database for code
// - Return true if exists, false otherwise
// - Cache results to avoid repeated queries
func validateSafetyCode(fl interface{}) bool {
	// TODO: Extract field value (safety code)
	// TODO: Check cache first
	// TODO: If not cached, query database
	// TODO: Cache result
	// TODO: Return validation result
	return false
}

// validateUniqueEmail validates that email doesn't already exist.
//
// Purpose:
// - Catch duplicate emails early (before database constraint)
// - Provide better error message than DB constraint error
//
// Note: Race condition possible (email could be created between validation and insert)
// Database constraint is still required as final check.
func validateUniqueEmail(fl interface{}) bool {
	// TODO: Extract email from field
	// TODO: Query database for existing user with email
	// TODO: Return true if not found, false if exists
	return false
}

// validateOrganizationAccess validates user has access to organization.
//
// Purpose:
// - Ensure user can only create/modify resources in their organizations
// - Prevent unauthorized access
//
// Implementation:
// - Extract user ID from context
// - Extract organization ID from field
// - Query organization_members table
// - Return true if user is member, false otherwise
func validateOrganizationAccess(fl interface{}) bool {
	// TODO: Get user ID from context (need to pass context to validator)
	// TODO: Extract organization ID from field
	// TODO: Query organization_members for membership
	// TODO: Return validation result
	return false
}

// FormatValidationErrors converts validator errors to user-friendly messages.
//
// Purpose:
// - Transform technical validation errors into readable messages
// - Map field names to human-readable labels
// - Provide specific error messages for each validation rule
//
// Example output:
//
//	{
//	  "email": "must be a valid email address",
//	  "password": "must be at least 12 characters",
//	  "username": "must be between 3 and 50 characters"
//	}
//
// Parameters:
//
//	err - validation error from validator.Struct()
//
// Returns map of field -> error message.
func FormatValidationErrors(err error) map[string]string {
	errors := make(map[string]string)

	// Type assert to validator.ValidationErrors
	validationErrors, ok := err.(validator.ValidationErrors)
	if !ok {
		// If it's not validation errors, return generic error
		errors["_error"] = err.Error()
		return errors
	}

	// Iterate over each field error
	for _, fieldErr := range validationErrors {
		fieldName := strings.ToLower(fieldErr.Field())

		// Generate user-friendly message based on validation tag
		switch fieldErr.Tag() {
		case "required":
			errors[fieldName] = "is required"
		case "email":
			errors[fieldName] = "must be a valid email address"
		case "min":
			if fieldErr.Type().Kind() == 24 { // string type
				errors[fieldName] = fmt.Sprintf("must be at least %s characters", fieldErr.Param())
			} else {
				errors[fieldName] = fmt.Sprintf("must be at least %s", fieldErr.Param())
			}
		case "max":
			if fieldErr.Type().Kind() == 24 { // string type
				errors[fieldName] = fmt.Sprintf("must be no more than %s characters", fieldErr.Param())
			} else {
				errors[fieldName] = fmt.Sprintf("must be no more than %s", fieldErr.Param())
			}
		case "uuid":
			errors[fieldName] = "must be a valid UUID"
		case "alphanum":
			errors[fieldName] = "must contain only letters and numbers"
		case "gte":
			errors[fieldName] = fmt.Sprintf("must be greater than or equal to %s", fieldErr.Param())
		case "lte":
			errors[fieldName] = fmt.Sprintf("must be less than or equal to %s", fieldErr.Param())
		case "gt":
			errors[fieldName] = fmt.Sprintf("must be greater than %s", fieldErr.Param())
		case "lt":
			errors[fieldName] = fmt.Sprintf("must be less than %s", fieldErr.Param())
		case "len":
			errors[fieldName] = fmt.Sprintf("must be exactly %s characters", fieldErr.Param())
		case "oneof":
			errors[fieldName] = fmt.Sprintf("must be one of: %s", fieldErr.Param())
		default:
			// Generic error message for unknown validation tags
			errors[fieldName] = fmt.Sprintf("failed validation: %s", fieldErr.Tag())
		}
	}

	return errors
}

// PasswordComplexity validates password complexity requirements.
//
// Purpose:
// - Enforce strong password policy
// - Check for uppercase, lowercase, number, special character
// - Called separately from struct validation (not a validator tag)
//
// Requirements:
// - At least 12 characters (checked by min tag)
// - Contains uppercase letter
// - Contains lowercase letter
// - Contains number
// - Contains special character
//
// Usage in handlers:
//
//	if err := validation.PasswordComplexity(req.Password); err != nil {
//	    return echo.NewHTTPError(http.StatusBadRequest, err.Error())
//	}
func PasswordComplexity(password string) error {
	// Check minimum length (12 characters)
	if len(password) < 12 {
		return errors.New("password must be at least 12 characters long")
	}

	var (
		hasUpper   bool
		hasLower   bool
		hasNumber  bool
		hasSpecial bool
	)

	// Check for required character types
	for _, char := range password {
		switch {
		case unicode.IsUpper(char):
			hasUpper = true
		case unicode.IsLower(char):
			hasLower = true
		case unicode.IsNumber(char):
			hasNumber = true
		case unicode.IsPunct(char) || unicode.IsSymbol(char):
			hasSpecial = true
		}
	}

	// Return specific error messages for missing requirements
	if !hasUpper {
		return errors.New("password must contain at least one uppercase letter")
	}
	if !hasLower {
		return errors.New("password must contain at least one lowercase letter")
	}
	if !hasNumber {
		return errors.New("password must contain at least one number")
	}
	if !hasSpecial {
		return errors.New("password must contain at least one special character")
	}

	return nil
}

// SanitizeInput removes potentially dangerous characters from user input.
//
// Purpose:
// - Prevent XSS attacks
// - Remove control characters
// - Trim whitespace
//
// Note: This is defense in depth - output encoding is primary XSS defense.
//
// Usage:
//
//	req.Username = validation.SanitizeInput(req.Username)
func SanitizeInput(input string) string {
	// Trim leading/trailing whitespace
	input = strings.TrimSpace(input)

	// Remove null bytes and other control characters
	var builder strings.Builder
	for _, r := range input {
		// Keep printable characters, tabs, newlines, and carriage returns
		// Filter out other control characters (including null bytes)
		if r == '\t' || r == '\n' || r == '\r' || !unicode.IsControl(r) {
			builder.WriteRune(r)
		}
	}

	return builder.String()
}

// ValidateFileUpload validates file uploads.
//
// Purpose:
// - Check file size limits
// - Verify file type (MIME type)
// - Prevent malicious uploads
//
// Parameters:
//
//	fileHeader - multipart.FileHeader from form upload
//	maxSize - maximum file size in bytes
//	allowedTypes - slice of allowed MIME types
//
// Returns error if validation fails.
//
// Usage in upload handler:
//
//	err := validation.ValidateFileUpload(fileHeader, 10*1024*1024, []string{"image/jpeg", "image/png", "image/webp"})
func ValidateFileUpload(fileHeader interface{}, maxSize int64, allowedTypes []string) error {
	// Type assert to multipart.FileHeader
	header, ok := fileHeader.(*multipart.FileHeader)
	if !ok {
		return errors.New("invalid file header")
	}

	// Check file size against maxSize
	if header.Size > maxSize {
		return fmt.Errorf("file size exceeds maximum allowed size of %d bytes", maxSize)
	}

	// Open the file to read its content
	file, err := header.Open()
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	// Read first 512 bytes for content type detection
	buffer := make([]byte, 512)
	n, err := file.Read(buffer)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	// Use http.DetectContentType to get MIME type
	contentType := http.DetectContentType(buffer[:n])

	// Check if the detected MIME type is in the allowed list
	allowed := false
	for _, allowedType := range allowedTypes {
		if contentType == allowedType {
			allowed = true
			break
		}
	}

	if !allowed {
		return fmt.Errorf("file type %s is not allowed (allowed types: %s)", contentType, strings.Join(allowedTypes, ", "))
	}

	return nil
}

// ValidationErrorResponse formats validation errors for API responses.
//
// Purpose:
// - Consistent error response format
// - Include all field errors in single response
// - Easy for clients to parse and display
//
// Response format:
//
//	{
//	  "error": "validation failed",
//	  "fields": {
//	    "email": "must be a valid email address",
//	    "password": "must be at least 12 characters"
//	  }
//	}
func ValidationErrorResponse(err error) map[string]interface{} {
	// Format the validation errors
	fieldErrors := FormatValidationErrors(err)

	// Build response map with "error" and "fields" keys
	response := map[string]interface{}{
		"error":  "validation failed",
		"fields": fieldErrors,
	}

	return response
}
