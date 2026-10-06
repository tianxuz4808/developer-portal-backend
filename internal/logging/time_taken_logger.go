package logging

import (
	"time"

	"go.uber.org/zap"
)

func LogTimeTaken(logger zap.Logger, startTime time.Time, endTime time.Time) {
	timeDelta := endTime.Sub(startTime)
	logger.Info("the process took: ",
		zap.String("time taken", timeDelta.String()),
	)
}
