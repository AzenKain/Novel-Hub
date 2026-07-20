package constants

type RoleType string

const (
	RoleTypeAdmin  RoleType = "ADMIN"
	RoleTypeMod    RoleType = "MOD"
	RoleTypeUser   RoleType = "USER"
	RoleTypeBanned RoleType = "BANNED"
)

func (r RoleType) String() string {
	return string(r)
}

func CheckValidRole(r RoleType) bool {
	return r == RoleTypeAdmin ||
		r == RoleTypeMod ||
		r == RoleTypeUser ||
		r == RoleTypeBanned
}

func ParseRole(s string) (RoleType, bool) {
	r := RoleType(s)
	if CheckValidRole(r) {
		return r, true
	}
	return "", false
}
