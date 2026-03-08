# Orka File Service API

Multi-tenant file service API with S3 storage, JWT authentication, project management, API keys, and email notifications.

## Tech Stack

- Go 1.24
- Echo v4 (HTTP framework)
- PostgreSQL (via `lib/pq`)
- AWS S3
- JWT authentication
- Email (Resend, SendGrid) with failover

## Getting Started

### Prerequisites

- Go 1.24+
- PostgreSQL database
- AWS S3 bucket

### Setup

1. Clone the repository:

```bash
git clone https://github.com/orkait/orkait-file-service.git
cd orkait-file-service/api
```

2. Install dependencies:

```bash
go mod download
```

3. Configure environment variables — copy `.env.example` to `.env` and fill in your values:

```bash
cp .env.example .env
```

Key variables:

| Variable | Description |
| --- | --- |
| `BUCKET_NAME` | S3 bucket name |
| `REGION` | AWS region |
| `AWS_ACCESS_KEY_ID` | AWS access key |
| `AWS_SECRET_ACCESS_KEY` | AWS secret key |
| `DATABASE_URL` | PostgreSQL connection string |
| `JWT_SECRET` | Secret for JWT signing |
| `MAIL_FROM` | Sender email address |
| `MAIL_PROVIDERS` | Comma-separated list (`resend,sendgrid`) |
| `PORT` | Server port (default: 8080) |

See `.env.example` for the full list.

## Usage

```bash
# Run directly
go run main.go

# Or use Make
make run

# Or use Air for hot reload
air
```
