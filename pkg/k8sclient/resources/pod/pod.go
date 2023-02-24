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

package pod

import (
	"cert-csi/pkg/utils"
	"context"
	"errors"
	"fmt"
	"io"
	"net/url"
	"strconv"
	"time"

	"golang.org/x/sync/errgroup"
	policyv1beta1 "k8s.io/api/policy/v1beta1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/cli-runtime/pkg/resource"

	"github.com/fatih/color"
	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"
	apierrs "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	v1core "k8s.io/client-go/kubernetes/typed/core/v1"
	restclient "k8s.io/client-go/rest"
	"k8s.io/client-go/tools/remotecommand"
)

const (
	// Poll is a poll interval for Pod
	Poll = 2 * time.Second
	// Timeout is a timeout for Pod operations
	Timeout = 1800 * time.Second
	// Block VolumeMode
	Block = "Block"
	// EvictionKind represents the kind of evictions object
	EvictionKind = "Eviction"
	// EvictionSubresource represents the kind of evictions object as pod's subresource
	EvictionSubresource = "pods/eviction"
	policy              = "policy"
)

// Config contains volume configuration parameters
type Config struct {
	Name            string
	NamePrefix      string
	VolumeName      string
	VolumeMode      string
	MountPath       string
	ContainerName   string
	ContainerImage  string
	PvcNames        []string
	Command         []string
	Args            []string
	Capabilities    []v1.Capability
	Annotations     map[string]string
	EnvVars         []v1.EnvVar
	CSIVolumeSource v1.CSIVolumeSource
	ReadOnlyFlag    bool
}

// Client contains node client information
type Client struct {
	Interface v1core.PodInterface
	ClientSet kubernetes.Interface
	Config    *restclient.Config
	Namespace string
	Timeout   int
	nodeInfos []*resource.Info
}

// Pod contains pod related information
type Pod struct {
	Client  *Client
	Object  *v1.Pod
	Deleted bool

	// Used when error arises in syncable methods
	error error
}

// MakePod creates pod
func (c *Client) MakePod(config *Config) *v1.Pod {
	if len(config.NamePrefix) == 0 {
		config.NamePrefix = "pod-"
	}

	if len(config.MountPath) == 0 {
		config.MountPath = "/data"
	}

	if len(config.ContainerName) == 0 {
		config.ContainerName = "test-container"
	}

	if len(config.ContainerImage) == 0 {
		config.ContainerImage = "docker.io/centos:latest"
	}

	if len(config.Command) == 0 {
		config.Command = append(config.Command, `/bin/bash`)
	}

	var volumes []v1.Volume
	var volumeMounts []v1.VolumeMount
	var volumeDevices []v1.VolumeDevice

	for num, pvcName := range config.PvcNames {
		if config.VolumeMode == Block {
			volumeDevice := v1.VolumeDevice{
				DevicePath: "/dev" + config.MountPath + strconv.Itoa(num),
				Name:       config.VolumeName + strconv.Itoa(num),
			}
			volumeDevices = append(volumeDevices, volumeDevice)
		} else {
			volumeMount := v1.VolumeMount{
				Name:      config.VolumeName + strconv.Itoa(num),
				MountPath: config.MountPath + strconv.Itoa(num),
			}
			volumeMounts = append(volumeMounts, volumeMount)
		}

		volume := v1.Volume{
			Name: config.VolumeName + strconv.Itoa(num),
			VolumeSource: v1.VolumeSource{
				PersistentVolumeClaim: &v1.PersistentVolumeClaimVolumeSource{
					ClaimName: pvcName,
					ReadOnly:  config.ReadOnlyFlag,
				},
			},
		}
		volumes = append(volumes, volume)
	}

	container := v1.Container{
		Name:            config.ContainerName,
		Image:           config.ContainerImage,
		Command:         config.Command,
		Args:            config.Args,
		Env:             config.EnvVars,
		ImagePullPolicy: "IfNotPresent",
		SecurityContext: &v1.SecurityContext{
			Capabilities: &v1.Capabilities{Add: config.Capabilities},
		},
	}

	container.VolumeMounts = volumeMounts
	if config.VolumeMode == Block {
		container.VolumeDevices = volumeDevices
	}

	ObjMeta := metav1.ObjectMeta{
		GenerateName: config.NamePrefix,
		Namespace:    c.Namespace,
		Annotations:  config.Annotations,
	}
	if len(config.Name) != 0 {
		ObjMeta = metav1.ObjectMeta{
			Name:        config.Name,
			Namespace:   c.Namespace,
			Annotations: config.Annotations,
		}
	}

	return &v1.Pod{
		ObjectMeta: ObjMeta,
		Spec: v1.PodSpec{
			Volumes:    volumes,
			Containers: []v1.Container{container},
		},
	}
}

