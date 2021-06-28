package elastic

import (
	"context"
	"fmt"
	"github.com/fluid-cloudnative/fluid/pkg/common"
	"github.com/fluid-cloudnative/fluid/pkg/ddc/alluxio/operations"
	elastictl "github.com/fluid-cloudnative/fluid/pkg/elastic"
	"github.com/fluid-cloudnative/fluid/pkg/utils"
	"github.com/fluid-cloudnative/fluid/pkg/utils/kubeclient"
	"os/exec"
	ctrl "sigs.k8s.io/controller-runtime"
	"strings"
	"time"
)

func (r *ElasticReconciler) reconcileNoneElastic(ctx reconcileRequestContext) (ctrl.Result, error) {
	elasticTrainJobToUpdate := ctx.ElasticTrainJob.DeepCopy()

	// 1. change the phase from none to pending
	elasticTrainJobToUpdate.Status.Phase = common.PhasePending

	// 2. decide the InitWorkerNum and Min/Max WorkerNum
	if ctx.ElasticTrainJob.Spec.InitWorkerNum == 0 {
		// TODO: get the suitable init worker numbers from python
		elasticTrainJobToUpdate.Spec.InitWorkerNum = 3
	} else if ctx.ElasticTrainJob.Spec.InitWorkerNum < 3 {
		// if InitWorkerNum < 3, horovod cannot start
		elasticTrainJobToUpdate.Spec.InitWorkerNum = 3
	}
	if ctx.ElasticTrainJob.Spec.MinWorkerNum == 0 {
		elasticTrainJobToUpdate.Spec.MinWorkerNum = 1
	}

	if err := r.Update(context.TODO(), elasticTrainJobToUpdate); err != nil {
		r.Log.Error(err, "failed to update the elasticTrainJob", "elasticTrainJob", ctx.ElasticTrainJob.Name)
		return utils.RequeueIfError(err)
	}
	r.Log.Info("Update phase of the elasticTrainJob to Pending successfully", "elasticTrainJob", ctx.ElasticTrainJob.Name)
	return utils.RequeueImmediately()
}

func (r *ElasticReconciler) reconcilePendingElastic(ctx reconcileRequestContext) (ctrl.Result, error) {
	fileUtil := operations.NewAlluxioFileUtils(ctx.ElasticTrainJob.Name+"-master", "worker", ctx.ElasticTrainJob.Namespace, r.Log)
	// 1. create a job with InitWorkerNumber workers
	// will record the IP in PodIPs
	r.PodIPs[ctx.NamespacedName] = elastictl.CreateJob(
		r.Client,
		ctx.ElasticTrainJob.Name,
		ctx.ElasticTrainJob.Namespace,
		ctx.ElasticTrainJob.Spec.ConfigmapName,
		ctx.ElasticTrainJob.Spec.InitWorkerNum,
	)

	// 2. inject the PodIPs into horovod
	for _, podIP := range r.PodIPs[ctx.NamespacedName] {
		// TODO: what to do if cannot add
		_ = fileUtil.AddWorker(podIP)
	}

	// 3. update the ElasticTrainJob
	elasticTrainJobToUpdate := ctx.ElasticTrainJob.DeepCopy()
	elasticTrainJobToUpdate.Status.Phase = common.PhaseExecuting
	elasticTrainJobToUpdate.Status.WorkerNum = len(r.PodIPs[ctx.NamespacedName])
	if err := r.Update(context.TODO(), elasticTrainJobToUpdate); err != nil {
		r.Log.Error(err, "failed to update the elasticTrainJob", "elasticTrainJob", ctx.ElasticTrainJob.Name)
		return utils.RequeueIfError(err)
	}
	r.Log.Info("Update phase of the elasticTrainJob to Pending successfully", "elasticTrainJob", ctx.ElasticTrainJob.Name)
	return utils.RequeueAfterInterval(5 * time.Second)
}

