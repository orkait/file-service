# File Service - Current Status

## ✅ System Status: OPERATIONAL

### Database
- ✅ Connection: Working
- ✅ Tables: All created successfully
  - clients
  - projects
  - assets
  - api_keys
  - project_members
  - refresh_tokens
- ✅ Triggers: Active
  - Auto-create default project on user registration
  - Auto-add user as project owner

### Server
- ✅ Running on: http://localhost:8080
- ✅ Build: file-service.exe created
- ✅ Health check: /ping endpoint responding

### Authentication
- ✅ User Registration: Working
- ✅ User Login: Working
- ✅ JWT Tokens: Generated successfully
  - Access Token: 15 min expiry
  - Refresh Token: 7 days expiry
- ✅ Multi-tenant: Client isolation working

### Test Results
```
Client ID: e74cbc71-090e-489e-97c2-9865484e1e99
Project ID: a8dde0cf-d5fe-4429-af91-bbab52dd4c1d
Project Name: Default Project
Access Token: eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...
```

## ⚠️ Pending Configuration

### AWS S3 Credentials
Update `.env` with your actual AWS credentials:
```env
BUCKET_NAME=your-actual-bucket-name
AWS_ACCESS_KEY_ID=your-actual-access-key
AWS_SECRET_ACCESS_KEY=your-actual-secret-key
```

Current values are placeholders and will cause upload failures.

## 📝 Quick Start Commands

### Start Server
```bash
.\file-service.exe
```

### Run Tests
```powershell
powershell -ExecutionPolicy Bypass -File test_api.ps1
```

### Setup Database (if needed)
```bash
go run scripts/setup_db.go
```

### Test Database Connection
```bash
go run scripts/test_db.go
```

## 🔑 Test Credentials

**Email:** test@example.com
**Password:** password123

**Access Token (valid for 15 min):**
```
eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJjbGllbnRfaWQiOiJlNzRjYmM3MS0wOTBlLTQ4OWUtOTdjMi05ODY1NDg0ZTFlOTkiLCJlbWFpbCI6InRlc3RAZXhhbXBsZS5jb20iLCJleHAiOjE3NzEyMTAyMjEsImlhdCI6MTc3MTIwOTMyMX0.kwqgY1r49kro8xPunG_yQXYw8VNtJvb-7kUyds-oiBQ
```

## 📚 API Endpoints

### Public (No Auth)
- `GET /ping` - Health check
- `POST /auth/register` - User registration
- `POST /auth/login` - User login
- `POST /auth/refresh` - Refresh access token

### Protected (JWT Required)
- `GET /api/projects` - List projects
- `POST /api/projects` - Create project
- `GET /api/projects/:id` - Get project details
- `POST /api/projects/:project_id/members` - Invite member
- `GET /api/projects/:project_id/members` - List members
- `DELETE /api/projects/:project_id/members/:member_id` - Remove member
- `POST /api/assets` - Upload asset
- `GET /api/assets` - List assets
- `GET /api/assets/:id` - Get asset
- `GET /api/assets/:id/versions` - Get version history
- `DELETE /api/assets/:id` - Delete asset
- `GET /api/folders` - List folders
- `GET /upload-url` - Get presigned upload URL
- `POST /assets/confirm` - Confirm direct upload

### API Key Routes (X-API-Key header)
- `/v1/*` - Same as protected routes but use API key instead of JWT

## 🧪 Manual Testing

### Test Upload (requires AWS credentials)
```bash
curl -X POST http://localhost:8080/api/assets \
  -H "Authorization: Bearer <your_token>" \
  -F "project_id=a8dde0cf-d5fe-4429-af91-bbab52dd4c1d" \
  -F "folder_path=/test/" \
  -F "file=@C:\codingFiles\orkait\orka-file-service\api\example\test_image.jpeg"
```

### Test List Assets
```bash
curl -X GET "http://localhost:8080/api/assets?project_id=a8dde0cf-d5fe-4429-af91-bbab52dd4c1d" \
  -H "Authorization: Bearer <your_token>"
```

## 📖 Documentation

- `docs/AUTH_AND_INVITES.md` - Authentication & email invite system
- `docs/VERSIONING.md` - Asset versioning guide
- `docs/IMPROVEMENTS.md` - Recent code improvements
- `TESTING_GUIDE.md` - Complete testing guide

## 🎯 Next Steps

1. **Update AWS Credentials** in `.env`
2. **Test File Upload** with actual S3 bucket
3. **Test Email Invites:**
   - Register second user
   - Invite them to project
   - Verify they can access shared project
4. **Test Versioning:**
   - Upload file
   - Upload new version with `create_version=true`
   - Check version history

## 🔧 Architecture

### S3 Bucket Structure
```
main-bucket/
  client-uuid/
    project-uuid/
      /folder-path/
        asset-uuid/
          filename.ext
```

### Multi-Tenant Security
- All queries filtered by `client_id`
- Project access via `project_members` table
- JWT middleware on all protected routes
- API key alternative for programmatic access

### Database Triggers
1. **create_default_project** - Auto-creates "Default Project" on user registration
2. **add_project_owner** - Auto-adds user as project owner

## ⚠️ Important Notes

### Email Invites
- **No actual emails sent** - system assumes users are pre-registered
- Invitee MUST exist in database before invitation
- Lookup by email address only

### Token Expiry
- Access tokens expire after 15 minutes
- Use `/auth/refresh` endpoint with refresh token
- Or login again

### File Uploads
- Requires valid AWS S3 credentials
- Bucket must exist and be accessible
- Region must match configuration

## 🐛 Troubleshooting

### Upload Fails
- Check AWS credentials in `.env`
- Verify bucket exists
- Check bucket permissions
- Ensure region is correct

### "client not found with that email"
- User must register first
- Email is case-sensitive

### Token Expired
- Access tokens expire after 15 min
- Use refresh token or login again

### Database Connection Failed
- Check DATABASE_URL in `.env`
- Verify Supabase project is active
- Check network connectivity
