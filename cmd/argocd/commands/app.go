package commands

import (
	"context"
	"fmt"
	"os"
	"text/tabwriter"

	argocdclient "github.com/argoproj/argo-cd/client"
	"github.com/argoproj/argo-cd/errors"
	argoappv1 "github.com/argoproj/argo-cd/pkg/apis/application/v1alpha1"
	"github.com/argoproj/argo-cd/server/application"
	"github.com/argoproj/argo-cd/util"
	"github.com/ghodss/yaml"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// NewApplicationCommand returns a new instance of an `argocd app` command
func NewApplicationCommand(clientOpts *argocdclient.ClientOptions) *cobra.Command {
	var command = &cobra.Command{
		Use:   "app",
		Short: fmt.Sprintf("%s app COMMAND", cliName),
		Run: func(c *cobra.Command, args []string) {
			c.HelpFunc()(c, args)
			os.Exit(1)
		},
	}

	command.AddCommand(NewApplicationAddCommand(clientOpts))
	command.AddCommand(NewApplicationGetCommand(clientOpts))
	command.AddCommand(NewApplicationSyncCommand(clientOpts))
	command.AddCommand(NewApplicationListCommand(clientOpts))
	command.AddCommand(NewApplicationRemoveCommand(clientOpts))
	return command
}

// NewApplicationAddCommand returns a new instance of an `argocd app add` command
func NewApplicationAddCommand(clientOpts *argocdclient.ClientOptions) *cobra.Command {
	var (
		repoURL string
		appPath string
		env     string
	)
	var command = &cobra.Command{
		Use:   "add",
		Short: fmt.Sprintf("%s app add APPNAME", cliName),
		Run: func(c *cobra.Command, args []string) {
			if len(args) != 1 {
				c.HelpFunc()(c, args)
				os.Exit(1)
			}
			app := argoappv1.Application{
				ObjectMeta: metav1.ObjectMeta{
					Name: args[0],
				},
				Spec: argoappv1.ApplicationSpec{
					Source: argoappv1.ApplicationSource{
						RepoURL:     repoURL,
						Path:        appPath,
						Environment: env,
					},
				},
			}
			conn, appIf := argocdclient.NewClient(clientOpts).NewApplicationClientOrDie()
			defer util.Close(conn)
			_, err := appIf.Create(context.Background(), &app)
			errors.CheckError(err)
		},
	}
	command.Flags().StringVar(&repoURL, "repo", "", "Repository URL")
	command.Flags().StringVar(&appPath, "path", "", "Path in repository to the ksonnet app directory")
	command.Flags().StringVar(&env, "env", "", "Application environment to monitor")

	return command
}

// NewApplicationGetCommand returns a new instance of an `argocd app get` command
func NewApplicationGetCommand(clientOpts *argocdclient.ClientOptions) *cobra.Command {
	var command = &cobra.Command{
		Use:   "get",
		Short: fmt.Sprintf("%s app get APPNAME", cliName),
		Run: func(c *cobra.Command, args []string) {
			if len(args) == 0 {
				c.HelpFunc()(c, args)
				os.Exit(1)
			}
			conn, appIf := argocdclient.NewClient(clientOpts).NewApplicationClientOrDie()
			defer util.Close(conn)
			for _, appName := range args {
				app, err := appIf.Get(context.Background(), &application.ApplicationQuery{Name: appName})
				errors.CheckError(err)
				yamlBytes, err := yaml.Marshal(app)
				errors.CheckError(err)
				fmt.Printf("%v\n", string(yamlBytes))
			}
		},
	}
	return command
}

// NewApplicationRemoveCommand returns a new instance of an `argocd app list` command
func NewApplicationRemoveCommand(clientOpts *argocdclient.ClientOptions) *cobra.Command {
	var command = &cobra.Command{
		Use:   "rm",
		Short: fmt.Sprintf("%s app rm APPNAME", cliName),
		Run: func(c *cobra.Command, args []string) {
			if len(args) == 0 {
				c.HelpFunc()(c, args)
				os.Exit(1)
			}
			conn, appIf := argocdclient.NewClient(clientOpts).NewApplicationClientOrDie()
			defer util.Close(conn)
			for _, appName := range args {
				_, err := appIf.Delete(context.Background(), &application.ApplicationQuery{Name: appName})
				errors.CheckError(err)
			}
		},
	}
	return command
}

// NewApplicationListCommand returns a new instance of an `argocd app rm` command
func NewApplicationListCommand(clientOpts *argocdclient.ClientOptions) *cobra.Command {
	var command = &cobra.Command{
		Use:   "list",
		Short: fmt.Sprintf("%s app list", cliName),
		Run: func(c *cobra.Command, args []string) {
			conn, appIf := argocdclient.NewClient(clientOpts).NewApplicationClientOrDie()
			defer util.Close(conn)
			apps, err := appIf.List(context.Background(), &application.ApplicationQuery{})
			errors.CheckError(err)
			w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			fmt.Fprintf(w, "NAME\tCLUSTER\tNAMESPACE\tSTATUS\n")
			for _, app := range apps.Items {
				fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", app.Name, app.Status.ComparisonResult.Server, app.Status.ComparisonResult.Namespace, app.Status.ComparisonResult.Status)
			}
			_ = w.Flush()
		},
	}
	return command
}

// NewApplicationSyncCommand returns a new instance of an `argocd app sync` command
func NewApplicationSyncCommand(clientOpts *argocdclient.ClientOptions) *cobra.Command {
	var (
		dryRun bool
	)
	var command = &cobra.Command{
		Use:   "sync",
		Short: fmt.Sprintf("%s app sync APPNAME", cliName),
		Run: func(c *cobra.Command, args []string) {
			if len(args) == 0 {
				c.HelpFunc()(c, args)
				os.Exit(1)
			}
			conn, appIf := argocdclient.NewClient(clientOpts).NewApplicationClientOrDie()
			defer util.Close(conn)
			appName := args[0]
			syncReq := application.ApplicationSyncRequest{
				Name:   appName,
				DryRun: dryRun,
			}
			syncRes, err := appIf.Sync(context.Background(), &syncReq)
			errors.CheckError(err)
			fmt.Printf("%s %s\n", appName, syncRes.Message)
			w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			fmt.Fprintf(w, "NAME\tKIND\tMESSAGE\n")
			for _, resDetails := range syncRes.Resources {
				fmt.Fprintf(w, "%s\t%s\t%s\n", resDetails.Name, resDetails.Kind, resDetails.Message)
			}
			_ = w.Flush()
		},
	}
	command.Flags().BoolVar(&dryRun, "dry-run", false, "Preview apply without affecting cluster")
	return command
}