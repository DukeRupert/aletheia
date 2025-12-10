package http

// registerRoutes sets up all routes for the server.
// All routes are defined in this single file for easy navigation.
func (s *Server) registerRoutes() {
	// Health check routes (public)
	s.echo.GET("/health", s.handleHealthCheck)
	s.echo.GET("/health/live", s.handleLivenessCheck)
	s.echo.GET("/health/ready", s.handleReadinessCheck)

	// Public auth routes
	auth := s.echo.Group("/api/auth")
	auth.POST("/register", s.handleRegister)
	auth.POST("/login", s.handleLogin)
	auth.POST("/verify-email", s.handleVerifyEmail)
	auth.POST("/resend-verification", s.handleResendVerification)
	auth.POST("/request-password-reset", s.handleRequestPasswordReset)
	auth.POST("/verify-reset-token", s.handleVerifyResetToken)
	auth.POST("/reset-password", s.handleResetPassword)

	// Protected routes (require authentication)
	protected := s.echo.Group("/api")
	protected.Use(s.RequireAuth())

	// Auth (authenticated)
	protected.POST("/auth/logout", s.handleLogout)
	protected.GET("/auth/me", s.handleMe)
	protected.PUT("/auth/profile", s.handleUpdateProfile)

	// Organizations
	protected.POST("/organizations", s.handleCreateOrganization)
	protected.GET("/organizations", s.handleListOrganizations)
	protected.GET("/organizations/:id", s.handleGetOrganization)
	protected.PUT("/organizations/:id", s.handleUpdateOrganization)
	protected.DELETE("/organizations/:id", s.handleDeleteOrganization)

	// Organization members
	protected.GET("/organizations/:id/members", s.handleListOrganizationMembers)
	protected.POST("/organizations/:id/members", s.handleAddOrganizationMember)
	protected.PUT("/organizations/:id/members/:memberId", s.handleUpdateOrganizationMember)
	protected.DELETE("/organizations/:id/members/:memberId", s.handleRemoveOrganizationMember)

	// Projects
	protected.POST("/projects", s.handleCreateProject)
	protected.GET("/projects/:id", s.handleGetProject)
	protected.GET("/organizations/:orgId/projects", s.handleListProjects)
	protected.PUT("/projects/:id", s.handleUpdateProject)
	protected.DELETE("/projects/:id", s.handleDeleteProject)

	// Inspections
	protected.POST("/inspections", s.handleCreateInspection)
	protected.GET("/inspections/:id", s.handleGetInspection)
	protected.GET("/projects/:projectId/inspections", s.handleListInspections)
	protected.PUT("/inspections/:id/status", s.handleUpdateInspectionStatus)

	// Photos
	protected.POST("/photos", s.handleUploadPhoto)
	protected.GET("/photos/:id", s.handleGetPhoto)
	protected.GET("/inspections/:inspectionId/photos", s.handleListPhotos)
	protected.DELETE("/photos/:id", s.handleDeletePhoto)
	protected.POST("/photos/analyze", s.handleAnalyzePhoto)
	protected.GET("/photos/analyze/:jobId", s.handleGetPhotoAnalysisStatus)

	// Safety codes
	protected.POST("/safety-codes", s.handleCreateSafetyCode)
	protected.GET("/safety-codes", s.handleListSafetyCodes)
	protected.GET("/safety-codes/:id", s.handleGetSafetyCode)
	protected.PUT("/safety-codes/:id", s.handleUpdateSafetyCode)
	protected.DELETE("/safety-codes/:id", s.handleDeleteSafetyCode)

	// Violations
	protected.POST("/violations/manual", s.handleCreateViolation)
	protected.GET("/inspections/:inspectionId/violations", s.handleListViolations)
	protected.POST("/violations/:id/confirm", s.handleConfirmViolation)
	protected.POST("/violations/:id/dismiss", s.handleDismissViolation)
	protected.POST("/violations/:id/pending", s.handleSetViolationPending)
	protected.PATCH("/violations/:id", s.handleUpdateViolation)
}
