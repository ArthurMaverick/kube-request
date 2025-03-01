package controller

import (
	"fmt"
	"html/template"
	"log"
	"net/http"
	"path/filepath"

	"github.com/ArthurMaverick/kube-request/pkg/kubernetes"
)

// Template functions map
var templateFuncs = template.FuncMap{
	"mul": func(a, b float64) float64 {
		return a * b
	},
	"div": func(a, b float64) float64 {
		if b == 0 {
			return 0
		}
		return a / b
	},
	"add": func(a, b float64) float64 {
		return a + b
	},
	"sub": func(a, b float64) float64 {
		return a - b
	},
	"min": func(a, b float64) float64 {
		if a < b {
			return a
		}
		return b
	},
	"float64": func(v interface{}) float64 {
		switch val := v.(type) {
		case int:
			return float64(val)
		case int32:
			return float64(val)
		case int64:
			return float64(val)
		case float32:
			return float64(val)
		case float64:
			return val
		default:
			return 0
		}
	},
	"formatCPU": func(millicores float64) string {
		if millicores >= 1000 {
			return fmt.Sprintf("%.2f cores", millicores/1000)
		}
		return fmt.Sprintf("%dm", int64(millicores))
	},
	"formatMemory": func(bytes float64) string {
		const (
			KiB = 1024
			MiB = 1024 * KiB
			GiB = 1024 * MiB
		)

		switch {
		case bytes >= GiB:
			return fmt.Sprintf("%.2f GiB", bytes/GiB)
		case bytes >= MiB:
			return fmt.Sprintf("%.2f MiB", bytes/MiB)
		case bytes >= KiB:
			return fmt.Sprintf("%.2f KiB", bytes/KiB)
		default:
			return fmt.Sprintf("%.2f B", bytes)
		}
	},
	"formatPercentage": func(value float64) string {
		return fmt.Sprintf("%.1f%%", value)
	},
}

// PageData contains the data to be rendered in the template.
type PageData struct {
	Title              string
	Aggregator         kubernetes.PodAggregator
	NodeAggregator     *kubernetes.NodeAggregator
	CurrentNamespace   string
	AllNamespaces      []string
	CurrentKubeContext string
	AllKubeContexts    []string
	ErrorMessage       string
	Filter             string
	TotalMetrics
}

type TotalMetrics struct {
	TotalCpuUsage       int64
	TotalCpuRequests    int64
	TotalCpuLimit       int64
	TotalMemoryUsage    float64
	TotalMemoryRequests float64
	TotalMemoryLimit    float64
}

// PodController  is a controller for handling pod-related requests.
type PodController struct {
	k8sClient *kubernetes.KubernetesClient
}

// NewPodController creates a new PodController.
func NewPodController() *PodController {
	k8sClient := kubernetes.NewK8sClient()
	return &PodController{k8sClient: k8sClient}
}

func GetTotalRequestsLimitsAndUsage(podList kubernetes.PodAggregator) TotalMetrics {
	var totalMetrics TotalMetrics

	for _, pod := range podList.PodList {
		totalMetrics.TotalCpuUsage += pod.CurrentCPUUsage
		totalMetrics.TotalMemoryUsage += pod.CurrentMemoryUsage
		totalMetrics.TotalCpuRequests += pod.CpuRequest
		totalMetrics.TotalMemoryRequests += pod.MemoryRequest
		totalMetrics.TotalCpuLimit += pod.CpuLimit
		totalMetrics.TotalMemoryLimit += pod.MemoryLimit
	}
	return totalMetrics
}

// HandlePods handles the request for the main page.
func (pc *PodController) HandlePods(w http.ResponseWriter, r *http.Request) {
	currentCtx, allContexts, err := kubernetes.GetKubeContexts()
	if err != nil {
		log.Printf("Erro ao obter contextos do kubeconfig: %v", err)
		currentCtx = "N/A"
		allContexts = []string{"N/A"}
	}

	allNamespaces := pc.k8sClient.ListNamespaces()

	// Query strings
	ns := r.URL.Query().Get("namespace")
	if ns == "" {
		ns = "default"
	}
	filterStr := r.URL.Query().Get("filter")
	errorMsg := r.URL.Query().Get("error")

	nodeAggregator, err := pc.k8sClient.GetNodesInfo(r.Context())
	if err != nil {
		log.Printf("Erro ao obter informações dos nodes: %v", err)
		nodeAggregator = &kubernetes.NodeAggregator{}
	}

	data := PageData{
		Title:              "Pods Overview",
		Aggregator:         pc.k8sClient.PodMetrics(ns),
		NodeAggregator:     nodeAggregator,
		CurrentNamespace:   ns,
		AllNamespaces:      allNamespaces,
		CurrentKubeContext: currentCtx,
		AllKubeContexts:    allContexts,
		ErrorMessage:       errorMsg,
		Filter:             filterStr,
		TotalMetrics:       GetTotalRequestsLimitsAndUsage(pc.k8sClient.PodMetrics(ns)),
	}

	// Load and parse templates with functions
	tmpl, err := template.New("").Funcs(templateFuncs).ParseFiles(
		filepath.Join("templates", "layout.gohtml"),
		filepath.Join("templates", "components", "header.gohtml"),
		filepath.Join("templates", "components", "main.gohtml"),
		filepath.Join("templates", "components", "footer.gohtml"),
		filepath.Join("templates", "components", "table.gohtml"),
	)
	if err != nil {
		log.Printf("Erro ao carregar templates: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	if err := tmpl.ExecuteTemplate(w, "layout", data); err != nil {
		log.Printf("Erro ao executar template: %v", err)
		return
	}
}

func (pc *PodController) HandleChangeContext(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Método não permitido", http.StatusMethodNotAllowed)
		return
	}
	novoContexto := r.FormValue("kubecontext")
	err := kubernetes.ChangeKubeContext(novoContexto)
	if err != nil {
		log.Printf("Erro ao alterar o contexto: %v", err)
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}
	pc.k8sClient = kubernetes.NewK8sClient()
	http.Redirect(w, r, "/", http.StatusSeeOther)
}
