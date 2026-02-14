# Architecture

Multi-tenant file storage API. Users sign up, get a tenant (client), organize files in projects, and upload/download via S3 presigned URLs. Two auth paths: JWT for humans, API keys for automation.

```
+--------+         +-------------+         +-----------+
| Client | ------> | File Service | ------> | PostgreSQL |
| (app)  | <------ |   (Go/Echo)  | ------> |            |
+--------+         +-------------+         +-----------+
     |                    |
     |  presigned URL     |  presigned URL generation
     |                    |
     +----------> +-------+-------+
                  |   AWS S3      |
                  | (file storage)|
                  +---------------+
```

---

## Layering

```
main.go                          Wires everything, starts server
internal/config/                 Reads env vars, validates, returns typed config
internal/domain/                 Pure data structs, no dependencies
internal/repository/             Interface definitions + Postgres implementations
internal/storage/s3/             AWS S3 presigned URL generation, bucket ops
internal/auth/                   JWT service, API key service, auth + RBAC middleware
internal/rbac/                   Permission engine, role hierarchy, capability matrix
internal/http/                   Echo server setup, route registration, handlers
pkg/errors/                      Sentinel errors and AppError type
pkg/password/                    bcrypt hash/verify
pkg/token/                       Crypto-random token generation (API keys, share tokens)
pkg/validator/                   Stateless input validation (email, password, filenames)
```

**Why this layout**: Domain has zero imports so it never causes circular deps. Repository defines interfaces that handlers depend on, not the Postgres concrete types. Handlers never import `postgres/` directly. `pkg/` holds generic utilities reusable outside this service. `internal/` holds service-specific code that Go prevents external packages from importing.

```
+-----------+     +------------+     +----------+     +---------+
|  handler  | --> | repository | --> |  domain  |     |   pkg   |
| (http/)   |     | (interfaces)|    | (structs)|     | (utils) |
+-----------+     +------------+     +----------+     +---------+
      |                 ^                  ^               ^
      |                 |                  |               |
      v           +-----+------+           |               |
  +------+        |  postgres/ |  ---------+               |
  | auth | -----> | (concrete) |  -------------------------+
  +------+        +------------+
```

All arrows point inward/downward. Nothing in domain imports anything else.

---

## Domain Models

**User** holds email, password hash, and a client_id linking them to their tenant. Update fields use pointers so the handler can distinguish "not sent" from "empty string".

**Client** represents a tenant/organization. Created automatically during signup. Has an owner_user_id. One client can have many users and projects.

**Project** groups files under one S3 bucket. Each project has members with roles. The s3_bucket_name is auto-generated from client+project UUID prefixes to guarantee uniqueness without user input.

**File** stores metadata only (name, size, mime type, s3 key). The actual bytes live in S3. Files belong to a project and optionally to a folder.

**Folder** is a metadata record with an s3_prefix. Folders can nest via parent_folder_id. Deleting a folder cascades to children via the database FK.

**APIKey** stores a SHA256 hash of the key, never the key itself. A prefix (first 11 chars) is kept for display in lists. Each key is scoped to one project and carries a permission array (read, write, delete).

**ShareLink** maps a random 64-char hex token to a file. Public endpoint, no auth needed. Deleting the record revokes the link.

```
                         +----------+
                         |  Client  |
                         | (tenant) |
                         +----+-----+
                              |
                    +---------+---------+
                    |                   |
               +----+----+        +----+----+
               |  User   |        | Project |
               +---------+        +----+----+
                    |                   |
                    |    +--------------+---------------+
                    |    |              |               |
               +----+---+---+    +-----+-----+   +----+----+
               |  project_   |    |   File    |   | API Key |
               |  members    |    +-----+-----+   +---------+
               | (role: v/e/a)|         |
               +-------------+    +-----+-----+
                                  |   Folder  |
                                  +-----------+
                                        |
                                  +-----+------+
                                  | ShareLink  |
                                  +------------+
```

---

## Multi-Tenancy

Every user belongs to exactly one client. Users access projects through the project_members table. Without a membership row, there is no access. API keys are scoped to a single project, so they cannot cross project boundaries. Handlers always verify client_id or membership before returning data. Cascade deletes in the schema ensure that removing a client removes all its projects, members, files, and keys with no orphans.

