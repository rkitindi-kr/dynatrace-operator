package daemonset

import (
	"testing"

	"github.com/rkitindi-kr/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/rkitindi-kr/dynatrace-operator/pkg/api/latest/dynakube/activegate"
)

func getDynaKubeWithCerts(t *testing.T) dynakube.DynaKube {
	t.Helper()

	dk := dynakube.DynaKube{}
	dk.ActiveGate().TLSSecretName = "test"
	dk.ActiveGate().Capabilities = []activegate.CapabilityDisplayName{activegate.KubeMonCapability.DisplayName}

	return dk
}

func getDynaKubeWithAutomaticCerts(t *testing.T) dynakube.DynaKube {
	t.Helper()

	dk := dynakube.DynaKube{}
	dk.ActiveGate().Capabilities = []activegate.CapabilityDisplayName{activegate.KubeMonCapability.DisplayName}

	return dk
}