func (r *ElasticReconciler) reconcileExecutingElastic(ctx reconcileRequestContext) (ctrl.Result, error) {
	// 1. get horovod pod, enter into completed or failed logic
	horovodPod, err := kubeclient.GetPodByName(r.Client, ctx.ElasticTrainJob.Name+"-master", ctx.ElasticTrainJob.Namespace)
	if err != nil {
		r.Log.Error(err, "Failed to get horovod pod")
		return utils.RequeueIfError(err)
	}
	if kubeclient.IsSucceededPod(horovodPod) {
		elasticTrainJobToUpdate := ctx.ElasticTrainJob.DeepCopy()
		elasticTrainJobToUpdate.Status.Phase = common.PhaseComplete
		if err := r.Status().Update(context.TODO(), elasticTrainJobToUpdate); err != nil {
			r.Log.Error(err, "the horovod pod has completed, but failed to update the ElasticTrainJob")
			return utils.RequeueIfError(err)
		}
		r.Log.Info("Update phase of the ElasticTrainJob to Complete successfully")
		return utils.RequeueImmediately()
	} else if kubeclient.IsFailedPod(horovodPod) {
		elasticTrainJobToUpdate := ctx.ElasticTrainJob.DeepCopy()
		elasticTrainJobToUpdate.Status.Phase = common.PhaseFailed
		if err := r.Status().Update(context.TODO(), elasticTrainJobToUpdate); err != nil {
			r.Log.Error(err, "the horovod pod has failed, but failed to update the ElasticTrainJob")
			return utils.RequeueIfError(err)
		}
		r.Log.Info("Update phase of the ElasticTrainJob to Failed successfully")
		return utils.RequeueImmediately()
	}

	// 2. update the status of ElasticTrainJob
	cmd := exec.Command("kubectl", "logs", ctx.ElasticTrainJob.Name+"-master", "-n", ctx.ElasticTrainJob.Namespace, "--tail=1")
	stdout, err := cmd.Output()
	if err != nil {
		r.Log.Error(err, "fail to get logs", "output", stdout)
		return utils.RequeueIfError(err)
	}
	lastLineLog := string(stdout)
	if strings.HasPrefix(lastLineLog, "[0]<stdout>:status of epoch ") {
		currentEpoch, currentTimeCost, currentSpeed, speedUnit, err := elastictl.ParseLastLineLog(lastLineLog)
		if err != nil {
			return utils.RequeueIfError(err)
		}
		if currentEpoch == 0 {
			r.SpeedUnits[ctx.NamespacedName] = speedUnit
		}

		// it's a new epoch, need to update status of ElasticTrainJob
		if ctx.ElasticTrainJob.Status.CurrentEpoch.Sequence != currentEpoch {
			r.EpochStatuses[ctx.NamespacedName] = append(r.EpochStatuses[ctx.NamespacedName],
				elastictl.NewEpochStatus(currentTimeCost, currentSpeed))

			totalTime, meanSpeed := elastictl.CalMeanAndTotal(r.EpochStatuses[ctx.NamespacedName])
			elasticTrainJobToUpdate := ctx.ElasticTrainJob.DeepCopy()
			elasticTrainJobToUpdate.Status.CurrentEpoch.Sequence = currentEpoch
			elasticTrainJobToUpdate.Status.CurrentEpoch.TimeCost = fmt.Sprintf("%.1f", currentTimeCost) + "s"
			elasticTrainJobToUpdate.Status.CurrentEpoch.Speed = fmt.Sprintf("%.1f", currentSpeed) + " " + speedUnit
			elasticTrainJobToUpdate.Status.UptoNow.TotalTimeCost = fmt.Sprintf("%.1f", totalTime) + "s"
			elasticTrainJobToUpdate.Status.UptoNow.MeanSpeed = fmt.Sprintf("%.1f", meanSpeed) + " " + speedUnit

			if err := r.Update(context.TODO(), elasticTrainJobToUpdate); err != nil {
				r.Log.Error(err, "failed to update the elasticTrainJob", "elasticTrainJob", ctx.ElasticTrainJob.Name)
				return utils.RequeueIfError(err)
			}
		}
	}

	return utils.RequeueAfterInterval(5 * time.Second)
}

func (r *ElasticReconciler) reconcileCompleteElastic(ctx reconcileRequestContext) (ctrl.Result, error) {
	r.Log.Info("job has succeed", "Name", ctx.ElasticTrainJob.Name, "Namespace", ctx.ElasticTrainJob.Namespace)
	return utils.NoRequeue()
}

func (r *ElasticReconciler) reconcileFailedElastic(ctx reconcileRequestContext) (ctrl.Result, error) {
	r.Log.Info("job has failed", "Name", ctx.ElasticTrainJob.Name, "Namespace", ctx.ElasticTrainJob.Namespace)
	return utils.NoRequeue()
}