**Why client_id on user is immutable**: Transferring users between tenants would break ownership assumptions across projects and files. Simpler to keep it fixed.

```
Tenant Isolation Boundaries
============================

 Tenant A (client_id=aaa)          Tenant B (client_id=bbb)
+---------------------------+     +---------------------------+
| User A1    User A2        |     | User B1                  |
|                           |     |                          |
| Project "default"         |     | Project "default"        |
|   +-- files/              |     |   +-- files/             |
|   +-- api-keys/           |     |   +-- api-keys/          |
|   +-- members: A1(admin)  |     |   +-- members: B1(admin) |
|                A2(viewer) |     |                          |
|                           |     | Project "staging"        |
| Project "prod"            |     |   +-- files/             |
|   +-- files/              |     |   +-- members: B1(admin) |
|   +-- members: A1(admin)  |     +---------------------------+
+---------------------------+

 X  User B1 cannot see Tenant A's projects (no membership row)
 X  API key for Project "default" in A cannot access Project "prod"
```

---

## Signup Flow

Signup creates four records atomically in one Postgres transaction: user, client, default project, and a membership row making the user an admin of that project.

```
Client                    Server                        Postgres            S3
  |                         |                              |                 |
  |  POST /auth/signup      |                              |                 |
  |  {email, password}      |                              |                 |
  | ----------------------> |                              |                 |
  |                         |  hash password (bcrypt)      |                 |
  |                         |                              |                 |
  |                         |  BEGIN TX                    |                 |
  |                         | ---------------------------> |                 |
  |                         |  INSERT user (client_id=X)   |                 |
  |                         | ---------------------------> |  (FK deferred)  |
  |                         |  INSERT client (id=X)        |                 |
  |                         | ---------------------------> |                 |
  |                         |  INSERT project (default)    |                 |
  |                         | ---------------------------> |                 |
  |                         |  INSERT member (admin)       |                 |
  |                         | ---------------------------> |                 |
  |                         |  COMMIT                      |                 |
  |                         | ---------------------------> |  (FKs checked)  |
  |                         |                              |                 |
  |                         |  CreateBucket (async)        |                 |
  |                         | ------------------------------------------>  |
  |                         |                              |                 |
  |                         |  Generate JWT                |                 |
  |                         |                              |                 |
  |  {user_id, token, ...}  |                              |                 |
  | <---------------------- |                              |                 |
```

**Why a transaction**: If any step fails (duplicate email, DB error), nothing is committed. The user never ends up with a half-created account.

**Why pre-generated UUIDs**: The user row references client_id before the client row exists. PostgreSQL supports this with `DEFERRABLE INITIALLY DEFERRED` foreign keys, which delay constraint checks until commit. Both UUIDs are generated before any INSERT runs, so each row can reference the other.

After the transaction commits, an S3 bucket is created for the default project. This runs outside the transaction because S3 is an external service that cannot participate in a Postgres transaction. If bucket creation fails, the signup still succeeds and the bucket can be retried later. The error is logged.

---

## Authentication

### JWT Path

User logs in with email+password. Server verifies bcrypt hash, generates an HS256 JWT containing user_id, client_id, and email. The token expires after a configurable duration (default 60 minutes).

**Why HS256**: Single service, no need for asymmetric keys. Simpler and faster than RS256. The secret must be at least 32 chars, enforced at startup.

**Why client_id in the JWT**: Handlers need the tenant context on every request. Embedding it avoids a database lookup per request.

### API Key Path

Admins create API keys scoped to a project. The full key (`pk_<40 hex chars>`) is shown once during creation. The server stores only the SHA256 hash. On each request, the middleware hashes the provided key and looks it up. If found, it checks that the key is not revoked, not expired, and has the required permission.

**Why hash-only storage**: If the database leaks, attackers cannot reconstruct valid keys. Same principle as password hashing but SHA256 is sufficient here because the input has high entropy (20 random bytes).

**Why a prefix**: Users need to identify which key is which in the list view. Storing the first 11 characters of the key allows display without exposing the full secret.

**Why last_used_at is updated asynchronously**: The update is fire-and-forget in a goroutine. It should not add latency to the API response or cause a failure if the update query errors.

