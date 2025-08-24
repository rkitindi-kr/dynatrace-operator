package validation

import (
	"context"

	"github.com/rkitindi-kr/dynatrace-operator/pkg/api/v1alpha2/edgeconnect"
	"github.com/rkitindi-kr/dynatrace-operator/pkg/util/installconfig"
)

var (
	errorModuleDisabled = installconfig.GetModuleValidationErrorMessage("EdgeConnect")
)

func isModuleDisabled(_ context.Context, v *Validator, _ *edgeconnect.EdgeConnect) string {
	if v.modules.EdgeConnect {
		return ""
	}

	return errorModuleDisabled
}
