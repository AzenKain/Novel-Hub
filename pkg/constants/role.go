package constants

type RoleType string

const (
	RoleTypeAdmin  RoleType = "ADMIN"
	RoleTypeMod    RoleType = "MOD"
	RoleTypeUser   RoleType = "USER"
	RoleTypeGuest  RoleType = "GUEST"
	RoleTypeBanned RoleType = "BANNED"
)

func (r RoleType) String() string {
	return string(r)
}

const (
	//Book Reading & Discovery
	PermBookRead       = "book.read"
	PermBookTTS        = "book.tts"
	PermBookSearchDeep = "book.search.deep"
	PermBookDownload   = "book.download"
	PermBookOffline    = "book.offline"
	PermBookSendEmail  = "book.send_email"
	PermBookShare      = "book.share"

	//Interactions & Personal
	PermBookBookmark     = "book.bookmark"
	PermBookCollection   = "book.collection"
	PermBookHighlight    = "book.highlight"
	PermBookReviewCreate = "book.review.create"
	PermBookReviewDelete = "book.review.delete"
	PermUserStatsRead    = "user.stats.read"
	PermTrackerSync      = "tracker.sync"

	//Book Content Management
	PermBookUpload          = "book.upload"
	PermBookEdit            = "book.edit"
	PermBookRepair          = "book.repair"
	PermBookMetadataFetch   = "book.metadata.fetch"
	PermBookDelete          = "book.delete"
	PermBookDuplicateManage = "book.duplicate.manage"
	PermBookArchive         = "book.archive"
	PermBookBulkManage      = "book.bulk.manage"

	//Library Management
	PermLibraryRead   = "library.read"
	PermLibraryManage = "library.manage"

	//External Sync & Integration
	PermOPDSRead       = "opds.read"
	PermOPDSDownload   = "opds.download"
	PermWebDAVRead     = "webdav.read"
	PermWebDAVDownload = "webdav.download"
	PermKoboSync       = "kobo.sync"
	PermKomgaSync      = "komga.sync"
	PermCalibreSync    = "calibre.sync"
	PermPodcastManage  = "podcast.manage"

	//Personal Customization & Soundscape
	PermUserFontManage       = "user.font.manage"
	PermUserSoundscapeManage = "user.soundscape.manage"
	PermUserThemeManage      = "user.theme.manage"

	//System Administration
	PermAdminAccess           = "admin.access"
	PermAdminSoundscapeManage = "admin.soundscape.manage"
	PermAdminFontManage       = "admin.font.manage"
	PermAdminThemeManage      = "admin.theme.manage"
	PermUserManage            = "user.manage"
	PermRoleManage            = "role.manage"
	PermSettingManage         = "setting.manage"
	PermJobRead               = "job.read"
	PermJobManage             = "job.manage"
	PermSystemLogRead         = "system.log.read"
	PermSystemBackup          = "system.backup"
	PermWebhookManage         = "webhook.manage"
)

type PermissionInfo struct {
	Key         string `json:"key"`
	Description string `json:"description"`
}

type PermissionCategory struct {
	ID          string           `json:"id"`
	Name        string           `json:"name"`
	Permissions []PermissionInfo `json:"permissions"`
}

func GetDefaultPermissionsForRole(role RoleType) []string {
	switch role {
	case RoleTypeGuest:
		return []string{
			PermBookRead,
			PermBookTTS,
			PermLibraryRead,
			PermOPDSRead,
		}
	case RoleTypeUser:
		return []string{
			PermBookRead,
			PermBookTTS,
			PermBookSearchDeep,
			PermBookDownload,
			PermBookOffline,
			PermBookSendEmail,
			PermBookShare,
			PermBookBookmark,
			PermBookCollection,
			PermBookHighlight,
			PermBookReviewCreate,
			PermUserStatsRead,
			PermTrackerSync,
			PermLibraryRead,
			PermOPDSRead,
			PermOPDSDownload,
			PermWebDAVRead,
			PermWebDAVDownload,
			PermKoboSync,
			PermKomgaSync,
			PermUserFontManage,
			PermUserSoundscapeManage,
			PermUserThemeManage,
		}
	case RoleTypeMod:
		return []string{
			PermBookRead,
			PermBookTTS,
			PermBookSearchDeep,
			PermBookDownload,
			PermBookOffline,
			PermBookSendEmail,
			PermBookShare,
			PermBookBookmark,
			PermBookCollection,
			PermBookHighlight,
			PermBookReviewCreate,
			PermBookReviewDelete,
			PermUserStatsRead,
			PermTrackerSync,
			PermBookUpload,
			PermBookEdit,
			PermBookRepair,
			PermBookMetadataFetch,
			PermBookDelete,
			PermBookDuplicateManage,
			PermBookArchive,
			PermBookBulkManage,
			PermLibraryRead,
			PermLibraryManage,
			PermOPDSRead,
			PermOPDSDownload,
			PermWebDAVRead,
			PermWebDAVDownload,
			PermKoboSync,
			PermKomgaSync,
			PermCalibreSync,
			PermPodcastManage,
			PermUserFontManage,
			PermUserSoundscapeManage,
			PermUserThemeManage,
			PermAdminAccess,
			PermJobRead,
		}
	case RoleTypeAdmin:
		return []string{
			PermBookRead, PermBookTTS, PermBookSearchDeep, PermBookDownload, PermBookOffline, PermBookSendEmail, PermBookShare,
			PermBookBookmark, PermBookCollection, PermBookHighlight, PermBookReviewCreate, PermBookReviewDelete, PermUserStatsRead, PermTrackerSync,
			PermBookUpload, PermBookEdit, PermBookRepair, PermBookMetadataFetch, PermBookDelete, PermBookDuplicateManage, PermBookArchive, PermBookBulkManage,
			PermLibraryRead, PermLibraryManage,
			PermOPDSRead, PermOPDSDownload, PermWebDAVRead, PermWebDAVDownload, PermKoboSync, PermKomgaSync, PermCalibreSync, PermPodcastManage,
			PermUserFontManage, PermUserSoundscapeManage, PermUserThemeManage,
			PermAdminAccess, PermAdminSoundscapeManage, PermAdminFontManage, PermAdminThemeManage,
			PermUserManage, PermRoleManage, PermSettingManage, PermJobRead, PermJobManage, PermSystemLogRead, PermSystemBackup, PermWebhookManage,
		}
	default:
		return []string{}
	}
}