```
JWT Auth Flow
=============
Client                         Server                     Postgres
  |                              |                            |
  |  Authorization: Bearer <jwt> |                            |
  | ---------------------------> |                            |
  |                              |  Verify HS256 signature    |
  |                              |  Extract: user_id,         |
  |                              |    client_id, email        |
  |                              |  Check expiry              |
  |                              |                            |
  |                              |  Store in echo context     |
  |                              |  --> next middleware        |
  |                              |                            |


API Key Auth Flow
=================
Client                         Server                     Postgres
  |                              |                            |
  |  X-API-Key: pk_a1b2c3...    |                            |
  | ---------------------------> |                            |
  |                              |  SHA256(pk_a1b2c3...)      |
  |                              |                            |
  |                              |  SELECT * FROM api_keys    |
  |                              |    WHERE key_hash = ?      |
  |                              | -------------------------> |
  |                              | <------------------------- |
  |                              |                            |
  |                              |  Check: not revoked?       |
  |                              |  Check: not expired?       |
  |                              |  Check: has permission?    |
  |                              |                            |
  |                              |  UPDATE last_used_at       |
  |                              |    (async goroutine)       |
  |                              |                            |
  |                              |  Store project_id,         |
  |                              |    permissions in context  |
  |                              |  --> next middleware        |
```

---

## RBAC

Two authorization models exist because JWT users and API keys have different trust levels and access patterns.

### Role-Based (JWT Users)

Three roles with a numeric hierarchy: viewer (1) < editor (2) < admin (3). A user's role is per-project, stored in project_members. The RBAC engine has a capability matrix mapping role + resource + action to allowed/denied.

- **Viewer**: Read files, folders, members, keys.
- **Editor**: Everything viewer can do, plus write/delete files and folders.
- **Admin**: Everything editor can do, plus manage members and API keys.

**Why numeric levels**: Role elevation checks (`admin > editor`) reduce to integer comparison. No string parsing or ordered lists needed.

### Permission-Based (API Keys)

API keys carry explicit permissions (read, write, delete) and are restricted to file and folder resources only. Member and API key management is JWT-only because those operations require human accountability.

**Why two models**: Roles make sense for humans (one role per project, capabilities implied). Permissions make sense for automation (explicit list, no concept of "role"). The RBAC engine handles both through a single `Authorize` method with branching logic.

```
Role Capability Matrix (JWT Users)
===================================

              | File        | Folder      | API Key     | Member      |
              | R  W  D     | R  W  D     | R  W  D  M  | R  W  D  M  |
--------------+-------------+-------------+-------------+-------------+
 Admin (3)    | Y  Y  Y     | Y  Y  Y     | Y  Y  Y  Y  | Y  Y  Y  Y  |
 Editor (2)   | Y  Y  Y     | Y  Y  Y     | Y  Y  -  -  | Y  -  -  -  |
 Viewer (1)   | Y  -  -     | Y  -  -     | Y  -  -  -  | Y  -  -  -  |

R=Read  W=Write  D=Delete  M=Manage  Y=Allowed  -=Denied


Permission-to-Action Map (API Keys)
=====================================

 Permission    Action    Allowed Resources
 ----------    ------    -----------------
 read     -->  read      file, folder
 write    -->  write     file, folder
 delete   -->  delete    file, folder

 API keys CANNOT access: api_key, member (JWT-only)
```

### Middleware Structure

`RequireJWT` / `RequireAPIKey`: Applied at the route group level. Extracts identity and stores it in the echo context.

`RequireProjectRole(minRole)`: For JWT users, looks up the user's membership in the project and checks their role meets the minimum. For API keys, skips this check because API key permissions are validated separately.

`RequireProjectAction(resource, action)`: For JWT users, uses the capability matrix. For API keys, maps the permission to the action and checks.

`RequireProjectRoleForFile(minRole)`: Same as RequireProjectRole but first resolves the project_id by looking up the file record. Used on endpoints like `GET /files/:id` where the URL has a file ID, not a project ID.

**Why resolveJWTSubject helper**: Three middleware methods repeated the same 20-line sequence (get user ID, look up membership, build subject, store in context). Extracted to a single method to avoid divergence if the logic changes.

