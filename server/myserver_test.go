package server

import (
	"testing"
	"context"
	"flag"
	"time"


	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/argoproj/argo-cd/v2/test"
	"github.com/argoproj/argo-cd/v2/reposerver/apiclient/mocks"
	apps "github.com/argoproj/argo-cd/v2/pkg/client/clientset/versioned/fake"
	servercache "github.com/argoproj/argo-cd/v2/server/cache"
	appstatecache "github.com/argoproj/argo-cd/v2/util/cache/appstate"
	cacheutil "github.com/argoproj/argo-cd/v2/util/cache"
)

type TestArgoCDServer struct {
	*ArgoCDServer
}

func testServer(t *testing.T) (*TestArgoCDServer, func()) {

	redis, closer := test.NewInMemoryRedis()
	appClientSet := apps.NewSimpleClientset()
	port, err := test.GetFreePort()

	if err != nil {
		panic(err)
	}
	
	
	argoCDOpts := ArgoCDServerOpts{
		Namespace: "argocd",
		ListenPort: port,
		AppClientset: appClientSet,
		RedisClient: redis,
		RepoClientset: &mocks.Clientset{RepoServerServiceClient: &mocks.RepoServerServiceClient{}},
		KubeClientset: getKubernetesClient(),
		Cache: servercache.NewCache(
			appstatecache.NewCache(
				cacheutil.NewCache(cacheutil.NewInMemoryCache(1*time.Hour)),
				1*time.Minute,
			),
			1*time.Minute,
			1*time.Minute,
			1*time.Minute,
		),
		StaticAssetsDir: "/workspaces/argo-tests/argo-cd/ui/src/app",
	}
	
	server := NewServer(context.Background(), argoCDOpts)

	s := &TestArgoCDServer{
		ArgoCDServer: server,
	}

	return s, closer
}

func getKubernetesClient() *kubernetes.Clientset {
	kubeconfig := flag.String("kubeconfig", "/workspaces/argo-tests/config.yaml", "(optional) absolute path to the kubeconfig file")
	
	flag.Parse()
	
	config, err := clientcmd.BuildConfigFromFlags("", *kubeconfig)

	if err != nil {
		panic(err.Error())
	}

	return kubernetes.NewForConfigOrDie(config)
}


func Test_ServerCreation(t *testing.T) {
	s, closer := testServer(t)
	defer closer()

	ctx := context.Background() 
	lstns, err := s.Listen()
	
	if err != nil {
		panic(err)
	}
	
	t.Log(lstns)
	s.Init(ctx)

	s.Run(ctx, lstns)
	
	
	
	t.Log(s)	
}
