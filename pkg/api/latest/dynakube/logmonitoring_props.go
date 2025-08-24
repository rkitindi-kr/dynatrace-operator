package dynakube

import (
	"github.com/rkitindi-kr/dynatrace-operator/pkg/api/latest/dynakube/logmonitoring"
)

func (dk *DynaKube) LogMonitoring() *logmonitoring.LogMonitoring {
	lm := &logmonitoring.LogMonitoring{
		Spec:         dk.Spec.LogMonitoring,
		TemplateSpec: dk.Spec.Templates.LogMonitoring,
	}
	lm.SetName(dk.Name)
	lm.SetHostAgentDependency(dk.OneAgent().IsDaemonsetRequired())

	return lm
}
