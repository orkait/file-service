# Authentication & Email Invites

## Authentication Flow

### 1. User Registration (Required First Step)
Users MUST register before they can be invited to projects.

```bash
POST /auth/register
Content-Type: application/json

{
  "name": "John Doe",
  "email": "john@example.com",
  "password": "securepassword"
}
```

Response:
```json
{
  "client": {
    "id": "uuid",
    "name": "John Doe",
    "email": "john@example.com",
    "created_at": "2024-01-01T00:00:00Z"
  },
  "access_token": "eyJhbGc...",
  "refresh_token": "eyJhbGc..."
}
```

**What happens automatically:**
- Password is hashed using bcrypt
- Client record created in database
- Default project created automatically (via database trigger)
- Client added as owner of default project (via database trigger)
- JWT tokens generated
- Welcome email is sent (if mailer is configured)

### 2. User Login
```bash
POST /auth/login
Content-Type: application/json

{
  "email": "john@example.com",
  "password": "securepassword"
}
```

### 3. Token Refresh
Access tokens expire after 15 minutes. Use refresh token to get new access token:

```bash
POST /auth/refresh
Content-Type: application/json

{
  "refresh_token": "eyJhbGc..."
}
```

### 4. Forgot Password

```bash
POST /auth/forgot-password
Content-Type: application/json

{
  "email": "john@example.com"
}
```

This sends a reset link by email (expires in 1 hour).

### 5. Reset Password

```bash
POST /auth/reset-password
Content-Type: application/json

{
  "token": "<reset-token>",
  "new_password": "newsecurepassword"
}
```

### 6. Delete Client Account

Soft delete (default): account is paused now and permanently deleted after 90 days.

```bash
DELETE /api/clients/me
Authorization: Bearer <access_token>
```

Force delete (immediate): removes account, projects, records, and asset objects immediately.

```bash
DELETE /api/clients/me?force_delete=true
Authorization: Bearer <access_token>
```

You can also create client using:

```bash
POST /clients
Content-Type: application/json

{
  "name": "John Doe",
  "email": "john@example.com",
  "password": "securepassword"
}
```

### 7. Client CRUD

```bash
GET /api/clients
Authorization: Bearer <access_token>
```

Query params:
- `limit` (default 50, max 200)
- `offset` (default 0)
- `q` (optional name/email search)

```bash
GET /api/clients/me
Authorization: Bearer <access_token>
```

```bash
GET /api/clients/:id
Authorization: Bearer <access_token>
```

```bash
PATCH /api/clients/me
Authorization: Bearer <access_token>
Content-Type: application/json

{
  "name": "Updated Name",
  "email": "updated@example.com",
  "password": "newpassword123"
}
```

## Email Invite System

### How It Works

Invites send an email notification if the mailer is configured.

The invite flow:
1. User A (project owner) wants to invite User B
2. User B MUST already be registered in the system
3. User A invites User B by email address
4. System looks up User B by email
5. If found, User B is added to the project and receives email
6. User B can now access the project

### Invite a Member

```bash
POST /api/projects/:project_id/members
Authorization: Bearer <access_token>
Content-Type: application/json

{
  "email": "invitee@example.com",
  "role": "editor"
}
```

**Roles:**
- `owner` - Full control (can invite/remove members, delete project)
- `editor` - Can upload/delete assets
- `viewer` - Read-only access

**Requirements:**
- Only project owner can invite members
- Invitee must already be registered
- Invitee email must exist in clients table

### List Project Members

```bash
GET /api/projects/:project_id/members
Authorization: Bearer <access_token>
```

### Remove a Member

```bash
DELETE /api/projects/:project_id/members/:member_id
Authorization: Bearer <access_token>
```

**Note:** Cannot remove project owner

## Complete Workflow Example

### Scenario: User A invites User B to collaborate

**Step 1: User B registers (if not already registered)**
```bash
curl -X POST http://localhost:8080/auth/register \
  -H "Content-Type: application/json" \
  -d '{
    "name": "User B",
    "email": "userb@example.com",
    "password": "password123"
  }'
```

**Step 2: User A logs in**
```bash
curl -X POST http://localhost:8080/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "email": "usera@example.com",
    "password": "password123"
  }'
```

Save the `access_token` from response.

**Step 3: User A gets their projects**
```bash
curl -X GET http://localhost:8080/api/projects \
  -H "Authorization: Bearer <access_token>"
```

**Step 4: User A invites User B**
```bash
curl -X POST http://localhost:8080/api/projects/<project_id>/members \
  -H "Authorization: Bearer <access_token>" \
  -H "Content-Type: application/json" \
  -d '{
    "email": "userb@example.com",
    "role": "editor"
  }'
```

**Step 5: User B logs in and sees the shared project**
```bash
curl -X POST http://localhost:8080/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "email": "userb@example.com",
    "password": "password123"
  }'

# Then get projects
curl -X GET http://localhost:8080/api/projects \
  -H "Authorization: Bearer <userb_access_token>"
```

User B will now see both their default project AND the project they were invited to.

## API Key Authentication (Alternative)

For programmatic access without user login:

### Create API Key
```bash
POST /api/api-keys
Authorization: Bearer <access_token>
Content-Type: application/json

{
  "name": "Production API Key",
  "project_id": "uuid",
  "permissions": ["read", "write"]
}
```

Response includes the actual API key (only shown once):
```json
{
  "api_key": "ork_live_abc123...",
  "key_prefix": "ork_live_abc",
  "id": "uuid",
  "name": "Production API Key"
}
```

### Use API Key
```bash
GET /v1/assets?project_id=<uuid>
X-API-Key: ork_live_abc123...
```

**Note:** API key routes use `/v1` prefix instead of `/api`

## Security Notes

1. **Passwords:** Hashed with bcrypt before storage
2. **JWT Tokens:**
   - Access token: 15 minutes expiry
   - Refresh token: 7 days expiry
3. **API Keys:** Hashed before storage (like passwords)
4. **Multi-tenant:** All queries scoped by client_id
5. **Project Access:** Verified via project_members table

## Common Issues

### "client not found with that email"
- The user you're trying to invite hasn't registered yet
- They need to call `/auth/register` first

### "only project owner can invite members"
- Only users with role='owner' can invite
- Check your role: `GET /api/projects/:project_id/members`

### "invalid credentials"
- Wrong email or password
- Email is normalized to lowercase

### "token expired"
- Access token expired (15 min)
- Use refresh token to get new access token
