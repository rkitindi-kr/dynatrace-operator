package configsecret

import (
	"github.com/rkitindi-kr/dynatrace-operator/pkg/logd"
)

const (
	DeploymentConfigFilename = "deployment.conf"
)

var (
	log = logd.Get().WithName("logmonitoring-config-secret")
)
