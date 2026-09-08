package v1_13_0

import (
	"context"
	"errors"
	"github.com/SealinGp/sing-box-easy/app/pkg/config"
	"github.com/SealinGp/sing-box-easy/app/pkg/fault"
	"github.com/SealinGp/sing-box-easy/app/pkg/outbounds/rules"
	"github.com/cloudwego/hertz/pkg/app"
)

func respondOperationError(ctx context.Context, c *app.RequestContext, err error) {
	var validation *config.ValidationError
	if errors.As(err, &validation) {
		respondConfigError(ctx, c, err)
		return
	}
	code := CodeInternalError
	switch {
	case errors.Is(err, noderules.ErrNotFound):
		code = CodeNotFound
	case errors.Is(err, noderules.ErrInvalidInput):
		code = CodeValidationError
	case errors.Is(err, noderules.ErrDuplicateName), errors.Is(err, noderules.ErrFallbackProtected):
		code = CodeConflict
	}
	var e *fault.Error
	if errors.As(err, &e) {
		switch e.Kind {
		case fault.Input:
			code = CodeBadRequest
		case fault.Configuration:
			code = CodeConfigError
		case fault.Invalid:
			code = CodeValidationError
		case fault.Missing:
			code = CodeNotFound
		case fault.Conflict:
			code = CodeConflict
		case fault.Forbidden:
			code = CodeForbidden
		case fault.Unavailable:
			code = CodeServiceError
		case fault.Failed:
			code = CodeOperationFailed
		}
	}
	respErr(ctx, c, code, err.Error())
}

func isOperationFault(err error) bool { var e *fault.Error; return errors.As(err, &e) }
