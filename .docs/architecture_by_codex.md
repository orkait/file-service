# Architecture (Current Implementation)

This file describes the implementation that currently exists in `api/`. It focuses on what the system does now and why each major step exists.

## 1) Scope and Goal

The service is a multi-tenant file metadata API with direct S3 transfer.
The core goal is to keep tenant boundaries strict, keep project access simple, and keep file bytes off the backend server.

What is included:
- Multi-tenant clients
- Projects
- Project members and roles
- File and folder metadata
- Direct-to-S3 upload and download
- File overwrite by path replacement
- File and folder delete
- Project-scoped API keys
- Minimal viewer-only share links

What is intentionally not implemented:
- Zero Mode
- Time Travel/version history
- Advanced share policies (password, expiry, editor links)
- Audit logs

## 2) High-Level Component Map

1. `api/main.go`
   - What: Boots config, DB, repositories, S3 client, auth middleware, RBAC middleware, and HTTP server.
   - Why: Keeps wiring in one place and makes runtime dependencies explicit.

2. `api/internal/config`
   - What: Loads env vars, validates critical secrets, and builds typed config values.
   - Why: Fail fast on missing secrets and keep operational settings centralized.

3. `api/internal/http`
   - What: Defines public, JWT, and API key route groups and attaches middleware.
   - Why: Enforces auth boundaries at routing level before handlers run.

4. `api/internal/auth`
   - What: JWT generation/verification, API key verification, and middleware context injection.
   - Why: Keeps identity and machine auth logic consistent across all endpoints.

5. `api/internal/rbac`
   - What: Role hierarchy and permission engine used by middleware.
   - Why: Makes role checks deterministic and reusable.

6. `api/internal/http/handler`
   - What: Request parsing, validation, authorization-boundary checks, and use-case flow.
   - Why: Keeps HTTP concerns at the edge and business persistence in repositories.

7. `api/internal/repository/postgres`
   - What: SQL reads/writes for users, clients, projects, files, folders, API keys, share links, and signup transaction.
   - Why: Centralizes database behavior and keeps handlers free of SQL details.

8. `api/internal/storage/s3`
   - What: Presigned upload/download URL creation and S3 object/bucket operations.
   - Why: Keeps object-store behavior separate from metadata and auth logic.

9. `api/migrations`
   - What: Schema for tenant, project, member, file/folder, API key, and share link data.
   - Why: Defines hard data boundaries and database constraints for MVP behavior.

## 3) Startup Sequence (What + Why)

1. Load `.env` and process env config.
   - Why: Allows local/dev deployment and environment-driven runtime.

2. Validate config values (DB password, AWS keys, JWT secret length, etc.).
   - Why: Prevents booting a weak or broken runtime.

3. Open PostgreSQL pool and ping database.
   - Why: Ensures data path is alive before serving traffic.

4. Create repositories.
   - Why: Encapsulates data access per aggregate.

5. Create S3 client.
   - Why: Needed for presigned URLs and object/bucket operations.

6. Create JWT and API key services.
   - Why: Supports user auth and machine auth paths.

7. Create auth middleware and RBAC middleware.
   - Why: Enforces authentication and role checks before handler logic.

8. Build HTTP server with routes and handlers.
   - Why: Binds dependencies once and keeps runtime graph explicit.

9. Start server and wait for SIGINT/SIGTERM.
   - Why: Supports controlled shutdown in real deployments.

10. Graceful shutdown with timeout.
    - Why: Avoids abrupt connection drops and partial request handling.

## 4) Data Model and Why It Exists

1. `clients`
   - What: Tenant root.
   - Why: Hard isolation boundary between organizations.

2. `users`
   - What: Human identities with `client_id`.
   - Why: Every user belongs to exactly one tenant.

3. `projects`
   - What: Work container under a client, includes `s3_bucket_name`.
   - Why: Project is the main collaboration and storage boundary.

4. `project_members`
   - What: Membership and role (`viewer`, `editor`, `admin`).
   - Why: Internal access is membership-based, not file ACL based.

5. `folders`
   - What: Logical folder metadata and path prefix.
   - Why: Provides hierarchy without storing bytes on server.

6. `files`
   - What: File metadata with unique `(project_id, s3_key)`.
   - Why: Enables overwrite-by-path semantics and metadata queries.

7. `api_keys`
   - What: Project-scoped machine credentials with granular permissions.
   - Why: Automation access without impersonating users.

8. `share_links`
   - What: Token to file mapping only.
   - Why: Minimal public viewer access without advanced policies.

9. Deferred foreign keys between `users` and `clients`
   - What: Circular relation uses `DEFERRABLE INITIALLY DEFERRED`.
   - Why: Allows atomic signup transaction that creates both records safely.

## 5) Auth and Access Model

### 5.1 JWT user auth

What happens:
1. Client sends `Authorization: Bearer <jwt>`.
2. Middleware verifies signature and token validity.
3. Context receives `user_id`, `client_id`, `auth_type=jwt`.
4. RBAC middleware loads membership role for target project.

Why:
- JWT identifies a human user.
- Project membership decides internal access.
- Role checks remain simple and explicit.

### 5.2 API key auth

What happens:
1. Client sends `X-API-Key`.
2. Middleware checks format (`pk_`), hashes key, loads DB record.
3. Middleware enforces key status (not revoked/expired) and permission requirement.
4. Context receives `project_id`, key object, `auth_type=api_key`.

Why:
- Keys are machine credentials, not user identities.
- Scope is fixed to one project.
- Permission (`read|write|delete`) maps to endpoint groups.

### 5.3 Role model for members

