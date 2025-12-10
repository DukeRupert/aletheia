
## Storage

### AWS implementation
```go
// Load AWS config
cfg, err := config.LoadDefaultConfig(context.Background())
if err != nil {
    log.Fatal(err)
}

// Create S3 client
s3Client := s3.NewFromConfig(cfg)

// Use S3 storage instead
fileStorage := storage.NewS3Storage(
    s3Client,
    "my-bucket-name",
    "us-east-1",
    "https://cdn.myapp.com", // CloudFront URL
)
```

## Audit Logging

The audit logging system is implemented but not yet integrated into handlers. Here's how to enable it:

### 1. Enable Scheduled Cleanup Job

In `cmd/aletheiad/main.go`, add a cleanup job:

```go
// Enable daily cleanup at 2 AM
go func() {
	ticker := time.NewTicker(24 * time.Hour)
	defer ticker.Stop()
	for range ticker.C {
		now := time.Now()
		if now.Hour() == 2 {
			ctx := context.Background()
			deleted, err := auditLogger.CleanupOldAuditLogs(ctx, 2555)
			if err != nil {
				logger.Error("audit log cleanup failed", slog.String("error", err.Error()))
			} else {
				logger.Info("audit log cleanup completed", slog.Int64("deleted", deleted))
			}
		}
	}
}()
```

**Retention period**: 2555 days (7 years) - common compliance requirement

### 2. Add AuditLogger to Server

Add `auditLogger` field to the `http.Server` struct:

```go
// In http/server.go
type Server struct {
	// ... existing fields ...
	auditLogger *audit.AuditLogger
}
```

**Handler methods that need audit logging:**
- Auth handlers - user registration, login, logout, password changes
- Organization handlers - organization CRUD, member management
- Project handlers - project CRUD
- Inspection handlers - inspection CRUD, status changes
- Safety code handlers - safety code CRUD
- Photo handlers - photo uploads, deletions
- Violation handlers - violation confirmations, dismissals, manual creation

### 3. Add Audit Logging Calls in Handler Methods

**After creating a resource:**

```go
// In Register handler after creating user
auditLogger.LogCreate(ctx, user.ID, orgID, "user", user.ID,
	map[string]interface{}{
		"email": user.Email,
		"username": user.Username,
	}, c)
```

**Before and after updates:**

```go
// In UpdateInspection handler
// Get old values before update
oldInspection, _ := queries.GetInspection(ctx, inspectionID)

// Perform update
newInspection := updateInspection(...)

// Log the update
auditLogger.LogUpdate(ctx, userID, orgID, "inspection", inspectionID,
	map[string]interface{}{
		"status": oldInspection.Status,
		"title": oldInspection.Title,
	},
	map[string]interface{}{
		"status": newInspection.Status,
		"title": newInspection.Title,
	}, c)
```

**Before deletions:**

```go
// In DeleteProject handler
// Get values before deletion
project, _ := queries.GetProject(ctx, projectID)

// Delete the project
queries.DeleteProject(ctx, projectID)

// Log the deletion
auditLogger.LogDelete(ctx, userID, orgID, "project", projectID,
	map[string]interface{}{
		"name": project.Name,
		"status": project.Status,
	}, c)
```

**For sensitive data access:**

```go
// In GetReport handler (for compliance)
auditLogger.LogView(ctx, userID, orgID, "report", reportID, c)
```

### 5. Viewing Audit Logs

**Query user activity:**

```go
entries, err := auditLogger.GetUserAuditLog(ctx, userID, 50, 0)
```

**Query resource history:**

```go
entries, err := auditLogger.GetResourceAuditLog(ctx, "inspection", inspectionID, 50, 0)
```

**Search with filters:**

```go
filter := audit.AuditFilter{
	OrganizationID: &orgID,
	Action: ptr("create"),
	StartTime: &startTime,
	EndTime: &endTime,
	Limit: 100,
}
entries, err := auditLogger.SearchAuditLog(ctx, filter)
```

**Export for compliance:**

```go
var buf bytes.Buffer
err := auditLogger.ExportAuditLog(ctx, filter, &buf)
// buf now contains CSV data for regulators
```

### 6. Remove Unused Variable Suppression

Once audit logging is integrated, remove any unused variable suppressions in `cmd/aletheiad/main.go`:

```go
_ = auditLogger  // Delete this line when integrated
```

### Benefits

- **Compliance**: Meets OSHA, insurance, and legal audit requirements
- **Security**: Track all state changes and detect unauthorized access
- **Forensics**: Complete history for investigations and rollback
- **Accountability**: Know who did what, when, and from where
