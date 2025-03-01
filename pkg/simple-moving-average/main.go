package simplemovingaverage

import (
	"fmt"
)

// SimpleMovingAverage calcula a média simples para uma janela fixa.
func SimpleMovingAverage(data []float64, window int) float64 {
	if len(data) < window || window <= 0 {
		return 0
	}
	sum := 0.0
	// Neste exemplo, usamos a última janela dos dados.
	for i := len(data) - window; i < len(data); i++ {
		sum += data[i]
	}
	return sum / float64(window)
}

// RecommendResourcesSMA calcula os novos valores de request e limit usando SMA.
func RecommendResourcesSMA(data []float64, window int, currentRequest, currentLimit float64) (newRequest, newLimit float64) {
	avgUsage := SimpleMovingAverage(data, window)
	// Exemplo: adiciona 20% de buffer ao uso médio
	newRequest = avgUsage * 1.20
	// Define o limit como 1.5 vezes o novo request
	newLimit = newRequest * 1.50
	return newRequest, newLimit
}

func SimpleMovingAverageResult() {
	usage := []float64{100, 120, 130, 110, 115, 125, 1000, 1200, 1300, 1100, 1150, 1250, 100, 100, 100, 100}
	currentRequest := 100.0
	currentLimit := 200.0
	window := 3

	reqSMA, limSMA := RecommendResourcesSMA(usage, window, currentRequest, currentLimit)
	fmt.Printf("SMA (window %d):\n  Request atual: %.2f -> Novo recomendado: %.2f\n", window, currentRequest, reqSMA)
	fmt.Printf("  Limit atual:   %.2f -> Novo recomendado: %.2f\n\n", currentLimit, limSMA)
}