What:
- `viewer` read-only
- `editor` read/write/delete file/folder actions
- `admin` member and API key management + full control

Why:
- Minimal hierarchy for MVP with predictable behavior.

### 5.4 Project-scope enforcement for API keys

What:
- File/folder handlers call `ensureAPIKeyProjectScope`.
- Route/body `project_id` mismatch is rejected.
- File-ID routes resolve file first, then verify key project equals file project.

Why:
- Prevents key from crossing into another project through alternate params or file IDs.

## 6) Request Flows and Why Each Step Exists

### 6.1 Signup flow

1. Validate email and password format.
   - Why: Reject invalid identity input early.
2. Hash password with bcrypt.
   - Why: Never store plaintext credentials.
3. Run transaction to create user, client, default project, and admin membership.
   - Why: Avoid partial tenant creation.
4. Try to create project bucket in S3.
   - Why: Project needs storage target.
5. Generate JWT and return identifiers.
   - Why: User can call protected API immediately.

### 6.2 Login flow

1. Find user by email.
2. Verify bcrypt password.
3. Return JWT.
   - Why: Standard secure login with short server-side state.

### 6.3 Project create flow

1. Require JWT and resolve user/client.
2. Validate project name.
3. Insert project metadata and generated bucket name.
4. Create S3 bucket.
5. Add creator as admin member.
6. If bucket or membership step fails, rollback project resources.
   - Why: Keep project access and storage initialization consistent.

### 6.4 Project member management

1. Admin-only routes for add/update/remove.
2. Add member checks invitee is from same client.
3. Update/remove checks prevent removing/demoting last admin.
   - Why: Preserve tenant boundary and avoid ownerless projects.

### 6.5 API key management

1. Admin-only routes create/list/revoke by project.
2. Create validates requested permissions and stores only key hash + prefix.
3. Revoke validates key belongs to same project route.
   - Why: Prevent plaintext key storage and cross-project key operations.

### 6.6 Upload URL + overwrite flow

1. Validate file name and file size.
2. Resolve authoritative project ID from route/body/auth context.
3. Enforce API key project scope when auth is machine key.
4. Build S3 key from folder path + filename.
5. Create presigned PUT URL.
6. Check existing file by `(project_id, s3_key)`.
7. If exists, update metadata; if not, create metadata row.
8. Return upload URL + file ID + S3 key.
   - Why: Server stays out of file bytes while enforcing overwrite at data boundary.

### 6.7 Download URL flow

1. Resolve file metadata by file ID.
2. Enforce project scope for API keys.
3. Load project for bucket name.
4. Generate presigned GET URL.
   - Why: Read access is authorized on metadata before object access token is issued.

### 6.8 Share link flow (minimal viewer link)

1. Editor-or-above creates link from file ID.
2. Service generates fixed token and stores only `file_id + token + created_by`.
3. Public endpoint resolves token, then file, then project, then presigned GET URL.
   - Why: Keep sharing minimal and read-only by design.

### 6.9 Delete flows

1. File delete:
   - Delete S3 object first, then delete metadata row.
   - Why: Keeps store and metadata aligned for file deletes.
2. Folder delete:
   - Delete folder metadata row (DB cascade handles nested metadata links).
   - Why: Keeps folder tree metadata consistent.
3. Project delete:
   - Delete S3 bucket, then project row.
   - Why: Prevents orphan bucket after project removal.

## 7) Storage Boundary Guarantees

1. Upload/download use presigned URLs.
   - Why: Server does not stream file bytes.

2. Server stores only metadata in PostgreSQL.
   - Why: Clean separation of authorization metadata from object storage.

3. Authorization check happens before issuing presigned links.
   - Why: Prevents direct object access without policy decision.

## 8) Route Topology and Why It Is Split

1. Public routes
   - Signup, login, health, and share-token download-url.
   - Why: Needed for bootstrap and public viewer links.

2. JWT routes (`/api/...`)
   - Project/member/API-key management and member-based file/folder routes.
   - Why: Human collaboration uses role-based membership.

3. API key routes (`/api/key/...`)
   - Permission-scoped file/folder operations only.
   - Why: Machine automation stays project-scoped and narrow.

## 9) Deferred Features Confirmed Absent in Current Capability

1. Zero Mode
   - No encryption key envelope fields, no client-side key workflow, no mode switches.

2. Time Travel
   - No file version table, no version pointer, no restore endpoints.

3. Advanced sharing policies
   - No password, expiry, editor role, view caps, or unlock logic in schema/handlers.

4. Audit logs
   - No audit log table, writer, or query routes.

## 10) Current Operational Notes

1. There are no automated tests yet (`go test ./...` has no test files).
   - Why it matters: behavior depends on manual verification today.

2. Bucket creation in handlers uses hard-coded region string.
   - Why it matters: can drift from configured AWS region.

3. Folder delete removes metadata but does not explicitly delete S3 objects under folder prefix.
   - Why it matters: can leave object-store leftovers if not cleaned separately.

4. The initial migration file was edited during refactor (not forward-migrated).
   - Why it matters: existing deployed environments may need careful rollout planning.

## 11) Why This Architecture Was Chosen

1. Project membership is the only internal permission source.
   - Why: fewer hidden paths and easier correctness checks.

2. API keys are project-scoped and permission-scoped, not role-based users.
   - Why: avoids machine-to-user ambiguity.

3. Share links are intentionally minimal and read-only in behavior.
   - Why: MVP sharing without policy complexity.

4. S3 direct transfer is enforced by design.
   - Why: scalability and lower backend transfer risk.

5. Metadata-first repository design.
   - Why: clean boundaries for auth, persistence, and storage integration.
