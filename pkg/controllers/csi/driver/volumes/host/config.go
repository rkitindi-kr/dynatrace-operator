package host

import (
	"github.com/rkitindi-kr/dynatrace-operator/pkg/logd"
)

const Mode = "host"

var log = logd.Get().WithName("csi-hostvolume")
