package cloud

import (
	"fmt"
	"io"

	"github.com/astronomer/astro-cli/astro-client"

	airflowversions "github.com/astronomer/astro-cli/airflow_versions"
	astrocore "github.com/astronomer/astro-cli/astro-client-core"
	"github.com/astronomer/astro-cli/cloud/deployment"
	"github.com/astronomer/astro-cli/cloud/deployment/fromfile"
	"github.com/astronomer/astro-cli/cloud/organization"
	"github.com/astronomer/astro-cli/pkg/httputil"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

const (
	enable  = "enable"
	disable = "disable"
)

var (
	label                         string
	runtimeVersion                string
	deploymentID                  string
	forceDelete                   bool
	description                   string
	clusterID                     string
	dagDeploy                     string
	schedulerAU                   int
	schedulerReplicas             int
	updateSchedulerReplicas       int
	updateSchedulerAU             int
	forceUpdate                   bool
	allDeployments                bool
	warnLogs                      bool
	errorLogs                     bool
	infoLogs                      bool
	waitForStatus                 bool
	logCount                      = 500
	variableKey                   string
	variableValue                 string
	useEnvFile                    bool
	makeSecret                    bool
	executor                      string
	inputFile                     string
	cloudProvider                 string
	region                        string
	schedulerSize                 string
	highAvailability              bool
	deploymentVariableListExample = `
		# List a deployment's variables
		$ astro deployment variable list --deployment-id <deployment-id> --key FOO
		# List a deployment's variables and save them to a file
		$ astro deployment variable list  --deployment-id <deployment-id> --save --env .env.my-deployment
		`
	deploymentVariableCreateExample = `
		# Create a deployment variable
		$ astro deployment variable create FOO=BAR FOO2=BAR2 --deployment-id <deployment-id> --secret
		# Create a deployment variables from a file
		$ astro deployment variable create --deployment-id <deployment-id> --load --env .env.my-deployment
		`
	deploymentVariableUpdateExample = `
		# Update a deployment variable
		$ astro deployment variable update FOO=NEWBAR FOO2=NEWBAR2 --deployment-id <deployment-id> --secret
		# Update a deployment variables from a file
		$ astro deployment variable update --deployment-id <deployment-id> --load --env .env.my-deployment
		`
	httpClient              = httputil.NewHTTPClient()
	errFlag                 = errors.New("--deployment-file can not be used with other arguments")
	errInvalidExecutor      = errors.New("not a valid executor")
	errInvalidCloudProvider = errors.New("not a valid cloud provider. It can only be gcp")
	errNoRegion             = errors.New("region must be specified with --cloud-provider")
)

func newDeploymentRootCmd(out io.Writer) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "deployment",
		Aliases: []string{"de"},
		Short:   "Manage your Deployments running on Astronomer",
		Long:    "Create or manage Deployments running on Astro according to your Organization and Workspace permissions.",
	}
	cmd.PersistentFlags().StringVar(&workspaceID, "workspace-id", "", "workspace assigned to deployment")
	cmd.AddCommand(
		newDeploymentListCmd(out),
		newDeploymentDeleteCmd(),
		newDeploymentCreateCmd(out),
		newDeploymentLogsCmd(),
		newDeploymentUpdateCmd(out),
		newDeploymentVariableRootCmd(out),
		newDeploymentWorkerQueueRootCmd(out),
		newDeploymentInspectCmd(out),
	)
	return cmd
}

func newDeploymentListCmd(out io.Writer) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List all Deployments running in your Astronomer Workspace",
		Long:    "List all Deployments running in your Astronomer Workspace. Switch Workspaces to see other Deployments in your Organization.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return deploymentList(cmd, out)
		},
	}
	cmd.Flags().BoolVarP(&allDeployments, "all", "a", false, "Show deployments across all workspaces")
	return cmd
}

func newDeploymentLogsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "logs [Deployment-ID]",
		Aliases: []string{"l"},
		Short:   "Show an Astro Deployment's Scheduler logs",
		Long:    "Show an Astro Deployment's Scheduler logs. Use flags to determine what log level to show.",
		RunE:    deploymentLogs,
	}
	cmd.Flags().BoolVarP(&warnLogs, "warn", "w", false, "Show logs with a log level of 'warning'")
	cmd.Flags().BoolVarP(&errorLogs, "error", "e", false, "Show logs with a log level of 'error'")
	cmd.Flags().BoolVarP(&infoLogs, "info", "i", false, "Show logs with a log level of 'info'")
	cmd.Flags().IntVarP(&logCount, "log-count", "c", logCount, "Number of logs to show")
	cmd.Flags().StringVarP(&deploymentName, "deployment-name", "n", "", "Name of the deployment to show logs of")
	return cmd
}

func newDeploymentCreateCmd(out io.Writer) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "create",
		Aliases: []string{"cr"},
		Short:   "Create a new Astro Deployment",
		Long:    "Create a new Astro Deployment. All flags are optional",
		RunE: func(cmd *cobra.Command, args []string) error {
			return deploymentCreate(cmd, args, out)
		},
	}
	cmd.Flags().StringVarP(&label, "name", "n", "", "The Deployment's name. If the name contains a space, specify the entire name within quotes \"\" ")
	cmd.Flags().StringVarP(&workspaceID, "workspace-id", "w", "", "Workspace to create the Deployment in")
	cmd.Flags().StringVarP(&description, "description", "d", "", "Description of the Deployment. If the description contains a space, specify the entire description in quotes \"\"")
	cmd.Flags().StringVarP(&clusterID, "cluster-id", "c", "", "Cluster to create the Deployment in")
	cmd.Flags().StringVarP(&runtimeVersion, "runtime-version", "v", "", "Runtime version for the Deployment")
	cmd.Flags().StringVarP(&dagDeploy, "dag-deploy", "", "disable", "Enables DAG-only deploys for the deployment")
	cmd.Flags().StringVarP(&executor, "executor", "e", "", "The executor to use for the deployment. Possible values can be CeleryExecutor or KubernetesExecutor.")
	cmd.Flags().StringVarP(&inputFile, "deployment-file", "", "", "Location of file containing the deployment to create. File can be in either JSON or YAML format.")
	cmd.Flags().IntVarP(&schedulerAU, "scheduler-au", "s", 0, "The Deployment's Scheduler resources in AUs")
	cmd.Flags().IntVarP(&schedulerReplicas, "scheduler-replicas", "r", 0, "The number of Scheduler replicas for the Deployment")
	cmd.Flags().BoolVarP(&waitForStatus, "wait", "i", false, "Wait for the Deployment to become healthy before ending the command")
	cmd.Flags().BoolVarP(&cleanOutput, "clean-output", "", false, "clean output to only include inspect yaml or json file in any situation.")
	if organization.IsOrgHosted() {
		cmd.Flags().StringVarP(&cloudProvider, "cloud-provider", "p", "", "The Cloud Provider to use for the Deployment. Possible values can be gcp.")
		cmd.Flags().StringVarP(&region, "region", "", "", "The Cloud Provider region to use for the deployment.")
		cmd.Flags().StringVarP(&schedulerSize, "scheduler-size", "", "", "The size of Scheduler for the Deployment. Possible values can be small, medium, large")
		cmd.Flags().BoolVarP(&highAvailability, "high-availability", "a", false, "High Availability for the Deployment")
	}
	return cmd
}

