// SPDX-FileCopyrightText: (C) 2024 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package northbound

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/open-edge-platform/app-orch-deployment/app-deployment-manager/pkg/utils/k8serrors"
	"github.com/open-edge-platform/orch-library/go/dazl"
	"github.com/open-edge-platform/orch-library/go/pkg/errors"
	fleetv1alpha1 "github.com/rancher/fleet/pkg/apis/fleet.cattle.io/v1alpha1"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apiserver/pkg/storage/names"
	yaml2 "sigs.k8s.io/yaml"

	deploymentpb "github.com/open-edge-platform/app-orch-deployment/app-deployment-manager/api/nbi/v2/deployment/v1"
	deploymentv1beta1 "github.com/open-edge-platform/app-orch-deployment/app-deployment-manager/api/v1beta1"
	"github.com/open-edge-platform/app-orch-deployment/app-deployment-manager/internal/catalogclient"
	"github.com/open-edge-platform/app-orch-deployment/app-deployment-manager/pkg/utils"
)

var log = dazl.GetPackageLogger()
var matchUIDDeploymentFn = matchUIDDeployment

type Deployment struct {
	Name                       string                                             `yaml:"name"`
	AppName                    string                                             `yaml:"appName"`
	DisplayName                string                                             `yaml:"displayName"`
	AppVersion                 string                                             `yaml:"appVersion"`
	ProfileName                string                                             `yaml:"profileName"`
	DeployID                   string                                             `yaml:"deployId"`
	DeploymentType             string                                             `yaml:"deploymentType"`
	Project                    string                                             `yaml:"project"`
	ValueSecretName            map[string]string                                  `yaml:"valueSecretName"`
	ProfileSecretName          map[string]string                                  `yaml:"profileSecretName"`
	RepoSecretName             map[string]string                                  `yaml:"repoSecretName"`
	ImageRegistrySecretName    map[string]string                                  `yaml:"imageRegistrySecretName"`
	Namespace                  string                                             `yaml:"namespace"`
	HelmApps                   *[]catalogclient.HelmApp                           `yaml:"helmApps"`
	Status                     []*deploymentpb.Deployment_Status                  `yaml:"status"`
	OverrideValues             []*deploymentpb.OverrideValues                     `yaml:"overrideValues"`
	OverrideValuesMasked       []*deploymentpb.OverrideValues                     `yaml:"overrideValuesMasked"`
	TargetClusters             []*deploymentpb.TargetClusters                     `yaml:"targetClusters"`
	AllAppTargetClusters       *deploymentpb.TargetClusters                       `yaml:"allAppTargetClusters"`
	CreateTime                 *timestamppb.Timestamp                             `yaml:"createTime"`
	ForbidsMultipleDeployments bool                                               `yaml:"forbidsMultipleDeployments"`
	RequiredDeploymentPackage  map[string]*deploymentv1beta1.DeploymentPackageRef `yaml:"requiredDeploymentPackage"`
	ChildDeploymentList        map[string]*Deployment                             `yaml:"childDeploymentList"`
	ActiveProjectID            string                                             `yaml:"activeProjectID"`
	ServiceExports             []*deploymentpb.ServiceExport                      `yaml:"serviceExports"`
	NetworkName                string                                             `yaml:"networkName"`
	Namespaces                 []deploymentv1beta1.Namespace                      `yaml:"namespaces"`
	ParameterTemplateSecrets   map[string]string                                  `yaml:"parameterTemplateSecret"`
}

// formatAppNameValidationError creates a standardized error message for app name validation failures.
// It returns a formatted error message indicating that the app name at the specified index is invalid
// due to either being empty or not matching the required regex pattern.
func formatAppNameValidationError(index int) string {
	return fmt.Sprintf("validation error:\n - deployment.override_values[%d].app_name: value length must be at least 1 characters [string.min_len]\n - deployment.override_values[%d].app_name: value does not match regex pattern `^[a-z0-9][a-z0-9-]{0,38}[a-z0-9]{0,1}$` [string.pattern]", index, index)
}

// Sets values to create or update Deployment CR.
func initDeployment(ctx context.Context, s *DeploymentSvc, scenario string, in *deploymentpb.Deployment, dependentDepls map[string]*Deployment, activeProjectID string) (*Deployment, error) {
	d := &Deployment{}

	d.AppName = in.GetAppName()
	d.AppVersion = in.GetAppVersion()
	d.ProfileName = in.GetProfileName()
	d.ServiceExports = in.GetServiceExports()
	d.NetworkName = in.GetNetworkName()
	d.OverrideValues = in.GetOverrideValues()
	d.TargetClusters = in.GetTargetClusters()
	d.AllAppTargetClusters = in.GetAllAppTargetClusters()

	// DeploymentType is optional as input but defaults to auto-scaling if omitted or if input is invalid
	d.DeploymentType = string(deploymentType(in.GetDeploymentType()))

	d.ActiveProjectID = activeProjectID

	if (len(d.TargetClusters) == 0) && (d.AllAppTargetClusters == nil) {
		return d, errors.NewInvalid("missing targetClusters in request")
	}

	for _, val := range d.TargetClusters {
		if val.AppName == "" {
			return d, errors.NewInvalid("missing targetClusters.appName in request")
		}

		if (val.Labels) == nil && val.ClusterId == "" {
			return d, errors.NewInvalid("missing targetClusters.labels or targetClusters.clusterId in request")
		}

		if d.DeploymentType == string(deploymentv1beta1.AutoScaling) && val.Labels == nil {
			return d, errors.NewInvalid("deployment type is auto-scaling but missing targetClusters.labels")
		}

		if d.DeploymentType == string(deploymentv1beta1.Targeted) && val.ClusterId == "" {
			return d, errors.NewInvalid("deployment type is targeted but missing targetClusters.clusterId")
		}

		// Declare labels map if deployment type is targeted.
		if d.DeploymentType == string(deploymentv1beta1.Targeted) {
			val.Labels = make(map[string]string, 0)
		}

		val.Labels[deploymentv1beta1.ClusterOrchKeyProjectID] = d.ActiveProjectID
	}

	if d.AllAppTargetClusters != nil {
		if (d.AllAppTargetClusters.Labels) == nil && d.AllAppTargetClusters.ClusterId == "" {
			return d, errors.NewInvalid("missing allAppTargetClusters.labels or allAppTargetClusters.clusterId in request")
		}

		if d.DeploymentType == string(deploymentv1beta1.AutoScaling) && d.AllAppTargetClusters.Labels == nil {
			return d, errors.NewInvalid("deployment type is auto-scaling but missing allAppTargetClusters.labels")
		}

		if d.DeploymentType == string(deploymentv1beta1.Targeted) && d.AllAppTargetClusters.ClusterId == "" {
			return d, errors.NewInvalid("deployment type is targeted but missing allAppTargetClusters.clusterId")
		}

		// Declare labels map if deployment type is targeted.
		if d.DeploymentType == string(deploymentv1beta1.Targeted) {
			d.AllAppTargetClusters.Labels = make(map[string]string, 0)
		}

		d.AllAppTargetClusters.Labels[deploymentv1beta1.ClusterOrchKeyProjectID] = d.ActiveProjectID
	}

	allOverrideKeys := make(map[string][]string)
	if len(d.OverrideValues) != 0 {
		for i, val := range d.OverrideValues {
			if (val.AppName) == "" {
				return d, errors.NewInvalid(formatAppNameValidationError(i))
			}
			if (val.Values) == nil && val.TargetNamespace == "" {
				return d, errors.NewInvalid("missing overrideValues.targetNamespace or overrideValues.values in request")
			}

			if (val.Values) != nil {
				var emptyValKeys []string
				// Map each override values with app name to enforce mandatory parameter template values
				allOverrideKeys[val.AppName], _ = getAllPbStructKeys(val.Values, emptyValKeys, 0)
			}
		}
	}

	// fixme: remove hardcode and get value from context of JWT
	d.Project = "app.edge-orchestrator.intel.com"

	dp, helmApps, defaultProfileName, err := catalogclient.CatalogLookupDPAndHelmApps(ctx, s.catalogClient,
		d.AppName,
		d.AppVersion,
		d.ProfileName)
	if err != nil {
		return d, errors.NewNotFound("%v", err)
	}

	d.HelmApps = helmApps

	// Validate namespaces
	if len(dp.Namespaces) > 0 {
		nsNameRegex := regexp.MustCompile("^[a-z0-9][a-z0-9-]{0,38}[a-z0-9]{0,1}$")
		for _, ns := range dp.Namespaces {
			if ns.Name == "" {
				return d, errors.NewInvalid("missing namespace name")
			}

			// Check ns name is not default
			if ns.Name == "default" {
				return d, errors.NewInvalid("namespace name \"%s\" is invalid. Namespace name cannot be \"default\"", ns.Name)
			}

			// Check ns name that doesn't have prefix kube-
			checkKindName := strings.Split(ns.Name, "-")
			if len(checkKindName) > 0 && checkKindName[0] == "kind" {
				return d, errors.NewInvalid("namespace name \"%s\" is invalid. Prefix \"kube-\", is reserved for Kubernetes system namespaces", ns.Name)
			}

			// Validate ns name with RFC 1123 label
			if !(nsNameRegex.MatchString(ns.Name)) {
				return d, errors.NewInvalid("namespace name \"%s\" is invalid. Use names that conforms with RFC 1123 label", ns.Name)
			}

			d.Namespaces = append(d.Namespaces, deploymentv1beta1.Namespace{
				Name:        ns.Name,
				Labels:      ns.Labels,
				Annotations: ns.Annotations,
			})
		}
	}

	// Unmask secrets before createSecrets for Update Deployment
	if scenario == "update" {
		// Set deployment ID
		if in.GetDeployId() != "" {
			d.DeployID = in.GetDeployId()
		}

		// Ensuring have a valid deployment name or ID
		if d.Name == "" && d.DeployID != "" {
			deployment, err := matchUIDDeploymentFn(ctx, d.DeployID, activeProjectID, s, metav1.ListOptions{})
			if err != nil {
				log.Warnf("Failed to get deployment by UID: %v", err)
			} else if deployment != nil && deployment.Name != "" {
				d.Name = deployment.Name
				// Set the namespace from the deployment
				d.Namespace = deployment.Namespace
			}
		}

		// If namespace is still empty, set it to activeProjectID
		if d.Namespace == "" {
			d.Namespace = activeProjectID
		}

		// Now process each app with the deployment name
		for _, app := range *d.HelmApps {
			secretName := fmt.Sprintf("%s-%s-%s-secret", d.Name, app.Name, d.ProfileName)

			secretValue, err := utils.GetSecretValue(ctx, s.k8sClient, d.Namespace, secretName)
			if err != nil {
				continue
			}

			// Convert data values back to JSON to unmarshal
			val, err := yaml2.YAMLToJSON(secretValue.Data["values"])
			if err != nil {
				continue
			}

			var valuesStrPb *structpb.Struct
			if err := json.Unmarshal(val, &valuesStrPb); err != nil {
				continue
			}

			for _, oVal := range d.OverrideValues {
				if oVal.AppName == app.Name && oVal.Values != nil && valuesStrPb != nil {
					UnmaskSecrets(oVal.Values, valuesStrPb, "")
				}
			}
		}
	}

	d, err = checkParameterTemplate(d, allOverrideKeys)
	if err != nil {
		return d, err
	}

	// if no profilename was provided, use default profilename from dp
	if d.ProfileName == "" {
		d.ProfileName = defaultProfileName
	}

	d.ForbidsMultipleDeployments = dp.ForbidsMultipleDeployments

	// Set the namespace as the project ID
	d.Namespace = d.ActiveProjectID

	if scenario == "create" {
		d.Name = names.SimpleNameGenerator.GenerateName("deployment-")

		// Create namespace if needed
		err = utils.CreateNamespace(ctx, s.k8sClient, d.Namespace)
		if err != nil {
			return d, err
		}

		if in.GetDisplayName() != "" {
			d.DisplayName = in.GetDisplayName()
			if d.DisplayName != strings.TrimSpace(d.DisplayName) {
				return d, errors.NewInvalid("display-name cannot contain leading or trailing spaces")
			}
		} else {
			d.DisplayName = d.Name
		}
	} else if scenario == "update" {
		d.DisplayName = in.GetDisplayName()
	}

	// Add entries from AllAppTargetClusters into TargetClusters
	if d.AllAppTargetClusters != nil {
		err = mergeAllAppTargetClusters(ctx, d)
		if err != nil {
			return d, errors.Status(err).Err()
		}
	}

	d, err = addDepAndRelationships(ctx, d, s, in, dependentDepls, activeProjectID)
	if err != nil {
		return d, errors.Status(err).Err()
	}

	return d, nil
}

