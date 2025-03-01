package kubernetes

import (
	"context"
	"fmt"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// NodeAggregator contém informações agregadas sobre os nodes do cluster
type NodeAggregator struct {
	TotalNodes   int
	HealthyNodes int
	TotalCPU     int64 // em millicores
	UsedCPU      int64 // em millicores
	TotalMemory  int64 // em bytes
	UsedMemory   int64 // em bytes
	NodeList     []v1.Node
}

// GetNodesInfo retorna informações agregadas sobre os nodes do cluster
func (k *KubernetesClient) GetNodesInfo(ctx context.Context) (*NodeAggregator, error) {
	nodes, err := k.clientset.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("erro ao listar nodes: %v", err)
	}

	aggregator := &NodeAggregator{
		TotalNodes: len(nodes.Items),
		NodeList:   nodes.Items,
	}

	for _, node := range nodes.Items {
		// Contagem de nodes saudáveis
		if isNodeHealthy(node) {
			aggregator.HealthyNodes++
		}

		// Recursos totais
		cpuQuantity := node.Status.Capacity[v1.ResourceCPU]
		memoryQuantity := node.Status.Capacity[v1.ResourceMemory]

		aggregator.TotalCPU += cpuQuantity.MilliValue()
		aggregator.TotalMemory += memoryQuantity.Value()

		// Recursos utilizados
		if node.Status.Allocatable != nil {
			allocatableCPU := node.Status.Allocatable[v1.ResourceCPU]
			allocatableMemory := node.Status.Allocatable[v1.ResourceMemory]

			aggregator.UsedCPU += cpuQuantity.MilliValue() - allocatableCPU.MilliValue()
			aggregator.UsedMemory += memoryQuantity.Value() - allocatableMemory.Value()
		}
	}

	return aggregator, nil
}

// isNodeHealthy verifica se um node está saudável
func isNodeHealthy(node v1.Node) bool {
	for _, condition := range node.Status.Conditions {
		if condition.Type == v1.NodeReady {
			return condition.Status == v1.ConditionTrue
		}
	}
	return false
}