func newDeploymentUpdateCmd(out io.Writer) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "update [DEPLOYMENT-ID]",
		Aliases: []string{"up"},
		Short:   "Update an Astro Deployment",
		Long:    "Update the configuration for an Astro Deployment. All flags are optional",
		RunE: func(cmd *cobra.Command, args []string) error {
			return deploymentUpdate(cmd, args, out)
		},
	}
	cmd.Flags().StringVarP(&label, "name", "n", "", "Update the Deployment's name. If the new name contains a space, specify the entire name within quotes \"\" ")
	cmd.Flags().StringVarP(&workspaceID, "workspace-id", "w", "", "Workspace the Deployment is located in")
	cmd.Flags().StringVarP(&description, "description", "d", "", "Description of the Deployment. If the description contains a space, specify the entire description in quotes \"\"")
	cmd.Flags().StringVarP(&executor, "executor", "e", "", "The executor to use for the deployment. Possible values can be CeleryExecutor or KubernetesExecutor.")
	cmd.Flags().StringVarP(&inputFile, "deployment-file", "", "", "Location of file containing the deployment to update. File can be in either JSON or YAML format.")
	cmd.Flags().IntVarP(&updateSchedulerAU, "scheduler-au", "s", 0, "The Deployment's Scheduler resources in AUs")
	cmd.Flags().IntVarP(&updateSchedulerReplicas, "scheduler-replicas", "r", 0, "The number of Scheduler replicas for the Deployment")
	cmd.Flags().BoolVarP(&forceUpdate, "force", "f", false, "Force update: Don't prompt a user before Deployment update")
	cmd.Flags().StringVarP(&deploymentName, "deployment-name", "", "", "Name of the deployment to update")
	cmd.Flags().StringVarP(&dagDeploy, "dag-deploy", "", "", "Enables DAG-only deploys for the deployment")
	cmd.Flags().BoolVarP(&cleanOutput, "clean-output", "c", false, "clean output to only include inspect yaml or json file in any situation.")
	return cmd
}

func newDeploymentDeleteCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "delete DEPLOYMENT-ID",
		Aliases: []string{"de"},
		Short:   "Delete an Astro Deployment",
		Long:    "Delete an Astro Deployment",
		RunE:    deploymentDelete,
	}
	cmd.Flags().BoolVarP(&forceDelete, "force", "f", false, "Force delete. Don't prompt a user before Deployment deletion")
	cmd.Flags().StringVarP(&deploymentName, "deployment-name", "n", "", "Name of the deployment to delete")
	return cmd
}

func newDeploymentVariableRootCmd(out io.Writer) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "variable",
		Aliases: []string{"var"},
		Short:   "Manage deployment variables",
		Long:    "Manage environment variables for an Astro Deployment. These variables can be used in DAGs or to customize your Airflow environment",
	}
	cmd.AddCommand(
		newDeploymentVariableListCmd(out),
		newDeploymentVariableCreateCmd(out),
		newDeploymentVariableUpdateCmd(out),
	)
	return cmd
}

func newDeploymentVariableListCmd(out io.Writer) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "list",
		Short:   "List a Deployment's variables",
		Long:    "List the keys and values for a Deployment's variables and save them to an environment file",
		Args:    cobra.NoArgs,
		Example: deploymentVariableListExample,
		RunE: func(cmd *cobra.Command, args []string) error {
			return deploymentVariableList(cmd, args, out)
		},
	}
	cmd.Flags().StringVarP(&deploymentID, "deployment-id", "d", "", "deployment to list variables for")
	cmd.Flags().StringVarP(&variableKey, "key", "k", "", "Specify a key to find a specifc variable")
	cmd.Flags().BoolVarP(&useEnvFile, "save", "s", false, "Save deployment variables to an environment file")
	cmd.Flags().StringVarP(&envFile, "env", "e", ".env", "Location of the file to save environment variables to")
	cmd.Flags().StringVarP(&deploymentName, "deployment-name", "n", "", "Name of the deployment to list variables from")

	return cmd
}

//nolint:dupl
func newDeploymentVariableCreateCmd(out io.Writer) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create [key1=val1 key2=val2]",
		Short: "Create Deployment-level environment variables",
		Long:  "Create Deployment-level environment variables by supplying either a key and value or an environment file with a list of keys and values",
		// Args:    cobra.NoArgs,
		Example: deploymentVariableCreateExample,
		RunE: func(cmd *cobra.Command, args []string) error {
			return deploymentVariableCreate(cmd, args, out)
		},
	}
	cmd.Flags().StringVarP(&deploymentID, "deployment-id", "d", "", "Deployment assigned to variables")
	cmd.Flags().StringVarP(&variableKey, "key", "k", "", "Key for the new variable")
	cmd.Flags().StringVarP(&variableValue, "value", "v", "", "Value for the new variable")
	cmd.Flags().BoolVarP(&useEnvFile, "load", "l", false, "Create environment variables loaded from an environment file")
	cmd.Flags().BoolVarP(&makeSecret, "secret", "s", false, "Set the new environment variables as secrets")
	cmd.Flags().StringVarP(&envFile, "env", "e", ".env", "Location of file to load environment variables from")
	_ = cmd.Flags().MarkHidden("key")
	_ = cmd.Flags().MarkHidden("value")
	cmd.Flags().StringVarP(&deploymentName, "deployment-name", "n", "", "Name of the deployment to create variables from")

	return cmd
}

