package constants

const (
	AgeRatingG    = "G"
	AgeRatingPG   = "PG"
	AgeRatingPG13 = "PG-13"
	AgeRatingR17  = "R17+"
	AgeRatingR18  = "R18+"
)

var AgeRatingLevels = map[string]int{
	AgeRatingG:    1,
	AgeRatingPG:   2,
	AgeRatingPG13: 3,
	AgeRatingR17:  4,
	AgeRatingR18:  5,
}
