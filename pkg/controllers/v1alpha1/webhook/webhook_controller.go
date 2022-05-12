/*
 Copyright 2022 The Fluid Authors.

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

package webhook

import (
	"context"
	"github.com/fluid-cloudnative/fluid/pkg/common"
	"github.com/fluid-cloudnative/fluid/pkg/utils"
	fluidwebhook "github.com/fluid-cloudnative/fluid/pkg/webhook"
	"github.com/go-logr/logr"
	"os"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const controllerName string = "WebhookController"

type WebhookReconciler struct {
	Client      client.Client
	WebhookName string
	CertDir     string
	caCert      []byte
	Log         logr.Logger
}

func (r *WebhookReconciler) Reconcile(context context.Context, req ctrl.Request) (ctrl.Result, error) {

	certBuilder := fluidwebhook.NewCertificateBuilder(r.Client, r.Log)
	caCert, err := certBuilder.BuildAndSyncCABundle(common.WebhookServiceName, common.WebhookName, r.CertDir)
	if err != nil || len(caCert) == 0 {
		r.Log.Error(err, "patch webhook CABundle failed")
		os.Exit(1)
	}

	err = certBuilder.PatchCABundle(r.WebhookName, r.caCert)
	if err != nil {
		r.Log.Error(err, "fail to patch CABundle of MutatingWebhookConfiguration on update")
	}

	return utils.NoRequeue()

}