// Checks if label exists within Deployment.
func (c *DeploymentInstance) checkFilter(labelCheck map[string]string) bool {
	foundFilter := false

	// Handle the list of lables, separated by comma.
	for _, labelSet := range c.checkFilters {
		l := strings.Split(labelSet, ",")
		for _, v := range l {
			// kv returns key (k[0]) value (k[1]).
			kv := strings.Split(v, "=")

			// If value missing in the label then continue
			if len(kv) == 1 {
				continue
			}

			// Check if labelCheck keys match the key of z.
			value, exists := labelCheck[kv[0]]

			// First check if key exists.
			if exists {
				// Then if key is present then check if value matches.
				if value == kv[1] {
					foundFilter = true
				}
			}
		}
	}

	return foundFilter
}

// Checks if label is applied to query.
func (c *DeploymentInstance) queryFilter(ctx context.Context, labelSet []string, s *DeploymentSvc) ([]*deploymentpb.Deployment, string) {
	var (
		deploy     *deploymentpb.Deployment
		deployList []*deploymentpb.Deployment
	)

	logFilter := ""
	// If filters are provided in query parameters.
	if len(labelSet) > 0 {
		c.checkFilters = labelSet
		var isFilterFound bool
		for i, deployment := range c.deployments.Items {
			for j, deploymentCluster := range c.deploymentClusters.Items {
				// If matched.
				if string(deployment.ObjectMeta.UID) == deploymentCluster.Labels[string(deploymentv1beta1.DeploymentID)] {
					c.deploymentCluster = &c.deploymentClusters.Items[j]
				}
			}

			c.deployment = &c.deployments.Items[i]
			deploy, isFilterFound = c.createDeploymentObject(ctx, s)
			if isFilterFound {
				deployList = append(deployList, deploy)
			}

			c.deploymentCluster = nil
		}

		logFilter = "labels=" + strings.Join(c.checkFilters, ",")
	} else {
		for i, deployment := range c.deployments.Items {
			for j, deploymentCluster := range c.deploymentClusters.Items {
				// If matched.
				if string(deployment.ObjectMeta.UID) == deploymentCluster.Labels[string(deploymentv1beta1.DeploymentID)] {
					c.deploymentCluster = &c.deploymentClusters.Items[j]
				}
			}

			c.deployment = &c.deployments.Items[i]
			deploy, _ = c.createDeploymentObject(ctx, s)
			deployList = append(deployList, deploy)
			c.deploymentCluster = nil
		}
	}

	return deployList, logFilter
}

