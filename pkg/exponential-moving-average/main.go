package exponentialmovingaverage

import "fmt"

// ExponentialMovingAverage calcula a média móvel exponencial para um slice de dados.
func ExponentialMovingAverage(data []float64, alpha float64) float64 {
	if len(data) == 0 || alpha <= 0 || alpha > 1 {
		return 0
	}
	ema := data[0]
	for i := 1; i < len(data); i++ {
		ema = alpha*data[i] + (1-alpha)*ema
	}
	return ema
}

// RecommendResourcesEMA calcula os novos valores de request e limit usando EMA.
// currentRequest e currentLimit são valores atuais, mas a recomendação é baseada
// no histórico de uso (data). Os fatores de buffer podem ser ajustados.
func RecommendResourcesEMA(data []float64, currentRequest, currentLimit, alpha float64) (newRequest, newLimit float64) {
	avgUsage := ExponentialMovingAverage(data, alpha)
	// Adiciona um buffer de 20% ao uso médio para definir o request recomendado.
	newRequest = avgUsage * 1.20
	// Define o limit como 1.5 vezes o novo request (pode ser ajustado conforme política)
	newLimit = newRequest * 1.50
	return newRequest, newLimit
}

func ExponentialMovingAverageResult() {
	usage := []float64{100, 120, 130, 110, 115, 125}
	currentRequest := 100.0
	currentLimit := 200.0
	alpha := 0.3

	reqEMA, limEMA := RecommendResourcesEMA(usage, currentRequest, currentLimit, alpha)
	fmt.Printf("EMA:\n  Request atual: %.2f -> Novo recomendado: %.2f\n", currentRequest, reqEMA)
	fmt.Printf("  Limit atual:   %.2f -> Novo recomendado: %.2f\n\n", currentLimit, limEMA)
}
