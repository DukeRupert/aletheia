# Aletheia MVP Roadmap

A construction safety inspection platform using AI to detect safety violations from photos.

## Foundation (Completed)

### Project Setup
- [x] Go project structure established
- [x] Echo web framework integrated
- [x] PostgreSQL database with pgxpool connection
- [x] Database migrations with Goose (all tables created)
- [x] sqlc configured and integrated
- [x] Configuration system (.env + command-line flags)
- [x] Structured logging (slog)
- [x] Docker Compose for local development
- [x] Makefile with dev, build, and migration commands
- [x] Storage abstraction layer (local + S3)

## Phase 1: Core Data Layer

### Database Operations
- [x] Create sqlc queries for users
- [x] Create sqlc queries for organizations
- [x] Create sqlc queries for organization members
- [x] Create sqlc queries for projects
- [x] Create sqlc queries for inspections
- [x] Create sqlc queries for photos
- [x] Create sqlc queries for detected violations
- [x] Create sqlc queries for safety codes
- [x] Create sqlc queries for reports

## Phase 2: Authentication & User Management (Completed ✓)

### User Registration & Login
- [x] Implement password hashing (bcrypt)
- [x] Create user registration endpoint
- [x] Create user login endpoint with session-based auth
- [x] Create session middleware for protected routes
- [x] Implement email verification flow
- [x] Implement password reset flow
- [x] Create user profile endpoints (get, update)

### Session Management
- [x] Configure session storage
- [x] Implement logout endpoint
- [ ] Add token refresh mechanism (optional - skipped for MVP)

## Phase 3: Organization & Project Management

### Organization Setup (Completed ✓)
- [x] Create organization creation endpoint
- [x] Create organization retrieval endpoints (get, list)
- [x] Create organization update endpoint
- [x] Implement organization member invitation flow (add by email)
- [x] Create organization member management endpoints (list, add, update role, remove)
- [x] Add role-based access control (owner, admin, member)

### Project Management (Completed ✓)
- [x] Create project creation endpoint
- [x] Create project listing endpoint (by organization)
- [x] Create project detail endpoint
- [x] Create project update endpoint
- [x] Create project deletion/archival endpoint

## Phase 4: Inspection Workflow (Completed ✓)

### Inspection Management (Completed ✓)
- [x] Create inspection creation endpoint
- [x] Create inspection listing endpoint (by project)
- [x] Create inspection detail endpoint
- [x] Create inspection update endpoint (status)
- [x] Implement inspection status workflow (draft, in_progress, completed)

### Photo Upload & Storage (Completed ✓)
- [x] Basic photo upload endpoint implemented
- [x] Local storage implementation complete
- [x] S3 storage implementation complete
- [x] Storage service made modular/configurable (local or S3 via env var)
- [x] Associate photos with inspections
- [x] Create photo listing endpoint (by inspection)
- [x] Create photo detail endpoint
- [x] Implement photo deletion endpoint
- [x] Add thumbnail generation (abstracted interface for local/S3)

## Phase 5: Safety Code Configuration

### Safety Standards Management
- [ ] Create safety code creation endpoint
- [ ] Create safety code listing endpoint
- [ ] Create safety code update endpoint
- [ ] Seed database with common OSHA/safety standards
- [ ] Add safety code categorization/tagging

## Phase 6: AI Integration

### AI Vision Processing
- [ ] Choose AI provider (OpenAI Vision, Anthropic Claude, etc.)
- [ ] Implement AI service client
- [ ] Create prompt engineering for safety violation detection
- [ ] Build photo analysis queue/job system
- [ ] Create endpoint to trigger AI analysis on photos
- [ ] Store detected violations in database
- [ ] Map AI findings to safety codes
- [ ] Add confidence scoring for violations

### Violation Management
- [ ] Create violation listing endpoint (by inspection)
- [ ] Create violation detail endpoint
- [ ] Implement violation confirmation/dismissal workflow
- [ ] Add violation severity levels
- [ ] Create violation update endpoint (add notes, change status)

