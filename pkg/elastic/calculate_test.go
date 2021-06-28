package elastic

import (
	"testing"
)

func TestCalMeanAndTotal(t *testing.T) {
	var epochStatuses = []EpochStatus{
		{
			TimeCost: 2.4,
			Speed:    3.2,
		},
		{
			TimeCost: 2.3,
			Speed:    3.5,
		},
	}
	totalTime, meanSpeed := CalMeanAndTotal(epochStatuses)

	if totalTime != 4.7 {
		t.Errorf("totalTime calculate err")
	}
	if meanSpeed != 3.4 {
		t.Errorf("meanSpeed calculate err")
	}

}