// Return the deployment object.
func (c *DeploymentInstance) createDeploymentObject(ctx context.Context, s *DeploymentSvc) (*deploymentpb.Deployment, bool) {
	// Create the TargetClusters list
	targetClustersList := make([]*deploymentpb.TargetClusters, 0)
	for _, app := range c.deployment.Spec.Applications {
		labelCheckList := app.Targets
		for _, l := range labelCheckList {
			if len(c.checkFilters) > 0 {
				foundFilter := c.checkFilter(l)
				if !foundFilter {
					return &deploymentpb.Deployment{}, false
				}
			}

			if _, ok := l[string(deploymentv1beta1.ClusterName)]; ok {
				targetClustersList = append(targetClustersList, &deploymentpb.TargetClusters{
					AppName:   app.Name,
					ClusterId: l[string(deploymentv1beta1.ClusterName)],
					Labels:    l,
				})
			} else {
				targetClustersList = append(targetClustersList, &deploymentpb.TargetClusters{
					AppName: app.Name,
					Labels:  l,
				})
			}
		}
	}

	// Create the OverrideValues list
	var overrideValuesList []*deploymentpb.OverrideValues
	for _, app := range c.deployment.Spec.Applications {
		// App has no override values
		if app.ValueSecretName == "" {
			continue
		}

		secretValue, err := utils.GetSecretValue(ctx, s.k8sClient, c.deployment.ObjectMeta.Namespace, app.ValueSecretName+"-masked")
		if err != nil {
			if apierrors.IsNotFound(err) {
				secretValue, err = utils.GetSecretValue(ctx, s.k8sClient, c.deployment.ObjectMeta.Namespace, app.ValueSecretName)
				if err != nil {
					utils.LogActivity(ctx, "get", "ADM", fmt.Sprintf("cannot get secret %s, error: %v", app.ValueSecretName, err))
					continue
				}
			} else {
				utils.LogActivity(ctx, "get", "ADM", fmt.Sprintf("cannot get secret %s-masked, error: %v", app.ValueSecretName, err))
				continue
			}
		}

		// Convert data values back to JSON to unmarshal
		val, err := yaml2.YAMLToJSON(secretValue.Data["values"])
		if err != nil {
			utils.LogActivity(ctx, "get", "ADM", fmt.Sprintf("cannot convert values to JSON %v", err))
			continue
		}

		var valuesStrPb *structpb.Struct
		_ = json.Unmarshal(val, &valuesStrPb)

		overrideValuesList = append(overrideValuesList, &deploymentpb.OverrideValues{
			AppName:         app.Name,
			TargetNamespace: app.Namespace,
			Values:          valuesStrPb,
		})
	}

	var appList []*deploymentpb.App
	if c.deploymentClusters != nil && len(c.deploymentClusters.Items) != 0 {
		for clIndex := range c.deploymentClusters.Items {
			clusterObj := createDeploymentClusterCr(&c.deploymentClusters.Items[clIndex])
			appList = append(appList, clusterObj.Apps...)
		}
	}

	// Set summary
	summary := &deploymentpb.Summary{
		Total:   utils.ToInt32Clamped(c.deployment.Status.Summary.Total),
		Running: utils.ToInt32Clamped(c.deployment.Status.Summary.Running),
		Down:    utils.ToInt32Clamped(c.deployment.Status.Summary.Down),
		Unknown: utils.ToInt32Clamped(c.deployment.Status.Summary.Unknown),
		Type:    string(deploymentv1beta1.ClusterCounts),
	}

	deployState := c.deployment.Status.State

	// Set status
	status := &deploymentpb.Deployment_Status{
		State:   deploymentState(string(deployState)),
		Message: c.deployment.Status.Message,
		Summary: summary,
	}

	// Convert metav1.Time to protobuf time and return secs
	setPbTime := c.deployment.ObjectMeta.CreationTimestamp.ProtoTime()

	// Return Timestamp from the provided time.Time in unix
	createTimePbUnix := timestamppb.New(time.Unix(setPbTime.Seconds, 0))

	// Append to deployment object
	deployResponse := &deploymentpb.Deployment{
		Name:           c.deployment.ObjectMeta.Name,
		DisplayName:    c.deployment.Spec.DisplayName,
		AppName:        c.deployment.Spec.DeploymentPackageRef.Name,
		AppVersion:     c.deployment.Spec.DeploymentPackageRef.Version,
		ProfileName:    c.deployment.Spec.DeploymentPackageRef.ProfileName,
		DeploymentType: string(c.deployment.Spec.DeploymentType),
		CreateTime:     createTimePbUnix,
		DeployId:       string(c.deployment.ObjectMeta.UID),
		OverrideValues: overrideValuesList,
		TargetClusters: targetClustersList,
		Status:         status,
		Apps:           appList,
	}

	return deployResponse, true
}

// Gets all deployment clusters count status.
func (s *DeploymentSvc) GetDeploymentsStatus(ctx context.Context, in *deploymentpb.GetDeploymentsStatusRequest) (*deploymentpb.GetDeploymentsStatusResponse, error) {
	if err := s.protoValidator.Validate(in); err != nil {
		log.Warnf("%v", err)
		return nil, errors.Status(errors.NewInvalid("%v", err)).Err()
	}

	// RBAC auth
	if err := s.AuthCheckAllowed(ctx, in); err != nil {
		log.Warnf("cannot get status of deployments: %v", err)
		return nil, errors.Status(errors.NewForbidden("cannot get status of deployments: %v", err)).Err()
	}

	activeProjectIDKey := string(deploymentv1beta1.AppOrchActiveProjectID)

	activeProjectID, err := s.GetActiveProjectID(ctx)
	if err != nil {
		msg := fmt.Sprintf("failed to get tenant project ID %s", err.Error())
		return nil, errors.Status(errors.NewUnavailable(msg)).Err()
	}

	labelSelector := metav1.LabelSelector{
		MatchLabels: map[string]string{activeProjectIDKey: activeProjectID},
	}

	listOpts := metav1.ListOptions{
		LabelSelector: labels.Set(labelSelector.MatchLabels).String(),
	}

	namespace := activeProjectID

	deployments, err := s.crClient.Deployments(namespace).List(ctx, listOpts)
	if err != nil {
		log.Warnf("cannot get status of deployments: %v", err)
		return nil, errors.Status(k8serrors.K8sToTypedError(err)).Err()
	}

	// fixme: when deployment is in errored, etc, then no deploymentcluster CR is created or it's deleted
	// need to figure out how to account for the missing deploymentcluster CR
	// Per deploymentCluster Controller
	// Check if any map to the DeploymentCluster in the request
	// If so, create a new DeploymentCluster for this BundleDeployment
	// delete CR when dc.Status.Status.Summary.Total == 0 // Delete this DeploymentCluster since it has no Apps
	deploymentClusters, err := s.crClient.DeploymentClusters("").List(ctx, listOpts)
	if err != nil {
		log.Warnf("cannot get status of deployments: %v", err)
		return nil, errors.Status(k8serrors.K8sToTypedError(err)).Err()
	}

	// Filter out orphaned DeploymentClusters to prevent status query errors
	var validDeploymentClusters []deploymentv1beta1.DeploymentCluster
	deploymentUIDs := make(map[string]bool)
	for _, depl := range deployments.Items {
		deploymentUIDs[string(depl.ObjectMeta.UID)] = true
	}
	for _, dc := range deploymentClusters.Items {
		deploymentID := dc.Labels[string(deploymentv1beta1.DeploymentID)]
		if deploymentUIDs[deploymentID] {
			validDeploymentClusters = append(validDeploymentClusters, dc)
		} else {
			log.Warnf("GetDeploymentsStatus: skipping orphaned DeploymentCluster %s with deployment ID %s", dc.Name, deploymentID)
		}
	}
	// Replace orphaned list with valid list
	deploymentClusters.Items = validDeploymentClusters

	var c DeploymentInstance
	c.deployments = deployments
	c.deploymentClusters = deploymentClusters
	deployList, logFilter := c.queryFilter(ctx, in.Labels, s)

	var (
		total       int32
		running     int32
		down        int32
		deploying   int32
		updating    int32
		terminating int32
		unknown     int32
		errored     int32
	)

	// Loop for thru all deployments and capture total cluster count
	for index := range deployList {
		status := deployList[index].Status

		// Total number of all deployment clusters
		total = total + status.Summary.Total
		if total < 0 {
			total = 0
		}

		// LPOD-1965: Expand the states reported in the cluster summary counts, as required by LP GUI
		// These summary counts don't really make much sense in the context of ADM.
		// The intention of LPOD-1965 seems to be to count *clusters* by state DEPLOYING, ERROR, etc.,
		// but ADM considers clusters as either RUNNING or DOWN or UNKNOWN; other states like
		// DEPLOYING and ERROR are associated with Deployments, not clusters.  Since these
		// expanded cluster states don't fit into ADM's view of the world, they are calculated
		// in the API rather than reported by the ADM controller.
		//
		// The approach is to use the state of each Deployment to map its clusters to the appropriate
		// categories based on the table below:
		//
		// Deployment status		Status of Deployment's clusters
		// -----------------		-------------------------------
		// ERROR					All clusters are counted as ERROR
		// TERMINATING				All clusters are counted as TERMINATING
		// RUNNING					All clusters are (should be) counted as RUNNING
		// DEPLOYING				Each cluster is counted as RUNNING or DEPLOYING
		// UPDATING					Each cluster is counted as RUNNING or UPDATING
		// DOWN				Each cluster is counted as RUNNING or DOWN
		// UNKNOWN				    Each cluster is counted as UNKNOWN

		switch status.State {
		case deploymentpb.State_ERROR:
			errored += status.Summary.Total
			if errored < 0 {
				errored = 0
			}
		case deploymentpb.State_TERMINATING:
			terminating += status.Summary.Total
			if terminating < 0 {
				terminating = 0
			}
		default:
			running += status.Summary.Running
			if running < 0 {
				running = 0
			}
			switch status.State {
			case deploymentpb.State_DEPLOYING:
				deploying += status.Summary.Down
				if deploying < 0 {
					deploying = 0
				}
			case deploymentpb.State_UPDATING:
				updating += status.Summary.Down
				if updating < 0 {
					updating = 0
				}
			case deploymentpb.State_UNKNOWN:
				unknown += status.Summary.Unknown
				if unknown < 0 {
					unknown = 0
				}
			default:
				down += status.Summary.Down
				if down < 0 {
					down = 0
				}
			}
		}
	}

	utils.LogActivity(ctx, "get deployment status", "ADM", logFilter)
	return &deploymentpb.GetDeploymentsStatusResponse{
		Total:       total,
		Running:     running,
		Down:        down,
		Deploying:   deploying,
		Updating:    updating,
		Terminating: terminating,
		Unknown:     unknown,
		Error:       errored,
	}, nil
}

