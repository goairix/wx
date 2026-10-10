package documents

import (
	"errors"
	"strconv"

	coreerrors "github.com/goairix/wx/v2/core/errors"
	"github.com/goairix/wx/v2/work/aibot"
)

func wrapError(operation string, result *ToolResult, cause error) error {
	if cause == nil {
		return nil
	}
	wrapped := aibot.WrapError("documents."+operation, "", cause)
	var metadata *coreerrors.Error
	if errors.As(wrapped, &metadata) {
		if result != nil {
			if result.RequestID != "" {
				metadata.RequestID = result.RequestID
			}
			if result.HTTPStatus != 0 {
				metadata.HTTPStatus = result.HTTPStatus
			}
		}
		var rpc *RPCError
		if errors.As(cause, &rpc) {
			metadata.Code = strconv.Itoa(rpc.Code)
		}
	}
	return wrapped
}
