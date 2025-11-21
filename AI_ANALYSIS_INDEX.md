# AI Analysis Documentation Index

## Quick Navigation

**Start here**: [AI_ANALYSIS_README.md](AI_ANALYSIS_README.md) - Overview and summary

### By Use Case

**I want to understand how a feature works**
→ [AI_ANALYSIS_FEATURES.md](AI_ANALYSIS_FEATURES.md)
- Detailed explanations of all 18 implemented features
- Code examples and architecture details
- How each component integrates

**I need to quickly look something up**
→ [AI_ANALYSIS_QUICK_REFERENCE.md](AI_ANALYSIS_QUICK_REFERENCE.md)
- Endpoints reference table
- Feature checklist
- Configuration variables
- Color codes and UI reference
- Performance notes

**I want to understand the flow of data/process**
→ [AI_ANALYSIS_FLOWS.md](AI_ANALYSIS_FLOWS.md)
- Complete photo analysis workflow diagram
- Job queue lifecycle
- Violation state machine
- HTMX polling diagram
- Database relationships
- Error handling flow

**I need to find which file does what**
→ [AI_ANALYSIS_FILE_GUIDE.md](AI_ANALYSIS_FILE_GUIDE.md)
- Detailed file descriptions
- Function mapping
- File tree structure
- Statistics and entry points

## Implementation Summary

18 features fully implemented:
- Photo upload with validation
- AI analysis via job queue
- HTMX real-time polling
- Violation detection and management
- Inspector review workflow
- Manual violation creation
- Smart re-analysis
- Regulation citations
- And 10 more...

See [AI_ANALYSIS_README.md](AI_ANALYSIS_README.md) for complete list.

## Key Files

**Handlers** (1,300+ lines):
- `internal/handlers/photos.go` - Analysis endpoints
- `internal/handlers/violations.go` - Review & management
- `internal/handlers/photo_analysis_job.go` - Background processing

**Templates** (585+ lines):
- `web/templates/pages/photo-detail.html` - Full review UI
- `web/templates/pages/inspection-detail.html` - Overview UI

**Services**:
- `internal/ai/ai.go` - AI interface
- `internal/ai/claude.go` - Claude integration
- `internal/queue/postgres.go` - Job queue
- `internal/queue/worker.go` - Worker pool

**Database**:
- `internal/migrations/` - Schema migrations (3 key files)
- `internal/database/` - Auto-generated queries

## Key Endpoints

```
POST   /api/photos/analyze              Trigger analysis
GET    /api/photos/analyze/{job_id}     Poll for results
POST   /api/violations/{id}/confirm     Confirm violation
POST   /api/violations/{id}/dismiss     Dismiss violation
POST   /api/violations/manual           Add manual violation
GET    /api/inspections/{id}/violations List violations
```

## Key Technologies

- Go 1.25.1 + Echo v4
- PostgreSQL (data + queue)
- Claude 3.5 Sonnet (vision)
- HTMX (polling)
- Alpine.js (reactivity)

## Architecture

```
Browser → HTMX Endpoints → Handlers → Job Queue → Claude AI → Database
```

## Getting Started

1. **Understand the Big Picture**: Read `AI_ANALYSIS_README.md`
2. **Deep Dive on a Feature**: Look in `AI_ANALYSIS_FEATURES.md`
3. **Visual Understanding**: Check `AI_ANALYSIS_FLOWS.md`
4. **Find a File**: Use `AI_ANALYSIS_FILE_GUIDE.md`
5. **Quick Reference**: Keep `AI_ANALYSIS_QUICK_REFERENCE.md` handy

## Documentation Stats

- **AI_ANALYSIS_README.md**: 3 KB (overview)
- **AI_ANALYSIS_FEATURES.md**: 18 KB (comprehensive)
- **AI_ANALYSIS_QUICK_REFERENCE.md**: 6 KB (lookup)
- **AI_ANALYSIS_FLOWS.md**: 18 KB (diagrams)
- **AI_ANALYSIS_FILE_GUIDE.md**: 16 KB (mapping)
- **AI_ANALYSIS_INDEX.md**: This file

**Total**: 61 KB of documentation

## Code Stats

- **Handlers**: 3 files, 1,300+ lines
- **Templates**: 2 files, 585+ lines
- **Services**: 5+ files
- **Tests**: 2 test files
- **Migrations**: 3 key migrations

**Total Implementation**: ~2,000+ lines of code

## Feature Checklist

All 18 features implemented:
- [x] Photo upload with validation
- [x] Asynchronous AI analysis
- [x] Real-time HTMX polling
- [x] Violation detection
- [x] Severity levels
- [x] Confidence scoring
- [x] Location annotations
- [x] Violation status workflow
- [x] Inspector context hints
- [x] Manual violation creation
- [x] Soft-delete for dismissed
- [x] Smart re-analysis
- [x] Regulation citations
- [x] Location-specific codes
- [x] Multiple views
- [x] HTMX forms
- [x] Job tracking
- [x] Rate limiting

## Recent Git Commits

Implementation timeline:
1. `feat: user may add context to assist ai violation detection`
2. `feat: improved regulation citation`
3. `feat: more intelligent handling of past violations when a new analysis is requested`
4. `feat: violations that are dismissed are treated as a soft delete`
5. `feat: manually create violations`

## Next Steps

For development or updates:
1. Refer to documentation for context
2. Check `AI_ANALYSIS_FILE_GUIDE.md` for file locations
3. Use `AI_ANALYSIS_QUICK_REFERENCE.md` for API reference
4. Review `AI_ANALYSIS_FLOWS.md` for process understanding

## Questions?

- **"How does X feature work?"** → `AI_ANALYSIS_FEATURES.md`
- **"What endpoint does Y?"** → `AI_ANALYSIS_QUICK_REFERENCE.md`
- **"Where is function Z?"** → `AI_ANALYSIS_FILE_GUIDE.md`
- **"What's the flow for X?"** → `AI_ANALYSIS_FLOWS.md`
- **"What's implemented?"** → This file (checklist above)

---

**Generated**: November 21, 2025
**Status**: Feature-complete (18/18 implemented)
**Quality**: Production-ready
