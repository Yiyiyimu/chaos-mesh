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

package networkchaos

import (
	"context"
	"fmt"
	"io/ioutil"
	"regexp"
	"strconv"
	"strings"

	"google.golang.org/grpc/grpclog"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/chaos-mesh/chaos-mesh/api/v1alpha1"
	cm "github.com/chaos-mesh/chaos-mesh/pkg/debug/common"
)

// Debug get chaos debug information
func Debug(ctx context.Context, chaos runtime.Object, c *cm.ClientSet, result *cm.ChaosResult) error {
	networkChaos, ok := chaos.(*v1alpha1.NetworkChaos)
	if !ok {
		return fmt.Errorf("chaos is not network")
	}
	chaosStatus := networkChaos.Status.ChaosStatus
	chaosSelector := networkChaos.Spec.GetSelector()

	pods, daemons, err := cm.GetPods(ctx, chaosStatus, chaosSelector, c.CtrlCli)
	if err != nil {
		return err
	}

	for i := range pods {
		podName := pods[i].GetObjectMeta().GetName()
		podResult := cm.PodResult{Name: podName}
		err := debugEachPod(ctx, pods[i], daemons[i], networkChaos, c, &podResult)
		result.Pods = append(result.Pods, podResult)
		if err != nil {
			return fmt.Errorf("for %s: %s", podName, err.Error())
		}
	}
	return nil
}

func debugEachPod(ctx context.Context, pod v1.Pod, daemon v1.Pod, chaos *v1alpha1.NetworkChaos, c *cm.ClientSet, result *cm.PodResult) error {
	podName := pod.GetObjectMeta().GetName()
	podNamespace := pod.GetObjectMeta().GetNamespace()

	// To disable printing irrelevant log from grpc/clientconn.go
	// see grpc/grpc-go#3918 for detail. could be resolved in the future
	grpclog.SetLoggerV2(grpclog.NewLoggerV2(ioutil.Discard, ioutil.Discard, ioutil.Discard))
	pid, err := cm.GetPidFromPod(ctx, pod, daemon)
	if err != nil {
		return err
	}
	nsenterPath := "-n/proc/" + strconv.Itoa(pid) + "/ns/net"

	// print out debug info
	cmd := fmt.Sprintf("/usr/bin/nsenter %s -- ipset list", nsenterPath)
	out, err := cm.Exec(ctx, daemon, daemon, cmd, c.KubeCli)
	if err != nil {
		return fmt.Errorf("run command '%s' failed with: %s", cmd, err.Error())
	}
	result.Items = append(result.Items, cm.ItemResult{Name: "ipset list", Value: string(out)})

	cmd = fmt.Sprintf("/usr/bin/nsenter %s -- tc qdisc list", nsenterPath)
	out, err = cm.Exec(ctx, daemon, daemon, cmd, c.KubeCli)
	if err != nil {
		return fmt.Errorf("run command '%s' failed with: %s", cmd, err.Error())
	}
	itemResult := cm.ItemResult{Name: "tc qdisc list", Value: string(out)}

	action := chaos.Spec.Action
	var netemExpect string
	switch action {
	case "delay":
		latency := chaos.Spec.Delay.Latency
		jitter := chaos.Spec.Delay.Jitter
		correlation := chaos.Spec.Delay.Correlation
		netemExpect = fmt.Sprintf("%v %v %v %v%%", action, latency, jitter, correlation)
	default:
		return fmt.Errorf("chaos not supported")
	}

	netemCurrent := regexp.MustCompile("(?:limit 1000)(.*)").FindStringSubmatch(string(out))
	if len(netemCurrent) == 0 {
		return fmt.Errorf("No NetworkChaos is applied")
	}
	for i, netem := range strings.Fields(netemCurrent[1]) {
		itemCurrent := netem
		itemExpect := strings.Fields(netemExpect)[i]
		if itemCurrent != itemExpect {
			r := regexp.MustCompile("([0-9]*[.])?[0-9]+")
			numCurrent, err := strconv.ParseFloat(r.FindString(itemCurrent), 64)
			if err != nil {
				return fmt.Errorf("parse itemCurrent failed: %s", err.Error())
			}
			numExpect, err := strconv.ParseFloat(r.FindString(itemExpect), 64)
			if err != nil {
				return fmt.Errorf("parse itemExpect failed: %s", err.Error())
			}
			if numCurrent == numExpect {
				continue
			}
			alpCurrent := regexp.MustCompile("[[:alpha:]]+").FindString(itemCurrent)
			alpExpect := regexp.MustCompile("[[:alpha:]]+").FindString(itemExpect)
			if alpCurrent == alpExpect {
				continue
			}
			itemResult.Status = cm.ItemFailure
			itemResult.ErrInfo = fmt.Sprintf("expect: %s, got: %v", netemExpect, netemCurrent)
			result.Items = append(result.Items, itemResult)
			return nil
		}
	}
	itemResult.Status = cm.ItemSuccess
	result.Items = append(result.Items, itemResult)

	cmd = fmt.Sprintf("/usr/bin/nsenter %s -- iptables --list", nsenterPath)
	out, err = cm.Exec(ctx, daemon, daemon, cmd, c.KubeCli)
	if err != nil {
		return fmt.Errorf("cmd.Run() failed with: %s", err.Error())
	}
	result.Items = append(result.Items, cm.ItemResult{Name: "iptables list", Value: string(out)})

	podNetworkChaos := &v1alpha1.PodNetworkChaos{}
	objectKey := client.ObjectKey{
		Namespace: podNamespace,
		Name:      podName,
	}

	if err = c.CtrlCli.Get(ctx, objectKey, podNetworkChaos); err != nil {
		return fmt.Errorf("failed to get chaos: %s", err.Error())
	}
	mar, err := cm.MarshalChaos(podNetworkChaos.Spec)
	if err != nil {
		return err
	}
	result.Items = append(result.Items, cm.ItemResult{Name: "podnetworkchaos", Value: mar})

	return nil
}