// Create a new deployment object and returns the new object.
func (s *DeploymentSvc) CreateDeployment(ctx context.Context, in *deploymentpb.CreateDeploymentRequest) (*deploymentpb.CreateDeploymentResponse, error) {
	// Handle concurrency of shared resources to allow parallel deployment creation requests with Mutex.
	// This can prevent race conditions, creation of deployments with same deployId, data corruption and
	// unpredictable program behaviour. Also prevents DoS on deployments created by other users, affecting availability.
	s.apiMutex.Lock()
	defer s.apiMutex.Unlock()

	if in == nil || in.Deployment == nil {
		log.Warnf("incomplete request")
		return nil, errors.Status(errors.NewInvalid("incomplete request")).Err()
	}

	if err := s.protoValidator.Validate(in); err != nil {
		log.Warnf("%v", err)
		return nil, errors.Status(errors.NewInvalid("%v", err)).Err()
	}

	// RBAC auth
	if err := s.AuthCheckAllowed(ctx, in); err != nil {
		log.Warnf("cannot create deployment: %v", err)
		return nil, errors.Status(errors.NewForbidden("cannot create deployment: %v", err)).Err()
	}
	activeProjectIDKey := string(deploymentv1beta1.AppOrchActiveProjectID)

	activeProjectID, err := s.GetActiveProjectID(ctx)
	if err != nil {
		msg := fmt.Sprintf("failed to get tenant project ID %s", err.Error())
		return nil, errors.Status(errors.NewUnavailable(msg)).Err()
	}

	labelSelector := metav1.LabelSelector{
		MatchLabels: map[string]string{activeProjectIDKey: activeProjectID},
	}

	listOpts := metav1.ListOptions{
		LabelSelector: labels.Set(labelSelector.MatchLabels).String(),
	}

	ctx, cancel, err := utils.AddToOutgoingContext(ctx, s.vaultAuthClient, activeProjectID, true)
	if err != nil {
		log.Warnf("cannot create deployment: %v", err)
		return nil, errors.Status(err).Err()
	}
	defer cancel()

	dependentDepls := make(map[string]*Deployment)
	d, err := initDeployment(ctx, s, "create", in.GetDeployment(), dependentDepls, activeProjectID)
	if err != nil {
		log.Warnf("cannot create deployment: %v", err)
		return nil, errors.Status(err).Err()
	}

	err = createGitClientSecret(ctx, s.k8sClient, d)
	if err != nil {
		log.Warnf("cannot create deployment: %v", err)
		return nil, errors.Status(err).Err()
	}

	// for dependents
	for k, v := range dependentDepls {
		var dependentDeploymentCR *deploymentv1beta1.Deployment
		if v.DeployID != "" {
			// case 1: need to update CR - adding dependency and targetCluster
			dependentDeploymentCR, err = s.crClient.Deployments(d.Namespace).Get(ctx, v.Name, metav1.GetOptions{})
			if err != nil {
				msg := fmt.Sprintf("failed to get pre-created dependent deployment %v (forbidsMultipleDeployment: %v): %v", d.Name, v.ForbidsMultipleDeployments, err)
				log.Warnf(msg)
				return nil, errors.Status(k8serrors.K8sToTypedError(err)).Err()
			}

			// fill out child deployment list in Deployment CR spec
			addRelationshipToDeploymentCR(dependentDepls[k], dependentDeploymentCR)

			// update new TargetCluster to children Deployments if children Deployments have forbidMultipleDeployments:true
			addTargetClusterEntry(v.TargetClusters, dependentDeploymentCR, activeProjectID)
			_, err = s.crClient.Deployments(d.Namespace).Update(ctx, dependentDeploymentCR.Name, dependentDeploymentCR, metav1.UpdateOptions{})
			if err != nil {
				msg := fmt.Sprintf("failed to get pre-created deployment %v (forbidsMultipleDeployment: %v): %v", d.Name, d.ForbidsMultipleDeployments, err)
				log.Warnf(msg)
				return nil, errors.Status(k8serrors.K8sToTypedError(err)).Err()
			}
		} else {
			// case 2: need to create CR
			dependentDepls[k], err = createSecrets(ctx, s.k8sClient, dependentDepls[k])
			if err != nil {
				log.Warnf("cannot create deployment: %v", err)
				return nil, errors.Status(err).Err()
			}

			depCreateInstance, err := createDeploymentCr(dependentDepls[k], "create", "")
			if err != nil {
				log.Warnf("cannot create dependent deployment: %v", err)
				return nil, errors.Status(err).Err()
			}

			// fill out child deployment list in Deployment CR spec
			addRelationshipToDeploymentCR(dependentDepls[k], depCreateInstance)

			dependentDeploymentCR, err = s.crClient.Deployments(d.Namespace).Create(ctx, depCreateInstance, metav1.CreateOptions{})
			if err != nil {
				log.Warnf("cannot create dependent deployment %v: %v", v.Name, err)
				return nil, errors.Status(k8serrors.K8sToTypedError(err)).Err()
			}
			dependentDepls[k].DeployID = string(dependentDeploymentCR.ObjectMeta.UID)

			// Need to update Owner Reference to clean up. Cannot add required details during secret creation
			// since deployment UID is required.
			dependentDepls[k], err = updateOwnerRefSecrets(ctx, s.k8sClient, dependentDepls[k])
			if err != nil {
				log.Warnf("cannot create deployment: %v", err)
				return nil, errors.Status(err).Err()
			}
		}
	}

	// for root
	var deploymentCR *deploymentv1beta1.Deployment
	// case 2: need to create CR

	// fetch deploymentCR for this deployment package
	deplCRs, err := getDeploymentCRWithDeploymentPackage(ctx, s, d.Namespace, catalogclient.GetDeploymentPackageID(d.AppName, d.AppVersion, d.ProfileName), listOpts)
	if err != nil {
		log.Warnf("error when getting existing deployments - err %v", err)
		return nil, errors.Status(err).Err()
	}
	if len(deplCRs) > 0 {
		// find deploymentCR with same displayName as new create request
		for idx := range deplCRs {
			if d.DisplayName == deplCRs[idx].Spec.DisplayName {
				// If deploymentCR exists with same displayName as new create request
				// then return failure else continue
				msg := fmt.Sprintf("Duplicate deployment exists with same name %v, deployID: %v", d.DisplayName, string(deplCRs[idx].ObjectMeta.UID))
				log.Warnf(msg)
				return nil, errors.Status(errors.NewAlreadyExists(msg)).Err()
			}
		}
	}

	d, err = createSecrets(ctx, s.k8sClient, d)
	if err != nil {
		log.Warnf("cannot create deployment: %v", err)
		return nil, errors.Status(err).Err()
	}

	createInstance, err := createDeploymentCr(d, "create", "")
	if err != nil {
		log.Warnf("cannot create deployment: %v", err)
		return nil, errors.Status(err).Err()
	}

	// fill out child deployment list in Deployment CR spec
	addRelationshipToDeploymentCR(d, createInstance)

	deploymentCR, err = s.crClient.Deployments(d.Namespace).Create(ctx, createInstance, metav1.CreateOptions{})
	if err != nil {
		log.Warnf("cannot create deployment %v: %v", d.Name, err)
		return nil, errors.Status(k8serrors.K8sToTypedError(err)).Err()
	}
	d.DeployID = string(deploymentCR.ObjectMeta.UID)

	// Need to update Owner Reference to clean up. Cannot add required details during secret creation
	// since deployment UID is required.
	d, err = updateOwnerRefSecrets(ctx, s.k8sClient, d)
	if err != nil {
		log.Warnf("cannot create deployment: %v", err)
		return nil, errors.Status(err).Err()
	}

	utils.LogActivity(ctx, "create", "ADM", "deployment-name "+d.Name, "deploy-id "+d.DeployID, "deployment-app-version "+d.AppVersion)
	return &deploymentpb.CreateDeploymentResponse{DeploymentId: d.DeployID}, nil
}

