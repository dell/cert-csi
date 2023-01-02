/*
 *
 * Copyright © 2022-2023 Dell Inc. or its subsidiaries. All Rights Reserved.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *      http://www.apache.org/licenses/LICENSE-2.0
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 */

package statefulset

import (
	"cert-csi/pkg/k8sclient/resources/pod"
	"cert-csi/pkg/utils"
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/fatih/color"
	apierrs "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/client-go/kubernetes"

	"github.com/sirupsen/logrus"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	tappsv1 "k8s.io/client-go/kubernetes/typed/apps/v1"
	"sigs.k8s.io/yaml"
)

const (
	// Poll is a poll interval for StatefulSet tests
	Poll = 2 * time.Second
	// Timeout is a timeout interval for StatefulSet operations
	Timeout = 1800 * time.Second
	// K8sMissScaleError command's been lost by the k8s, need to retry
	K8sMissScaleError = "the object has been modified; please apply your changes to the latest version and try again"
)

// Config can be used as config for creation of new StatefulSets
type Config struct {
	VolumeNumber        int
	Replicas            int32
	VolumeName          string
	MountPath           string
	StorageClassName    string
	PodManagementPolicy string
	ClaimSize           string
	ContainerName       string
	ContainerImage      string
	NamePrefix          string
	Command             []string
	Args                []string
	Annotations         map[string]string
	Labels              map[string]string
}

// Client contains go-client interface for making namespaced StatefulSet Kubernetes API calls
type Client struct {
	// KubeClient *core.KubeClient
	Interface tappsv1.StatefulSetInterface
	ClientSet kubernetes.Interface
	Namespace string
	Timeout   int
}

// StatefulSet bounds go-client StatefulSet object and client
type StatefulSet struct {
	Client  *Client
	Set     *appsv1.StatefulSet
	Deleted bool

	// Used when error arises in syncable methods
	error error
}

// MakeStatefulSet returns new go-client StatefulSet object from provided config
func (c *Client) MakeStatefulSet(config *Config) *appsv1.StatefulSet {
	if config.Replicas <= 0 {
		config.Replicas = 1
	}

	if len(config.VolumeName) == 0 {
		config.VolumeName = "vol"
	}

	if config.VolumeNumber <= 0 {
		config.VolumeNumber = 1
	}

	if len(config.ClaimSize) == 0 {
		config.ClaimSize = "3Gi"
	}

	if len(config.PodManagementPolicy) == 0 {
		config.PodManagementPolicy = "Parallel"
	}

	if len(config.NamePrefix) == 0 {
		config.NamePrefix = "sts-"
	}

	if len(config.MountPath) == 0 {
		config.MountPath = "/data"
	}

	if len(config.ContainerName) == 0 {
		config.ContainerName = "test-container"
	}

	if len(config.ContainerImage) == 0 {
		config.ContainerImage = "amaas-eos-mw1.cec.lab.emc.com:5028/centos:latest"
	}

	if len(config.Command) == 0 {
		config.Command = append(config.Command, `/bin/bash`)
	}

	var volumeMounts []v1.VolumeMount
	var volumeClaimTemplates []v1.PersistentVolumeClaim

	for i := 0; i < config.VolumeNumber; i++ {
		volumeName := config.VolumeName + strconv.Itoa(i)
		volumeMount := v1.VolumeMount{
			Name:      volumeName,
			MountPath: config.MountPath + strconv.Itoa(i),
		}
		volumeMounts = append(volumeMounts, volumeMount)

		volumeClaim := v1.PersistentVolumeClaim{
			ObjectMeta: metav1.ObjectMeta{
				Name: volumeName,
			},
			Spec: v1.PersistentVolumeClaimSpec{
				AccessModes: []v1.PersistentVolumeAccessMode{"ReadWriteOnce"},
				Resources: v1.ResourceRequirements{
					Requests: v1.ResourceList{
						v1.ResourceStorage: resource.MustParse(config.ClaimSize),
					},
				},
				StorageClassName: &config.StorageClassName,
			},
		}

		volumeClaimTemplates = append(volumeClaimTemplates, volumeClaim)
	}

	container := v1.Container{
		Name:            config.ContainerName,
		Image:           config.ContainerImage,
		Command:         config.Command,
		Args:            config.Args,
		VolumeMounts:    volumeMounts,
		ImagePullPolicy: "IfNotPresent",
	}

	return &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: config.NamePrefix,
			Namespace:    c.Namespace,
			Annotations:  config.Annotations,
		},
		Spec: appsv1.StatefulSetSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: config.Labels,
			},
			Replicas: &config.Replicas,
			Template: v1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: config.Labels,
				},
				Spec: v1.PodSpec{
					Containers: []v1.Container{container},
				},
			},
			VolumeClaimTemplates: volumeClaimTemplates,
			PodManagementPolicy:  appsv1.PodManagementPolicyType(config.PodManagementPolicy),
		},
	}
}

