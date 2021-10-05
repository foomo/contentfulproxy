package config

import (
	keelconfig "github.com/foomo/keel/config"
	"github.com/spf13/viper"
)

const (
	WebserverAddress = "webserver.address"
	WebserverPath    = "webserver.path"
	BackendURL       = "backend.url"
	WebhookURLs      = "webhook.urls"
)

func DefaultWebhookURLs(c *viper.Viper) func() []string {
	return keelconfig.GetStringSlice(c, WebhookURLs, []string{})
}

func DefaultWebserverAddress(c *viper.Viper) func() string {
	return keelconfig.GetString(c, WebserverAddress, ":80")
}

func DefaultWebserverPath(c *viper.Viper) func() string {
	return keelconfig.GetString(c, WebserverPath, "")
}

func DefaultBackendURL(c *viper.Viper) func() string {
	return keelconfig.GetString(c, BackendURL, "https://cdn.contentful.com")
}
