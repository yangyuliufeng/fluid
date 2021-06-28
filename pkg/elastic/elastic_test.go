package elastic

import (
	"testing"
)

func TestParseLastLineLog(t *testing.T) {
	var log = "[0]<stdout>:status of epoch 99: time cost 12.4 sec, speed is 25.7 img/sec per GPU"
	currentEpoch, timeCost, speed, speedUnit, err := ParseLastLineLog(log)
	if err != nil {
		t.Errorf("%s", err)
	}
	if currentEpoch != 99 {
		t.Errorf("currentEpoch parse err")
	}
	if timeCost != 12.4 {
		t.Errorf("timeCost parse err")
	}
	if speed != 25.7 {
		t.Errorf("speed parse err")
	}
	if speedUnit != "img/sec per GPU" {
		t.Errorf("speedUnit parse err")
	}
}
