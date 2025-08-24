package endpoint

import "github.com/rkitindi-kr/dynatrace-operator/pkg/logd"

var (
	log = logd.Get().WithName("telemetry-ingest-api-credentials-secret")
)