// MakeStatefulSetFromYaml creates and returns new go-client StatefulSet object by parsing provided .yaml file
func (c *Client) MakeStatefulSetFromYaml(filePath string) *appsv1.StatefulSet {
	sts := &appsv1.StatefulSet{}

	file, _ := ioutil.ReadFile(filepath.Clean(filePath))
	fmt.Printf("%s \n", string(file))

	errUnmarshal := yaml.Unmarshal(file, sts)
	if errUnmarshal != nil {
		logrus.Errorf("Can't unmarshal yaml file; error=%v", errUnmarshal)
		return nil
	}

	return sts
}

// Create uses client interface to make API call for creating provided StatefulSet
func (c *Client) Create(ctx context.Context, sts *appsv1.StatefulSet) *StatefulSet {
	log := utils.GetLoggerFromContext(ctx)
	var funcErr error
	newSTS, err := c.Interface.Create(ctx, sts, metav1.CreateOptions{})
	if err != nil {
		funcErr = err
	}
	log.Infof("Created stateful set %s", newSTS.GetName())
	return &StatefulSet{
		Client:  c,
		Set:     newSTS,
		Deleted: false,
		error:   funcErr,
	}
}

// Delete method deletes StatefulSet from namespace and returns StatefulSet object with Deleted field set to true
func (c *Client) Delete(ctx context.Context, sts *appsv1.StatefulSet) *StatefulSet {
	return c.DeleteWithOptions(ctx, sts, metav1.DeleteOptions{})
}

// DeleteWithOptions method deletes StatefulSet from namespace and returns StatefulSet object with Deleted field set to true
func (c *Client) DeleteWithOptions(ctx context.Context, sts *appsv1.StatefulSet, opts metav1.DeleteOptions) *StatefulSet {
	log := utils.GetLoggerFromContext(ctx)
	var funcErr error
	if sts.Status.String() == "Terminating" {
		return &StatefulSet{
			Client:  c,
			Set:     sts,
			Deleted: true,
			error:   funcErr,
		}
	}
	err := c.Interface.Delete(ctx, sts.GetName(), opts)
	if err != nil {
		logrus.Debugf(err.Error())
		funcErr = err
	}
	log.Debugf("Deleted stateful set %s", sts.GetName())

	return &StatefulSet{
		Client:  c,
		Set:     sts,
		Deleted: true,
		error:   funcErr,
	}
}

// DeleteAll deletes all statefulsets
func (c *Client) DeleteAll(ctx context.Context) error {
	log := utils.GetLoggerFromContext(ctx)
	stsList, stsErr := c.Interface.List(ctx, metav1.ListOptions{})
	if stsErr != nil {
		return stsErr
	}
	log.Debugf("Deleting all STS")
	for i, stset := range stsList.Items {
		if strings.Contains(stset.Name, "-test-") {
			log.Debugf("Deleting STS %s", stset.Name)
			err := c.Delete(ctx, &stsList.Items[i]).Sync(ctx).GetError()
			if err != nil {
				log.Errorf("Can't delete STS %s; error=%v", stset.Name, err)
			}
		}
	}

	return nil
}

