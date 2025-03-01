package kubernetes

import (
	"context"
	"log"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/metrics/pkg/apis/metrics/v1beta1"
)

// IPod é uma interface para manipular pods e suas métricas.
type IPod interface {
	GetPodsList(namespace string) *v1.PodList
	PodMetricsList(namespace string) *v1beta1.PodMetricsList
}

// PodAggregator agrega informações consolidadas de pods.
type PodAggregator struct {
	PodList []PodInfo
}

// PodInfo armazena dados consolidados de uso e configurações de recursos.
// O campo DeploymentName armazena, genericamente, o nome do recurso (Deployment, DaemonSet ou StatefulSet).
type PodInfo struct {
	DeploymentName     string // Nome do recurso
	PodName            string // Nome individual do pod
	Namespace          string
	Replicas           int32   // Valor do recurso (não alterado)
	CurrentCPUUsage    int64   // em millicores (m) – soma total de usage
	CurrentMemoryUsage float64 // em Mi – soma total de usage
	CpuRequest         int64   // em millicores (m) – fixo do recurso
	CpuLimit           int64   // em millicores (m) – fixo do recurso
	MemoryRequest      float64 // em Mi – fixo do recurso
	MemoryLimit        float64 // em Mi – fixo do recurso
}

// PodMetrics coleta as métricas dos pods associados aos recursos Deployments, DaemonSets e StatefulSets.
func (k *KubernetesClient) PodMetrics(namespace string) PodAggregator {
	aggregator := PodAggregator{}

	// Lista os pods e suas métricas (única listagem para o namespace)
	pods, err := k.coreClient.CoreV1().Pods(namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		log.Fatalf("Erro ao listar pods: %v", err)
	}
	podMetricsList, err := k.metrisClient.MetricsV1beta1().PodMetricses(namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		log.Fatalf("Erro ao listar métricas de pods: %v", err)
	}

	// Processa Deployments
	deployments, err := k.coreClient.AppsV1().Deployments(namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		log.Fatalf("Erro ao listar deployments: %v", err)
	}
	for _, d := range deployments.Items {
		if len(d.Spec.Template.Spec.Containers) == 0 {
			log.Printf("[WARN] Deployment %s não possui containers", d.Name)
			continue
		}
		mainContainerName := d.Spec.Template.Spec.Containers[0].Name
		replicas := d.Status.Replicas
		processResourcePods(d.Name, d.Namespace, d.Spec.Selector.MatchLabels, d.Spec.Template.Spec.Containers, replicas, mainContainerName, pods, podMetricsList, &aggregator)
	}

	// Processa DaemonSets
	daemonSets, err := k.coreClient.AppsV1().DaemonSets(namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		log.Fatalf("Erro ao listar daemonsets: %v", err)
	}
	for _, ds := range daemonSets.Items {
		if len(ds.Spec.Template.Spec.Containers) == 0 {
			log.Printf("[WARN] DaemonSet %s não possui containers", ds.Name)
			continue
		}
		mainContainerName := ds.Spec.Template.Spec.Containers[0].Name
		// Para DaemonSet, usamos o número atual de pods agendados
		replicas := ds.Status.CurrentNumberScheduled
		processResourcePods(ds.Name, ds.Namespace, ds.Spec.Selector.MatchLabels, ds.Spec.Template.Spec.Containers, replicas, mainContainerName, pods, podMetricsList, &aggregator)
	}

	// Processa StatefulSets
	statefulSets, err := k.coreClient.AppsV1().StatefulSets(namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		log.Fatalf("Erro ao listar statefulsets: %v", err)
	}
	for _, ss := range statefulSets.Items {
		if len(ss.Spec.Template.Spec.Containers) == 0 {
			log.Printf("[WARN] StatefulSet %s não possui containers", ss.Name)
			continue
		}
		mainContainerName := ss.Spec.Template.Spec.Containers[0].Name
		replicas := ss.Status.Replicas
		processResourcePods(ss.Name, ss.Namespace, ss.Spec.Selector.MatchLabels, ss.Spec.Template.Spec.Containers, replicas, mainContainerName, pods, podMetricsList, &aggregator)
	}

	// Agrupa os PodInfo por recurso (DeploymentName)
	aggregator.PodList = groupPodsByResource(aggregator.PodList)

	return aggregator
}

