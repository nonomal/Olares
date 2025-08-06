package user

import (
	"bytetrade.io/web3os/app-service/api/sys.bytetrade.io/v1alpha1"
	"fmt"
	iamv1alpha2 "github.com/beclab/api/iam/v1alpha2"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/clientcmd"
	"os"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func newUserClientFromKubeConfig(kubeconfig string) (client.Client, error) {
	if kubeconfig == "" {
		kubeconfig = os.Getenv("KUBECONFIG")
		if kubeconfig == "" {
			kubeconfig = clientcmd.RecommendedHomeFile
		}
	}

	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		return nil, fmt.Errorf("failed to get kubeconfig: %w", err)
	}

	scheme := runtime.NewScheme()

	if err := iamv1alpha2.AddToScheme(scheme); err != nil {
		return nil, fmt.Errorf("failed to add user scheme: %w", err)
	}

	if err := v1alpha1.AddToScheme(scheme); err != nil {
		return nil, fmt.Errorf("failed to add system scheme: %w", err)
	}

	userClient, err := client.New(config, client.Options{Scheme: scheme})
	if err != nil {
		return nil, fmt.Errorf("failed to create user client: %w", err)
	}
	return userClient, nil
}

func validateResourceLimit(limit resourceLimit) error {
	if limit.memoryLimit != "" {
		memLimit, err := resource.ParseQuantity(limit.memoryLimit)
		if err != nil {
			return fmt.Errorf("invalid memory limit: %v", err)
		}
		minMemLimit, _ := resource.ParseQuantity(defaultMemoryLimit)
		if memLimit.Cmp(minMemLimit) < 0 {
			return fmt.Errorf("invalid memory limit: %s is less than minimum required: %s", memLimit.String(), minMemLimit.String())
		}
	}

	if limit.cpuLimit != "" {
		cpuLimit, err := resource.ParseQuantity(limit.cpuLimit)
		if err != nil {
			return fmt.Errorf("invalid cpu limit: %v", err)
		}
		minCPULimit, _ := resource.ParseQuantity(defaultCPULimit)
		if cpuLimit.Cmp(minCPULimit) < 0 {
			return fmt.Errorf("invalid cpu limit: %s is less than minimum required: %s", cpuLimit.String(), minCPULimit.String())
		}
	}

	return nil
}
