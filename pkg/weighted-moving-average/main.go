package weightedmovingaverage

import (
	"fmt"
)

// WeightedMovingAverage calcula a média ponderada móvel para uma janela fixa.
func WeightedMovingAverage(data []float64, window int) float64 {
	if len(data) < window || window <= 0 {
		return 0
	}
	totalWeight := float64((window * (window + 1)) / 2)
	weightedSum := 0.0
	// Usa os últimos "window" elementos do slice
	start := len(data) - window
	for i := 0; i < window; i++ {
		weight := float64(i + 1) // Peso aumenta com a posição
		weightedSum += weight * data[start+i]
	}
	return weightedSum / totalWeight
}

// RecommendResourcesWMA calcula os novos valores de request e limit usando WMA.
func RecommendResourcesWMA(data []float64, window int, currentRequest, currentLimit float64) (newRequest, newLimit float64) {
	avgUsage := WeightedMovingAverage(data, window)
	// Aplica um buffer de 20% ao uso médio para definir o novo request
	newRequest = avgUsage * 1.20
	// Define o limit como 1.5 vezes o novo request
	newLimit = newRequest * 1.50
	return newRequest, newLimit
}

func WeightedMovingAverageResult() {
	usage := []float64{100, 120, 130, 110, 115, 125}
	currentRequest := 100.0
	currentLimit := 200.0
	window := 3

	reqWMA, limWMA := RecommendResourcesWMA(usage, window, currentRequest, currentLimit)
	fmt.Printf("WMA (window %d):\n  Request atual: %.2f -> Novo recomendado: %.2f\n", window, currentRequest, reqWMA)
	fmt.Printf("  Limit atual:   %.2f -> Novo recomendado: %.2f\n", currentLimit, limWMA)
}
