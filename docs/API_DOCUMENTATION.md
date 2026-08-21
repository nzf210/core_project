# API Documentation

WCH Platform menggunakan **Swagger/OpenAPI 3.0** untuk dokumentasi REST API.

## Quick Start

**View documentation:**
```bash
# Development
http://localhost:8000/swagger/

# Staging
http://157.15.40.27:21000/swagger/
```

## Generate/Update Documentation

```bash
# Install swag CLI (one-time)
go install github.com/swaggo/swag/cmd/swag@latest

# Generate Swagger docs from code annotations
make swagger

# Or manually:
swag init -g services/api-gateway/main.go -o docs/swagger
```

Swagger JSON/YAML files akan di-generate di `docs/swagger/`.

## Adding API Documentation

Tambahkan Swagger annotations di handler functions:

```go
// @Summary      Login user
// @Description  Authenticate user with email and password
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        request body LoginRequest true "Login credentials"
// @Success      200 {object} LoginResponse
// @Failure      401 {object} ErrorResponse
// @Router       /auth/login [post]
func handleLogin(w http.ResponseWriter, r *http.Request) {
    // handler implementation
}
```

**Common annotations:**
- `@Summary` — Short description (1 line)
- `@Description` — Detailed description
- `@Tags` — Group endpoints by tag
- `@Accept` — Request content type
- `@Produce` — Response content type
- `@Param` — Request parameters (query, path, header, body)
- `@Success` — Success response (status code + type)
- `@Failure` — Error response (status code + type)
- `@Router` — Endpoint path + HTTP method

## API Structure

**Base URL:**
- Dev: `http://localhost:8000`
- Staging: `http://157.15.40.27:21000`
- Production: `https://api.wch.id`

**Authentication:**
```
Authorization: Bearer <jwt_token>
X-Tenant-ID: <tenant_uuid>
```

**Endpoints by service:**
- `/auth/*` — Authentication (login, register, OTP)
- `/api/umkm/*` — UMKM app (POS, accounting, chatbot)
- `/api/campaign/*` — Campaign app (volunteers, voters)
- `/api/billing/*` — Billing & subscriptions
- `/api/superadmin/*` — Superadmin dashboard

## Response Format

**Success:**
```json
{
  "success": true,
  "message": "Operation successful",
  "data": { ... }
}
```

**Error:**
```json
{
  "success": false,
  "message": "Error description"
}
```

## Testing APIs

**cURL:**
```bash
curl -X POST http://localhost:8000/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"test@example.com","password":"pass123"}'
```

**Postman Collection:**
Import `docs/postman/WCH-Platform.postman_collection.json` (TODO)

## Swagger UI Configuration

Swagger UI di-serve via api-gateway di `/swagger/` endpoint dengan:
- Dark theme support
- Try-it-out functionality
- Model schema explorer
- Authentication header injection

## Future Enhancements

- [ ] Generate Postman collection from OpenAPI spec
- [ ] Add request/response examples for complex endpoints
- [ ] Setup API versioning (v1, v2)
- [ ] Add rate limit documentation
- [ ] WebSocket endpoint documentation