// Update makes API call to update existing resource with new Spec provided in `sts`
func (c *Client) Update(sts *appsv1.StatefulSet) *StatefulSet {
	return &StatefulSet{}
}

// Scale makes API call to update existing resource replicas to new value of `count`
func (c *Client) Scale(ctx context.Context, sts *appsv1.StatefulSet, count int32) *StatefulSet {
	log := utils.GetLoggerFromContext(ctx)
	log.Infof("Scaling to %d replicas", count)
	scale, scaleErr := c.Interface.GetScale(ctx, sts.GetName(), metav1.GetOptions{})
	if scaleErr != nil {
		return &StatefulSet{
			Client:  c,
			Set:     sts,
			Deleted: false,
			error:   scaleErr,
		}
	}
	scale.Spec.Replicas = count
	sts.Spec.Replicas = &count
	_, updateErr := c.Interface.UpdateScale(ctx, sts.GetName(), scale, metav1.UpdateOptions{})
	if updateErr != nil {
		if !strings.Contains(updateErr.Error(), K8sMissScaleError) {
			return &StatefulSet{
				Client:  c,
				Set:     sts,
				Deleted: false,
				error:   updateErr,
			}
		}
	}

	for {
		select {
		case <-ctx.Done():
			log.Infof("Stopping pod wait polling")
			return &StatefulSet{
				Client:  c,
				Set:     sts,
				Deleted: false,
				error:   errors.New("stopping waiting for sts to scale"),
			}
		default:
			break
		}
		actualState, actualStateError := c.Interface.GetScale(ctx, sts.GetName(), metav1.GetOptions{})
		if actualStateError != nil || actualState == nil {
			log.Warnf("Failed to get sts %s", sts.GetName())
			continue
		}
		if actualState.Spec.Replicas == count {
			return &StatefulSet{
				Client:  c,
				Set:     sts,
				Deleted: false,
				error:   nil,
			}
		}

		log.Warnf("Scale command's been lost by the k8s, retrying.. desired=%d, actual=%d", count, actualState.Spec.Replicas)
		actualState.Spec.Replicas = count
		sts.Spec.Replicas = &count
		_, updateErr := c.Interface.UpdateScale(ctx, sts.GetName(), actualState, metav1.UpdateOptions{})
		if updateErr != nil {
			if !strings.Contains(updateErr.Error(), K8sMissScaleError) {
				return &StatefulSet{
					Client:  c,
					Set:     sts,
					Deleted: false,
					error:   updateErr,
				}
			}
		} else {
			continue
		}
	}
}

// GetPodList returns all pods that belong to StatefulSet
func (sts *StatefulSet) GetPodList(ctx context.Context) (*v1.PodList, error) {
	selector, selectorErr := metav1.LabelSelectorAsSelector(sts.Set.Spec.Selector)
	if selectorErr != nil {
		return nil, selectorErr
	}
	// Maybe this can be done slightly better and we can use pod client from here
	podList, err := sts.Client.ClientSet.CoreV1().Pods(sts.Client.Namespace).List(ctx, metav1.ListOptions{LabelSelector: selector.String()})
	if err != nil {
		return nil, err
	}
	return podList, nil
}

