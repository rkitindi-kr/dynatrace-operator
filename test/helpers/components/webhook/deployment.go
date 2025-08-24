//go:build e2e

package webhook

import (
	"github.com/rkitindi-kr/dynatrace-operator/pkg/webhook"
	"github.com/rkitindi-kr/dynatrace-operator/test/helpers/kubeobjects/deployment"
	"sigs.k8s.io/e2e-framework/pkg/env"
)

const (
	DeploymentName = webhook.DeploymentName
)

func WaitForDeployment(namespace string) env.Func {
	return deployment.WaitFor(DeploymentName, namespace)
}
