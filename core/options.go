package openapi

type Option func(*API)

func WithTitle(title string) Option {
	return func(api *API) {
		api.doc.Info.Title = title
	}
}

func WithDescription(description string) Option {
	return func(api *API) {
		api.doc.Info.Description = description
	}
}

func WithVersion(version string) Option {
	return func(api *API) {
		api.doc.Info.Version = version
	}
}

func WithServer(url, description string) Option {
	return func(api *API) {
		api.doc.Servers = append(api.doc.Servers, Server{
			URL:         url,
			Description: description,
		})
	}
}
