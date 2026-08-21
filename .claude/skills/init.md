# init

Initialize CLAUDE.md untuk project baru.

## When to invoke

- User mengetik `/init`
- Setup project baru yang belum punya CLAUDE.md
- Migrate existing project ke Claude Code

## What this does

Generate `CLAUDE.md` file dengan template untuk:
1. **Project identity** - Nama, deskripsi, tech stack
2. **Architecture overview** - Struktur direktori, pola kode
3. **Development workflow** - Build, test, deploy commands
4. **Coding conventions** - Style guide, best practices
5. **Common tasks** - FAQ, troubleshooting

## Template Structure

```markdown
# Project Name

Brief description of what this project does.

## Tech Stack

- Backend: Go 1.21+
- Frontend: Vue 3 + TypeScript
- Database: PostgreSQL 15
- Cache: Redis 7

## Architecture

Describe monorepo/microservices structure, key directories.

## Development

### Setup
\`\`\`bash
# Clone and install
git clone ...
make setup
\`\`\`

### Common Commands
\`\`\`bash
make dev          # Start dev server
make test         # Run tests
make build        # Build for production
\`\`\`

## Coding Conventions

- **Go:** Use `net/http`, avoid frameworks
- **TypeScript:** Strict mode, no `any`
- **Git:** Conventional commits

## Common Tasks

### Add new API endpoint
1. Define handler in `handlers/`
2. Register route in `main.go`
3. Add tests

### Run migrations
\`\`\`bash
make migrate-up
\`\`\`

## Reference

- API Docs: /docs/api.md
- Architecture: /docs/architecture.md
\`\`\`

## Execution

Run interactively to gather project info:

```bash
echo "Initializing CLAUDE.md..."
read -p "Project name: " PROJECT_NAME
read -p "Primary language (Go/TypeScript/Python): " PRIMARY_LANG
read -p "Framework (if any): " FRAMEWORK
# ... generate CLAUDE.md
```

## Integration

- Run `/init` on new project setup
- Update CLAUDE.md as project evolves
- Reference in onboarding docs
