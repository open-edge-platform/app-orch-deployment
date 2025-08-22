// SPDX-FileCopyrightText: (C) 2024 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package northbound

import (
	"context"
	"fmt"
	"regexp"
	"time"

	deploymentpb "github.com/open-edge-platform/app-orch-deployment/app-deployment-manager/api/nbi/v2/deployment/v1"
	"github.com/open-edge-platform/orch-library/go/dazl"
	"github.com/open-edge-platform/orch-library/go/pkg/errors"
	"google.golang.org/protobuf/types/known/timestamppb"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"

	deploymentv1beta1 "github.com/open-edge-platform/app-orch-deployment/app-deployment-manager/api/v1beta1"
	"github.com/open-edge-platform/app-orch-deployment/app-deployment-manager/pkg/utils"
	"github.com/open-edge-platform/app-orch-deployment/app-deployment-manager/pkg/utils/k8serrors"
)

type ClusterInfo struct {
	ID         string                 `yaml:"id"`
	Labels     map[string]string      `yaml:"labels"`
	Name       string                 `yaml:"name"`
	CreateTime *timestamppb.Timestamp `yaml:"createTime"`
}

func (s *DeploymentSvc) GetKubeConfig(ctx context.Context, in *deploymentpb.GetKubeConfigRequest) (*deploymentpb.GetKubeConfigResponse, error) {
	if in == nil || in.ClusterId == "" {
		log.Warnf("incomplete request")
		return nil, errors.Status(errors.NewInvalid("incomplete request")).Err()
	}

	if err := s.protoValidator.Validate(in); err != nil {
		log.Warnf("%v", err)
		return nil, errors.Status(errors.NewInvalid("%v", err)).Err()
	}

	// Validate cluster ID pattern
	idRegex := regexp.MustCompile(IDPattern)
	if !idRegex.MatchString(in.ClusterId) {
		log.Warnf("cluster ID does not match pattern: %s", in.ClusterId)
		return nil, errors.Status(errors.NewInvalid("validation error:\n - cluster_id: value does not match regex pattern `%s` [string.pattern]", IDPattern)).Err()
	}

	// RBAC auth
	if err := s.AuthCheckAllowed(ctx, in); err != nil {
		log.Warnf("cannot get kubeConfig info: %v", err)
		return nil, errors.Status(errors.NewForbidden("cannot get kubeConfig info")).Err()
	}

	log.Infow("Received GetKubeConfig Request", dazl.String("Cluster ID", in.ClusterId))

	namespace, err := s.GetActiveProjectID(ctx)
	if err != nil {
		msg := fmt.Sprintf("failed to get tenant project ID %s", err.Error())
		return nil, errors.Status(errors.NewUnavailable(msg)).Err()
	}

	cluster, err := s.crClient.Clusters(namespace).Get(ctx, in.ClusterId, metav1.GetOptions{})
	if err != nil {
		log.Warnf("cannot get cluster info: %v", err)
		return nil, errors.Status(k8serrors.K8sToTypedError(err)).Err()
	}

	secretName := cluster.Spec.KubeConfigSecretName
	kubeConfigValue, err := utils.GetSecretValue(ctx, s.k8sClient, namespace, secretName)
	if err != nil {
		log.Warnf("cannot get kubeConfig value: %v", err)
		return nil, errors.Status(err).Err()
	}

	kubeConfigInfo := &deploymentpb.KubeConfigInfo{
		KubeConfig: kubeConfigValue.Data["value"],
	}

	utils.LogActivity(ctx, "get kubeConfig", "ADM")

	return &deploymentpb.GetKubeConfigResponse{
		KubeConfigInfo: kubeConfigInfo,
	}, nil
}

