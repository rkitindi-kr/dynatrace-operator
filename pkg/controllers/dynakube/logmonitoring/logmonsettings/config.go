package logmonsettings

import (
	"github.com/rkitindi-kr/dynatrace-operator/pkg/logd"
)

var (
	log = logd.Get().WithName("logmonitoring-settings")
)