// Gets a deployment object and returns the object.
func (s *DeploymentSvc) GetDeployment(ctx context.Context, in *deploymentpb.GetDeploymentRequest) (*deploymentpb.GetDeploymentResponse, error) {
	if in == nil || in.DeplId == "" {
		log.Warnf("incomplete request")
		return nil, errors.Status(errors.NewInvalid("incomplete request")).Err()
	}

	if err := s.protoValidator.Validate(in); err != nil {
		log.Warnf("%v", err)
		return nil, errors.Status(errors.NewInvalid("%v", err)).Err()
	}

	// RBAC auth
	if err := s.AuthCheckAllowed(ctx, in); err != nil {
		log.Warnf("cannot get deployment: %v", err)
		return nil, errors.Status(errors.NewForbidden("cannot get deployment: %v", err)).Err()
	}

	UID := in.DeplId
	activeProjectIDKey := string(deploymentv1beta1.AppOrchActiveProjectID)

	activeProjectID, err := s.GetActiveProjectID(ctx)
	if err != nil {
		msg := fmt.Sprintf("failed to get tenant project ID %s", err.Error())
		return nil, errors.Status(errors.NewUnavailable(msg)).Err()
	}

	labelSelector := metav1.LabelSelector{
		MatchLabels: map[string]string{activeProjectIDKey: activeProjectID},
	}

	listOpts := metav1.ListOptions{
		LabelSelector: labels.Set(labelSelector.MatchLabels).String(),
	}

	namespace := activeProjectID

	deployment, err := matchUIDDeployment(ctx, UID, namespace, s, listOpts)
	if err != nil {
		log.Warnf("cannot get deployment: %v", err)
		return nil, errors.Status(err).Err()
	} else if deployment.ObjectMeta.Name == "" {
		log.Warnf("deployment id %v not found", UID)
		return nil, errors.Status(errors.NewNotFound("deployment id %v not found", UID)).Err()
	}

	name := deployment.ObjectMeta.Name
	var c DeploymentInstance

	labelSelector = metav1.LabelSelector{
		MatchLabels: map[string]string{string(deploymentv1beta1.DeploymentID): UID},
	}

	labelSelector.MatchLabels[string(deploymentv1beta1.AppOrchActiveProjectID)] = activeProjectID

	listOpts = metav1.ListOptions{
		LabelSelector: labels.Set(labelSelector.MatchLabels).String(),
	}

	// List all deploymentCluster objects and match the deployment-id of deployment
	// deploymentCluster objects are in same namespace of bundledeployment and deploymentCluster object name
	// is generated based on bundledeployment id. NB would need to either parse through bundledeployment object
	// and somehow map CRs to get information. Or deploymentCluster objects can live in same NS as all ADM CRs
	// future improvement since load will be heavy when looping thru 500+ edgenodes and mapping all ADM CRs with deployments
	deploymentClusters, err := s.crClient.DeploymentClusters("").List(ctx, listOpts)
	if err != nil {
		log.Warnf("cannot get deployment cluster: %v", err)
		return nil, errors.Status(k8serrors.K8sToTypedError(err)).Err()
	}

	// Filter out orphaned DeploymentClusters to prevent get deployment errors
	var validDeploymentClusters []deploymentv1beta1.DeploymentCluster
	for _, dc := range deploymentClusters.Items {
		deploymentID := dc.Labels[string(deploymentv1beta1.DeploymentID)]
		if deploymentID == UID {
			validDeploymentClusters = append(validDeploymentClusters, dc)
		} else {
			log.Warnf("GetDeployment: skipping DeploymentCluster %s with mismatched deployment ID %s (expected %s)", dc.Name, deploymentID, UID)
		}
	}
	// Replace list with filtered list
	deploymentClusters.Items = validDeploymentClusters

	c.deploymentClusters = deploymentClusters
	c.deployment = deployment
	deployResponse, _ := c.createDeploymentObject(ctx, s)

	utils.LogActivity(ctx, "list", "ADM", "deployment name "+name, "deploy id "+UID, "deployment app version "+deployResponse.AppVersion)
	return &deploymentpb.GetDeploymentResponse{
		Deployment: deployResponse,
	}, nil
}

func (s *DeploymentSvc) ListDeploymentsPerCluster(ctx context.Context, in *deploymentpb.ListDeploymentsPerClusterRequest) (*deploymentpb.ListDeploymentsPerClusterResponse, error) {
	if err := s.protoValidator.Validate(in); err != nil {
		log.Warnf("%v", err)
		return nil, errors.Status(errors.NewInvalid("%v", err)).Err()
	}

	// RBAC auth
	if err := s.AuthCheckAllowed(ctx, in); err != nil {
		log.Warnf("cannot list deployments: %v", err)
		return nil, errors.Status(errors.NewForbidden("cannot list deployments: %v", err)).Err()
	}

	if in == nil || in.ClusterId == "" {
		log.Warnf("incomplete request - cluster ID is missing")
		return nil, errors.Status(errors.NewInvalid("incomplete request - cluster ID is missing")).Err()
	}

	activeProjectID, err := s.GetActiveProjectID(ctx)
	if err != nil {
		msg := fmt.Sprintf("failed to get tenant project ID %s", err.Error())
		return nil, errors.Status(errors.NewUnavailable(msg)).Err()
	}

	namespace := activeProjectID

	// Filter deployments with only project id
	labelSelector := metav1.LabelSelector{
		MatchLabels: map[string]string{string(deploymentv1beta1.AppOrchActiveProjectID): activeProjectID},
	}

	listOpts := metav1.ListOptions{
		LabelSelector: labels.Set(labelSelector.MatchLabels).String(),
	}

	deployments, err := s.crClient.Deployments(namespace).List(ctx, listOpts)
	if err != nil {
		msg := fmt.Sprintf("failed to get deployment list %s", err.Error())
		return nil, errors.Status(errors.NewUnavailable(msg)).Err()
	}

	deploymentMap := map[string]*deploymentv1beta1.Deployment{}
	if deployments != nil && len(deployments.Items) > 0 {
		for i, deployment := range deployments.Items {
			deploymentMap[string(deployment.UID)] = &deployments.Items[i]
		}
	}

	// Filter deploymentclusters with project id and cluster name
	labelSelector.MatchLabels[string(deploymentv1beta1.ClusterName)] = in.ClusterId

	listOpts = metav1.ListOptions{
		LabelSelector: labels.Set(labelSelector.MatchLabels).String(),
	}

	deploymentClusters, err := s.crClient.DeploymentClusters("").List(ctx, listOpts)
	if err != nil {
		log.Warnf("cannot get deployment cluster: %v", err)
		return nil, errors.Status(k8serrors.K8sToTypedError(err)).Err()
	}

	deploymentList := make([]*deploymentpb.DeploymentInstancesCluster, 0)

	if deploymentClusters != nil && len(deploymentClusters.Items) > 0 {
		for _, dc := range deploymentClusters.Items {
			if dc.Spec.DeploymentID == "" || dc.Spec.ClusterID == "" {
				// skip weird CR
				continue
			}
			deplUID := dc.Spec.DeploymentID

			if _, ok := deploymentMap[deplUID]; !ok {
				// Skip orphaned DeploymentCluster - parent Deployment not found
				// The DeploymentCluster controller will clean this up in the background
				log.Infof("Skipping orphaned DeploymentCluster %s - parent deployment %s not found", dc.Name, deplUID)
				continue
			}

			deplName := deploymentMap[deplUID].Name
			deplDisplayName := deploymentMap[deplUID].Spec.DisplayName
			deploymentList = append(deploymentList, createDeploymentInstanceCluster(&dc, deplUID, deplName, deplDisplayName))
		}
	}

	totalNumDeployments := len(deploymentList)

	resp := &deploymentpb.ListDeploymentsPerClusterResponse{
		DeploymentInstancesCluster: deploymentList,
		TotalElements:              utils.ToInt32Clamped(totalNumDeployments),
	}

	return resp, nil
}

