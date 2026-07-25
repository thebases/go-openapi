package docs

type Provider string

const (
	Swagger Provider = "swagger"
	Base    Provider = "base"
	Scalar  Provider = "scalar"
)

type Config struct {
	Provider    Provider
	Title       string
	DocumentURL string
	CDNBaseURL  string
}
