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
	PermBookMetadataFetch   = "book.metadata.fetch"
	PermBookDelete          = "book.delete"
	PermBookDuplicateManage = "book.duplicate.manage"
	PermBookArchive         = "book.archive"
	PermBookBulkManage      = "book.bulk.manage"

	//Library Management
	PermLibraryRead   = "library.read"
	PermLibraryManage = "library.manage"

	//External Sync & Integration
	PermOPDSRead     = "opds.read"
	PermOPDSDownload = "opds.download"
	PermKoboSync     = "kobo.sync"
	PermKomgaSync    = "komga.sync"
	PermCalibreSync  = "calibre.sync"
	PermPodcastManage = "podcast.manage"

	//System Administration
	PermAdminAccess   = "admin.access"
	PermUserManage    = "user.manage"
	PermRoleManage    = "role.manage"
	PermSettingManage = "setting.manage"
	PermJobRead       = "job.read"
	PermJobManage     = "job.manage"
	PermSystemLogRead = "system.log.read"
	PermSystemBackup  = "system.backup"
	PermWebhookManage = "webhook.manage"
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
			PermKoboSync,
			PermKomgaSync,
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
			PermBookMetadataFetch,
			PermBookDelete,
			PermBookDuplicateManage,
			PermBookArchive,
			PermBookBulkManage,
			PermLibraryRead,
			PermLibraryManage,
			PermOPDSRead,
			PermOPDSDownload,
			PermKoboSync,
			PermKomgaSync,
			PermCalibreSync,
			PermPodcastManage,
			PermAdminAccess,
			PermJobRead,
		}
	case RoleTypeAdmin:
		return []string{
			PermBookRead, PermBookTTS, PermBookSearchDeep, PermBookDownload, PermBookOffline, PermBookSendEmail, PermBookShare,
			PermBookBookmark, PermBookCollection, PermBookHighlight, PermBookReviewCreate, PermBookReviewDelete, PermUserStatsRead, PermTrackerSync,
			PermBookUpload, PermBookEdit, PermBookMetadataFetch, PermBookDelete, PermBookDuplicateManage, PermBookArchive, PermBookBulkManage,
			PermLibraryRead, PermLibraryManage,
			PermOPDSRead, PermOPDSDownload, PermKoboSync, PermKomgaSync, PermCalibreSync, PermPodcastManage,
			PermAdminAccess, PermUserManage, PermRoleManage, PermSettingManage, PermJobRead, PermJobManage, PermSystemLogRead, PermSystemBackup, PermWebhookManage,
		}
	default:
		return []string{}
	}
}