```
Request Middleware Pipeline
============================

  Request
    |
    v
+------------------+
| Echo Logger      |  Log method, path, status, latency
+------------------+
    |
    v
+------------------+
| Echo Recover     |  Catch panics, return 500
+------------------+
    |
    v
+------------------+
| Echo CORS        |  Cross-origin headers
+------------------+
    |
    +---> /auth/*  (no more middleware, public)
    |
    +---> /shares/* (no more middleware, public)
    |
    +---> /api/* (JWT routes)
    |       |
    |       v
    |   +------------------+
    |   | RequireJWT       |  Verify token, set user_id/client_id
    |   +------------------+
    |       |
    |       v
    |   +------------------+
    |   | RequireProject   |  Look up membership, check role
    |   | Role/Action      |  (or RequireProjectRoleForFile)
    |   +------------------+
    |       |
    |       v
    |     Handler
    |
    +---> /api/key/* (API key routes)
            |
            v
        +------------------+
        | RequireAPIKey    |  Hash key, DB lookup, check
        | (permission)     |  active + permission
        +------------------+
            |
            v
          Handler (same handler functions as JWT routes)
```

---

## File Operations

### Upload

The client requests a presigned PUT URL. The server validates the filename (no path traversal, no control chars), checks size limits, builds the S3 key, generates the presigned URL, and creates (or updates) the file metadata record. The client then uploads directly to S3 using the presigned URL.

```
File Upload (presigned URL)
============================

Client                      Server                   Postgres        S3
  |                           |                          |             |
  | POST /files/upload-url    |                          |             |
  | {name, size, mime, folder}|                          |             |
  | ----------------------->  |                          |             |
  |                           | validate filename        |             |
  |                           | validate size <= 100GB   |             |
  |                           | build s3_key             |             |
  |                           |                          |             |
  |                           | GeneratePresignedPutURL  |             |
  |                           | ------------------------------------------->
  |                           | <-------------------------------------------
  |                           |                          |             |
  |                           | INSERT/UPDATE file       |             |
  |                           | -----------------------> |             |
  |                           |                          |             |
  | {upload_url, file_id}     |                          |             |
  | <------------------------ |                          |             |
  |                                                                    |
  | PUT <upload_url>                                                   |
  | (binary file data, direct to S3, server not involved)              |
  | -----------------------------------------------------------------> |
  | <----------------------------------------------------------------- |
  |   200 OK                                                           |
```

**Why presigned URLs**: The server never handles file bytes. This eliminates bandwidth costs on the API server, removes file size limits imposed by the HTTP framework, and lets S3 handle the heavy lifting. Upload and download scale independently of the API server.

**Why metadata is created before the upload completes**: The presigned URL is valid for 15 minutes. If the client never uploads, the metadata record exists but the S3 object does not. This is an acceptable trade-off. Cleaning up orphaned records can be a background job. The alternative (creating metadata after upload) would require a callback or polling mechanism, adding complexity.

### Download

The client requests a presigned GET URL for a file. The server checks authorization, fetches the file metadata to get the S3 key and bucket name, generates the presigned URL, and returns it. The client downloads directly from S3.

```
File Download (presigned URL)
==============================

Client                      Server                   Postgres        S3
  |                           |                          |             |
  | GET /files/:id/           |                          |             |
  |     download-url          |                          |             |
  | ----------------------->  |                          |             |
  |                           | SELECT file (s3_key)     |             |
  |                           | -----------------------> |             |
  |                           | SELECT project (bucket)  |             |
  |                           | -----------------------> |             |
  |                           |                          |             |
  |                           | GeneratePresignedGetURL  |             |
  |                           | ------------------------------------------->
  |                           | <-------------------------------------------
  |                           |                          |             |
  | {download_url, file_name} |                          |             |
  | <------------------------ |                          |             |
  |                                                                    |
  | GET <download_url>        (direct from S3)                         |
  | -----------------------------------------------------------------> |
  | <----------------------------------------------------------------- |
  |   file bytes                                                       |
```

### Folders

Folders are metadata-only records with an s3_prefix field. Creating a folder does not create an S3 object. Files reference their folder via folder_id. Deleting a folder deletes all S3 objects matching the prefix, then removes the folder record.

**Why metadata-only folders**: S3 is a flat object store. Folder semantics are imposed by the application through key prefixes. Storing folder records in the database allows querying folder contents without listing S3 objects.