// MakePodFromYaml creates pod from yaml manifest
func (c *Client) MakePodFromYaml(filepath string) *v1.Pod {
	return &v1.Pod{}
}

// Create creates a new Pod
func (c *Client) Create(ctx context.Context, pod *v1.Pod) *Pod {
	log := utils.GetLoggerFromContext(ctx)
	var funcErr error
	newPod, err := c.Interface.Create(ctx, pod, metav1.CreateOptions{})

	if err != nil {
		funcErr = err
	} else {
		log.Debugf("Created Pod %s", newPod.GetName())
	}
	return &Pod{
		Client:  c,
		Object:  newPod,
		Deleted: false,
		error:   funcErr,
	}
}

// Delete deletes the specified pod
func (c *Client) Delete(ctx context.Context, pod *v1.Pod) *Pod {
	var funcErr error
	err := c.Interface.Delete(ctx, pod.GetName(), metav1.DeleteOptions{})
	if err != nil {
		funcErr = err
	}
	logrus.Debugf("Deleted Pod %s", pod.GetName())
	return &Pod{
		Client:  c,
		Object:  pod,
		Deleted: true,
		error:   funcErr,
	}
}

// Update updates a pod, to be implemented
func (c *Client) Update(pod *v1.Pod) {
	// TODO ?
}

// DeleteAll deletes all the pods
func (c *Client) DeleteAll(ctx context.Context) error {
	podList, podErr := c.Interface.List(ctx, metav1.ListOptions{})
	if podErr != nil {
		return podErr
	}

	return c.deleteAllFromList(ctx, podList)
}

// Exec runs the pod
func (c *Client) Exec(ctx context.Context, pod *v1.Pod, command []string, stdout, stderr io.Writer, quiet bool) error {
	log := utils.GetLoggerFromContext(ctx)
	executor := DefaultRemoteExecutor{}

	restClient := c.ClientSet.CoreV1().RESTClient()
	if !quiet {
		log.Infof("Executing command: %v", command)
	}
	log.Debugf("Executing command: %v", command)

	req := restClient.Post().
		Resource("pods").
		Name(pod.Name).
		Namespace(pod.Namespace).
		SubResource("exec")

	req.VersionedParams(&corev1.PodExecOptions{
		Container: pod.Spec.Containers[0].Name,
		Command:   command,
		Stdout:    true,
		Stderr:    true,
		TTY:       true,
	}, scheme.ParameterCodec)
	return executor.Execute("POST", req.URL(), c.Config, nil, stdout, stderr, false, nil)
}

// ReadyPodsCount returns the number of Pods in Ready state
func (c *Client) ReadyPodsCount(ctx context.Context) (int, error) {
	podList, err := c.Interface.List(ctx, metav1.ListOptions{})

	if err != nil {
		return 0, err
	}

	var readyCount int = 0
	for i, p := range podList.Items {
		isReady := IsPodReady(&podList.Items[i])
		if p.Status.Phase == v1.PodRunning || isReady {
			readyCount = readyCount + 1
		}
	}
	return readyCount, nil
}

// WaitForAllToBeReady waits for all Pods to be in Ready state
func (c *Client) WaitForAllToBeReady(ctx context.Context) error {
	log := utils.GetLoggerFromContext(ctx)
	log.Infof("Waiting for all pods in %s to be %s", c.Namespace, color.GreenString("READY"))
	startTime := time.Now()
	timeout := Timeout
	if c.Timeout != 0 {
		timeout = time.Duration(c.Timeout) * time.Second
	}

	pollErr := wait.PollImmediate(Poll, timeout,
		func() (bool, error) {
			select {
			case <-ctx.Done():
				log.Infof("Stopping pod wait polling")
				return true, fmt.Errorf("stopped waiting to be ready")
			default:
				break
			}

			podList, err := c.Interface.List(ctx, metav1.ListOptions{})

			if err != nil {
				return false, err
			}

			for i, p := range podList.Items {
				isReady := IsPodReady(&podList.Items[i])
				log.Debugf("Waiting for pod %s to be ready", p.Name)
				if p.Status.Phase != v1.PodRunning || !isReady {
					return false, nil
				}
			}

			return true, nil
		})

	if pollErr != nil {
		return pollErr
	}

	yellow := color.New(color.FgHiYellow)
	log.Infof("All pods are ready in %s", yellow.Sprint(time.Since(startTime)))
	return nil
}

