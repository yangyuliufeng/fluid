/*

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1alpha1

import (
	"github.com/fluid-cloudnative/fluid/pkg/common"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// CurrentEpoch tells user the current epoch's status of ElasticTrainJob
type CurrentEpoch struct {
	Sequence int    `json:"sequence,omitempty"`
	Speed    string `json:"speed,omitempty"`
	TimeCost string `json:"timeCost,omitempty"`
}

// UpToNow tells user the up to now status of ElasticTrainJob
type UpToNow struct {
	MeanSpeed     string `json:"meanSpeed,omitempty"`
	TotalTimeCost string `json:"totalTimeCost,omitempty"`
}

// Resource is the resource cost of ElasticTrainJob
type Resource struct {
	CPUCore  int `json:"CPUCore,omitempty"`
	MemGi    int `json:"MemGi,omitempty"`
	GPUCard  int `json:"GPUCard,omitempty"`
	GPUMemGi int `json:"GPUMemGi,omitempty"`
}

// ElasticTrainJobSpec defines the desired state of ElasticTrainJob
type ElasticTrainJobSpec struct {
	// user should create a configmap with the main.py file
	ConfigmapName string `json:"configmapName,omitempty"`
	EpochNumbers  int    `json:"epochNumbers,omitempty"`
	BatchSize     int    `json:"batchSize,omitempty"`
	// use should tell the system the ufsTotal of dataset
	UFSTotal string `json:"ufsTotal,omitempty"`
	// user should submit a spec TimeCost
	// it should be in the unit of sec
	TimeCost string `json:"timeCost,omitempty"`

	MinWorkerNum    int      `json:"minWorkerNum,omitempty"`
	MaxWorkerNum    int      `json:"maxWorkerNum,omitempty"`
	InitWorkerNum   int      `json:"initWorkerNum,omitempty"`
	ResourceRequest Resource `json:"resourceRequest,omitempty"`
}

// ElasticTrainJobStatus defines the observed state of ElasticTrainJob
type ElasticTrainJobStatus struct {
	CurrentEpoch CurrentEpoch `json:"currentEpoch,omitempty"`
	UptoNow      UpToNow      `json:"upToNow,omitempty"`
	WorkerNum    int          `json:"workerNum,omitempty"`
	Phase        common.Phase `json:"phase,omitempty"`
	ResourceUsed Resource     `json:"resourceUsed,omitempty"`
}

// +kubebuilder:printcolumn:name="WorkerNum",type="string",JSONPath=`.status.workerNum`
// +kubebuilder:printcolumn:name="Phase",type="string",JSONPath=`.status.phase`
// +kubebuilder:printcolumn:name="Sequence",type="string",JSONPath=`.status.currentEpoch.sequence`
// +kubebuilder:printcolumn:name="Speed",type="string",JSONPath=`.status.currentEpoch.speed`
// +kubebuilder:printcolumn:name="TimeCost",type="string",JSONPath=`.status.currentEpoch.timeCost`
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=`.metadata.creationTimestamp`
// +kubebuilder:object:root=true
// +genclient

// ElasticTrainJob is the Schema for the elastictrainjobs API
type ElasticTrainJob struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ElasticTrainJobSpec   `json:"spec,omitempty"`
	Status ElasticTrainJobStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// ElasticTrainJobList contains a list of ElasticTrainJob
type ElasticTrainJobList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ElasticTrainJob `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ElasticTrainJob{}, &ElasticTrainJobList{})
}