func createDeploymentInstanceCluster(dc *deploymentv1beta1.DeploymentCluster, deploymentUID, deploymentName, deploymentDisplayName string) *deploymentpb.DeploymentInstancesCluster {
	appList := make([]*deploymentpb.App, 0)
	if dc != nil && len(dc.Status.Apps) != 0 {
		appList = make([]*deploymentpb.App, len(dc.Status.Apps))
		for i, app := range dc.Status.Apps {
			// Create app summary
			appSummary := &deploymentpb.Summary{
				Total:   utils.ToInt32Clamped(app.Status.Summary.Total),
				Running: utils.ToInt32Clamped(app.Status.Summary.Running),
				Down:    utils.ToInt32Clamped(app.Status.Summary.Down),
				Unknown: utils.ToInt32Clamped(app.Status.Summary.Unknown),
				Type:    string(app.Status.Summary.Type),
			}

			appState := dc.Status.Apps[i].Status.State

			// Create app status
			appStatus := &deploymentpb.Deployment_Status{
				State:   deploymentState(string(appState)),
				Message: app.Status.Message,
				Summary: appSummary,
			}

			// Append to app list
			appList[i] = &deploymentpb.App{
				Id:     app.Id,
				Name:   app.Name,
				Status: appStatus,
			}
		}
	}

	summary := &deploymentpb.Summary{
		Total:   utils.ToInt32Clamped(dc.Status.Status.Summary.Total),
		Running: utils.ToInt32Clamped(dc.Status.Status.Summary.Running),
		Down:    utils.ToInt32Clamped(dc.Status.Status.Summary.Down),
		Unknown: utils.ToInt32Clamped(dc.Status.Status.Summary.Unknown),
		Type:    string(dc.Status.Status.Summary.Type),
	}

	deployState := dc.Status.Status.State

	status := &deploymentpb.Deployment_Status{
		State:   deploymentState(string(deployState)),
		Message: dc.Status.Status.Message,
		Summary: summary,
	}

	deploymentInstanceCluster := &deploymentpb.DeploymentInstancesCluster{
		DeploymentUid:         deploymentUID,
		DeploymentName:        deploymentName,
		DeploymentDisplayName: deploymentDisplayName,
		Status:                status,
		Apps:                  appList,
	}

	return deploymentInstanceCluster
}

// Gets a list of all deployment objects and returns all the objects.
func (s *DeploymentSvc) ListDeployments(ctx context.Context, in *deploymentpb.ListDeploymentsRequest) (*deploymentpb.ListDeploymentsResponse, error) {
	if err := s.protoValidator.Validate(in); err != nil {
		log.Warnf("%v", err)
		return nil, errors.Status(errors.NewInvalid("%v", err)).Err()
	}

	// RBAC auth
	if err := s.AuthCheckAllowed(ctx, in); err != nil {
		log.Warnf("cannot list deployments: %v", err)
		return nil, errors.Status(errors.NewForbidden("cannot list deployments: %v", err)).Err()
	}

	activeProjectIDKey := string(deploymentv1beta1.AppOrchActiveProjectID)

	activeProjectID, err := s.GetActiveProjectID(ctx)
	if err != nil {
		msg := fmt.Sprintf("failed to get tenant project ID %s", err.Error())
		return nil, errors.Status(errors.NewUnavailable(msg)).Err()
	}

	labelSelector := metav1.LabelSelector{
		MatchLabels: map[string]string{activeProjectIDKey: activeProjectID},
	}

	listOpts := metav1.ListOptions{
		LabelSelector: labels.Set(labelSelector.MatchLabels).String(),
	}

	namespace := activeProjectID
	deployments, err := s.crClient.Deployments(namespace).List(ctx, listOpts)
	if err != nil {
		log.Warnf("cannot list deployments: %v", err)
		return nil, errors.Status(k8serrors.K8sToTypedError(err)).Err()
	}

	// NEX-5503 Don't return DeploymentClusters as part of ListDeployments
	deploymentClusters := &deploymentv1beta1.DeploymentClusterList{}

	var c DeploymentInstance
	c.deployments = deployments
	c.deploymentClusters = deploymentClusters
	deployList, logFilter := c.queryFilter(ctx, in.Labels, s)

	totalNumDeployments := len(deployList)

	// Paginate, sort, and filter list of deployments
	selectedDeployments, err := selectDeployments(in, deployList)
	if err != nil {
		log.Warnf("cannot list deployments: %v", err)
		return nil, errors.Status(err).Err()
	}

	utils.LogActivity(ctx, "list", "ADM", "Filter"+logFilter, "Total-Deployments: "+strconv.Itoa(totalNumDeployments))
	return &deploymentpb.ListDeploymentsResponse{
		Deployments:   selectedDeployments,
		TotalElements: utils.ToInt32Clamped(totalNumDeployments),
	}, nil
}

// Deletes a deployment object and nothing is returned.
func (s *DeploymentSvc) DeleteDeployment(ctx context.Context, in *deploymentpb.DeleteDeploymentRequest) (*emptypb.Empty, error) {
	// Handle concurrency of shared resources to allow parallel deployment deletion requests with Mutex.
	// This can prevent race conditions, deletion of deployments, data corruption and unpredictable
	// program behaviour. Also prevents DoS on deployments being deleted by other users, affecting availability.
	s.apiMutex.Lock()
	defer s.apiMutex.Unlock()

	if in == nil || in.DeplId == "" {
		log.Warnf("incomplete request")
		return nil, errors.Status(errors.NewInvalid("incomplete request")).Err()
	}

	if err := s.protoValidator.Validate(in); err != nil {
		log.Warnf("%v", err)
		return nil, errors.Status(errors.NewInvalid("%v", err)).Err()
	}

	// RBAC auth
	if err := s.AuthCheckAllowed(ctx, in); err != nil {
		log.Warnf("cannot delete deployment: %v", err)
		return nil, errors.Status(errors.NewForbidden("cannot delete deployment: %v", err)).Err()
	}

	d := &Deployment{}

	d.DeployID = in.DeplId

	activeProjectIDKey := string(deploymentv1beta1.AppOrchActiveProjectID)

	activeProjectID, err := s.GetActiveProjectID(ctx)
	if err != nil {
		msg := fmt.Sprintf("failed to get tenant project ID %s", err.Error())
		return nil, errors.Status(errors.NewUnavailable(msg)).Err()
	}

	labelSelector := metav1.LabelSelector{
		MatchLabels: map[string]string{activeProjectIDKey: activeProjectID},
	}

	listOpts := metav1.ListOptions{
		LabelSelector: labels.Set(labelSelector.MatchLabels).String(),
	}

	d.Namespace = activeProjectID

	deployment, err := matchUIDDeployment(ctx, d.DeployID, d.Namespace, s, listOpts)
	if err != nil {
		log.Warnf("cannot delete deployment: %v", err)
		return nil, errors.Status(err).Err()
	} else if deployment.ObjectMeta.Name == "" {
		log.Warnf("deployment %v not found while deleting deployment", d.DeployID)
		return nil, errors.Status(errors.NewNotFound("deployment %v not found while deleting deployment", d.DeployID)).Err()
	}

	d.Name = deployment.ObjectMeta.Name

	// dependency support
	// case 1. if there is any parent, do not delete it
	// case 2. if there is no parent, and delete type is PARENT_ONLY
	// case 3. if there is no parent and delete type is ALL
	if len(deployment.Status.ParentDeploymentList) > 0 {
		// case 1
		// must not delete this deployment regardless of delete type
		return nil, errors.Status(errors.NewAborted("cannot delete deployment %v - %d parent deployment(s) is running",
			d.Name, len(deployment.Status.ParentDeploymentList))).Err()
	}

	if in.DeleteType == deploymentpb.DeleteType_PARENT_ONLY {
		// case 2
		// delete this Deployment
		err = s.crClient.Deployments(d.Namespace).Delete(ctx, d.Name, metav1.DeleteOptions{})
		if err != nil {
			log.Warnf("cannot delete deployment: %v", err)
			return nil, errors.Status(k8serrors.K8sToTypedError(err)).Err()
		}

		err = deleteSecrets(ctx, s.k8sClient, deployment)
		if err != nil {
			utils.LogActivity(ctx, "delete", "ADM", fmt.Sprintf("%v", err))
		}

		utils.LogActivity(ctx, "delete", "ADM", "deployment-name "+d.Name, "deploy-id "+d.DeployID, "delete-type "+in.DeleteType.String())
	} else {
		// case 3
		labelSelector = metav1.LabelSelector{
			MatchLabels: map[string]string{activeProjectIDKey: activeProjectID},
		}

		listOpts = metav1.ListOptions{
			LabelSelector: labels.Set(labelSelector.MatchLabels).String(),
		}

		// do dependency graph traversal, collect all target deployments, verify if there is any parent deployment CR outside of dependency graph
		allDeployments, err := s.crClient.Deployments(d.Namespace).List(ctx, listOpts)
		if err != nil {
			msg := fmt.Sprintf("failed to get list of deployment CRs to delete deploment %v (deleteType: %v): err %v",
				d.Name, in.DeleteType.String(), err)
			log.Warnf(msg)
			return nil, errors.Status(k8serrors.K8sToTypedError(err)).Err()
		}

		// to enhance time complexity, convert list to map data structure
		allDeploymentMap := make(map[string]*deploymentv1beta1.Deployment)
		for idx, depl := range allDeployments.Items {
			allDeploymentMap[depl.Name] = &allDeployments.Items[idx]
		}

		targetList := make(map[string]*deploymentv1beta1.Deployment)
		err = getTargetDeploymentsForDeleteDeployment(d.Name, targetList, allDeploymentMap, 0)
		if err != nil {
			msg := fmt.Sprintf("failed to get target deployment CRs to delete deployment %v (deleteType: %v): err %v",
				d.Name, in.DeleteType.String(), err)
			log.Warnf(msg)
			return nil, errors.Status(errors.NewNotFound(msg)).Err()
		}

		if !validateTargetDeployments(targetList) {
			msg := fmt.Sprintf("validation failed to delete deployment %v (deleteType: %v) - one of target deployment CRs has a parent deployment CR outside of dependency graph", d.Name, in.DeleteType)
			log.Warnf(msg)
			return nil, errors.Status(errors.NewNotFound(msg)).Err()
		}

		for k := range targetList {
			err = s.crClient.Deployments(d.Namespace).Delete(ctx, k, metav1.DeleteOptions{})
			if err != nil {
				log.Warnf("cannot delete deployment: %v", err)
				return nil, errors.Status(k8serrors.K8sToTypedError(err)).Err()
			}
		}
	}

	return &emptypb.Empty{}, nil
}

