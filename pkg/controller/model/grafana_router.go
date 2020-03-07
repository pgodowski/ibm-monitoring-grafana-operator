//
// Copyright 2020 IBM Corporation
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//
package model

import (
	"fmt"

	conf "github.com/IBM/ibm-grafana-operator/pkg/controller/config"
	corev1 "k8s.io/api/core/v1"

	"github.com/IBM/ibm-grafana-operator/pkg/apis/operator/v1alpha1"
)

func getVolumeMountsForRouter() []corev1.VolumeMount {
	return []corev1.VolumeMount{
		corev1.VolumeMount{
			Name:      "router-config",
			MountPath: "/opt/ibm/router/conf",
		},
		corev1.VolumeMount{
			Name:      "router-entry",
			MountPath: "/opt/ibm/router/entry",
		},
		corev1.VolumeMount{
			Name:      "grafana-storage",
			MountPath: "/test",
			ReadOnly:  true,
		},
		corev1.VolumeMount{
			Name:      "monitoring-ca-certs",
			MountPath: "/opt/ibm/router/ca-certs",
		},
		corev1.VolumeMount{
			Name:      "monitoring-certs",
			MountPath: "/opt/ibm/router/certs",
		},
		corev1.VolumeMount{
			Name:      "grafana-lua-script-config",
			MountPath: "/opt/lua-scripts",
		},
		corev1.VolumeMount{
			Name:      "util-lua-script-config",
			MountPath: "/opt/ibm/router/nginx/conf/monitoring-util.lua",
			SubPath:   "monitoring-util.lua",
		},
	}
}

// hardcode the setting
func getGrafanaRouterSC() *corev1.SecurityContext {

	True := true
	False := false
	return &corev1.SecurityContext{
		Capabilities: &corev1.Capabilities{
			Add: []corev1.Capability{"ALL"},
			Drop: []corev1.Capability{"CHOWN", "NET_ADMIN",
				"NET_RAW", "LEASE",
				"SETGID", "SETUID"},
		},
		Privileged:               &True,
		AllowPrivilegeEscalation: &False,
		ReadOnlyRootFilesystem:   &True,
	}

}

func getRouterProbe(delay, period int) *corev1.Probe {
	config := conf.GetControllerConfig()
	iamNamespace := config.GetConfigString(conf.IAMNamespaceName, "")
	iamServicePort := config.GetConfigString(conf.IAMServicePortName, "")
	wget := "wget --spider --no-check-certificate -S 'https://platform-identity-provider"
	checkURL := wget + iamNamespace + ".svc." + ClusterDomain + ":" + iamServicePort + "/v1/info"
	checkCMD := []string{"sh", "-c", checkURL}

	handler := corev1.Handler{}
	handler.Exec = &corev1.ExecAction{}
	handler.Exec.Command = checkCMD
	return &corev1.Probe{
		Handler: corev1.Handler{
			Exec: &corev1.ExecAction{
				Command: checkCMD,
			},
		},
		InitialDelaySeconds: int32(delay),
		PeriodSeconds:       int32(period),
	}
}

func createRouterContainer(cr *v1alpha1.Grafana) corev1.Container {

	return corev1.Container{
		Name:    "router",
		Image:   fmt.Sprintf("%s:%s", RouterImage, RouterImageTag),
		Command: []string{"/opt/ibm/router/entry/entrypoint.sh"},
		Ports: []corev1.ContainerPort{
			{
				Name:          "router",
				ContainerPort: DefaultRouterPort,
				Protocol:      "TCP",
			},
		},
		Resources:                getContainerResource(cr, "Router"),
		LivenessProbe:            getRouterProbe(30, 20),
		ReadinessProbe:           getRouterProbe(32, 10),
		SecurityContext:          getGrafanaRouterSC(),
		VolumeMounts:             getVolumeMountsForRouter(),
		Env:                      setupAdminEnv("GF_SECURITY_ADMIN_USER", "GF_SECURITY_ADMIN_PASSWORD"),
		TerminationMessagePath:   "/dev/termination-log",
		TerminationMessagePolicy: "File",
		ImagePullPolicy:          "IfNotPresent",
	}
}