//nolint:dupl
func newDeploymentVariableUpdateCmd(out io.Writer) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "update [key1=update_val1 key2=update_val2]",
		Short:   "Update Deployment-level environment variables",
		Long:    "Update Deployment-level environment variables by supplying either a key and value or an environment file with a list of keys and values, variables that don't already exist will be created",
		Example: deploymentVariableUpdateExample,
		RunE: func(cmd *cobra.Command, args []string) error {
			return deploymentVariableUpdate(cmd, args, out)
		},
	}
	cmd.Flags().StringVarP(&deploymentID, "deployment-id", "d", "", "Deployment assigned to variables")
	cmd.Flags().StringVarP(&variableKey, "key", "k", "", "Key of the variable to update")
	cmd.Flags().StringVarP(&variableValue, "value", "v", "", "Value of the variable to update")
	cmd.Flags().BoolVarP(&useEnvFile, "load", "l", false, "Update environment variables loaded from an environment file")
	cmd.Flags().BoolVarP(&makeSecret, "secret", "s", false, "Set updated environment variables as secrets")
	cmd.Flags().StringVarP(&envFile, "env", "e", ".env", "Location of file to load environment variables to update from")
	_ = cmd.Flags().MarkHidden("key")
	_ = cmd.Flags().MarkHidden("value")
	cmd.Flags().StringVarP(&deploymentName, "deployment-name", "n", "", "Name of the deployment to update varibles from")

	return cmd
}

func deploymentList(cmd *cobra.Command, out io.Writer) error {
	ws, err := coalesceWorkspace()
	if err != nil {
		return errors.Wrap(err, "failed to find a valid workspace")
	}

	// Don't validate workspace if viewing all deployments
	if allDeployments {
		ws = ""
	}

	// Silence Usage as we have now validated command input
	cmd.SilenceUsage = true

	return deployment.List(ws, allDeployments, astroClient, out)
}

func deploymentLogs(cmd *cobra.Command, args []string) error {
	// Get release name from args, if passed
	if len(args) > 0 {
		deploymentID = args[0]
	}

	ws, err := coalesceWorkspace()
	if err != nil {
		return errors.Wrap(err, "failed to find a valid Workspace")
	}

	return deployment.Logs(deploymentID, ws, deploymentName, warnLogs, errorLogs, infoLogs, logCount, astroClient)
}

func deploymentCreate(cmd *cobra.Command, _ []string, out io.Writer) error {
	// Find Workspace ID
	ws, err := coalesceWorkspace()
	if err != nil {
		return errors.Wrap(err, "failed to find a valid Workspace")
	}
	workspaceID = ws

	// Silence Usage as we have now validated command input
	cmd.SilenceUsage = true

	// clean output
	deployment.CleanOutput = cleanOutput

	// set default executor if none was specified
	if executor == "" {
		executor = deployment.CeleryExecutor
	}
	// check if executor is valid
	if !isValidExecutor(executor) {
		return fmt.Errorf("%s is %w", executor, errInvalidExecutor)
	}

	// request is to create from a file
	if inputFile != "" {
		requestedFlags := cmd.Flags().NFlag()
		if requestedFlags > 1 {
			// other flags were requested
			return errFlag
		}

		return fromfile.CreateOrUpdate(inputFile, cmd.Name(), astroClient, out)
	}
	if dagDeploy != "" && !(dagDeploy == enable || dagDeploy == disable) {
		return errors.New("Invalid --dag-deploy value)")
	}

	// Get latest runtime version
	if runtimeVersion == "" {
		airflowVersionClient := airflowversions.NewClient(httpClient, false)
		runtimeVersion, err = airflowversions.GetDefaultImageTag(airflowVersionClient, "")
		if err != nil {
			return err
		}
	}
	// validate cloudProvider
	if cloudProvider != "" {
		if !isValidCloudProvider(astrocore.SharedClusterCloudProvider(cloudProvider)) {
			return fmt.Errorf("%s is %w", cloudProvider, errInvalidCloudProvider)
		}
		if region == "" {
			return errNoRegion
		}
	}
	return deployment.Create(label, workspaceID, description, clusterID, runtimeVersion, dagDeploy, executor, cloudProvider, region, schedulerSize, schedulerAU, schedulerReplicas, astroClient, astroCoreClient, waitForStatus, highAvailability)
}