// Updates a Deployment object and returns the new object.
// Deployment Name is required to update and cannot be changed.
func (s *DeploymentSvc) UpdateDeployment(ctx context.Context, in *deploymentpb.UpdateDeploymentRequest) (*deploymentpb.UpdateDeploymentResponse, error) {
	// Handle concurrency of shared resources to allow parallel deployment update requests with Mutex.
	// This can prevent race conditions, updating secrets of deployments, data corruption and
	// unpredictable program behaviour. Also prevents DoS on deployments updated by other users, affecting availability.
	s.apiMutex.Lock()
	defer s.apiMutex.Unlock()

	if in == nil || in.Deployment == nil || in.DeplId == "" {
		log.Warnf("incomplete request")
		return nil, errors.Status(errors.NewInvalid("incomplete request")).Err()
	}

	if err := s.protoValidator.Validate(in); err != nil {
		log.Warnf("%v", err)
		return nil, errors.Status(errors.NewInvalid("%v", err)).Err()
	}

	// RBAC auth
	if err := s.AuthCheckAllowed(ctx, in); err != nil {
		log.Warnf("cannot update deployment: %v", err)
		return nil, errors.Status(errors.NewForbidden("cannot update deployment: %v", err)).Err()
	}

	activeProjectIDKey := string(deploymentv1beta1.AppOrchActiveProjectID)
	activeProjectID, err := s.GetActiveProjectID(ctx)
	if err != nil {
		msg := fmt.Sprintf("failed to get tenant project ID %s", err.Error())
		return nil, errors.Status(errors.NewUnavailable(msg)).Err()
	}

	labelSelector := metav1.LabelSelector{
		MatchLabels: map[string]string{activeProjectIDKey: activeProjectID},
	}

	listOpts := metav1.ListOptions{
		LabelSelector: labels.Set(labelSelector.MatchLabels).String(),
	}

	ctx, cancel, err := utils.AddToOutgoingContext(ctx, s.vaultAuthClient, activeProjectID, true)
	if err != nil {
		log.Warnf("cannot update deployment: %v", err)
		return nil, errors.Status(err).Err()
	}
	defer cancel()

	// Explicitly set the deployment ID in the Deployment object
	if in.Deployment != nil {
		in.Deployment.DeployId = in.DeplId
	}

	dependentDepls := make(map[string]*Deployment)
	d, err := initDeployment(ctx, s, "update", in.GetDeployment(), dependentDepls, activeProjectID)
	if err != nil {
		log.Warnf("cannot update deployment: %v", err)
		return nil, errors.Status(err).Err()
	}

	// Ensure the deployment ID is set from the request
	d.DeployID = in.DeplId

	deployment, err := matchUIDDeployment(ctx, d.DeployID, d.Namespace, s, listOpts)
	if err != nil {
		log.Warnf("cannot update deployment: %v", err)
		return nil, errors.Status(err).Err()
	} else if deployment.ObjectMeta.Name == "" {
		log.Warnf(fmt.Sprintf("deployment %s not found while updating deployment", d.DeployID))
		return nil, errors.Status(errors.NewNotFound("deployment %s not found while updating deployment", d.DeployID)).Err()
	} else if deployment.ObjectMeta.ResourceVersion == "" {
		log.Warnf("resourceVersion not found while updating deployment")
		return nil, errors.Status(errors.NewNotFound("resourceVersion not found while updating deployment")).Err()
	}

	d.Name = deployment.ObjectMeta.Name

	if d.DisplayName == "" {
		d.DisplayName = deployment.Spec.DisplayName
	}

	// dependency support
	// case 1. if there is any parent, do not update it
	// case 2. if there is no parent but child deployment should be updated (different version, profile, set of required deployment package), do not update it
	//         todo: update this to allow the children deployment CRs to be updated with this deployment CR
	// case 3. if there is no parent and child deployment does not need to be updated, update it

	if len(deployment.Status.ParentDeploymentList) > 0 {
		// case 1
		// must not update this deployment
		return nil, errors.Status(errors.NewAborted("cannot update deployment %v - %d parent deployment(s) is running",
			d.Name, len(deployment.Status.ParentDeploymentList))).Err()
	}

	// case 2
	// compare current child deployment and new required deployment package
	// if version or profile is different for the same deployment package or
	// if deployment package is missing or
	// if new deployment package is added,
	// must not update
	childDeplMap := make(map[string]string)
	newDepDeplMap := make(map[string]string)

	for _, childDepl := range deployment.Spec.ChildDeploymentList {
		childDeplMap[childDepl.DeploymentPackageRef.Name] = fmt.Sprintf("%s/%s", childDepl.DeploymentPackageRef.Version, childDepl.DeploymentPackageRef.ProfileName)
	}

	for _, reqDepl := range d.RequiredDeploymentPackage {
		newDepDeplMap[reqDepl.Name] = fmt.Sprintf("%s/%s", reqDepl.Version, reqDepl.ProfileName)
	}

	if !reflect.DeepEqual(childDeplMap, newDepDeplMap) {
		return nil, errors.Status(errors.NewAborted("cannot update deployment %v - current deployment package for child and new deployment package are not matched: current - %v, new - %v", d.Name, childDeplMap, newDepDeplMap)).Err()
	}

	// case 3
	err = deleteSecrets(ctx, s.k8sClient, deployment)
	if err != nil {
		log.Warnf("cannot update deployment: %v", err)
		return nil, errors.Status(err).Err()
	}

	d, err = createSecrets(ctx, s.k8sClient, d)
	if err != nil {
		log.Warnf("cannot update deployment: %v", err)
		return nil, errors.Status(err).Err()
	}

	updateInstance, err := createDeploymentCr(d, "update", deployment.ObjectMeta.ResourceVersion)
	if err != nil {
		log.Warnf("cannot update deployment: %v", err)
		return nil, errors.Status(err).Err()
	}

	// copy relationship lists
	updateInstance.Spec.ChildDeploymentList = deployment.Spec.ChildDeploymentList

	// Update api doesn't support with UID
	deployment, err = s.crClient.Deployments(d.Namespace).Update(ctx, d.Name, updateInstance, metav1.UpdateOptions{})
	if err != nil {
		log.Warnf("cannot update deployment: %v", err)
		return nil, errors.Status(k8serrors.K8sToTypedError(err)).Err()
	}

	// Propagate targets to child deployments (dependencies)
	// When the main deployment is updated with new clusters, ensure child deployments
	// (dependencies) are also deployed to the same clusters
	if len(updateInstance.Spec.ChildDeploymentList) > 0 {
		err = s.propagateTargetsToChildDeployments(ctx, updateInstance, d)
		if err != nil {
			log.Warnf("failed to propagate targets to child deployments: %v", err)
		}
	}

	// Need to update Owner Reference to clean up secrets. Cannot add required details during secret creation
	// since deployment UID is required.
	d, err = updateOwnerRefSecrets(ctx, s.k8sClient, d)
	if err != nil {
		log.Warnf("cannot update deployment: %v", err)
		return nil, errors.Status(err).Err()
	}

	var c DeploymentInstance
	c.deployment = deployment
	deployResponse, _ := c.createDeploymentObject(ctx, s)

	utils.LogActivity(ctx, "update", "ADM", "deployment name "+d.Name, "deploy id "+d.DeployID, "deployment app version "+in.Deployment.AppVersion)
	return &deploymentpb.UpdateDeploymentResponse{
		Deployment: deployResponse,
	}, nil
}

