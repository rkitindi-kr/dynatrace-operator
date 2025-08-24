package apimonitoring

import (
	"github.com/rkitindi-kr/dynatrace-operator/pkg/logd"
)

var (
	log = logd.Get().WithName("automatic-api-monitoring")
)
