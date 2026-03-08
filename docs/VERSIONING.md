# Asset Versioning

## Overview

The file service supports automatic versioning of assets. When you upload a new version of an existing file, the system maintains a complete version history while marking only the latest version as current.

## How It Works

### Creating a New Asset (Version 1)
```bash
POST /api/assets/upload
Content-Type: multipart/form-data

project_id=<uuid>
folder_path=/images/
file=<binary>
```

Response includes `version: 1` and `is_latest: true`

### Creating a New Version
```bash
POST /api/assets/upload
Content-Type: multipart/form-data

project_id=<uuid>
folder_path=/images/
file=<binary>
create_version=true
parent_asset_id=<parent-uuid>
```

This will:
1. Mark all previous versions as `is_latest: false`
2. Create new version with incremented version number
3. Link to parent via `parent_asset_id`
4. Mark new version as `is_latest: true`

### Retrieving Version History
```bash
GET /api/assets/:id/versions
```

Returns all versions sorted by version number (descending), with presigned URLs for each.

## Database Schema

```sql
CREATE TABLE assets (
    id UUID PRIMARY KEY,
    version INTEGER DEFAULT 1,
    is_latest BOOLEAN DEFAULT TRUE,
    parent_asset_id UUID REFERENCES assets(id),
    -- other fields...
);
```

## Key Features

- **Automatic version numbering**: System increments version numbers automatically
- **Latest version tracking**: Only one version marked as `is_latest` per asset chain
- **Complete history**: All versions preserved with full metadata
- **Presigned URLs**: Each version gets its own presigned download URL
- **Transaction safety**: Version creation uses database transactions for consistency

## API Examples

### Direct Upload with Versioning
```bash
# Step 1: Get presigned upload URL
GET /api/assets/upload-url?project_id=<uuid>&filename=logo.png

# Step 2: Upload to S3 (client-side)
POST <presigned_url>

# Step 3: Confirm upload with versioning
POST /api/assets/confirm-upload
{
  "project_id": "<uuid>",
  "s3_key": "<s3-key>",
  "filename": "logo.png",
  "original_filename": "logo.png",
  "file_size": 12345,
  "mime_type": "image/png",
  "create_version": true,
  "parent_asset_id": "<parent-uuid>"
}
```

## Best Practices

1. **Always specify parent_asset_id** when creating versions
2. **Use create_version=true** flag to enable versioning
3. **Query by is_latest=true** to get current versions only
4. **Store parent_asset_id** on client side for version chains
