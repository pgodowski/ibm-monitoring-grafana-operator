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
	"crypto/md5"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"path"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/IBM/ibm-grafana-operator/pkg/apis/operator/v1alpha1"
)

var log = logf.Log.WithName("data-source")

const (
	grafanaDatasourceFile = "datasource.yaml"
	certBasePath          = "/opt/ibm/monitoring/certs/"
	caBasePath            = "/opt/ibm/monitoring/caCerts/"
	certFile              = "tls.crt"
	keyFile               = "tls.key"
)

type grafanaDatasource struct {
	APIVersion int                        `json:"apiVersion,omitempty"`
	Datasource v1alpha1.GrafanaDatasource `json:"datasources,omitempty"`
}

// GrafanaDatasourceConfig create configmap for datasource
func GrafanaDatasourceConfig(cr *v1alpha1.Grafana) (*corev1.ConfigMap, error) {

	caCert, err := ioutil.ReadFile(path.Join(caBasePath, certFile))
	if err != nil {
		return nil, err
	}

	clientCert, err := ioutil.ReadFile(path.Join(certBasePath, certFile))
	if err != nil {
		return nil, err
	}

	clientKey, err := ioutil.ReadFile(path.Join(certBasePath, keyFile))
	if err != nil {
		return nil, err
	}

	cfg := cr.Spec.Datasource
	cfg.SecureJSONData.TLSCACert = string(caCert)
	cfg.SecureJSONData.TLSClientCert = string(clientCert)
	cfg.SecureJSONData.TLSClientKey = string(clientKey)

	dataSource := grafanaDatasource{
		APIVersion: 1,
		Datasource: *cfg,
	}
	bytesData, err := json.Marshal(dataSource)
	if err != nil {
		return nil, err
	}

	configMap := corev1.ConfigMap{}
	configMap.ObjectMeta = metav1.ObjectMeta{
		Name:      "grafana-datasource",
		Namespace: cr.Namespace,
		Labels:    map[string]string{"app": "grafana", "component": "grafana"},
	}
	hash := md5.New()
	_, err = io.WriteString(hash, string(bytesData))
	if err != nil {
		return nil, err
	}
	hashMark := fmt.Sprintf("%x", hash.Sum(nil))

	configMap.Annotations = map[string]string{
		"lastConfig": hashMark,
	}
	configMap.Data[grafanaDatasourceFile] = string(bytesData)

	return &configMap, nil
}

func ReconciledGrafanaDatasource(cr *v1alpha1.Grafana, current *corev1.ConfigMap) (*corev1.ConfigMap, error) {

	reconciled := current.DeepCopy()
	newConfig, err := GrafanaDatasourceConfig(cr)
	if err != nil {
		return nil, err
	}

	newHash := newConfig.Annotations["lastConfig"]
	newData := newConfig.Data[grafanaDatasourceFile]
	if reconciled.Annotations["lastConfig"] != newHash {
		reconciled.Annotations["lastConfig"] = newHash
		reconciled.Data[grafanaDatasourceFile] = newData
	}

	return reconciled, nil

}

func GrafanaDatasourceSelector(cr *v1alpha1.Grafana) client.ObjectKey {

	return client.ObjectKey{
		Name:      GrafanaDatasourceName,
		Namespace: cr.Namespace,
	}
}