func (s *DeploymentSvc) GetAppNamespace(ctx context.Context, in *deploymentpb.GetAppNamespaceRequest) (*deploymentpb.GetAppNamespaceResponse, error) {
	if err := s.protoValidator.Validate(in); err != nil {
		log.Warnf("%v", err)
		return nil, errors.Status(errors.NewInvalid("%v", err)).Err()
	}

	resp := &deploymentpb.GetAppNamespaceResponse{}
	bundle := &fleetv1alpha1.Bundle{}

	clusterNameSpace, err := s.GetActiveProjectID(ctx)
	if err != nil {
		msg := fmt.Sprintf("failed to get tenant project ID %s", err.Error())
		return nil, errors.Status(errors.NewUnavailable(msg)).Err()
	}

	err = s.fleetBundleClient.Client.Get(ctx, clusterNameSpace, in.AppId, bundle, metav1.GetOptions{})
	if err != nil {
		log.Warnw("Failed to get namespace info for an application",
			dazl.String("AppID", in.AppId),
			dazl.Error(err))
		return nil, errors.Status(k8serrors.K8sToTypedError(err)).Err()
	}
	if bundle.Spec.TargetNamespace != "" {
		resp.Namespace = bundle.Spec.TargetNamespace
	} else {
		resp.Namespace = bundle.Spec.DefaultNamespace
	}
	return resp, nil
}

func (s *DeploymentSvc) ListDeploymentClusters(ctx context.Context, in *deploymentpb.ListDeploymentClustersRequest) (*deploymentpb.ListDeploymentClustersResponse, error) {
	resp := &deploymentpb.ListDeploymentClustersResponse{}
	if in == nil || in.DeplId == "" {
		log.Warnf("incomplete request")
		return nil, errors.Status(errors.NewInvalid("incomplete request")).Err()
	}

	if err := s.protoValidator.Validate(in); err != nil {
		log.Warnf("%v", err)
		return nil, errors.Status(errors.NewInvalid("%v", err)).Err()
	}

	// RBAC auth
	if err := s.AuthCheckAllowed(ctx, in); err != nil {
		log.Warnf("cannot get deployment clusters: %v", err)
		return nil, errors.Status(errors.NewForbidden("cannot get deployment clusters: %v", err)).Err()
	}

	activeProjectID, err := s.GetActiveProjectID(ctx)
	if err != nil {
		msg := fmt.Sprintf("failed to get tenant project ID %s", err.Error())
		return nil, errors.Status(errors.NewUnavailable(msg)).Err()
	}

	activeProjectIDKey := string(deploymentv1beta1.AppOrchActiveProjectID)

	labelSelector := metav1.LabelSelector{
		MatchLabels: map[string]string{activeProjectIDKey: activeProjectID},
	}

	listOpts := metav1.ListOptions{
		LabelSelector: labels.Set(labelSelector.MatchLabels).String(),
	}

	UID := in.DeplId

	namespace := activeProjectID

	deployment, err := matchUIDDeployment(ctx, UID, namespace, s, listOpts)
	if err != nil {
		log.Warnf("cannot get deployment: %v", err)
		return nil, errors.Status(err).Err()
	} else if deployment.ObjectMeta.Name == "" {
		log.Warnf("deployment id %v not found", UID)
		return nil, errors.Status(errors.NewNotFound("deployment id %v not found", UID)).Err()
	}

	name := deployment.ObjectMeta.Name

	labelSelector = metav1.LabelSelector{
		MatchLabels: map[string]string{string(deploymentv1beta1.DeploymentID): UID},
	}

	labelSelector.MatchLabels[activeProjectIDKey] = activeProjectID

	listOpts = metav1.ListOptions{
		LabelSelector: labels.Set(labelSelector.MatchLabels).String(),
	}

	// List all deploymentCluster objects and match the deployment-id of deployment
	deploymentClusters, err := s.crClient.DeploymentClusters("").List(ctx, listOpts)
	if err != nil {
		log.Warnf("cannot get deployment cluster: %v", err)
		return nil, errors.Status(k8serrors.K8sToTypedError(err)).Err()
	}
	clusterList := make([]*deploymentpb.Cluster, 0)

	if deploymentClusters != nil && len(deploymentClusters.Items) != 0 {
		clusterList = make([]*deploymentpb.Cluster, len(deploymentClusters.Items))
		for clIndex := range deploymentClusters.Items {
			clusterList[clIndex] = createDeploymentClusterCr(&deploymentClusters.Items[clIndex])
		}
	}

	totalNumClusters := len(clusterList)
	selectedClusters, err := selectClustersPerDeployment(in, clusterList)
	if err != nil {
		log.Warnf("cannot list clusters for given deployment: %s, %v", UID, err)
		return nil, errors.Status(errors.NewInvalid("cannot list clusters for given deployment: %s, %v", in.DeplId, err)).Err()
	}

	resp.Clusters = selectedClusters
	resp.TotalElements = utils.ToInt32Clamped(totalNumClusters)
	utils.LogActivity(ctx, "get", "Clusters for A Deployment", "deployment name "+name, "deploy id "+UID)
	return resp, nil
}

// propagateTargetsToChildDeployments updates child deployments (dependencies) to ensure
// they are deployed to the same clusters as the parent deployment
func (s *DeploymentSvc) propagateTargetsToChildDeployments(ctx context.Context, parentDeployment *deploymentv1beta1.Deployment, parentDeploymentData *Deployment) error {
	// Collect all targets from parent deployment applications
	allTargets := make([]map[string]string, 0)
	for _, app := range parentDeployment.Spec.Applications {
		allTargets = append(allTargets, app.Targets...)
	}

	// Remove duplicate targets
	uniqueTargets := make([]map[string]string, 0)
	targetSet := make(map[string]bool)
	for _, target := range allTargets {
		clusterName := target[string(deploymentv1beta1.ClusterName)]
		if !targetSet[clusterName] {
			targetSet[clusterName] = true
			uniqueTargets = append(uniqueTargets, target)
		}
	}

	log.Infof("Propagating targets to child deployments: %d unique clusters", len(uniqueTargets))

	// Update each child deployment to include all targets
	for childName := range parentDeployment.Spec.ChildDeploymentList {
		log.Infof("Updating child deployment: %s", childName)

		// Get the child deployment
		childDeployment, err := s.crClient.Deployments(parentDeployment.Namespace).Get(ctx, childName, metav1.GetOptions{})
		if err != nil {
			log.Warnf("Failed to get child deployment %s: %v", childName, err)
			continue
		}

		// Update targets for all applications in the child deployment
		updated := false
		for i := range childDeployment.Spec.Applications {
			app := &childDeployment.Spec.Applications[i]

			// Add missing targets to the child application
			for _, newTarget := range uniqueTargets {
				targetExists := false
				newClusterName := newTarget[string(deploymentv1beta1.ClusterName)]

				// Check if this target already exists
				for _, existingTarget := range app.Targets {
					existingClusterName := existingTarget[string(deploymentv1beta1.ClusterName)]
					if existingClusterName == newClusterName {
						targetExists = true
						break
					}
				}

				// Add the target if it doesn't exist
				if !targetExists {
					app.Targets = append(app.Targets, newTarget)
					updated = true
					log.Infof("Added cluster %s to child app %s", newClusterName, app.Name)
				}
			}
		}

		// Update the child deployment if targets were modified
		if updated {
			_, err = s.crClient.Deployments(parentDeployment.Namespace).Update(ctx, childName, childDeployment, metav1.UpdateOptions{})
			if err != nil {
				log.Warnf("Failed to update child deployment %s: %v", childName, err)
			} else {
				log.Infof("Successfully updated child deployment %s with new targets", childName)
			}
		}
	}

	return nil
}