// WaitForRunningAndReady stalls provided number of pods is running and ready
func (sts *StatefulSet) WaitForRunningAndReady(ctx context.Context, numPodsRunning, numPodsReady int32) error {
	log := utils.GetLoggerFromContext(ctx)
	log.Infof("Waiting for sts pods to become ready")
	timeout := Timeout
	if sts.Client.Timeout != 0 {
		timeout = time.Duration(sts.Client.Timeout) * time.Second
	}
	pollErr := wait.PollImmediate(Poll, timeout,
		func() (bool, error) {
			select {
			case <-ctx.Done():
				log.Infof("Stopping sts wait polling")
				return true, fmt.Errorf("stopped waiting to be ready")
			default:
				break
			}

			podList, err := sts.GetPodList(ctx)
			if err != nil {
				log.Errorf("Can't get pod list; error=%v", err)
				return false, err
			}

			if int32(len(podList.Items)) < numPodsRunning {
				log.Debugf("Found %d stateful pods, waiting for %d", len(podList.Items), numPodsRunning)
				return false, nil
			}
			if int32(len(podList.Items)) > numPodsRunning {
				log.Debugf("Too many pods scheduled, expected %d got %d", numPodsRunning, len(podList.Items))
				return false, nil
			}

			if int32(len(podList.Items)) == numPodsRunning && numPodsRunning == 0 {
				return true, nil
			}

			for i, p := range podList.Items {
				isReady := pod.IsPodReady(&podList.Items[i])
				log.Debugf("Waiting for pod %v to be ready, currently = %v", p.Name, isReady)
				if p.Status.Phase != v1.PodRunning || !isReady {
					return false, nil
				}
			}
			return true, nil
		})

	if pollErr != nil {
		return pollErr
	}
	return nil
}

func (sts *StatefulSet) pollWait(ctx context.Context) (bool, error) {
	log := utils.GetLoggerFromContext(ctx)
	select {
	case <-ctx.Done():
		log.Infof("Stopping sts wait polling")
		return true, fmt.Errorf("stopped waiting to be ready")
	default:
		break
	}

	if _, err := sts.Client.Interface.Get(ctx, sts.Set.Name, metav1.GetOptions{}); err != nil {
		if apierrs.IsNotFound(err) {
			return true, nil
		}
		log.Errorf("Error while waiting for STS to be terminated: %v", err)
		return false, err
	}
	return false, nil
}

// WaitUntilGone stalls until said StatefulSet no longer can be found in Kubernetes
func (sts *StatefulSet) WaitUntilGone(ctx context.Context) error {
	log := utils.GetLoggerFromContext(ctx)
	startTime := time.Now()
	timeout := Timeout
	if sts.Client.Timeout != 0 {
		timeout = time.Duration(sts.Client.Timeout) * time.Second
	}

	pollErr := wait.PollImmediate(Poll, timeout/100*95, func() (done bool, err error) {
		done, err = sts.pollWait(ctx)
		return done, err
	})
	if pollErr != nil {
		gotsts, err := sts.Client.Interface.Get(ctx, sts.Set.Name, metav1.GetOptions{})
		if err != nil {
			return err
		}
		log.Errorf("Failed to delete STS: %v \n", pollErr)
		log.Info("Forcing finalizers cleanup")
		gotsts.SetFinalizers([]string{})
		gotsts.Finalizers = []string{}
		_, er := sts.Client.Interface.Update(ctx, gotsts, metav1.UpdateOptions{})
		if er != nil {
			return er
		}
		pollErr = wait.PollImmediate(Poll, timeout/2, func() (done bool, err error) {
			done, err = sts.pollWait(ctx)
			return done, err
		})

		if pollErr != nil {
			log.Errorf("Failed to delete STS: %v \n", pollErr)
			log.Info("Forcing finalizers cleanup")
			return errors.New("failed to delete even with finalizers cleaned up")
		}
	}
	yellow := color.New(color.FgHiYellow)
	log.Debugf("StatefulSet %s was deleted in %s", sts.Set.Name, yellow.Sprint(time.Since(startTime)))
	return nil
}

// Sync makes sure that previous StatefulSet operation is completed
func (sts *StatefulSet) Sync(ctx context.Context) *StatefulSet {
	if sts.Deleted {
		sts.error = sts.WaitUntilGone(ctx)
	} else {
		sts.error = sts.WaitForRunningAndReady(ctx, *sts.Set.Spec.Replicas, *sts.Set.Spec.Replicas)
	}
	return sts
}

// HasError checks if statefulset contains error
func (sts *StatefulSet) HasError() bool {
	if sts.error != nil {
		return true
	}
	return false
}

// GetError returns statefulset error
func (sts *StatefulSet) GetError() error {
	return sts.error
}
