package operations

import (
	ctrl "sigs.k8s.io/controller-runtime"
	"testing"
)

func TestDeleteWorker(t *testing.T) {
	log := ctrl.Log.WithName("ScaleOut")


	fileUtil := NewAlluxioFileUtils("benchmark-master", "worker", "default", log)
	_ = fileUtil.DeleteWorker("182.230.235.203")


}