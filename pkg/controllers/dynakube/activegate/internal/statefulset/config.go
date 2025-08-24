package statefulset

import (
	"github.com/rkitindi-kr/dynatrace-operator/pkg/logd"
)

const (
	InternalProxySecretVolumeName = "internal-proxy-secret-volume"
)

var (
	log = logd.Get().WithName("activegate-statefulset")
)
