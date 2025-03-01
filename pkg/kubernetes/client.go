package kubernetes

import (
	"flag"
	"path/filepath"
	"sync"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
	metricsclient "k8s.io/metrics/pkg/client/clientset/versioned"
)

var (
	kubeconfigPath string
	once           sync.Once
)

// initKubeconfig define o caminho do kubeconfig apenas uma vez.
func initKubeconfig() {
	var kubeconfig *string
	if home := homedir.HomeDir(); home != "" {
		kubeconfig = flag.String("kubeconfig", filepath.Join(home, ".kube", "config"), "(optional) absolute path to the kubeconfig file")
	} else {
		kubeconfig = flag.String("kubeconfig", "", "absolute path to the kubeconfig file")
	}
	flag.Parse()
	kubeconfigPath = *kubeconfig
}

// KubernetesClient encapsula os clients do Kubernetes e Metrics.
type KubernetesClient struct {
	coreClient   *kubernetes.Clientset
	metrisClient *metricsclient.Clientset
	clientset    *kubernetes.Clientset
}

type IResources interface {
	IPod
}

// NewK8sClient cria e retorna uma nova instância de KubernetesClient,
// garantindo que o caminho do kubeconfig seja definido apenas uma vez.
func NewK8sClient() *KubernetesClient {
	once.Do(initKubeconfig)

	// Usa o contexto atual do kubeconfig.
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfigPath)
	if err != nil {
		panic(err.Error())
	}

	coreClientSet, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err.Error())
	}

	metricsClientSet, err := metricsclient.NewForConfig(config)
	if err != nil {
		panic(err.Error())
	}

	return &KubernetesClient{
		coreClient:   coreClientSet,
		metrisClient: metricsClientSet,
		clientset:    coreClientSet,
	}
}