// WaitForRunning stalls until pod is ready
func (pod *Pod) WaitForRunning(ctx context.Context) error {
	log := utils.GetLoggerFromContext(ctx)
	log.Infof("Waiting for pod %s to be READY", pod.Object.Name)
	timeout := Timeout
	if pod.Client.Timeout != 0 {
		timeout = time.Duration(pod.Client.Timeout) * time.Second
	}

	pollErr := wait.PollImmediate(Poll, timeout,
		func() (bool, error) {
			select {
			case <-ctx.Done():
				log.Infof("Stopping pod wait polling")
				return true, fmt.Errorf("stopped waiting to be ready")
			default:
				break
			}

			p, err := pod.Client.ClientSet.CoreV1().Pods(pod.Object.Namespace).Get(ctx, pod.Object.Name, metav1.GetOptions{})
			if err != nil {
				log.Errorf("Can't find pod %s", pod.Object.Name)
				return false, err
			}

			ready := IsPodReady(p)
			return ready, nil
		})

	if pollErr != nil {
		return pollErr
	}
	return nil
}

func (pod *Pod) pollWait(ctx context.Context) (bool, error) {
	log := utils.GetLoggerFromContext(ctx)
	select {
	case <-ctx.Done():
		log.Infof("Stopping pod wait polling")
		return true, fmt.Errorf("stopped waiting to be ready")
	default:
		break
	}
	if _, err := pod.Client.Interface.Get(ctx, pod.Object.Name, metav1.GetOptions{}); err != nil {
		if apierrs.IsNotFound(err) {
			return true, nil
		}
		log.Errorf("Error while waiting for pod to be terminated: %v", err)
		return false, err
	}
	return false, nil
}

// WaitUntilGone stalls until said Pod no longer can be found in Kubernetes
func (pod *Pod) WaitUntilGone(ctx context.Context) error {
	log := utils.GetLoggerFromContext(ctx)
	startTime := time.Now()
	timeout := Timeout
	if pod.Client.Timeout != 0 {
		timeout = time.Duration(pod.Client.Timeout) * time.Second
	}

	pollErr := wait.PollImmediate(Poll, timeout/100*95, func() (done bool, err error) {
		done, err = pod.pollWait(ctx)
		return done, err
	})

	if pollErr != nil {
		gotpod, err := pod.Client.Interface.Get(ctx, pod.Object.Name, metav1.GetOptions{})
		if err != nil {
			return err
		}
		log.Errorf("Failed to delete pod: %v \n", pollErr)
		log.Info("Forcing finalizers cleanup")
		gotpod.SetFinalizers([]string{})
		_, er := pod.Client.Interface.Update(ctx, gotpod, metav1.UpdateOptions{})
		if er != nil {
			return er
		}
		var graceper int64 = 0
		err = pod.Client.Interface.Delete(ctx, gotpod.GetName(), metav1.DeleteOptions{
			GracePeriodSeconds: &graceper,
		})
		if err != nil {
			return err
		}
		pollErr = wait.PollImmediate(Poll, timeout, func() (done bool, err error) {
			done, err = pod.pollWait(ctx)
			return done, err
		})

		if pollErr != nil {
			log.Errorf("Failed to delete pod: %v \n", pollErr)
			log.Info("Forcing finalizers cleanup")
			return errors.New("failed to delete even with finalizers cleaned up")
		}
	}
	yellow := color.New(color.FgHiYellow)
	log.Debugf("Pod %s was deleted in %s", pod.Object.Name, yellow.Sprint(time.Since(startTime)))
	return nil

}

// Sync waits until Pods in expected state
func (pod *Pod) Sync(ctx context.Context) *Pod {
	if pod.Deleted {
		pod.error = pod.WaitUntilGone(ctx)
	} else {
		pod.error = pod.WaitForRunning(ctx)
	}
	return pod
}

