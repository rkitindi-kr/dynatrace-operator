package supportarchive

import (
	"github.com/rkitindi-kr/dynatrace-operator/pkg/logd"
)

type collector interface {
	Name() string
	Do() error
}

type collectorCommon struct {
	supportArchive archiver
	log            logd.Logger
}
