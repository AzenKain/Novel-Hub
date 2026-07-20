package constants

type ProviderType string

const (
	LocalProvider ProviderType = "LOCAL"
)

func (p ProviderType) String() string {
	return string(p)
}
