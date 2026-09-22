package app

import (
	"fmt"

	"github.com/goairix/wx/v2/kernel/contracts"
)

func (app *App) AccessToken() (contracts.AccessToken, error) {
	return contracts.AccessToken{}, nil
}

func (app *App) AccessTokenCacheKey() string {
	return fmt.Sprintf("%s%s.%s", app.option.cacheKeyPrefix, "access_token", app.config.appId)
}