func deploymentUpdate(cmd *cobra.Command, args []string, out io.Writer) error {
	// Find Workspace ID
	ws, err := coalesceWorkspace()
	if err != nil {
		return errors.Wrap(err, "failed to find a valid workspace")
	}

	// Silence Usage as we have now validated command input
	cmd.SilenceUsage = true

	// clean output
	deployment.CleanOutput = cleanOutput

	// check if executor is valid
	if !isValidExecutor(executor) {
		return fmt.Errorf("%s is %w", executor, errInvalidExecutor)
	}
	// request is to update from a file
	if inputFile != "" {
		requestedFlags := cmd.Flags().NFlag()
		if requestedFlags > 1 {
			// other flags were requested
			return errFlag
		}
		return fromfile.CreateOrUpdate(inputFile, cmd.Name(), astroClient, out)
	}
	if dagDeploy != "" && !(dagDeploy == enable || dagDeploy == disable) {
		return errors.New("Invalid --dag-deploy value)")
	}

	// Get release name from args, if passed
	if len(args) > 0 {
		deploymentID = args[0]
	}

	return deployment.Update(deploymentID, label, ws, description, deploymentName, dagDeploy, executor, updateSchedulerAU, updateSchedulerReplicas, []astro.WorkerQueue{}, forceUpdate, astroClient)
}

func deploymentDelete(cmd *cobra.Command, args []string) error {
	ws, err := coalesceWorkspace()
	if err != nil {
		return errors.Wrap(err, "failed to find a valid workspace")
	}

	// Silence Usage as we have now validated command input
	cmd.SilenceUsage = true

	// Get release name from args, if passed
	if len(args) > 0 {
		deploymentID = args[0]
	}

	return deployment.Delete(deploymentID, ws, deploymentName, forceDelete, astroClient)
}

func deploymentVariableList(cmd *cobra.Command, _ []string, out io.Writer) error {
	ws, err := coalesceWorkspace()
	if err != nil {
		return errors.Wrap(err, "failed to find a valid workspace")
	}

	// Silence Usage as we have now validated command input
	cmd.SilenceUsage = true

	return deployment.VariableList(deploymentID, variableKey, ws, envFile, deploymentName, useEnvFile, astroClient, out)
}

func deploymentVariableCreate(cmd *cobra.Command, args []string, out io.Writer) error {
	ws, err := coalesceWorkspace()
	if err != nil {
		return errors.Wrap(err, "failed to find a valid workspace")
	}

	variableList := args

	// Silence Usage as we have now validated command input
	cmd.SilenceUsage = true

	return deployment.VariableModify(deploymentID, variableKey, variableValue, ws, envFile, deploymentName, variableList, useEnvFile, makeSecret, false, astroClient, out)
}

func deploymentVariableUpdate(cmd *cobra.Command, args []string, out io.Writer) error {
	ws, err := coalesceWorkspace()
	if err != nil {
		return errors.Wrap(err, "failed to find a valid workspace")
	}

	variableList := args

	// Silence Usage as we have now validated command input
	cmd.SilenceUsage = true

	return deployment.VariableModify(deploymentID, variableKey, variableValue, ws, envFile, deploymentName, variableList, useEnvFile, makeSecret, true, astroClient, out)
}

func isValidExecutor(executor string) bool {
	return executor == deployment.KubeExecutor || executor == deployment.CeleryExecutor || executor == ""
}

// isValidCloudProvider returns true for valid CloudProvider values and false if not.
func isValidCloudProvider(cloudProvider astrocore.SharedClusterCloudProvider) bool {
	return cloudProvider == astrocore.SharedClusterCloudProviderGcp
}
