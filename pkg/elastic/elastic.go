package elastic

import (
	"context"
	"fmt"
	"github.com/fluid-cloudnative/fluid/pkg/utils/kubeclient"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"strconv"
	"strings"
	"time"
)

const (
	waitTimes = 100
)

var setupLog = ctrl.Log.WithName("elastic")

func CreateJob(client client.Client, jobName string, namespace string, configmap string, initWorkerNum int) map[string]string {
	podRecord := map[string]string{}
	podIP, err := createPod(client, jobName+"-master", namespace, configmap, true)
	if err != nil {
		return podRecord
	}
	podRecord[jobName+"-master"] = podIP

	// if no need to create other workers
	if initWorkerNum < 2 {
		return podRecord
	}

	for i := 1; i <= initWorkerNum-1; i++ {
		podName := jobName + "-" + strconv.FormatInt(time.Now().Unix(), 10)
		podIP, err := createPod(client, podName, namespace, configmap, false)
		if err == nil {
			podRecord[podName] = podIP
		} else {
			setupLog.Error(err, "cannot create pod or get pod ip", "podName", podName)
		}
	}
	return podRecord
}

// createPod create a training Pod and get the podName and podIp
func createPod(client client.Client, podName string, namespace string, configmap string, master bool) (podIP string, err error) {
	setupLog.Info("start to create pod", "podName", podName)

	imageUrl := "reg.harbor.com/public-test/horovod"
	imageTag := "v13"

	var container = corev1.Container{
		Name:            "worker",
		Image:           imageUrl + ":" + imageTag,
		ImagePullPolicy: corev1.PullIfNotPresent,
		VolumeMounts: []corev1.VolumeMount{
			{
				Name:      "python-file",
				MountPath: "/project",
			},
		},
	}
	if master {
		container.Command = []string{"bash", "/entry-master.sh"}
	}

	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      podName,
			Namespace: namespace,
		},
		Spec: corev1.PodSpec{
			RestartPolicy: corev1.RestartPolicyNever,
			Containers: []corev1.Container{
				container,
			},
			Volumes: []corev1.Volume{
				{
					Name: "python-file",
					VolumeSource: corev1.VolumeSource{
						ConfigMap: &corev1.ConfigMapVolumeSource{
							LocalObjectReference: corev1.LocalObjectReference{
								Name: configmap,
							},
						},
					},
				},
			},
		},
	}

	found, err := kubeclient.IsPodExist(client, podName, namespace)

	// if cannot found or unable to sure, create a new pod
	if err != nil || !found {
		err = client.Create(context.TODO(), pod)
		if err != nil {
			setupLog.Error(err, "cannot create pod", "podName", podName, "namespace", namespace)
			return
		}
	}

	// wait until pod having ip
	for i := 1; i <= waitTimes; i++ {
		time.Sleep(10 * time.Second)
		pod, err = kubeclient.GetPodByName(client, podName, "default")
		if err != nil {
			setupLog.Error(err, "cannot get pod, wait 10 seconds", "podName", podName, "waitTimes", i)
		}
		if pod.Status.PodIP == "" {
			setupLog.Info("pod does not have ip, wait 10 seconds", "waitTimes", i)
		} else {
			break
		}
	}

	if pod.Status.PodIP == "" {
		err = fmt.Errorf("cannot get ip of pod")
		return
	} else {
		podIP = pod.Status.PodIP
		err = nil
		return
	}
}

func ParseLastLineLog(log string) (currentEpoch int, timeCost float64, speed float64, speedUnit string, err error) {
	if !strings.Contains(log, "<stdout>:status of epoch") {
		err = fmt.Errorf("log is not supported to be parsed")
		return
	}
	str :=  strings.Split(log, "]")
	if len(str) < 1 {
		err = fmt.Errorf("log is not supported to be parsed")
		return
	}
	log = strings.TrimPrefix(str[1], "<stdout>:status of epoch ")
	str = strings.Split(log, ":")
	currentEpoch, err = strconv.Atoi(str[0])
	if err != nil {
		return
	}
	str = strings.Split(str[1], ",")
	temp := strings.TrimPrefix(str[0], " time cost ")
	temp = strings.TrimSuffix(temp, " sec")
	timeCost, err = strconv.ParseFloat(temp, 64)
	if err != nil {
		return
	}
	temp = strings.TrimPrefix(str[1], " speed is ")
	str = strings.Split(temp, " ")
	speedUnit = strings.TrimPrefix(temp, str[0]+" ")
	speed, err = strconv.ParseFloat(str[0], 64)
	if err != nil {
		return
	}
	return
}
