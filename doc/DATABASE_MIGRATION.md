# Database Migration Guide

## Overview

sing-box-easy has migrated from JSON file storage to SQLite database using **XORM** (xorm.io) for better performance, reliability, and data integrity. This guide explains the migration process and new database structure.

## What Changed

### Before (JSON-based)
- Init state: `/etc/sing-box/init_state.json`
- Subscriptions: `/etc/sing-box/subscriptions.json`
- File-based storage with potential race conditions
- Manual file management

### After (SQLite + XORM)
- Single database: `/etc/sing-box/sing-box-easy.db`
- XORM ORM for struct-based models
- Automatic schema migrations via `Sync2()`
- ACID-compliant transactions
- Better concurrent access handling

## Automatic Migration

The application automatically migrates existing JSON data on startup:

1. **First startup with new version**:
   - Creates SQLite database
   - Runs schema migrations
   - Imports data from JSON files (if they exist)
   - Preserves all existing data

2. **Subsequent startups**:
   - Connects to existing database
   - Skips migration if data already exists
   - No data duplication

## Configuration Changes

### app.yml

**New configuration** (add this):
```yaml
sing_box:
  # SQLite database file path
  database_path: "/etc/sing-box/sing-box-easy.db"
```

**Deprecated configurations** (kept for backward compatibility):
```yaml
sing_box:
  subscription_path: "/etc/sing-box/subscriptions.json"  # Used for migration only
  init_state_path: "/etc/sing-box/init_state.json"      # Used for migration only
```

The deprecated paths are still used during the initial migration to locate your JSON files.

## Migration Process

### Step 1: Update Configuration

Edit your `app.yml`:

```yaml
server:
  port: "8080"

sing_box:
  config_path: "/etc/sing-box/config.json"
  binary_path: "sing-box"
  database_path: "/etc/sing-box/sing-box-easy.db"  # Add this line

  # Keep these for migration (optional after first run)
  subscription_path: "/etc/sing-box/subscriptions.json"
  init_state_path: "/etc/sing-box/init_state.json"
```

### Step 2: Backup Existing Data (Recommended)

```bash
# Backup your JSON files
cp /etc/sing-box/subscriptions.json /etc/sing-box/subscriptions.json.backup
cp /etc/sing-box/init_state.json /etc/sing-box/init_state.json.backup
```

### Step 3: Start the Application

```bash
./sing-box-easy
```

The application will:
1. Create `/etc/sing-box/sing-box-easy.db`
2. Run database migrations
3. Import data from JSON files
4. Log migration results

### Step 4: Verify Migration

Check the logs for migration success:
```
INFO: Database initialized path=/etc/sing-box/sing-box-easy.db
INFO: Starting JSON to SQLite migration...
INFO: Init state migrated successfully from JSON
INFO: Subscriptions migrated successfully from JSON
INFO: JSON to SQLite migration completed
```

### Step 5: Test Your Data

1. Access the application: `http://localhost:8080`
2. Verify subscriptions are present
3. Check initialization status
4. Test adding/updating subscriptions

### Step 6: Clean Up (Optional)

After confirming the migration worked:

```bash
# Archive old JSON files
mkdir -p /etc/sing-box/archive
mv /etc/sing-box/subscriptions.json /etc/sing-box/archive/
mv /etc/sing-box/init_state.json /etc/sing-box/archive/
```

## Database Schema

Schema is automatically managed by XORM based on Go struct models.

### init_state Table

```go
type InitState struct {
    ID                 int       `xorm:"pk autoincr 'id'"`
    Initialized        bool      `xorm:"notnull default(0)"`
    SingBoxInstalled   bool      `xorm:"'sing_box_installed' notnull default(0)"`
    ConfigGenerated    bool      `xorm:"'config_generated' notnull default(0)"`
    DashboardInstalled bool      `xorm:"'dashboard_installed' notnull default(0)"`
    SingBoxVersion     string    `xorm:"'sing_box_version' default('')"`
    InitTime           time.Time `xorm:"'init_time' null"`
    CreatedAt          time.Time `xorm:"created"`
    UpdatedAt          time.Time `xorm:"updated"`
}
```

