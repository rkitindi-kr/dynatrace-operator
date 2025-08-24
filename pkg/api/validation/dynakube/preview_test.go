package validation

import (
	"testing"

	"github.com/rkitindi-kr/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/rkitindi-kr/dynatrace-operator/pkg/api/latest/dynakube/oneagent"
)

func TestPreviewWarning(t *testing.T) {
	t.Run("no warning", func(t *testing.T) {
		assertAllowedWithoutWarnings(t, &dynakube.DynaKube{
			ObjectMeta: defaultDynakubeObjectMeta,
			Spec: dynakube.DynaKubeSpec{
				APIURL: testAPIURL,
				OneAgent: oneagent.Spec{
					ApplicationMonitoring: &oneagent.ApplicationMonitoringSpec{},
				},
			},
		})
	})
}