// processResourcePods filtra os pods que pertencem ao recurso (via match de labels)
// e coleta as métricas do container principal.
func processResourcePods(resourceName, namespace string, selector map[string]string, containers []v1.Container, replicas int32, mainContainerName string, pods *v1.PodList, podMetricsList *v1beta1.PodMetricsList, aggregator *PodAggregator) {
	for _, pod := range pods.Items {
		if !podMatchesSelector(pod, selector) {
			continue
		}

		cpuUsage, memoryUsage, found := getPodUsage(pod, podMetricsList, mainContainerName)
		if !found {
			log.Printf("[WARN] Métricas não encontradas para o pod %s", pod.Name)
		}

		// Usa os valores de requests/limits do recurso (do primeiro container)
		cpuRequest, cpuLimit, memoryRequest, memoryLimit := getContainerRequestsAndLimitsGeneric(containers, mainContainerName)

		podInfo := PodInfo{
			DeploymentName:     resourceName,
			PodName:            pod.Name,
			Namespace:          namespace,
			Replicas:           replicas,
			CurrentCPUUsage:    cpuUsage,
			CurrentMemoryUsage: memoryUsage,
			CpuRequest:         cpuRequest,
			CpuLimit:           cpuLimit,
			MemoryRequest:      memoryRequest,
			MemoryLimit:        memoryLimit,
		}
		aggregator.PodList = append(aggregator.PodList, podInfo)
	}
}

// podMatchesSelector verifica se o pod satisfaz o seletor fornecido.
func podMatchesSelector(pod v1.Pod, selector map[string]string) bool {
	for key, value := range selector {
		if pod.Labels[key] != value {
			return false
		}
	}
	return true
}

// getPodUsage extrai o uso de CPU (em millicores) e memória (em Mi) do container principal do pod.
func getPodUsage(pod v1.Pod, podMetricsList *v1beta1.PodMetricsList, mainContainerName string) (int64, float64, bool) {
	for _, pm := range podMetricsList.Items {
		if pm.Name == pod.Name {
			for _, containerMetric := range pm.Containers {
				if containerMetric.Name == mainContainerName {
					cpuUsage := containerMetric.Usage.Cpu().MilliValue()
					memoryUsage := bytesToMi(containerMetric.Usage.Memory().Value())
					return cpuUsage, memoryUsage, true
				}
			}
		}
	}
	return 0, 0, false
}

// getContainerRequestsAndLimitsGeneric retorna os valores de requests e limits (CPU e memória)
// do container principal do recurso, fazendo a conversão de memória para Mi.
func getContainerRequestsAndLimitsGeneric(containers []v1.Container, mainContainerName string) (int64, int64, float64, float64) {
	var cpuRequest, cpuLimit int64
	var memoryRequest, memoryLimit float64
	for _, container := range containers {
		if container.Name == mainContainerName {
			if container.Resources.Requests != nil {
				cpuRequest = container.Resources.Requests.Cpu().MilliValue()
				memoryRequest = bytesToMi(container.Resources.Requests.Memory().Value())
			}
			if container.Resources.Limits != nil {
				cpuLimit = container.Resources.Limits.Cpu().MilliValue()
				memoryLimit = bytesToMi(container.Resources.Limits.Memory().Value())
			}
			break
		}
	}
	return cpuRequest, cpuLimit, memoryRequest, memoryLimit
}

// groupPodsByResource agrupa os PodInfo pelo nome do recurso (DeploymentName) e
// soma somente os valores de CPU/Memory usage, mantendo os demais valores (requests/limits e Replicas)
// do primeiro item do grupo.
func groupPodsByResource(pods []PodInfo) []PodInfo {
	grouped := make(map[string][]PodInfo)
	for _, p := range pods {
		grouped[p.DeploymentName] = append(grouped[p.DeploymentName], p)
	}
	var result []PodInfo
	for resourceName, podList := range grouped {
		if len(podList) == 1 {
			result = append(result, podList[0])
			continue
		}
		first := podList[0]
		var sumCPUUsage int64
		var sumMemoryUsage float64
		for _, item := range podList {
			sumCPUUsage += item.CurrentCPUUsage
			sumMemoryUsage += item.CurrentMemoryUsage
		}
		groupedInfo := PodInfo{
			DeploymentName:     resourceName,
			PodName:            resourceName, // Exibe o nome do recurso
			Namespace:          first.Namespace,
			Replicas:           first.Replicas,
			CurrentCPUUsage:    sumCPUUsage,    // soma total de usage
			CurrentMemoryUsage: sumMemoryUsage, // soma total de usage
			CpuRequest:         first.CpuRequest,
			CpuLimit:           first.CpuLimit,
			MemoryRequest:      first.MemoryRequest,
			MemoryLimit:        first.MemoryLimit,
		}
		result = append(result, groupedInfo)
	}
	return result
}

// bytesToMi converte um valor em bytes para Mebibytes (Mi).
func bytesToMi(value int64) float64 {
	const Mi = 1024 * 1024
	return float64(value) / Mi
}
