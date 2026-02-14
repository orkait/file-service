package postgres

import (
	"fmt"
	"time"
)

const (
	bucketNameIDSegmentLength = 8
	defaultProjectName        = "default"
	defaultProjectMemberRole  = "admin"

	poolHealthCheckPeriod = time.Minute
	poolMaxConnLifetime   = time.Hour
	poolMaxConnIdleTime   = 30 * time.Minute
	dbPingTimeout         = 5 * time.Second

	errClientNotFound    = "client not found"
	errUserNotFound      = "user not found"
	errProjectNotFound   = "project not found"
	errMemberNotFound    = "member not found"
	errFileNotFound      = "file not found"
	errFolderNotFound    = "folder not found"
	errAPIKeyNotFound    = "API key not found"
	errShareLinkNotFound = "share link not found"
	errPrefixEmpty       = "prefix cannot be empty"

	errFailedParseDatabaseConfigFmt  = "failed to parse database config: %w"
	errFailedCreateConnectionPoolFmt = "failed to create connection pool: %w"
	errFailedPingDatabaseFmt         = "failed to ping database: %w"

	errFailedStartTransactionFmt       = "failed to start transaction: %w"
	errFailedCreateDefaultProjectFmt   = "failed to create default project: %w"
	errFailedAddUserAsProjectMemberFmt = "failed to add user as project member: %w"
	errFailedCommitTransactionFmt      = "failed to commit transaction: %w"

	errFailedCreateUserFmt = "failed to create user: %w"
	errFailedGetUserFmt    = "failed to get user: %w"
	errFailedListUsersFmt  = "failed to list users: %w"
	errFailedScanUserFmt   = "failed to scan user: %w"
	errIterateUsersFmt     = "error iterating users: %w"
	errFailedUpdateUserFmt = "failed to update user: %w"
	errFailedDeleteUserFmt = "failed to delete user: %w"

	errFailedCreateClientFmt = "failed to create client: %w"
	errFailedGetClientFmt    = "failed to get client: %w"
	errFailedDeleteClientFmt = "failed to delete client: %w"

	errFailedCreateProjectFmt      = "failed to create project: %w"
	errFailedGetProjectFmt         = "failed to get project: %w"
	errFailedListProjectsFmt       = "failed to list projects: %w"
	errFailedScanProjectFmt        = "failed to scan project: %w"
	errFailedListProjectsByUserFmt = "failed to list projects by user: %w"
	errFailedGetDefaultProjectFmt  = "failed to get default project: %w"
	errFailedUpdateProjectFmt      = "failed to update project: %w"
	errFailedDeleteProjectFmt      = "failed to delete project: %w"
	errFailedAddMemberFmt          = "failed to add member: %w"
	errFailedGetMemberFmt          = "failed to get member: %w"
	errFailedListMembersFmt        = "failed to list members: %w"
	errFailedScanMemberFmt         = "failed to scan member: %w"
	errFailedCountAdminsFmt        = "failed to count admins: %w"
	errFailedUpdateMemberRoleFmt   = "failed to update member role: %w"
	errFailedRemoveMemberFmt       = "failed to remove member: %w"

	errFailedCreateFileFmt          = "failed to create file: %w"
	errFailedGetFileFmt             = "failed to get file: %w"
	errFailedListFilesFmt           = "failed to list files: %w"
	errFailedScanFileFmt            = "failed to scan file: %w"
	errFailedDeleteFilesByPrefixFmt = "failed to delete files by prefix: %w"
	errFailedCountFilesByPrefixFmt  = "failed to count files by prefix: %w"
	errFailedUpdateFileFmt          = "failed to update file: %w"
	errFailedDeleteFileFmt          = "failed to delete file: %w"
	errFailedCreateFolderFmt        = "failed to create folder: %w"
	errFailedGetFolderFmt           = "failed to get folder: %w"
	errFailedListFoldersFmt         = "failed to list folders: %w"
	errFailedScanFolderFmt          = "failed to scan folder: %w"
	errFailedDeleteFolderFmt        = "failed to delete folder: %w"

	errFailedCreateAPIKeyFmt   = "failed to create API key: %w"
	errFailedGetAPIKeyFmt      = "failed to get API key: %w"
	errFailedListAPIKeysFmt    = "failed to list API keys: %w"
	errFailedScanAPIKeyFmt     = "failed to scan API key: %w"
	errFailedUpdateLastUsedFmt = "failed to update last used: %w"
	errFailedRevokeAPIKeyFmt   = "failed to revoke API key: %w"
	errFailedDeleteAPIKeyFmt   = "failed to delete API key: %w"

	errFailedCreateShareLinkFmt     = "failed to create share link: %w"
	errFailedGetShareLinkFmt        = "failed to get share link: %w"
	errFailedGetShareLinkByTokenFmt = "failed to get share link by token: %w"
	errFailedListShareLinksFmt      = "failed to list share links: %w"
	errFailedScanShareLinkFmt       = "failed to scan share link: %w"
	errFailedDeleteShareLinkFmt     = "failed to delete share link: %w"
)

