package main

import (
	"flag"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strings"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	metrics "k8s.io/metrics/pkg/client/clientset/versioned"
)

func main() {
	var kubeconfig *string
	if home := homeDir(); home != "" {
		kubeconfig = flag.String("kubeconfig", filepath.Join(home, ".kube", "config"), "(optional) absolute path to the kubeconfig file")
	} else {
		kubeconfig = flag.String("kubeconfig", "", "absolute path to the kubeconfig file")
	}
	flag.Parse()

	// use the current context in kubeconfig
	config, err := clientcmd.BuildConfigFromFlags("", *kubeconfig)
	if err != nil {
		panic(err.Error())
	}

	// create the clientset
	clientset, err := kubernetes.NewForConfig(config)
	metricsClient, _ := metrics.NewForConfig(config)
	if err != nil {
		panic(err.Error())
	}

	namespace := "applications"

	for {
		deployments, _ := clientset.AppsV1beta2().Deployments(namespace).List(metav1.ListOptions{})
		for _, deployment := range deployments.Items {
			if deployment.Name != "blog" {
				continue
			}

			deploymentLabels := deployment.Spec.Template.Labels
			var labelSelectors []string
			for key, value := range deploymentLabels {
				labelSelectors = append(labelSelectors, fmt.Sprintf("%s=%s", key, value))
			}
			labelSelector := strings.Join(labelSelectors, ",")
			pods, _ := clientset.CoreV1().Pods(namespace).List(metav1.ListOptions{LabelSelector: labelSelector})

			for _, pod := range pods.Items {
				fmt.Printf("Pod name: %s\n", pod.Name)
				podMetrics, _ := metricsClient.MetricsV1beta1().PodMetricses(namespace).Get(pod.Name, metav1.GetOptions{})
				for _, container := range podMetrics.Containers {
					fmt.Printf("\tContainer name: %s\n", container.Name)
					cpuQuantity := container.Usage.Cpu().AsDec()
					memQuantityRaw, _ := container.Usage.Memory().AsInt64()
					memQuantity := int64(math.Ceil(float64(memQuantityRaw) / 1024 / 1024))
					fmt.Printf("\t\tCPU: %d\n\t\tMemory: %d\n", cpuQuantity, memQuantity)
				}
			}
		}
		fmt.Printf("---\n\n")
		time.Sleep(2 * time.Second)
	}
}

func homeDir() string {
	if h := os.Getenv("HOME"); h != "" {
		return h
	}
	return os.Getenv("USERPROFILE") // windows
}