// HasError checks whether pod has any error
func (pod *Pod) HasError() bool {
	if pod.error != nil {
		return true
	}
	return false
}

// GetError returns pod error
func (pod *Pod) GetError() error {
	return pod.error
}

// IsPodReady checks if Pod is in Ready state
func IsPodReady(pod *v1.Pod) bool {
	return IsPodReadyConditionTrue(pod.Status)
}

// IsPodReadyConditionTrue returns true if Pod is in Ready state
func IsPodReadyConditionTrue(status v1.PodStatus) bool {
	condition := GetPodReadyCondition(status)
	return condition != nil && condition.Status == v1.ConditionTrue
}

// GetPodReadyCondition returns Pod status
func GetPodReadyCondition(status v1.PodStatus) *v1.PodCondition {
	_, condition := GetPodCondition(&status, v1.PodReady)
	return condition
}

// GetPodCondition returns pod condition
func GetPodCondition(status *v1.PodStatus, conditionType v1.PodConditionType) (int, *v1.PodCondition) {
	if status == nil {
		return -1, nil
	}
	return GetPodConditionFromList(status.Conditions, conditionType)
}

// GetPodConditionFromList returns pods and conditions
func GetPodConditionFromList(conditions []v1.PodCondition, conditionType v1.PodConditionType) (int, *v1.PodCondition) {
	if conditions == nil {
		return -1, nil
	}
	for i := range conditions {
		if conditions[i].Type == conditionType {
			return i, &conditions[i]
		}
	}
	return -1, nil
}

// DefaultRemoteExecutor represents default remote executor
type DefaultRemoteExecutor struct{}

// Execute executes remote command
func (*DefaultRemoteExecutor) Execute(method string, url *url.URL, config *restclient.Config, stdin io.Reader, stdout, stderr io.Writer, tty bool, terminalSizeQueue remotecommand.TerminalSizeQueue) error {
	exec, err := remotecommand.NewSPDYExecutor(config, method, url)
	if err != nil {
		return err
	}
	return exec.Stream(remotecommand.StreamOptions{
		Stdin:             stdin,
		Stdout:            stdout,
		Stderr:            stderr,
		Tty:               tty,
		TerminalSizeQueue: terminalSizeQueue,
	})
}

// MakeEphemeralPod creates an ephemeral pod
func (c *Client) MakeEphemeralPod(config *Config) *v1.Pod {
	if len(config.NamePrefix) == 0 {
		config.NamePrefix = "pod-"
	}

	if len(config.MountPath) == 0 {
		config.MountPath = "/data"
	}

	if len(config.ContainerName) == 0 {
		config.ContainerName = "test-container"
	}

	if len(config.ContainerImage) == 0 {
		config.ContainerImage = "docker.io/centos:latest"
	}

	if len(config.Command) == 0 {
		config.Command = append(config.Command, `/bin/bash`)
	}

	var volumes []v1.Volume
	var volumeMounts []v1.VolumeMount

	volumeMount := v1.VolumeMount{
		Name:      config.VolumeName,
		MountPath: config.MountPath,
	}
	volumeMounts = append(volumeMounts, volumeMount)
	volume := v1.Volume{
		Name: config.VolumeName,
		VolumeSource: v1.VolumeSource{
			CSI: &config.CSIVolumeSource,
		},
	}
	volumes = append(volumes, volume)

	container := v1.Container{
		Name:            config.ContainerName,
		Image:           config.ContainerImage,
		Command:         config.Command,
		Args:            config.Args,
		Env:             config.EnvVars,
		VolumeMounts:    volumeMounts,
		ImagePullPolicy: "IfNotPresent",
		SecurityContext: &v1.SecurityContext{
			Capabilities: &v1.Capabilities{Add: config.Capabilities},
		},
	}

	ObjMeta := metav1.ObjectMeta{
		GenerateName: config.NamePrefix,
		Namespace:    c.Namespace,
		Annotations:  config.Annotations,
	}
	if len(config.Name) != 0 {
		ObjMeta = metav1.ObjectMeta{
			Name:        config.Name,
			Namespace:   c.Namespace,
			Annotations: config.Annotations,
		}
	}

	return &v1.Pod{
		ObjectMeta: ObjMeta,
		Spec: v1.PodSpec{
			Containers: []v1.Container{container},
			Volumes:    volumes,
		},
	}
}

