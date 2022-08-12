package main

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"time"
)

// k8s-namespace-deleter deletes Kubernetes namespaces
//
// It takes as input the namespace as an argument and a few optional arguments.
//
// The following arguments are mandatory:
// * --namespace the namespace to delete
//
// The following flags are optional:
// * --kube-context (will use current context if not specified)
// * --kube-config  (kubectl will use its default algorithm to search for credentials)
// * --port         (kubectl proxy port, default to 8080)

// Example command:
//
/*
k8s-namespace-deleter --namespace obsolete-namespace  /
                      --kube-context some-context     /
                      --kube-config  ~/.kube/config   /
                      --port         8888
*/

const (
	maxAttempts = 5
	delay       = time.Millisecond * 1000
)

func startKubeProxy(kubeContext string, port int) *exec.Cmd {
	cmd := exec.Command("kubectl", "proxy", "--context", kubeContext, "--port", strconv.Itoa(port))
	err := cmd.Start()
	if err != nil {
		log.Fatalf("failed to start kubectl proxy: %v\n", err)
	}

	return cmd
}

func killKubeProxy(cmd *exec.Cmd) {
	err := cmd.Process.Kill()
	if err != nil {
		log.Fatalf("failed to kill the proxy. everything else is fine, err: %v\n", err)
	}
}

// composeURL composes the URL for the namespace
func composeURL(port int, namespace string) string {
	format := "http://localhost:%d/api/v1/namespaces/%s"
	return fmt.Sprintf(format, port, namespace)
}

func createNamespacePayload(namespace string) []byte {
	format := `
{
  "kind": "Namespace",
  "apiVersion": "v1",
  "metadata": {
    "name": "%s"
    },
    "spec": {
        "finalizers": []
    }
}
`
	return []byte(fmt.Sprintf(format, namespace))
}

// getNamespace fetches the namespace resource as raw JSON
func doesNamespaceExist(kubeContext string, url string) bool {
	cmd := exec.Command("kubectl", "get", "--context", kubeContext, "--raw", url)
	err := cmd.Run()
	if err != nil {
		return false
	}

	return true
}

// updateNamespace takes a url and a namespace
//
// It replace the namespace (that might have finalizers) with
// a simple namespace that just contains the name of the original
// namespace with no spec or status.
func updateNamespace(kubeContext string, url string, namespace string) {
	payload := createNamespacePayload(namespace)

	cli := http.DefaultClient

	// Retry several times in case the proxy is not ready yet
	var err error
	var req *http.Request
	var r *http.Response
	verb := http.MethodPut
	contentType := "application/json"
	for i := 0; i < maxAttempts; i++ {
		// Create a new request for each attempt
		req, err = http.NewRequest(verb, url+"/finalize", bytes.NewReader(payload))
		if err != nil {
			log.Fatalf("failed to create HTTP request: %v\n", err)
		}

		req.Header.Set("Content-Type", contentType)

		r, err = cli.Do(req)
		// All is well, break early
		if err == nil && r != nil && r.StatusCode == 200 {
			break
		}

		// Wait a little before trying again
		time.Sleep(delay)
	}
	if err != nil || r == nil {
		log.Fatalf("failed to update namespace: %v\n", err)
	}

	if r.StatusCode >= 400 {
		// Sometimes the namespace is already deleted and status code is 404. Just exit in this case.
		if !doesNamespaceExist(kubeContext, namespace) {
			log.Printf("namespace %s was deleted successfully.\n", namespace)
			os.Exit(0)
		}
		log.Fatalf("failed to update namespace: %v\n", r.Status)
	}
}

func deleteNamespace(kubeContext string, namespace string) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "kubectl", "delete", "--context", kubeContext, "ns", namespace)
	err := cmd.Run()
	if err != nil && ctx.Err() != context.DeadlineExceeded {
		log.Fatalf("failed to delete namespace %s, err: %v\n", namespace, err)
	}
}

func main() {
	var (
		namespace      string
		kubeConfigPath string
		kubeContext    string
		port           int
	)
	flag.StringVar(&namespace, "namespace", "", "the obsolete namespace to delete (mandatory)")
	flag.StringVar(&kubeConfigPath, "kube-config", "", "path to the kubernetes config file, if unset, will use $HOME/.kube/config")
	flag.StringVar(&kubeContext, "kube-context", "", "the context to use in the kube config file")
	flag.IntVar(&port, "port", 8888, "the port to start the kubectl proxy with")
	flag.Parse()

	// Verify arguments and get namespace
	if namespace == "" {
		log.Fatalf("usage: %s --namespace <namespace> [<flags>].\n", os.Args[0])
	}

	// Set KUBECONFIG to the provided kube config file (if no file is provided use default)
	if kubeConfigPath != "" {
		err := os.Setenv("KUBECONFIG", kubeConfigPath)
		if err != nil {
			log.Fatalf("failed to set KUBECONFIG to %s, err: %v\n", kubeConfigPath, err)
		}
	}

	// Start the proxy, so it can access the cluster on localhost
	cmd := startKubeProxy(kubeContext, port)

	// Kill the proxy when we're done
	defer killKubeProxy(cmd)

	// Prepare the base URL
	url := composeURL(port, namespace)

	// Verify the namespace exists (will fail fatally if it doesn't exist)
	ok := doesNamespaceExist(kubeContext, url)
	if !ok {
		log.Fatalf("namespace %s doesn't exist\n", namespace)
	}

	deleteNamespace(kubeContext, namespace)
	updateNamespace(kubeContext, url, namespace)

	// Verify the namespace doesn't exist anymore
	ok = doesNamespaceExist(kubeContext, url)
	if !ok {
		log.Printf("namespace %s was deleted successfully.\n", namespace)
	} else {
		log.Fatalf("oh, no! namespace %s still exists.\n", namespace)
	}
}
