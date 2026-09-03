package request

type KoboLocationDto struct {
	Value  string `json:"Value"`
	Type   string `json:"Type"`
	Source string `json:"Source"`
}

type KoboBookmarkDto struct {
	ProgressPercent              *float64         `json:"ProgressPercent"`
	ContentSourceProgressPercent *float64         `json:"ContentSourceProgressPercent"`
	Location                     *KoboLocationDto `json:"Location"`
}

type KoboStatisticsDto struct {
	SpentReadingMinutes  *float64 `json:"SpentReadingMinutes"`
	RemainingTimeMinutes *float64 `json:"RemainingTimeMinutes"`
}

type KoboStatusInfoDto struct {
	Status string `json:"Status"`
}

type KoboReadingStateDto struct {
	CurrentBookmark *KoboBookmarkDto   `json:"CurrentBookmark"`
	Statistics      *KoboStatisticsDto `json:"Statistics"`
	StatusInfo      *KoboStatusInfoDto `json:"StatusInfo"`
}

type PutKoboStateDto struct {
	ReadingStates []KoboReadingStateDto `json:"ReadingStates"`
}

type KoboSyncDto struct {
	UserID      string
	SyncToken   string
	EndpointURL string
}

type KoboAuthDeviceDto struct {
	UserKey string `json:"UserKey"`
}