// DeleteOrEvictPods deletes or evicts pod from a node
func (c *Client) DeleteOrEvictPods(ctx context.Context, nodeName string, gracePeriodSeconds int) error {
	podList, podErr := c.Interface.List(ctx, metav1.ListOptions{
		FieldSelector: fields.SelectorFromSet(fields.Set{"spec.nodeName": nodeName}).String()})

	if podErr != nil {
		return podErr
	}

	if len(podList.Items) == 0 {
		return nil
	}

	policyGroupVersion, err := checkEvictionSupport(c.ClientSet)
	if err != nil {
		return err
	}
	if len(policyGroupVersion) > 0 {
		return c.evictPods(ctx, podList, policyGroupVersion, gracePeriodSeconds)
	}
	return nil
}

func (c *Client) deleteAllFromList(ctx context.Context, podList *v1.PodList) error {
	log := utils.GetLoggerFromContext(ctx)
	log.Debugf("Deleting all pods")
	for i, pod := range podList.Items {
		err := c.Delete(ctx, &podList.Items[i]).Sync(ctx).GetError()
		if err != nil {
			log.Errorf("Can't delete pod %s; error=%v", pod.Name, err)
		}
	}
	return nil
}

func checkEvictionSupport(clientSet kubernetes.Interface) (string, error) {

	discoveryClient := clientSet.Discovery()
	groupList, err := discoveryClient.ServerGroups()
	if err != nil {
		return "", err
	}
	foundPolicyGroup := false
	var policyGroupVersion string
	for _, group := range groupList.Groups {
		if group.Name == policy {
			foundPolicyGroup = true
			policyGroupVersion = group.PreferredVersion.GroupVersion
			break
		}
	}
	if !foundPolicyGroup {
		return "", nil
	}
	resourceList, err := discoveryClient.ServerResourcesForGroupVersion("v1")
	if err != nil {
		return "", err
	}
	for _, resource := range resourceList.APIResources {
		if resource.Name == EvictionSubresource && resource.Kind == EvictionKind {
			return policyGroupVersion, nil
		}
	}
	return "", nil
}

func (c *Client) evictPods(ctx context.Context, podList *v1.PodList, policyGroupVersion string, gracePeriodSeconds int) error {
	log := utils.GetLoggerFromContext(ctx)
	g, errctx := errgroup.WithContext(ctx)

	for _, pod := range podList.Items {
		pod := pod
		g.Go(func() error {
			err := c.EvictPod(errctx, pod, policyGroupVersion, gracePeriodSeconds)
			if err != nil {
				log.Errorf("error when evicting pods/%q -n %q", pod.Name, pod.Namespace)
				return err
			}
			return nil
		})
	}
	return g.Wait()
}

// EvictPod evicts pod from a node
func (c *Client) EvictPod(ctx context.Context, pod corev1.Pod, policyGroupVersion string, gracePeriodSeconds int) error {
	log := utils.GetLoggerFromContext(ctx)
	log.Debugf("evicting the pod ")
	delOpts := metav1.DeleteOptions{}
	if gracePeriodSeconds >= 0 {
		gracePeriodSeconds := int64(gracePeriodSeconds)
		delOpts.GracePeriodSeconds = &gracePeriodSeconds
	}
	eviction := &policyv1beta1.Eviction{
		TypeMeta: metav1.TypeMeta{
			APIVersion: policyGroupVersion,
			Kind:       EvictionKind,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      pod.Name,
			Namespace: pod.Namespace,
		},
		DeleteOptions: &delOpts,
	}

	return c.ClientSet.PolicyV1beta1().Evictions(eviction.Namespace).Evict(ctx, eviction)
}

// IsInPendingState checks whether a pod is in Pending state
func (pod *Pod) IsInPendingState(ctx context.Context) error {
	updatedPod, err := pod.Client.Interface.Get(ctx, pod.Object.Name, metav1.GetOptions{})
	if err != nil {
		return err
	}
	pod.Object = updatedPod
	if updatedPod.Status.Phase != v1.PodPending {
		return fmt.Errorf("%s pod is in %s state", updatedPod.Name, updatedPod.Status.Phase)
	}
	return nil
}