func (s *DeploymentSvc) ListClusters(ctx context.Context, in *deploymentpb.ListClustersRequest) (*deploymentpb.ListClustersResponse, error) {
	if err := s.protoValidator.Validate(in); err != nil {
		log.Warnf("%v", err)
		return nil, errors.Status(errors.NewInvalid("%v", err)).Err()
	}

	// Validate maxItems for labels array
	if len(in.Labels) > MaxLabelsPerRequestClusters {
		log.Warnf("labels array exceeds maximum size: %d > %d", len(in.Labels), MaxLabelsPerRequestClusters)
		return nil, errors.Status(errors.NewInvalid("labels array exceeds maximum size of %d items", MaxLabelsPerRequestClusters)).Err()
	}

	// Validate page_size range
	if in.PageSize < 0 || in.PageSize > MaxPageSize {
		log.Warnf("page_size out of range: %d (must be 0 <= page_size <= %d)", in.PageSize, MaxPageSize)
		return nil, errors.Status(errors.NewInvalid("validation error:\n - page_size: value must be greater than or equal to 0 and less than or equal to %d [int32.gte_lte]", MaxPageSize)).Err()
	}

	// RBAC auth.
	if err := s.AuthCheckAllowed(ctx, in); err != nil {
		log.Warnf("cannot list clusters: %v", err)
		return nil, errors.Status(errors.NewForbidden("cannot list clusters: %v", err)).Err()
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

	namespace := activeProjectID

	clusters, err := s.crClient.Clusters(namespace).List(ctx, listOpts)
	if err != nil {
		log.Warnf("cannot list clusters: %v", err)
		return nil, errors.Status(k8serrors.K8sToTypedError(err)).Err()
	}

	clusterInfoList := make([]*deploymentpb.ClusterInfo, len(clusters.Items))
	for index, cluster := range clusters.Items {
		// Convert metav1.Time to protobuf time and return secs.
		setPbTime := cluster.ObjectMeta.CreationTimestamp.ProtoTime()

		// Return timestamp from the provided time.Time in unix.
		createTimePbUnix := timestamppb.New(time.Unix(setPbTime.Seconds, 0))

		clusterInfoList[index] = &deploymentpb.ClusterInfo{
			Id:         cluster.Name,
			Labels:     cluster.Labels,
			CreateTime: createTimePbUnix,
			Name:       cluster.Spec.DisplayName,
		}
	}

	totalNumClusters := len(clusterInfoList)
	// Paginate, sort, and filter list of clusters
	selectedClusters, err := selectClusters(in, clusterInfoList)
	if err != nil {
		log.Warnf("cannot list clusters: %v", err)
		return nil, errors.Status(err).Err()
	}

	// Enforce maxItems limit on final response
	if len(selectedClusters) > MaxClustersResponse {
		log.Warnf("response clusters array exceeds maximum size. Returning first %d entries from %d total",
			MaxClustersResponse, len(selectedClusters))
		selectedClusters = selectedClusters[:MaxClustersResponse]
	}

	utils.LogActivity(ctx, "list clusters", "ADM")

	return &deploymentpb.ListClustersResponse{
		Clusters:      selectedClusters,
		TotalElements: utils.ToInt32Clamped(totalNumClusters),
	}, nil
}

func (s *DeploymentSvc) GetCluster(ctx context.Context, in *deploymentpb.GetClusterRequest) (*deploymentpb.GetClusterResponse, error) {
	if in == nil || in.ClusterId == "" {
		log.Warnf("incomplete request")
		return nil, errors.Status(errors.NewInvalid("incomplete request")).Err()
	}

	if err := s.protoValidator.Validate(in); err != nil {
		log.Warnf("%v", err)
		return nil, errors.Status(errors.NewInvalid("%v", err)).Err()
	}

	// RBAC auth
	if err := s.AuthCheckAllowed(ctx, in); err != nil {
		log.Warnf("cannot get cluster: %v", err)
		return nil, errors.Status(errors.NewForbidden("cannot get cluster: %v", err)).Err()
	}

	clusterID := in.ClusterId

	// ListOptions filters with LabelSelector and FieldSelector only. FieldSelector only
	// supports metadata.namespace or metadata.name
	labelSelector := metav1.LabelSelector{
		MatchLabels: map[string]string{string(deploymentv1beta1.ClusterName): clusterID},
	}

	activeProjectID, err := s.GetActiveProjectID(ctx)
	if err != nil {
		msg := fmt.Sprintf("failed to get tenant project ID %s", err.Error())
		return nil, errors.Status(errors.NewUnavailable(msg)).Err()
	}

	activeProjectIDKey := string(deploymentv1beta1.AppOrchActiveProjectID)

	labelSelector.MatchLabels[activeProjectIDKey] = activeProjectID

	listOpts := metav1.ListOptions{
		LabelSelector: labels.Set(labelSelector.MatchLabels).String(),
	}

	clusters, err := s.crClient.DeploymentClusters("").List(ctx, listOpts)
	if err != nil {
		log.Warnf("cannot get cluster: %v", err)
		return nil, errors.Status(k8serrors.K8sToTypedError(err)).Err()
	}

	if len(clusters.Items) == 0 {
		log.Warnf("cluster id %v not found", clusterID)
		return nil, errors.Status(errors.NewNotFound("cluster id %v not found", clusterID)).Err()
	}

	clusterResponse := createDeploymentClusterCr(&clusters.Items[0])

	for i := 1; i < len(clusters.Items); i++ {
		tmpResp := createDeploymentClusterCr(&clusters.Items[i])
		// status
		clusterResponse.Status.Summary.Total = utils.ToInt32Clamped(int(tmpResp.Status.Summary.Total) + int(clusterResponse.Status.Summary.Total))
		clusterResponse.Status.Summary.Down = utils.ToInt32Clamped(int(tmpResp.Status.Summary.Down) + int(clusterResponse.Status.Summary.Down))
		clusterResponse.Status.Summary.Running = utils.ToInt32Clamped(int(tmpResp.Status.Summary.Running) + int(clusterResponse.Status.Summary.Running))
		clusterResponse.Status.Summary.Unknown = utils.ToInt32Clamped(int(tmpResp.Status.Summary.Unknown) + int(clusterResponse.Status.Summary.Unknown))
		if tmpResp.Status.State != deploymentpb.State_RUNNING {
			clusterResponse.Status.State = tmpResp.Status.State
		}

		// app
		clusterResponse.Apps = append(clusterResponse.Apps, tmpResp.Apps...)
	}

	utils.LogActivity(ctx, "get", "get Cluster", "cluster id "+clusterID)
	return &deploymentpb.GetClusterResponse{
		Cluster: clusterResponse,
	}, nil
}