```
S3 Key Structure
=================

Bucket: "a1b2c3d4-e5f6g7h8"  (client_prefix-project_prefix)
  |
  +-- documents/                     <-- folder (s3_prefix in DB)
  |     +-- report.pdf               <-- file (s3_key = "documents/report.pdf")
  |     +-- notes.txt                <-- file (s3_key = "documents/notes.txt")
  |
  +-- images/                        <-- folder
  |     +-- logo.png                 <-- file
  |
  +-- readme.txt                     <-- file (no folder, s3_key = "readme.txt")
```

---

## Public Sharing

An editor creates a share link for a file. The server generates a 64-char hex token (32 random bytes) and stores it with the file ID. Anyone with the token can hit the public endpoint to get a presigned download URL. No authentication required.

**Why no expiry on the share record**: The presigned download URL has its own 15-minute expiry. The share token just maps to a file. To revoke access, delete the share link record. This keeps the model simple: one record, one token, delete to revoke.

**Why 32 bytes of randomness**: 256 bits of entropy makes brute-forcing infeasible. The token space is large enough that collisions are negligible.

```
Public Share Flow
==================

Editor                    Server              Postgres        S3        Anyone
  |                         |                    |             |           |
  | POST /files/:id/        |                    |             |           |
  |   share-link            |                    |             |           |
  | ----------------------> |                    |             |           |
  |                         | generate 64-char   |             |           |
  |                         |   hex token        |             |           |
  |                         | INSERT share_link  |             |           |
  |                         | -----------------> |             |           |
  | {token, share_url}      |                    |             |           |
  | <---------------------- |                    |             |           |
  |                         |                    |             |           |
  |   (share URL given to anyone)                |             |           |
  |                         |                    |             |           |
  |                         |  GET /shares/<token>/download-url|           |
  |                         | <-------------------------------------------+
  |                         | SELECT share_link  |             |           |
  |                         | -----------------> |             |           |
  |                         | SELECT file, proj  |             |           |
  |                         | -----------------> |             |           |
  |                         | presigned GET URL  |             |           |
  |                         | ---------------------------->    |           |
  |                         | {download_url}     |             |           |
  |                         | ------------------------------------------> |
  |                         |                    |             |           |
  |                         |                    |       GET <presigned>   |
  |                         |                    |             | <-------- |
  |                         |                    |             | --------> |
  |                         |                    |             | file bytes|
```

---

## Error Handling

Sentinel errors (`ErrNotFound`, `ErrConflict`, `ErrEmailExists`, etc.) are defined in `pkg/errors`. Repositories wrap database errors into these sentinels. Handlers check with `errors.Is()` and map to HTTP status codes.

**Why sentinel errors instead of string comparison**: `errors.Is` works through wrapped error chains and survives message changes. String comparison (`err.Error() == "email already exists"`) breaks silently if the message is reworded.

**Why AppError struct**: Sentinels identify the category. The AppError wraps the sentinel with a human-readable message and an error code string. This lets the handler return a specific message to the client while the sentinel drives the HTTP status code decision.

**Why isUniqueViolation uses pgconn.PgError**: Postgres returns structured error objects with a code field. Checking `pgErr.Code == "23505"` is locale-independent and version-stable. String-matching the error message would break across Postgres versions or locale settings.

```
Error Flow: Repository --> Handler --> HTTP Response
=====================================================

  Postgres                 Repository             Handler              Client
     |                        |                      |                    |
     | unique_violation       |                      |                    |
     | (pgErr code 23505)     |                      |                    |
     | ---------------------> |                      |                    |
     |                        | isUniqueViolation()  |                    |
     |                        | return AppError{     |                    |
     |                        |   Code: "CONFLICT",  |                    |
     |                        |   Err: ErrConflict}  |                    |
     |                        | -------------------> |                    |
     |                        |                      | errors.Is(         |
     |                        |                      |   err, ErrConflict)|
     |                        |                      | --> true           |
     |                        |                      |                    |
     |                        |                      | 409 Conflict       |
     |                        |                      | {error: "..."}     |
     |                        |                      | -----------------> |


  Postgres                 Repository             Handler              Client
     |                        |                      |                    |
     | ErrNoRows              |                      |                    |
     | ---------------------> |                      |                    |
     |                        | return AppError{     |                    |
     |                        |   Code: "NOT_FOUND", |                    |
     |                        |   Err: ErrNotFound}  |                    |
     |                        | -------------------> |                    |
     |                        |                      | 404 Not Found      |
     |                        |                      | -----------------> |
```

