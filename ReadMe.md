# File Service API

Multi-tenant file storage service with S3 backend, direct-to-S3 uploads, and project-based access control.

## Features

- **Multi-tenant isolation** - Each client has isolated data
- **Project-based organization** - Files organized in projects (1 project = 1 S3 bucket)
- **Direct-to-S3 uploads** - Presigned URLs for fast, scalable file operations
- **Dual authentication** - JWT for users, API keys for automation
- **Project members** - Role-based access (viewer/editor/admin)
- **PostgreSQL storage** - Metadata in relational database
- **MVP-focused** - Security hardened with rate limiting and RBAC

## Quick Start

### 1. Install Dependencies

```bash
go mod download
```

### 2. Setup Database

```bash
# Install golang-migrate
brew install golang-migrate  # macOS

# Create database
createdb fileservice

# Run migrations
migrate -path migrations -database "postgres://user:password@localhost:5432/fileservice?sslmode=disable" up
```

### 3. Configure Environment

Create `.env` file:

```env
# Database
DB_HOST=localhost
DB_PORT=5432
DB_NAME=fileservice
DB_USER=fileservice_app
DB_PASSWORD=your_password
DB_SSL_MODE=disable

# AWS S3
REGION=ap-south-1
AWS_ACCESS_KEY_ID=your_aws_key
AWS_SECRET_ACCESS_KEY=your_aws_secret

# Security
JWT_SECRET=your_secret_min_32_chars
JWT_EXPIRY_MINUTES=60

# App
PORT=8080
DOWNLOAD_URL_TIME_LIMIT=15m
PAGINATION_PAGE_SIZE=100
```

### 4. Run

```bash
go run .
```

## API Endpoints

### Public

- `POST /auth/signup`
- `POST /auth/login`
- `GET /health`
- `GET /shares/:token/download-url` (viewer-only public link)

### JWT-protected

- Projects:
  - `GET /api/projects`
  - `POST /api/projects`
  - `GET /api/projects/:id`
  - `DELETE /api/projects/:id`
- Members:
  - `POST /api/projects/:project_id/members`
  - `GET /api/projects/:project_id/members`
  - `PUT /api/projects/:project_id/members/:user_id`
  - `DELETE /api/projects/:project_id/members/:user_id`
- API keys:
  - `POST /api/projects/:project_id/api-keys`
  - `GET /api/projects/:project_id/api-keys`
  - `DELETE /api/projects/:project_id/api-keys/:id`
- Files:
  - `POST /api/projects/:project_id/files/upload-url` (direct-to-S3)
  - `GET /api/projects/:project_id/files`
  - `GET /api/files/:id`
  - `GET /api/files/:id/download-url`
  - `DELETE /api/files/:id`
  - `POST /api/files/:id/share-link`
- Folders:
  - `POST /api/projects/:project_id/folders`
  - `GET /api/projects/:project_id/folders`
  - `DELETE /api/projects/:project_id/folders/:id`

### API key-protected

- Read:
  - `GET /api/key/projects/:project_id/files`
  - `GET /api/key/files/:id`
  - `GET /api/key/files/:id/download-url`
  - `GET /api/key/projects/:project_id/folders`
- Write:
  - `POST /api/key/projects/:project_id/files/upload-url`
  - `POST /api/key/projects/:project_id/folders`
- Delete:
  - `DELETE /api/key/files/:id`
  - `DELETE /api/key/projects/:project_id/folders/:id`

## Architecture

```
file-service/
├── main.go                    # Entry point
├── migrations/                # Database migrations
├── pkg/                       # Generic packages
│   ├── errors/               # Custom error types
│   ├── password/             # Password hashing (bcrypt)
│   ├── token/                # Token generation
│   └── validator/            # Input validation
└── internal/
    ├── config/               # Configuration
    ├── domain/               # Business models (user, client, project, file, apikey, share)
    ├── repository/           # Data access (PostgreSQL)
    ├── storage/s3/          # File storage (presigned URLs)
    ├── auth/                # Authentication & RBAC middleware
    ├── rbac/                # Role-based access control engine
    └── http/                # HTTP handlers & routes
```

## License

Proprietary