var (
	errFailedAddMember              = func(err error) error { return fmt.Errorf(errFailedAddMemberFmt, err) }
	errFailedAddUserAsProjectMember = func(err error) error { return fmt.Errorf(errFailedAddUserAsProjectMemberFmt, err) }
	errFailedCommitTransaction      = func(err error) error { return fmt.Errorf(errFailedCommitTransactionFmt, err) }
	errFailedCountFilesByPrefix     = func(err error) error { return fmt.Errorf(errFailedCountFilesByPrefixFmt, err) }
	errFailedCreateAPIKey           = func(err error) error { return fmt.Errorf(errFailedCreateAPIKeyFmt, err) }
	errFailedCreateClient           = func(err error) error { return fmt.Errorf(errFailedCreateClientFmt, err) }
	errFailedCreateConnectionPool   = func(err error) error { return fmt.Errorf(errFailedCreateConnectionPoolFmt, err) }
	errFailedCreateDefaultProject   = func(err error) error { return fmt.Errorf(errFailedCreateDefaultProjectFmt, err) }
	errFailedCreateFile             = func(err error) error { return fmt.Errorf(errFailedCreateFileFmt, err) }
	errFailedCreateFolder           = func(err error) error { return fmt.Errorf(errFailedCreateFolderFmt, err) }
	errFailedCreateProject          = func(err error) error { return fmt.Errorf(errFailedCreateProjectFmt, err) }
	errFailedCreateShareLink        = func(err error) error { return fmt.Errorf(errFailedCreateShareLinkFmt, err) }
	errFailedCreateUser             = func(err error) error { return fmt.Errorf(errFailedCreateUserFmt, err) }
	errFailedDeleteAPIKey           = func(err error) error { return fmt.Errorf(errFailedDeleteAPIKeyFmt, err) }
	errFailedDeleteClient           = func(err error) error { return fmt.Errorf(errFailedDeleteClientFmt, err) }
	errFailedDeleteFile             = func(err error) error { return fmt.Errorf(errFailedDeleteFileFmt, err) }
	errFailedDeleteFilesByPrefix    = func(err error) error { return fmt.Errorf(errFailedDeleteFilesByPrefixFmt, err) }
	errFailedDeleteFolder           = func(err error) error { return fmt.Errorf(errFailedDeleteFolderFmt, err) }
	errFailedDeleteProject          = func(err error) error { return fmt.Errorf(errFailedDeleteProjectFmt, err) }
	errFailedDeleteShareLink        = func(err error) error { return fmt.Errorf(errFailedDeleteShareLinkFmt, err) }
	errFailedDeleteUser             = func(err error) error { return fmt.Errorf(errFailedDeleteUserFmt, err) }
	errFailedGetAPIKey              = func(err error) error { return fmt.Errorf(errFailedGetAPIKeyFmt, err) }
	errFailedGetClient              = func(err error) error { return fmt.Errorf(errFailedGetClientFmt, err) }
	errFailedGetDefaultProject      = func(err error) error { return fmt.Errorf(errFailedGetDefaultProjectFmt, err) }
	errFailedGetFile                = func(err error) error { return fmt.Errorf(errFailedGetFileFmt, err) }
	errFailedGetFolder              = func(err error) error { return fmt.Errorf(errFailedGetFolderFmt, err) }
	errFailedGetMember              = func(err error) error { return fmt.Errorf(errFailedGetMemberFmt, err) }
	errFailedGetProject             = func(err error) error { return fmt.Errorf(errFailedGetProjectFmt, err) }
	errFailedGetShareLink           = func(err error) error { return fmt.Errorf(errFailedGetShareLinkFmt, err) }
	errFailedGetShareLinkByToken    = func(err error) error { return fmt.Errorf(errFailedGetShareLinkByTokenFmt, err) }
	errFailedGetUser                = func(err error) error { return fmt.Errorf(errFailedGetUserFmt, err) }
	errFailedListAPIKeys            = func(err error) error { return fmt.Errorf(errFailedListAPIKeysFmt, err) }
	errFailedListFiles              = func(err error) error { return fmt.Errorf(errFailedListFilesFmt, err) }
	errFailedListFolders            = func(err error) error { return fmt.Errorf(errFailedListFoldersFmt, err) }
	errFailedListMembers            = func(err error) error { return fmt.Errorf(errFailedListMembersFmt, err) }
	errFailedCountAdmins            = func(err error) error { return fmt.Errorf(errFailedCountAdminsFmt, err) }
	errFailedListProjects           = func(err error) error { return fmt.Errorf(errFailedListProjectsFmt, err) }
	errFailedListProjectsByUser     = func(err error) error { return fmt.Errorf(errFailedListProjectsByUserFmt, err) }
	errFailedListShareLinks         = func(err error) error { return fmt.Errorf(errFailedListShareLinksFmt, err) }
	errFailedListUsers              = func(err error) error { return fmt.Errorf(errFailedListUsersFmt, err) }
	errFailedParseDatabaseConfig    = func(err error) error { return fmt.Errorf(errFailedParseDatabaseConfigFmt, err) }
	errFailedPingDatabase           = func(err error) error { return fmt.Errorf(errFailedPingDatabaseFmt, err) }
	errFailedRemoveMember           = func(err error) error { return fmt.Errorf(errFailedRemoveMemberFmt, err) }
	errFailedRevokeAPIKey           = func(err error) error { return fmt.Errorf(errFailedRevokeAPIKeyFmt, err) }
	errFailedScanAPIKey             = func(err error) error { return fmt.Errorf(errFailedScanAPIKeyFmt, err) }
	errFailedScanFile               = func(err error) error { return fmt.Errorf(errFailedScanFileFmt, err) }
	errFailedScanFolder             = func(err error) error { return fmt.Errorf(errFailedScanFolderFmt, err) }
	errFailedScanMember             = func(err error) error { return fmt.Errorf(errFailedScanMemberFmt, err) }
	errFailedScanProject            = func(err error) error { return fmt.Errorf(errFailedScanProjectFmt, err) }
	errFailedScanShareLink          = func(err error) error { return fmt.Errorf(errFailedScanShareLinkFmt, err) }
	errFailedScanUser               = func(err error) error { return fmt.Errorf(errFailedScanUserFmt, err) }
	errFailedStartTransaction       = func(err error) error { return fmt.Errorf(errFailedStartTransactionFmt, err) }
	errFailedUpdateFile             = func(err error) error { return fmt.Errorf(errFailedUpdateFileFmt, err) }
	errFailedUpdateLastUsed         = func(err error) error { return fmt.Errorf(errFailedUpdateLastUsedFmt, err) }
	errFailedUpdateMemberRole       = func(err error) error { return fmt.Errorf(errFailedUpdateMemberRoleFmt, err) }
	errFailedUpdateProject          = func(err error) error { return fmt.Errorf(errFailedUpdateProjectFmt, err) }
	errFailedUpdateUser             = func(err error) error { return fmt.Errorf(errFailedUpdateUserFmt, err) }
	errIterateUsers                 = func(err error) error { return fmt.Errorf(errIterateUsersFmt, err) }
)