## Phase 7: Reporting

### Report Generation
- [ ] Design report template structure
- [ ] Create report generation endpoint
- [ ] Generate PDF reports with violations
- [ ] Include photos in reports
- [ ] Add report metadata (inspector, date, project info)
- [ ] Create report listing endpoint
- [ ] Create report download endpoint
- [ ] Implement report sharing/export

## Phase 8: Frontend Development

### Authentication UI
- [ ] Login page
- [ ] Registration page
- [ ] Email verification page
- [ ] Password reset page
- [ ] User profile page

### Dashboard
- [ ] Organization dashboard
- [ ] Project listing view
- [ ] Recent inspections view
- [ ] Violation statistics/charts

### Inspection Interface
- [ ] Create new inspection form
- [ ] Inspection detail view
- [ ] Photo upload interface
- [ ] Photo gallery view
- [ ] Trigger AI analysis button
- [ ] Violation review interface
- [ ] Mark violations as confirmed/dismissed

### Project Management UI
- [ ] Create project form
- [ ] Project detail view
- [ ] Project settings page
- [ ] Organization member management UI

### Reports UI
- [ ] Report generation form
- [ ] Report preview
- [ ] Report download/share
- [ ] Report history view

## Phase 9: Testing & Quality

### Backend Testing
- [ ] Write unit tests for handlers
- [ ] Write integration tests for database operations
- [ ] Write tests for AI service integration
- [ ] Write tests for file storage
- [ ] Test authentication flows
- [ ] Test authorization/permissions

### Frontend Testing
- [ ] Component unit tests
- [ ] Integration tests for key workflows
- [ ] End-to-end tests for critical paths
- [ ] Cross-browser testing
- [ ] Mobile responsive testing

## Phase 10: Deployment & Operations

### Infrastructure Setup
- [x] Set up development PostgreSQL database (Docker Compose)
- [x] Configure environment variables management (.env + flags)
- [x] Configure logging (slog with JSON/text modes)
- [ ] Set up production PostgreSQL database
- [ ] Configure S3 bucket and CloudFront for production
- [ ] Set up error tracking (Sentry, etc.)
- [ ] Configure backup strategy for database

### Deployment
- [ ] Create production build process
- [ ] Set up CI/CD pipeline
- [ ] Deploy backend to production environment
- [ ] Deploy frontend to production environment
- [ ] Run production migrations
- [ ] Configure custom domain and SSL
- [ ] Set up monitoring and alerts

### Documentation
- [ ] API documentation
- [ ] User guide/documentation
- [ ] Admin documentation
- [ ] Deployment runbook

## Phase 11: MVP Polish

### User Experience
- [ ] Add loading states and error messages
- [ ] Implement proper form validation
- [ ] Add success notifications
- [ ] Improve error handling and user feedback
- [ ] Add help text and tooltips

### Performance
- [ ] Optimize database queries
- [ ] Add database indexes (verify existing)
- [ ] Implement pagination for lists
- [ ] Optimize image loading
- [ ] Add caching where appropriate

### Security Audit
- [ ] Review authentication implementation
- [ ] Test authorization on all endpoints
- [ ] Validate input sanitization
- [ ] Check for SQL injection vulnerabilities
- [ ] Review file upload security
- [ ] Test rate limiting
- [ ] Review CORS configuration

## Launch Checklist

- [ ] All critical paths tested
- [ ] Production database backed up
- [ ] Monitoring and alerts configured
- [ ] Error tracking operational
- [ ] SSL certificates valid
- [ ] Environment variables secured
- [ ] Initial safety codes seeded
- [ ] At least one test organization created
- [ ] User documentation complete
- [ ] Support contact information configured

---

## Notes

- This roadmap is flexible - adjust as needed based on feedback and priorities
- Some phases can be worked on in parallel (e.g., frontend while backend is being built)
- Consider using feature flags for gradual rollout of AI features
- MVP should focus on core workflow: Upload photos → AI detects violations → Generate report
