package elastic

import (
	"github.com/fluid-cloudnative/fluid/pkg/ddc/alluxio/operations"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"strconv"
	"time"
)

func ScaleOut(client client.Client, jobName string, namespace string, configmap string) map[string]string{
	log := ctrl.Log.WithName("ScaleOut")
	podRecord := map[string]string{}
	podName := jobName + "-" + strconv.FormatInt(time.Now().Unix(), 10)
	podIP, err := createPod(client, podName, namespace, configmap, false)
	if err == nil {
		podRecord[podName] = podIP
	} else {
		setupLog.Error(err, "cannot create pod or get pod ip", "podName", podName)
	}

	fileUtil := operations.NewAlluxioFileUtils(jobName + "-master", "worker", namespace, log)
	_ = fileUtil.AddWorker(podIP)

	return podRecord

}
