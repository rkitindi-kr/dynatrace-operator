package validation

import (
	"github.com/rkitindi-kr/dynatrace-operator/pkg/logd"
)

const oneagentEnableVolumeStorageEnvVarName = "ONEAGENT_ENABLE_VOLUME_STORAGE"
const oneagentInstallerScriptURLEnvVarName = "ONEAGENT_INSTALLER_SCRIPT_URL"
const oneagentInstallerTokenEnvVarName = "ONEAGENT_INSTALLER_TOKEN"

var log = logd.Get().WithName("dynakube-validation")
