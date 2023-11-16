package oidc

import (
	"flag"
	"testing"
	"fmt"

	"net/http"
	"net/http/httptest"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"github.com/argoproj/argo-cd/v2/util/test"
	"github.com/argoproj/argo-cd/v2/util/settings"

)


func getKubernetesClient() *kubernetes.Clientset {
	kubeconfig := flag.String("kubeconfig", "/workspaces/argo-tests/config.yaml", "(optional) absolute path to the kubeconfig file")
	
	flag.Parse()
	
	config, err := clientcmd.BuildConfigFromFlags("", *kubeconfig)

	if err != nil {
		panic(err.Error())
	}

	return kubernetes.NewForConfigOrDie(config)
}


func Test_OurRedirect(t *testing.T) {
	oidcTestServer := test.GetOIDCTestServer(t)
	t.Cleanup(oidcTestServer.Close)

	dexTestServer := test.GetDexTestServer(t)
	t.Cleanup(dexTestServer.Close)

	t.Run("Our test handle URLS" , func(t *testing.T) {
		cdSettings := &settings.ArgoCDSettings{
			URL: "https://eaa-test-argocd-poc.go.akamai-access.com",
			OIDCConfigRAW: fmt.Sprintf(`
name: Test
issuer: %s
clientID: xxx
clientSecret: yyy
requestedScopes: ["oidc"]`, oidcTestServer.URL),
		}
		cdSettings.OIDCTLSInsecureSkipVerify = true
		app, err := NewClientApp(cdSettings, dexTestServer.URL, nil, "https://eaa-test-argocd-poc.go.akamai-access.com")
		if err != nil {
			t.Fatalf("Unable to create client app: %v", err)
		}
		req := httptest.NewRequest(http.MethodGet, "https://eaa-test-argocd-poc.go.akamai-access.com/auth/login?return_url=https%3A%2F%2Feaa-test-argocd-poc.go.akamai-access.com%2Fapplications", nil)
		w := httptest.NewRecorder()
		app.HandleLogin(w, req)

		t.Log(w.Body.String())
	})
}
