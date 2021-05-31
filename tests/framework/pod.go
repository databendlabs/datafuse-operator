// Copyright 2020-2021 The Datafuse Authors.
//
// SPDX-License-Identifier: Apache-2.0.
package framework

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/url"
	"strings"

	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/client-go/kubernetes"
	kscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/remotecommand"
)

func GetPod(kubeCilent kubernetes.Interface, ns, name string) (*corev1.Pod, error) {
	return kubeCilent.CoreV1().Pods(ns).Get(context.TODO(), name, metav1.GetOptions{})
}

func UpdatePod(kubeCilent kubernetes.Interface, pod *corev1.Pod) (*corev1.Pod, error) {
	return kubeCilent.CoreV1().Pods(pod.Namespace).Update(context.TODO(), pod, metav1.UpdateOptions{})
}

func MakePod(pathToYaml string) (*corev1.Pod, error) {
	manifest, err := PathToOSFile(pathToYaml)
	if err != nil {
		return nil, err
	}
	deployment := corev1.Pod{}
	if err := yaml.NewYAMLOrJSONDecoder(manifest, 100).Decode(&deployment); err != nil {
		return nil, errors.Wrap(err, fmt.Sprintf("failed to decode file %s", pathToYaml))
	}

	return &deployment, nil
}

func CreatePod(kubeClient kubernetes.Interface, namespace string, p *corev1.Pod) error {
	p.Namespace = namespace
	_, err := kubeClient.CoreV1().Pods(namespace).Create(context.TODO(), p, metav1.CreateOptions{})
	if err != nil {
		return errors.Wrap(err, fmt.Sprintf("failed to create deployment %s", p.Name))
	}
	return nil
}

func DeletePod(kubeClient kubernetes.Interface, namespace, name string) error {
	p, err := kubeClient.CoreV1().Pods(namespace).Get(context.TODO(), name, metav1.GetOptions{})
	if err != nil {
		return err
	}

	return kubeClient.CoreV1().Pods(namespace).Delete(context.TODO(), p.Name, metav1.DeleteOptions{})
}

// PrintPodLogs prints the logs of a specified Pod
func (f *Framework) PrintPodLogs(ns, p string) error {
	pod, err := f.KubeClient.CoreV1().Pods(ns).Get(context.TODO(), p, metav1.GetOptions{})
	if err != nil {
		return errors.Wrapf(err, "failed to print logs of pod '%v': failed to get pod", p)
	}

	for _, c := range pod.Spec.Containers {
		req := f.KubeClient.CoreV1().Pods(ns).GetLogs(p, &corev1.PodLogOptions{Container: c.Name})
		resp, err := req.DoRaw(context.TODO())
		if err != nil {
			return errors.Wrapf(err, "failed to retrieve logs of pod '%v'", p)
		}

		fmt.Printf("=== Logs of %v/%v/%v:", ns, p, c.Name)
		fmt.Println(string(resp))
	}

	return nil
}

// ExecOptions passed to ExecWithOptions
type ExecOptions struct {
	Command       []string
	Namespace     string
	PodName       string
	ContainerName string
	Stdin         io.Reader
	CaptureStdout bool
	CaptureStderr bool
	// If false, whitespace in std{err,out} will be removed.
	PreserveWhitespace bool
}

func (f *Framework) MakeExecOptions(containerName, podName, namespace string, commands []string) ExecOptions {
	return ExecOptions{
		Command:            commands,
		Namespace:          namespace,
		PodName:            podName,
		ContainerName:      containerName,
		CaptureStdout:      true,
		CaptureStderr:      true,
		PreserveWhitespace: true,
	}
}

// ExecWithOptions executes a command in the specified container, returning
// stdout, stderr and error. `options` allowed for additional parameters to be
// passed. Inspired by
// https://github.com/kubernetes/kubernetes/blob/dde6e8e7465468c32642659cb708a5cc922add64/test/e2e/framework/exec_util.go#L36-L51
func (f *Framework) ExecWithOptions(options ExecOptions) (string, string, error) {
	const tty = false

	req := f.KubeClient.CoreV1().RESTClient().Post().
		Resource("pods").
		Name(options.PodName).
		Namespace(options.Namespace).
		SubResource("exec").
		Param("container", options.ContainerName)
	req.VersionedParams(&v1.PodExecOptions{
		Container: options.ContainerName,
		Command:   options.Command,
		Stdin:     options.Stdin != nil,
		Stdout:    options.CaptureStdout,
		Stderr:    options.CaptureStderr,
		TTY:       tty,
	}, kscheme.ParameterCodec)

	var stdout, stderr bytes.Buffer
	err := execute("POST", req.URL(), f.RestConfig, options.Stdin, &stdout, &stderr, tty)

	if options.PreserveWhitespace {
		return stdout.String(), stderr.String(), err
	}
	return strings.TrimSpace(stdout.String()), strings.TrimSpace(stderr.String()), err
}

func execute(method string, url *url.URL, config *rest.Config, stdin io.Reader, stdout, stderr io.Writer, tty bool) error {
	exec, err := remotecommand.NewSPDYExecutor(config, method, url)
	if err != nil {
		return err
	}
	return exec.Stream(remotecommand.StreamOptions{
		Stdin:  stdin,
		Stdout: stdout,
		Stderr: stderr,
		Tty:    tty,
	})
}
