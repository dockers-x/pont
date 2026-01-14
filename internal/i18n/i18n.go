package i18n

import (
	"embed"
	"encoding/json"

	"github.com/nicksnyder/go-i18n/v2/i18n"
	"golang.org/x/text/language"
)

//go:embed locales/*.json
var localeFS embed.FS

var bundle *i18n.Bundle

// Init initializes the i18n bundle with embedded locale files
func Init() error {
	bundle = i18n.NewBundle(language.English)
	bundle.RegisterUnmarshalFunc("json", json.Unmarshal)

	// Load all locale files
	locales := []string{"en", "zh", "ja"}
	for _, locale := range locales {
		if _, err := bundle.LoadMessageFileFS(localeFS, "locales/"+locale+".json"); err != nil {
			return err
		}
	}

	return nil
}

// GetLocalizer returns a localizer for the given language tags
func GetLocalizer(langs ...string) *i18n.Localizer {
	return i18n.NewLocalizer(bundle, langs...)
}

// T translates a message ID with optional template data
func T(localizer *i18n.Localizer, messageID string, templateData map[string]interface{}) string {
	msg, err := localizer.Localize(&i18n.LocalizeConfig{
		MessageID:    messageID,
		TemplateData: templateData,
	})
	if err != nil {
		return messageID
	}
	return msg
}
