package elastic

import (
	"context"
	datav1alpha1 "github.com/fluid-cloudnative/fluid/api/v1alpha1"
	"github.com/fluid-cloudnative/fluid/pkg/common"
	elastictl "github.com/fluid-cloudnative/fluid/pkg/elastic"
	"github.com/fluid-cloudnative/fluid/pkg/utils"
	"github.com/fluid-cloudnative/fluid/pkg/utils/kubeclient"
	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"time"
)

const (
	finalizer = "fluid-elastic-controller-finalizer"
)

// ElasticReconciler reconciles a Elastic object
type ElasticReconciler struct {
	client.Client
	Recorder      record.EventRecorder
	Log           logr.Logger
	Scheme        *runtime.Scheme
	ResyncPeriod  time.Duration
	PodIPs        map[types.NamespacedName]map[string]string
	EpochStatuses map[types.NamespacedName][]elastictl.EpochStatus
	SpeedUnits    map[types.NamespacedName]string
}

type reconcileRequestContext struct {
	context.Context
	ElasticTrainJob datav1alpha1.ElasticTrainJob
	types.NamespacedName
}

// +kubebuilder:rbac:groups=data.fluid.io,resources=datasets,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=data.fluid.io,resources=datasets/status,verbs=get;update;patch

func (r *ElasticReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := reconcileRequestContext{
		Context:        context.Background(),
		NamespacedName: req.NamespacedName,
	}

	notFound := false
	r.Log.Info("process the request", "request", req)

	if err := r.Get(ctx, req.NamespacedName, &ctx.ElasticTrainJob); err != nil {
		r.Log.Info("Unable to fetch ElasticTrainJob", "reason", err)
		if utils.IgnoreNotFound(err) != nil {
			r.Log.Error(err, "failed to get ElasticTrainJob")
			return ctrl.Result{}, err
		} else {
			notFound = true
		}
	} else {
		return r.reconcileElastic(ctx)
	}

	if notFound {
		r.Log.Info("Not found!", "NamespacedName", req.NamespacedName)
	}
	return ctrl.Result{}, nil
}

// reconcile Elastic
func (r *ElasticReconciler) reconcileElastic(ctx reconcileRequestContext) (ctrl.Result, error) {

	// 1. Check if need to delete ElasticTrainJob
	if utils.HasDeletionTimestamp(ctx.ElasticTrainJob.ObjectMeta) {
		return r.reconcileElasticDeletion(ctx)
	}

	// 2.Add finalizer
	if !utils.ContainsString(ctx.ElasticTrainJob.ObjectMeta.GetFinalizers(), finalizer) {
		return r.addFinalizerAndRequeue(ctx)
	}

	// 3. init the record
	if _, find := r.PodIPs[ctx.NamespacedName]; !find {
		r.Log.Info("first create, init a record to store info", "ElasticTrainJob", ctx.ElasticTrainJob.Name)
		r.PodIPs[ctx.NamespacedName] = map[string]string{}
		r.EpochStatuses[ctx.NamespacedName] = []elastictl.EpochStatus{}
	}

	// 4. ElasticTrainJob's phase transition: None -> Pending -> Executing -> Complete or Failed
	switch ctx.ElasticTrainJob.Status.Phase {
	case common.PhaseNone:
		return r.reconcileNoneElastic(ctx)
	case common.PhasePending:
		return r.reconcilePendingElastic(ctx)
	case common.PhaseExecuting:
		return r.reconcileExecutingElastic(ctx)
	case common.PhaseComplete:
		return r.reconcileCompleteElastic(ctx)
	case common.PhaseFailed:
		return r.reconcileFailedElastic(ctx)
	default:
		r.Log.Info("Unknown Elastic phase, won't reconcile it", "Elastic", ctx.ElasticTrainJob)
	}
	return utils.NoRequeue()
}

// reconcile Elastic delete
func (r *ElasticReconciler) reconcileElasticDeletion(ctx reconcileRequestContext) (ctrl.Result, error) {
	if !ctx.ElasticTrainJob.ObjectMeta.GetDeletionTimestamp().IsZero() {
		// 1. delete all pods for this training job
		for podName, _ := range r.PodIPs[types.NamespacedName{Namespace: ctx.ElasticTrainJob.Namespace, Name: ctx.ElasticTrainJob.Name}] {
			_ = kubeclient.DeletePod(r.Client, podName, ctx.ElasticTrainJob.Namespace)
		}
		// 2. delete the records in ElasticReconciler
		delete(r.PodIPs, ctx.NamespacedName)
		delete(r.EpochStatuses, ctx.NamespacedName)
		delete(r.SpeedUnits, ctx.NamespacedName)
		// 3. delete the ElasticTrainJob
		ctx.ElasticTrainJob.ObjectMeta.Finalizers = utils.RemoveString(ctx.ElasticTrainJob.ObjectMeta.Finalizers, finalizer)

		if err := r.Update(ctx, &ctx.ElasticTrainJob); err != nil {
			r.Log.Error(err, "Failed to remove finalizer")
			return ctrl.Result{}, err
		}
		r.Log.Info("Finalizer is removed", "ElasticTrainJob", ctx.ElasticTrainJob)
	}

	r.Log.Info("delete the ElasticTrainJob successfully", "dataset", ctx.ElasticTrainJob)

	return ctrl.Result{}, nil
}

func (r *ElasticReconciler) addFinalizerAndRequeue(ctx reconcileRequestContext) (ctrl.Result, error) {
	ctx.ElasticTrainJob.ObjectMeta.Finalizers = append(ctx.ElasticTrainJob.ObjectMeta.Finalizers, finalizer)
	prevGeneration := ctx.ElasticTrainJob.ObjectMeta.GetGeneration()
	if err := r.Update(ctx, &ctx.ElasticTrainJob); err != nil {
		r.Log.Error(err, "Failed to add finalizer", "StatusUpdateError", ctx)
		return utils.RequeueIfError(err)
	}

	return utils.RequeueImmediatelyUnlessGenerationChanged(prevGeneration, ctx.ElasticTrainJob.ObjectMeta.GetGeneration())
}

func (r *ElasticReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&datav1alpha1.ElasticTrainJob{}).
		Complete(r)
}
