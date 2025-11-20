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

## Phase 5: Safety Code Configuration (Completed ✓)

### Safety Standards Management (Completed ✓)
- [x] Create safety code creation endpoint
- [x] Create safety code listing endpoint (with optional country/state filtering)
- [x] Create safety code retrieval endpoint
- [x] Create safety code update endpoint
- [x] Create safety code deletion endpoint
- [x] Seed database with common OSHA/safety standards (50+ construction safety codes)
- [x] Add safety code categorization (country and state/province support)

## Phase 6: AI Integration (Completed ✓)

### AI Vision Processing (Completed ✓)
- [x] Choose AI provider (Anthropic Claude selected)
- [x] Implement AI service client (modular interface with Claude and mock implementations)
- [x] Create prompt engineering for safety violation detection
- [x] Add confidence scoring for violations
- [x] Implement severity classification (critical/high/medium/low)
- [x] Add location detection within images
- [x] Build photo analysis queue/job system
- [x] Create endpoint to trigger AI analysis on photos
- [x] Store detected violations in database
- [x] Map AI findings to safety codes automatically

### Violation Management (Completed ✓)
- [x] Create violation listing endpoint (by inspection)
- [x] Create violation detail endpoint
- [x] Implement violation confirmation/dismissal workflow
- [x] Create violation update endpoint (add notes, change status)

## Phase 7: Frontend Development (In Progress)

> **Note:** Phase 7 (Reporting) has been deferred until after the UI is built. PDF generation is easier when HTML templates are already created.

See [UI_ROADMAP.md](./UI_ROADMAP.md) for detailed frontend implementation plan.
See [STYLE_GUIDE.md](./STYLE_GUIDE.md) for minimal styling guidelines.

**Tech Stack:**
- Go `html/template` for server-side rendering
- HTMX for dynamic interactions
- Alpine.js for client-side reactivity
- Minimal CSS (semantic HTML-first)

### Authentication UI ✓ COMPLETED
- [x] Login page (`/login`)
- [x] Registration page (`/register`)
- [x] Email verification page (`/verify`)
- [x] Forgot password page (`/forgot-password`)
- [x] Reset password page (`/reset-password`)
- [x] User profile page (`/profile`)

### Dashboard - Basic Complete
- [x] Basic dashboard page (`/dashboard`)
- [x] Navigation with user display name
- [ ] Organization selector/switcher
- [ ] Recent inspections widget
- [ ] Violation statistics/charts

### Organization & Project Management UI ✓ COMPLETED
- [x] Organization list page (`/organizations`)
- [x] Create organization form (`/organizations/new`)
- [x] Project list page (`/projects`)
- [x] Create project form (`/projects/new`)
- [x] Project detail/edit page (`/projects/:id`) with location data collection
- [ ] Organization detail page
- [ ] Organization member management UI

### Inspection Interface - Core Complete
- [x] Global inspection list (`/inspections`) - Aggregated across all projects
- [x] Project-specific inspection list (`/projects/:projectId/inspections`)
- [x] Create new inspection form (`/projects/:projectId/inspections/new`)
- [ ] Inspection detail view (`/inspections/:id`)
- [ ] Photo upload interface
- [ ] Photo gallery view
- [ ] Trigger AI analysis button
- [ ] Violation review interface
- [ ] Mark violations as confirmed/dismissed

### Reports UI - Not Started
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
