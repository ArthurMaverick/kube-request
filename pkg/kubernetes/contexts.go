package kubernetes

import (
	"fmt"
	"os"
	"path/filepath"

	"k8s.io/client-go/tools/clientcmd"
)

// ChangeKubeContext altera o contexto atual do kubeconfig para o valor informado.
func ChangeKubeContext(contextName string) error {
	kubeconfig := os.Getenv("KUBECONFIG")
	if kubeconfig == "" {
		kubeconfig = filepath.Join(os.Getenv("HOME"), ".kube", "config")
	}

	// Carrega o kubeconfig
	config, err := clientcmd.LoadFromFile(kubeconfig)
	if err != nil {
		return fmt.Errorf("failed to load kubeconfig: %v", err)
	}

	// Verifica se o contexto existe
	if _, ok := config.Contexts[contextName]; !ok {
		return fmt.Errorf("context %q not found in kubeconfig", contextName)
	}

	// Define o novo contexto atual
	config.CurrentContext = contextName

	// Escreve as alterações de volta no arquivo de kubeconfig
	if err := clientcmd.WriteToFile(*config, kubeconfig); err != nil {
		return fmt.Errorf("failed to write kubeconfig: %v", err)
	}

	return nil
}

// GetKubeContexts lê o kubeconfig e retorna o contexto atual e todos os contextos disponíveis.
func GetKubeContexts() (current string, allContexts []string, err error) {
	kubeconfig := os.Getenv("KUBECONFIG")
	if kubeconfig == "" {
		kubeconfig = filepath.Join(os.Getenv("HOME"), ".kube", "config")
	}
	config, err := clientcmd.LoadFromFile(kubeconfig)
	if err != nil {
		return "", nil, fmt.Errorf("failed to load kubeconfig: %v", err)
	}
	current = config.CurrentContext
	for ctx := range config.Contexts {
		allContexts = append(allContexts, ctx)
	}
	return current, allContexts, nil
}
