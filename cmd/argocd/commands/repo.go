package commands

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"syscall"
	"text/tabwriter"

	"github.com/argoproj/argo-cd/errors"
	appsv1 "github.com/argoproj/argo-cd/pkg/apis/application/v1alpha1"
	"github.com/argoproj/argo-cd/server/repository"
	"github.com/argoproj/argo-cd/util"
	"github.com/argoproj/argo-cd/util/git"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"golang.org/x/crypto/ssh/terminal"
	"google.golang.org/grpc"
)

// NewRepoCommand returns a new instance of an `argocd repo` command
func NewRepoCommand() *cobra.Command {
	var command = &cobra.Command{
		Use:   "repo",
		Short: fmt.Sprintf("%s repo COMMAND", cliName),
		Run: func(c *cobra.Command, args []string) {
			c.HelpFunc()(c, args)
			os.Exit(1)
		},
	}

	command.AddCommand(NewRepoAddCommand())
	command.AddCommand(NewRepoListCommand())
	command.AddCommand(NewRepoRemoveCommand())
	return command
}

// NewRepoAddCommand returns a new instance of an `argocd repo add` command
func NewRepoAddCommand() *cobra.Command {
	var (
		repo appsv1.Repository
	)
	var command = &cobra.Command{
		Use:   "add",
		Short: fmt.Sprintf("%s repo add REPO", cliName),
		Run: func(c *cobra.Command, args []string) {
			if len(args) != 1 {
				c.HelpFunc()(c, args)
				os.Exit(1)
			}
			repo.Repo = args[0]
			err := git.TestRepo(repo.Repo, repo.Username, repo.Password)
			if err != nil {
				if repo.Username != "" && repo.Password != "" {
					// if everything was supplied, one of the inputs was definitely bad
					log.Fatal(err)
				}
				// If we can't test the repo, it's probably private. Prompt for credentials and try again.
				promptCredentials(&repo)
				err = git.TestRepo(repo.Repo, repo.Username, repo.Password)
			}
			errors.CheckError(err)
			conn, repoIf := NewRepoClient()
			defer util.Close(conn)
			createdRepo, err := repoIf.Create(context.Background(), &repo)
			errors.CheckError(err)
			fmt.Printf("repository '%s' added\n", createdRepo.Repo)
		},
	}
	command.Flags().StringVar(&repo.Username, "username", "", "username to the repository")
	command.Flags().StringVar(&repo.Password, "password", "", "password to the repository")
	return command
}

func promptCredentials(repo *appsv1.Repository) {
	reader := bufio.NewReader(os.Stdin)
	if repo.Username == "" {
		fmt.Print("Username: ")
		username, _ := reader.ReadString('\n')
		repo.Username = username
	}
	if repo.Password == "" {
		fmt.Print("Password: ")
		bytePassword, _ := terminal.ReadPassword(syscall.Stdin)
		repo.Password = string(bytePassword)
	}
}

// NewRepoRemoveCommand returns a new instance of an `argocd repo list` command
func NewRepoRemoveCommand() *cobra.Command {
	var command = &cobra.Command{
		Use:   "rm",
		Short: fmt.Sprintf("%s repo rm REPO", cliName),
		Run: func(c *cobra.Command, args []string) {
			if len(args) == 0 {
				c.HelpFunc()(c, args)
				os.Exit(1)
			}
			conn, repoIf := NewRepoClient()
			defer util.Close(conn)
			for _, repoURL := range args {
				_, err := repoIf.Delete(context.Background(), &repository.RepoQuery{Repo: repoURL})
				errors.CheckError(err)
			}
		},
	}
	return command
}

// NewRepoListCommand returns a new instance of an `argocd repo rm` command
func NewRepoListCommand() *cobra.Command {
	var command = &cobra.Command{
		Use:   "list",
		Short: fmt.Sprintf("%s repo list", cliName),
		Run: func(c *cobra.Command, args []string) {
			conn, repoIf := NewRepoClient()
			defer util.Close(conn)
			repos, err := repoIf.List(context.Background(), &repository.RepoQuery{})
			errors.CheckError(err)
			w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			fmt.Fprintf(w, "REPO\tUSER\n")
			for _, r := range repos.Items {
				fmt.Fprintf(w, "%s\t%s\n", r.Repo, r.Username)
			}
			_ = w.Flush()
		},
	}
	return command
}

func NewRepoClient() (*grpc.ClientConn, repository.RepositoryServiceClient) {
	// TODO: get this from a config or command line flag
	serverAddr := "localhost:8080"
	var dialOpts []grpc.DialOption
	// TODO: add insecure config option and --insecure global flag
	if true {
		dialOpts = append(dialOpts, grpc.WithInsecure())
	} // else if opts.Credentials != nil {
	//	dialOpts = append(dialOpts, grpc.WithTransportCredentials(opts.Credentials))
	//}
	conn, err := grpc.Dial(serverAddr, dialOpts...)
	if err != nil {
		log.Fatalf("Failed to establish connection to %s: %v", serverAddr, err)
	}
	repoIf := repository.NewRepositoryServiceClient(conn)
	return conn, repoIf
}
