package multi_terminal

import (
	"fmt"

	"github.com/goairix/wx/v2/kernel/contracts"
)

func (m *MultiTerminal) AccessToken() (contracts.AccessToken, error) {
	return contracts.AccessToken{}, nil
}

func (m *MultiTerminal) AccessTokenCacheKey() string {
	return fmt.Sprintf("%s%s.%s", m.option.cacheKeyPrefix, "access_token", m.config.appId)
}
