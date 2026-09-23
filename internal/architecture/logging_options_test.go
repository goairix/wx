package architecture

import (
	"testing"

	"github.com/goairix/wx/v2/core/logging"
	"github.com/goairix/wx/v2/healthcard"
	"github.com/goairix/wx/v2/miniapp"
	"github.com/goairix/wx/v2/mobileapp"
	"github.com/goairix/wx/v2/official"
	"github.com/goairix/wx/v2/openplatform"
	"github.com/goairix/wx/v2/work"
)

func TestEveryPlatformExposesLoggerOption(t *testing.T) {
	logger := logging.Nop()

	var _ official.Option = official.WithLogger(logger)
	var _ miniapp.Option = miniapp.WithLogger(logger)
	var _ mobileapp.Option = mobileapp.WithLogger(logger)
	var _ openplatform.Option = openplatform.WithLogger(logger)
	var _ work.Option = work.WithLogger(logger)
	var _ healthcard.Option = healthcard.WithLogger(logger)
}
