// Copyright 2019 Chaos Mesh Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// See the License for the specific language governing permissions and
// limitations under the License.

package common

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"strconv"
	"strings"

	"google.golang.org/grpc/grpclog"
	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/remotecommand"
	kubectlscheme "k8s.io/kubectl/pkg/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
)

// Exec executes certain command and returns the result
// runtime-controller only support CRUD， use client-go client
func Exec(ctx context.Context, pod v1.Pod, daemon v1.Pod, cmd string, c *kubernetes.Clientset) (string, error) {
	out, err := exec(ctx, pod, daemon, cmd, c)

	if err != nil {
		// use daemon to enter namespace and execute command if command not found (which stream would failed)
		if strings.Contains(err.Error(), "streaming remotecommand") {
			outNs, errNs := nsEnterExec(ctx, err.Error(), pod, daemon, cmd, c)
			if errNs == nil {
				return outNs, nil
			}
			err = fmt.Errorf("%s\nnsenter also failed with: %s", err.Error(), errNs.Error())
		}
		return "", err
	}

	return out, nil
}

func exec(ctx context.Context, pod v1.Pod, daemon v1.Pod, cmd string, c *kubernetes.Clientset) (string, error) {
	name := pod.GetObjectMeta().GetName()
	namespace := pod.GetObjectMeta().GetNamespace()
	// TODO: if `containerNames` is set and specific container is injected chaos,
	// need to use THE name rather than the first one.
	// till 20/11/10 only podchaos and kernelchaos support `containerNames`, so not set it for now
	containerName := pod.Spec.Containers[0].Name

	req := c.CoreV1().RESTClient().Post().
		Resource("pods").
		Name(name).
		Namespace(namespace).
		SubResource("exec")

	req.VersionedParams(&v1.PodExecOptions{
		Container: containerName,
		Command:   []string{"/bin/sh", "-c", cmd},
		Stdin:     false,
		Stdout:    true,
		Stderr:    true,
		TTY:       false,
	}, kubectlscheme.ParameterCodec)

	var stdout, stderr bytes.Buffer
	exec, err := remotecommand.NewSPDYExecutor(config.GetConfigOrDie(), "POST", req.URL())
	if err != nil {
		return "", fmt.Errorf("error in creating NewSPDYExecutor: %s", err.Error())
	}
	err = exec.Stream(remotecommand.StreamOptions{
		Stdin:  nil,
		Stdout: &stdout,
		Stderr: &stderr,
	})
	if err != nil {
		return "", fmt.Errorf("error in streaming remotecommand: %s", err.Error())
	}
	if stderr.String() != "" {
		return "", fmt.Errorf(stderr.String())
	}
	return stdout.String(), nil
}

func nsEnterExec(ctx context.Context, stderr string, pod v1.Pod, daemon v1.Pod, cmd string, c *kubernetes.Clientset) (string, error) {
	cmdSubSlice := strings.Fields(cmd)
	if len(cmdSubSlice) == 0 {
		return "", fmt.Errorf("command should not be empty")
	}
	// To disable printing irrelevant log from grpc/clientconn.go
	// See grpc/grpc-go#3918 for detail. Could be resolved in the future
	grpclog.SetLoggerV2(grpclog.NewLoggerV2(ioutil.Discard, ioutil.Discard, ioutil.Discard))
	pid, err := GetPidFromPod(ctx, pod, daemon)
	if err != nil {
		return "", err
	}
	switch cmdSubSlice[0] {
	case "ps":
		nsenterPath := "-p/proc/" + strconv.Itoa(pid) + "/ns/pid"
		nsCmd := fmt.Sprintf("mount -t proc proc /proc && %s && umount proc", cmd)
		newCmd := fmt.Sprintf("/usr/bin/nsenter %s -- /bin/bash -c '%s'", nsenterPath, nsCmd)
		return exec(ctx, daemon, daemon, newCmd, c)
	case "cat", "ls":
		// we need to enter mount namespace to get file related infomation
		// but enter mnt ns would prevent us to access `cat`/`ls` in daemon
		// so use `nsexec` to achieve using nsenter and cat together
		if len(cmdSubSlice) < 2 {
			return "", fmt.Errorf("%s should have one argument at least", cmdSubSlice[0])
		}
		newCmd := fmt.Sprintf("/usr/local/bin/nsexec %s %s", strconv.Itoa(pid), cmd)
		return exec(ctx, daemon, daemon, newCmd, c)
	default:
		return "", fmt.Errorf("command not supported for nsenter")
	}
}