---

## Configuration

All config is loaded from environment variables at startup. Required vars (DB_PASSWORD, JWT_SECRET, AWS credentials) panic if missing. Optional vars have sensible defaults. The config struct is validated once, then passed by pointer to all components.

**Why env vars, not a config file**: Twelve-factor app convention. Works with Docker, Kubernetes, and CI/CD without mounting files. Secrets stay out of the repo.

**Why panic on missing required vars**: The service cannot function without a database password or JWT secret. Failing fast at startup is better than returning cryptic errors on the first request.

---

## Database

PostgreSQL with the pgx driver and connection pooling (pgxpool). Direct SQL queries, no ORM.

**Why no ORM**: The queries here are straightforward CRUD. An ORM adds a learning curve, hides query performance, and complicates transactions. Direct SQL gives full control and is easier to debug.

**Why pgx instead of database/sql**: pgx is Postgres-native, supports Postgres-specific types (arrays, UUIDs), connection pooling, and typed error handling (pgconn.PgError). database/sql would require additional drivers and type conversion.

**Why connection pooling (min 5, max 25)**: Prevents connection storms under load. Min connections keep warm connections ready. Max connections prevent exhausting Postgres connection slots.

```
Database Schema (FK relationships)
====================================

  +----------+        +----------+        +-----------+
  |  users   | -----> |  clients | <----- |  projects |
  +----------+  1:1   +----------+  1:N   +-----------+
  | id (PK)  |        | id (PK)  |        | id (PK)   |
  | email    |        | owner_   |        | client_id  |
  | pass_hash|        |  user_id |        | name       |
  | client_id|------->|          |        | s3_bucket  |
  +----------+        +----------+        | is_default |
       |                                  +-----------+
       |                                    |  |  |  |
       |    +-------------------------------+  |  |  |
       |    |       +------------------+       |  |  |
       |    |       |                  |       |  |  |
       v    v       v                  v       |  |  |
  +------------------+          +-----------+  |  |  |
  | project_members  |          | api_keys  |<-+  |  |
  +------------------+          +-----------+     |  |
  | project_id (FK)  |          | project_id|     |  |
  | user_id (FK)     |          | key_hash  |     |  |
  | role             |          | key_prefix|     |  |
  | invited_by       |          | perms[]   |     |  |
  +------------------+          | expires_at|     |  |
                                | revoked_at|     |  |
                                +-----------+     |  |
                                                  |  |
                         +----------+<------------+  |
                         |  files   |                |
                         +----------+                |
                         | id (PK)  |                |
                         | project_id|               |
                         | folder_id |---+           |
                         | name      |   |           |
                         | s3_key    |   |           |
                         | size_bytes|   |           |
                         | mime_type |   |           |
                         +----------+   |           |
                              |         |           |
                              v         v           |
                         +-------------+            |
                         |  folders    |<-----------+
                         +-------------+
                         | id (PK)     |
                         | project_id  |
                         | parent_     |---+
                         |  folder_id  |   | (self-ref)
                         | name        |<--+
                         | s3_prefix   |
                         +-------------+

                         +-------------+
                         | share_links |
                         +-------------+
                         | id (PK)     |
                         | file_id (FK)|-----> files
                         | token       |
                         | created_by  |-----> users
                         +-------------+

  All FKs use ON DELETE CASCADE.
  users.client_id is DEFERRABLE INITIALLY DEFERRED (for signup TX).
```

Schema highlights:
- Cascading deletes on all foreign keys ensure no orphaned records.
- Partial index on active API keys speeds up the auth lookup (most keys will eventually be revoked or expired).
- `updated_at` trigger fires automatically so application code never needs to set it.
- `DEFERRABLE INITIALLY DEFERRED` on user.client_id allows the signup transaction to insert user before client.

---

## S3 Integration

Uses AWS SDK v1. Presigned URLs are generated server-side with a configurable expiry (default 15 minutes). Bucket creation waits until the bucket exists (SDK waiter). Folder deletion lists all objects with the prefix and deletes them in a loop.

**Why AWS SDK v1**: The project was started with v1. v2 has a different API surface. Migration would be a separate effort with no functional benefit for this use case.

**Why 15-minute presigned URL expiry**: Long enough for large uploads over slow connections. Short enough that a leaked URL has limited exposure window.

