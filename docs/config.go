package docs

type Provider string

const (
	Swagger Provider = "swagger"
	Scalar  Provider = "scalar"
	Redoc   Provider = "redoc"
)

type Config struct {
	Provider    Provider
	Title       string
	DocumentURL string
	CDNBaseURL  string
}