### subscriptions Table

```go
type Subscription struct {
    ID             string    `xorm:"pk varchar(255)"`
    Name           string    `xorm:"notnull index"`
    URL            string    `xorm:"notnull text"`
    AutoUpdate     bool      `xorm:"'auto_update' notnull default(0)"`
    UpdateInterval string    `xorm:"'update_interval' default('24h')"`
    LastUpdate     time.Time `xorm:"'last_update' null"`
    CreatedAt      time.Time `xorm:"created"`
    UpdatedAt      time.Time `xorm:"updated"`
}
```

**XORM Features**:
- Automatic timestamps (`created`, `updated`)
- Struct-based schema definition
- No manual SQL migrations needed

## Accessing the Database

### Using SQLite CLI

```bash
# Open database
sqlite3 /etc/sing-box/sing-box-easy.db

# View tables
.tables

# Query subscriptions
SELECT * FROM subscriptions;

# Query init state
SELECT * FROM init_state;

# Exit
.quit
```

### Backup Database

```bash
# Create backup
sqlite3 /etc/sing-box/sing-box-easy.db ".backup /path/to/backup.db"

# Or copy file (when application is stopped)
cp /etc/sing-box/sing-box-easy.db /path/to/backup.db
```

### Restore Database

```bash
# Stop application
systemctl stop sing-box-easy

# Restore from backup
cp /path/to/backup.db /etc/sing-box/sing-box-easy.db

# Start application
systemctl start sing-box-easy
```

## Troubleshooting

### Migration Warnings

If you see warnings like:
```
WARN: Failed to migrate init state
WARN: Failed to migrate subscriptions
```

**Possible causes**:
1. JSON files don't exist (expected for new installations)
2. JSON files are corrupted
3. Data already migrated (safe to ignore)

### Database Locked

**Error**: `database is locked`

**Solutions**:
1. Ensure only one instance of sing-box-easy is running
2. Check file permissions: `chmod 644 /etc/sing-box/sing-box-easy.db`
3. Restart the application

### Migration Not Happening

**Checklist**:
1. Verify `database_path` in app.yml
2. Ensure JSON files exist at specified paths
3. Check application logs for errors
4. Verify database directory permissions

## Docker Considerations

The database file must be persisted using Docker volumes:

```yaml
services:
  sing-box-easy:
    volumes:
      - ./data:/etc/sing-box  # Persists database and config
```

The database will be created on first run and persisted across container restarts.

## API Compatibility

**Good news**: All existing APIs remain unchanged!

- Same endpoints
- Same request/response formats
- Same behavior

Only the underlying storage mechanism changed from JSON to SQLite.

## Performance Benefits

| Operation | JSON-based | SQLite |
|-----------|-----------|---------|
| Read single subscription | O(n) scan | O(1) indexed lookup |
| List all subscriptions | O(n) | O(n) with better caching |
| Update subscription | Read all + Write all | Update single row |
| Concurrent access | File locking | Database transactions |
| Data integrity | Manual validation | ACID guarantees |

## Migration Verification Checklist

- [ ] Updated app.yml with `database_path`
- [ ] Backed up existing JSON files
- [ ] Started application successfully
- [ ] Verified database file created
- [ ] Checked migration logs
- [ ] Tested subscription listing
- [ ] Tested subscription creation
- [ ] Tested init status endpoint
- [ ] Archived old JSON files (optional)

## Rollback (If Needed)

If you need to rollback to the old version:

1. **Stop the application**
2. **Restore JSON files** from backup
3. **Downgrade** to previous version
4. **Remove database**: `rm /etc/sing-box/sing-box-easy.db`
5. **Start old version**

## Support

If you encounter issues:

1. Check logs: `journalctl -u sing-box-easy -f`
2. Verify database: `sqlite3 /etc/sing-box/sing-box-easy.db "SELECT * FROM schema_migrations;"`
3. Report issues: https://github.com/SealinGp/sing-box-easy/issues

## Future Enhancements

With SQLite in place, future features will include:

- Subscription update history
- Usage statistics
- Configuration snapshots
- Advanced querying and filtering
- Data export/import tools
