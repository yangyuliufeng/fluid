package elastic

import (
	"github.com/shopspring/decimal"
)

type EpochStatus struct {
	TimeCost float64
	Speed    float64
}

func NewEpochStatus(timeCost float64, speed float64) EpochStatus {
	return EpochStatus{TimeCost: timeCost, Speed: speed}
}

func CalMeanAndTotal(epochStatuses []EpochStatus) (totalTime float64, meanSpeed float64) {
	decimal.DivisionPrecision = 1
	epochNumber := len(epochStatuses)
	for _, epochStatus := range epochStatuses {
		totalTime, _ = decimal.NewFromFloat(totalTime).Add(decimal.NewFromFloat(epochStatus.TimeCost)).Float64()
		meanSpeed, _ = decimal.NewFromFloat(meanSpeed).Add(decimal.NewFromFloat(epochStatus.Speed)).Float64()
	}
	meanSpeed, _ = decimal.NewFromFloat(meanSpeed).Div(decimal.NewFromFloat(float64(epochNumber))).Float64()
	return
}
