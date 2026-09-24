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

	requireOfficialOption(official.WithLogger(logger))
	requireMiniAppOption(miniapp.WithLogger(logger))
	requireMobileAppOption(mobileapp.WithLogger(logger))
	requireOpenPlatformOption(openplatform.WithLogger(logger))
	requireWorkOption(work.WithLogger(logger))
	requireHealthCardOption(healthcard.WithLogger(logger))
}

func requireOfficialOption(official.Option)         {}
func requireMiniAppOption(miniapp.Option)           {}
func requireMobileAppOption(mobileapp.Option)       {}
func requireOpenPlatformOption(openplatform.Option) {}
func requireWorkOption(work.Option)                 {}
func requireHealthCardOption(healthcard.Option)     {}
