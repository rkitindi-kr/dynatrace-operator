package otelc

import (
	"github.com/rkitindi-kr/dynatrace-operator/pkg/logd"
)

var (
	log = logd.Get().WithName("otel-collector")
)
