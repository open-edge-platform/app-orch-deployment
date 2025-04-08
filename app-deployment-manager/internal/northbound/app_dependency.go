// SPDX-FileCopyrightText: (C) 2024 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package northbound

import (
	"context"
	"fmt"
	"reflect"

	"github.com/open-edge-platform/app-orch-deployment/app-deployment-manager/pkg/utils/k8serrors"
	"github.com/open-edge-platform/orch-library/go/pkg/errors"
	"k8s.io/apimachinery/pkg/labels"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	deploymentpb "github.com/open-edge-platform/app-orch-deployment/app-deployment-manager/api/nbi/v2/deployment/v1"
	deploymentv1beta1 "github.com/open-edge-platform/app-orch-deployment/app-deployment-manager/api/v1beta1"
	"github.com/open-edge-platform/app-orch-deployment/app-deployment-manager/internal/catalogclient"
)

func getTargetDeploymentsForDeleteDeployment(rootName string, targetList map[string]*deploymentv1beta1.Deployment, allDeployments map[string]*deploymentv1beta1.Deployment, currentDepth int) error {
	// Check if the current depth exceeds the maximum depth
	if currentDepth > maxDepth {
		return fmt.Errorf("maximum recursion depth reached")
	}

	targetList[rootName] = allDeployments[rootName]

	if allDeployments[rootName] == nil || allDeployments[rootName].Spec.ChildDeploymentList == nil {
		return nil
	}

	for k := range allDeployments[rootName].Spec.ChildDeploymentList {
		err := getTargetDeploymentsForDeleteDeployment(k, targetList, allDeployments, currentDepth+1)
		if err != nil {
			return err
		}
	}
	return nil
}

func validateTargetDeployments(targetList map[string]*deploymentv1beta1.Deployment) bool {
	for _, v := range targetList {
		for k := range v.Status.ParentDeploymentList {
			// if one of target deployment CRs has a parent deployment CR outside of dependency graph, validation failed
			if _, ok := targetList[k]; !ok {
				return false
			}
		}
	}

	return true
}

func getDeploymentWithDeploymentPackage(dpID string, depls map[string]*Deployment) *Deployment {
	for k, v := range depls {
		if dpID == catalogclient.GetDeploymentPackageID(v.AppName, v.AppVersion, v.ProfileName) {
			return depls[k]
		}
	}

	return nil
}

func addRelationshipToDeploymentCR(d *Deployment, deploymentCR *deploymentv1beta1.Deployment) {
	// create map data structure if it is nil (CR does not have any entry in the list)
	if deploymentCR.Spec.ChildDeploymentList == nil {
		deploymentCR.Spec.ChildDeploymentList = make(map[string]deploymentv1beta1.DependentDeploymentRef)
	}
	for k, v := range d.ChildDeploymentList {
		deploymentCR.Spec.ChildDeploymentList[k] = deploymentv1beta1.DependentDeploymentRef{
			DeploymentPackageRef: deploymentv1beta1.DeploymentPackageRef{
				Name:                       v.AppName,
				Version:                    v.AppVersion,
				ProfileName:                v.ProfileName,
				ForbidsMultipleDeployments: false,
			},
			DeploymentName: k,
		}
	}
}

func getDeploymentCRWithDeploymentPackage(ctx context.Context, s *DeploymentSvc, namespace string, dpID string, listOpts metav1.ListOptions) ([]*deploymentv1beta1.Deployment, error) {
	results := make([]*deploymentv1beta1.Deployment, 0)

	deployments, err := s.crClient.Deployments(namespace).List(ctx, listOpts)
	if err != nil {
		return nil, k8serrors.K8sToTypedError(err)
	}

	for deplIdx := range deployments.Items {
		r := deployments.Items[deplIdx].Spec.DeploymentPackageRef
		if catalogclient.GetDeploymentPackageID(r.Name, r.Version, r.ProfileName) == dpID {
			results = append(results, &deployments.Items[deplIdx])
		}
	}

	return results, nil
}

func containTargetCluster(list []map[string]string, input map[string]string) bool {
	for _, e := range list {
		if reflect.DeepEqual(e, input) {
			return true
		}
	}
	return false
}

func getTargetClusterMap(input *deploymentpb.TargetClusters) map[string]string {
	result := make(map[string]string)
	if input.Labels != nil {
		result = input.Labels
	}
	if input.ClusterId != "" {
		result[string(deploymentv1beta1.ClusterName)] = input.ClusterId
	}

	return result
}

func addTargetClusterEntry(newTargetClusters []*deploymentpb.TargetClusters, deploymentCR *deploymentv1beta1.Deployment, activeProjectID string) {
	for _, tc := range newTargetClusters {
		for appIdx, app := range deploymentCR.Spec.Applications {
			tcForApp := getTargetClusterMap(tc)
			if tc.AppName == app.Name && !containTargetCluster(app.Targets, getTargetClusterMap(tc)) && len(tcForApp) > 0 {
				tcForApp[deploymentv1beta1.ClusterOrchKeyProjectID] = activeProjectID
				deploymentCR.Spec.Applications[appIdx].Targets = append(deploymentCR.Spec.Applications[appIdx].Targets, tcForApp)
			}
		}
	}
}