```
API Key Lifecycle
==================

 Creation (admin, JWT auth)            Usage (automation)           Revocation (admin)
 ============================          ==================           ==================

 POST /api-keys                        X-API-Key: pk_a1b2...       DELETE /api-keys/:id
   |                                      |                           |
   v                                      v                           v
 Generate 20 random bytes              SHA256(full key)             UPDATE api_keys
 Format: pk_<40 hex>                   DB lookup by hash            SET revoked_at=NOW()
 Store: SHA256 hash + prefix           Check active + perms            revoked_by=user
   |                                      |                           |
   v                                      v                           v
 Return full key ONCE                  Grant/deny access            Key becomes inactive
 (never retrievable again)             Update last_used_at          (record kept for audit)
```

---

## Dependency Flow

```
main.go
  -> config
  -> postgres (implements repository interfaces)
  -> s3
  -> auth (depends on repository interfaces, not postgres)
  -> rbac (standalone, no external deps)
  -> http (depends on repository interfaces, auth, s3)
```

**Why interfaces at the repository boundary**: Handlers depend on behavior (GetByID, Create), not on Postgres. This allows testing handlers with mock repositories and makes it possible to swap the database without touching handler code.

**Why no circular deps**: Domain packages import nothing. Repository interfaces import only domain. Auth imports repository interfaces. HTTP imports auth and repository interfaces. The dependency graph is a DAG.

```
Dependency Graph (imports)
===========================

  main.go
    |
    +-------> config
    |
    +-------> postgres/db  -------> repository (interfaces)
    |             |                       |
    |             +---------------------> domain/*
    |
    +-------> s3/client
    |
    +-------> auth/jwt  ----------> config
    |
    +-------> auth/apikey ---------> repository (interfaces)
    |                                     |
    +-------> auth/middleware -----> auth/jwt, auth/apikey
    |                                     |
    +-------> auth/rbac_middleware -> repository (interfaces), rbac/
    |
    +-------> rbac/ (standalone, imports nothing from this project)
    |
    +-------> http/server ---------> handler/*, auth/*, repository (interfaces)
                  |
                  +-> handler/* ---> repository (interfaces), auth/, pkg/*
                                          |
                                          +-> domain/* (structs only)

  pkg/errors, pkg/password, pkg/token, pkg/validator
    (leaf nodes, no project imports)
```

---

## What Is Not Here (and Why)

**No caching layer**: The query patterns are simple key lookups that Postgres handles efficiently. Adding Redis would increase operational complexity without a proven bottleneck.

**No message queue**: S3 bucket creation is the only async operation, and it is handled with a goroutine. A queue would be warranted if there were more background jobs or retry requirements.

**No rate limiting**: Mentioned as a feature goal but not yet implemented. Would be added as Echo middleware when needed.

**No tests**: The architecture supports testing through interfaces and dependency injection, but tests have not been written yet.

**No graceful shutdown**: The server has a `Shutdown` method but main.go does not wire up signal handling yet. Adding it is straightforward with `os/signal.NotifyContext`.

---

## Startup Wiring

```
main.go boots the entire system in order:

  config.Load()
       |
       v
  postgres.New(cfg.Database)  ---------> pgxpool (connection pool)
       |                                      |
       |   +----------------------------------+
       |   |
       v   v
  NewUserRepository(db)     ---+
  NewClientRepository(db)   ---+
  NewProjectRepository(db)  ---+--> all repos share the same *DB (pool)
  NewFileRepository(db)     ---+
  NewAPIKeyRepository(db)   ---+
  NewShareLinkRepository(db)---+
       |
       v
  s3.NewClient(cfg.AWS, cfg.App) -----> AWS session + S3 service
       |
       v
  auth.NewJWTService(cfg.JWT) --------> HS256 signer/verifier
       |
       v
  auth.NewAPIKeyService(apiKeyRepo) --> hash + lookup service
       |
       v
  auth.NewMiddleware(jwt, apiKey, apiKeyRepo)
       |
       v
  auth.NewRBACMiddleware(projectRepo, fileRepo)
       |
       v
  http.NewServer(ServerDependencies{...})
       |
       +---> creates Echo instance
       +---> creates all handlers
       +---> registers routes + middleware
       +---> returns *Server
       |
       v
  server.Start(":8080")
```