// Add dependent deployments and their relationships
func addDepAndRelationships(ctx context.Context, d *Deployment, s *DeploymentSvc, in *deploymentpb.Deployment, dependentDepls map[string]*Deployment, activeProjectID string) (*Deployment, error) {
	// initialize dependent lists
	d.RequiredDeploymentPackage = make(map[string]*deploymentv1beta1.DeploymentPackageRef)
	d.ChildDeploymentList = make(map[string]*Deployment)

	helmApps := d.HelmApps
	activeProjectIDKey := string(deploymentv1beta1.AppOrchActiveProjectID)

	// add dependent deployments and their relationships
	if helmApps != nil {
		for thisAppIdx := range *helmApps {
			for _, dep := range (*helmApps)[thisAppIdx].RequiredDeploymentPackages {
				depDP, depHelmApps, _, err := catalogclient.CatalogLookupDPAndHelmApps(ctx, s.catalogClient, dep.Name, dep.Version, dep.Profile)
				if err != nil {
					log.Warnf("failed to lookup dependent deployment package %+v", dep)
					return d, err
				}
				// convert dependent deployment package reference
				depDPRef := &deploymentv1beta1.DeploymentPackageRef{
					Name:                       dep.Name,
					Version:                    dep.Version,
					ProfileName:                dep.Profile,
					ForbidsMultipleDeployments: depDP.ForbidsMultipleDeployments,
				}
				d.RequiredDeploymentPackage[catalogclient.GetDeploymentPackageID(depDPRef.Name, depDPRef.Version, depDPRef.ProfileName)] = depDPRef

				// for the applications in the dependent deployment, add the targetCluster entries same as what parent application has
				// NOTE: assumption is that TargetClusters in parent Deployment object is not empty - it was checked at the beginning of this function
				depTargetClusters := make([]*deploymentpb.TargetClusters, 0)
				if depHelmApps != nil {
					for _, depApp := range *depHelmApps {
						depTargetClusters = append(depTargetClusters, &deploymentpb.TargetClusters{
							AppName:   depApp.Name,
							Labels:    in.TargetClusters[0].Labels,
							ClusterId: in.TargetClusters[0].ClusterId,
						})
					}
				}

				// case 1: if dependent deployment package has ForbidsMultipleDeployments == false (no matter if Deployment for dependent deployment package exists or not)
				// - action: Create dependent Deployment object with new name
				// case 2: if dependent deployment package has ForbidsMultipleDeployments == true and
				//         previously the Deployment object for this dependent deployment package is already created in this dependency graph
				// - action: Get existing Deployment object and update it
				// case 3: if dependent deployment package has ForbidsMultipleDeployments == true,
				//         previously the Deployment object for this dependent deployment package is not created in this dependency graph,
				//         and Deployment CR for dependent deployment package is not created
				// - action: Create dependent Deployment object with new name
				// case 4: if dependent deployment package has ForbidsMultipleDeployments == true,
				//         previously the Deployment object for this dependent deployment package is not created in this dependency graph,
				//         and Deployment CR for dependent deployment package exists
				// - action: Get existing dependent Deployment CR; dependent Deployment object with Deployment CR's name and ID

				var depDeployment *Deployment

				if tmpDepDepl := getDeploymentWithDeploymentPackage(catalogclient.GetDeploymentPackageID(depDPRef.Name, depDPRef.Version, depDPRef.ProfileName), dependentDepls); tmpDepDepl != nil && depDPRef.ForbidsMultipleDeployments {
					// case 2
					depDeployment = tmpDepDepl
				} else {
					// case 1, 3, 4
					// recursive call - it will traverse to the leaf deployments
					// todo: it would be good to add validation step to check if there is a cyclic reference among dependents
					// todo: the above one could be low priority since catalog already checks it when loading app and deployment packages
					depDeployment, err = initDeployment(ctx, s, "create", &deploymentpb.Deployment{
						AppName:        dep.Name,
						AppVersion:     dep.Version,
						ProfileName:    dep.Profile,
						OverrideValues: make([]*deploymentpb.OverrideValues, 0),
						TargetClusters: depTargetClusters,
						DeploymentType: in.DeploymentType,
					}, dependentDepls, activeProjectID)
					if err != nil {
						return d, errors.NewConflict("failed to create child deployment: %v", err)
					}
					// add childDeployment to dependentDepls map
					dependentDepls[depDeployment.Name] = depDeployment

					// no further action needed for case 1 and 3 since action for them is just create new Deployment object
					if depDPRef.ForbidsMultipleDeployments {
						listOpts := metav1.ListOptions{}
						
						labelSelector := metav1.LabelSelector{
							MatchLabels: map[string]string{activeProjectIDKey: activeProjectID},
						}

						listOpts = metav1.ListOptions{
							LabelSelector: labels.Set(labelSelector.MatchLabels).String(),
						}
						
						// case 4
						depDeplCRs, err := getDeploymentCRWithDeploymentPackage(ctx, s, d.Namespace, catalogclient.GetDeploymentPackageID(depDPRef.Name, depDPRef.Version, depDPRef.ProfileName), listOpts)
						if err != nil {
							return d, err
						}
						if len(depDeplCRs) == 1 {
							depDeployment.Name = depDeplCRs[0].Name
							depDeployment.DeployID = string(depDeplCRs[0].ObjectMeta.UID)
						} else if len(depDeplCRs) > 1 {
							// error case
							return d, errors.NewAlreadyExists("confused the target Deployment: the deployment package forbids multiple deployment package but there are multiple deployments for the deployment package")
						}
					}
				}

				// register childDeployment to this Deployment as a child
				d.ChildDeploymentList[depDeployment.Name] = depDeployment
			}
		}
	}

	return d, nil
}
