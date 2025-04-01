// SPDX-FileCopyrightText: (C) 2024 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package northbound

import (
	"context"
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"net/http"
	"net/http/httptest"

	deploymentpb "github.com/open-edge-platform/app-orch-deployment/app-deployment-manager/api/nbi/v2/deployment/v1"
	deploymentv1beta1 "github.com/open-edge-platform/app-orch-deployment/app-deployment-manager/api/v1beta1"
)

var _ = Describe("Gateway gRPC Service", func() {
	var (
		deploymentListSrc deploymentv1beta1.DeploymentList
		deployInstance    *deploymentv1beta1.Deployment
	)

	Describe("Gateway Utils", func() {
		BeforeEach(func() {
			setDeploymentListObject(&deploymentListSrc)
			deployInstance = SetDeployInstance(&deploymentListSrc, "")
		})

		It("successfully return deployment type targeted", func() {
			deploymentType := deploymentType(string(deploymentv1beta1.Targeted))

			Expect(deploymentType).Should(Equal(deploymentv1beta1.Targeted))
		})

		It("successfully return default deployment type auto-scaling", func() {
			deploymentType := deploymentType("test")

			Expect(deploymentType).Should(Equal(deploymentv1beta1.AutoScaling))
		})

		It("successfully return deployment state ERROR", func() {
			state := deploymentState("Error")

			Expect(state).Should(Equal(deploymentpb.State_ERROR))
		})

		It("successfully return deployment state INTERNAL_ERROR", func() {
			state := deploymentState("InternalError")

			Expect(state).Should(Equal(deploymentpb.State_INTERNAL_ERROR))
		})

		It("successfully return deployment state NO_TARGET_CLUSTERS", func() {
			state := deploymentState("NoTargetClusters")

			Expect(state).Should(Equal(deploymentpb.State_NO_TARGET_CLUSTERS))
		})

		It("successfully create all secrets", func() {
			ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				_, err := w.Write([]byte(`{"apiVersion": "v1", "kind": "Secret", "metadata": {"name": "test-name"}}`))
				Expect(err).ToNot(HaveOccurred())
			}))

			defer ts.Close()

			kc := mockK8Client(ts.URL)

			d := setDeployment()
			_, err := createSecrets(context.Background(), kc, d)

			Expect(err).ShouldNot(HaveOccurred())
			s, ok := status.FromError(err)
			Expect(s.Code()).To(Equal(codes.OK))
			Expect(ok).To(BeTrue())
		})

		It("fails due to missing namespace while creating secret", func() {
			Skip("It needs to be evaluated as empty namespace is acceptable with k8s client")
			ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				_, err := w.Write([]byte(`{"apiVersion": "v1", "kind": "Secret", "metadata": {"name": "test-name"}}`))
				Expect(err).ToNot(HaveOccurred())
			}))

			defer ts.Close()

			kc := mockK8Client(ts.URL)

			d := setDeployment()
			d.Namespace = ""
			_, err := createSecrets(context.Background(), kc, d)

			Expect(err).Should(HaveOccurred())
			s, ok := status.FromError(err)
			Expect(s.Code()).To(Equal(codes.Unknown))
			Expect(ok).To(BeFalse())
			Expect(s.Message()).Should(Equal("an empty namespace may not be set during creation"))
		})

		It("successfully update OwnerReference in secret", func() {
			ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				_, err := w.Write([]byte(`{"apiVersion": "v1", "kind": "Secret", "metadata": {"name": "test-name"}}`))
				Expect(err).ToNot(HaveOccurred())
			}))

			defer ts.Close()

			kc := mockK8Client(ts.URL)

			var ownerReferenceList []metav1.OwnerReference

			apiVersion := fmt.Sprintf("%s/%s", deploymentv1beta1.GroupVersion.Group, deploymentv1beta1.GroupVersion.Version)
			useController := true

			ownerReference := metav1.OwnerReference{
				Name:       "test-name",
				Kind:       "Deployment",
				APIVersion: apiVersion,
				UID:        types.UID(VALID_UID_DC),
				Controller: &useController,
			}

			ownerReferenceList = append(ownerReferenceList, ownerReference)

			err := updateOwnerRefSecret(context.Background(), kc, ownerReferenceList, "test-secretname", "test-namespace")

			Expect(err).ShouldNot(HaveOccurred())
			s, ok := status.FromError(err)
			Expect(s.Code()).To(Equal(codes.OK))
			Expect(ok).To(BeTrue())
		})

		It("fails due to missing namespace while updateOwnerRefSecret", func() {
			Skip("It needs to be evaluated as empty namespace is acceptable with k8s client")
			ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				_, err := w.Write([]byte(`{"apiVersion": "v1", "kind": "Secret", "metadata": {"name": "test-name"}}`))
				Expect(err).ToNot(HaveOccurred())
			}))

			defer ts.Close()

			kc := mockK8Client(ts.URL)

			var ownerReferenceList []metav1.OwnerReference

			err := updateOwnerRefSecret(context.Background(), kc, ownerReferenceList, "test-secretname", "")

			Expect(err).Should(HaveOccurred())
			s, ok := status.FromError(err)
			Expect(s.Code()).To(Equal(codes.Unknown))
			Expect(ok).To(BeFalse())
			Expect(s.Message()).Should(Equal("an empty namespace may not be set when a resource name is provided"))
		})

		It("fails due to missing namespace while updating all OwnerRefSecrets", func() {
			Skip("It needs to be evaluated as empty namespace is acceptable with k8s client")
			ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				_, err := w.Write([]byte(`{"apiVersion": "v1", "kind": "Secret", "metadata": {"name": "test-name"}}`))
				Expect(err).ToNot(HaveOccurred())
			}))

			defer ts.Close()

			kc := mockK8Client(ts.URL)

			d := setDeployment()
			d.Namespace = ""

			_, err := updateOwnerRefSecrets(context.Background(), kc, d)

			Expect(err).Should(HaveOccurred())
			s, ok := status.FromError(err)
			Expect(s.Code()).To(Equal(codes.Unknown))
			Expect(ok).To(BeFalse())
			Expect(s.Message()).Should(Equal("resource name may not be empty"))
		})

		It("successfully delete all secrets", func() {
			ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				_, err := w.Write([]byte(`{"apiVersion": "v1", "kind": "Secret", "metadata": {"name": "test-name"}}`))
				Expect(err).ToNot(HaveOccurred())
			}))

			defer ts.Close()

			kc := mockK8Client(ts.URL)

			deployInstance.Spec.Applications[0].ProfileSecretName = "test-profile"
			deployInstance.Spec.Applications[0].ValueSecretName = "test-values"
			deployInstance.Spec.Applications[0].HelmApp.RepoSecretName = "test-helmrepo"
			deployInstance.Spec.Applications[0].HelmApp.ImageRegistrySecretName = "test-imagerepo"

			err := deleteSecrets(context.Background(), kc, deployInstance)

			Expect(err).ShouldNot(HaveOccurred())
			s, ok := status.FromError(err)
			Expect(s.Code()).To(Equal(codes.OK))
			Expect(ok).To(BeTrue())
		})

		It("fails due to error returned when deleting secret", func() {
			ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusMethodNotAllowed)
				_, err := w.Write([]byte(`{"apiVersion": "v1", "kind": "Secret", "metadata": {"name": "test-name"}}`))
				Expect(err).ToNot(HaveOccurred())
			}))

			defer ts.Close()

			kc := mockK8Client(ts.URL)

			deployInstance.Spec.Applications[0].ProfileSecretName = "test-profile"

			err := deleteSecrets(context.Background(), kc, deployInstance)

			Expect(err).Should(HaveOccurred())
			s, ok := status.FromError(err)
			Expect(s.Code()).To(Equal(codes.Unknown))
			Expect(ok).To(BeFalse())
			Expect(s.Message()).Should(Equal("the server does not allow this method " +
				"on the requested resource (delete secrets test-profile)"))
		})

		It("fails due to ProfileSecretName is empty when deleting secrets", func() {
			ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				_, err := w.Write([]byte(`{"apiVersion": "v1", "kind": "Secret", "metadata": {"name": "test-name"}}`))
				Expect(err).ToNot(HaveOccurred())
			}))

			defer ts.Close()

			kc := mockK8Client(ts.URL)

			err := deleteSecrets(context.Background(), kc, deployInstance)

			Expect(err).Should(HaveOccurred())
			s, ok := status.FromError(err)
			Expect(s.Code()).To(Equal(codes.Unknown))
			Expect(ok).To(BeFalse())
			Expect(s.Message()).Should(Equal("resource name may not be empty"))
		})

		It("fails due to 1 application missing mandatory parameter template value", func() {
			d := setHelmApps(1)
			for _, i := range *d.HelmApps {
				i.ParameterTemplates[0].Mandatory = true
			}

			_, err := checkParameterTemplate(d, make(map[string][]string))

			Expect(err).Should(HaveOccurred())
			s, ok := status.FromError(err)
			Expect(s.Code()).To(Equal(codes.Unknown))
			Expect(ok).To(BeFalse())
			Expect(s.Message()).Should(Equal("application test-name-0 is missing mandatory override profile values"))
		})

		It("fails due to 3 applications missing mandatory parameter template value", func() {
			d := setHelmApps(3)
			for _, i := range *d.HelmApps {
				i.ParameterTemplates[0].Mandatory = true
			}

			_, err := checkParameterTemplate(d, make(map[string][]string))

			Expect(err).Should(HaveOccurred())
			s, ok := status.FromError(err)
			Expect(s.Code()).To(Equal(codes.Unknown))
			Expect(ok).To(BeFalse())
			Expect(s.Message()).Should(Equal("applications test-name-2, test-name-1, test-name-0 " +
				"are missing mandatory override profile values"))
		})

		It("successfully merges all targetclusters with targeted deployment type", func() {
			d := setDeployment()

			d.DeploymentType = string(deploymentv1beta1.Targeted)
			d.TargetClusters = make([]*deploymentpb.TargetClusters, 1)
			d.TargetClusters[0] = &deploymentpb.TargetClusters{
				AppName:   "test-appname",
				ClusterId: "cluster1",
			}

			d.AllAppTargetClusters = &deploymentpb.TargetClusters{
				AppName:   "test-appname",
				Labels:    map[string]string{"test1": "foo1"},
				ClusterId: "cluster2",
			}
			err := mergeAllAppTargetClusters(context.Background(), d)
			Expect(err).ShouldNot(HaveOccurred())
			s, ok := status.FromError(err)
			Expect(s.Code()).To(Equal(codes.OK))
			Expect(ok).To(BeTrue())
		})
		It("successfully merges all targetclusters with autoscale deployment type", func() {
			d := setDeployment()

			d.DeploymentType = string(deploymentv1beta1.AutoScaling)
			d.TargetClusters = make([]*deploymentpb.TargetClusters, 1)
			d.TargetClusters[0] = &deploymentpb.TargetClusters{
				AppName:   "test-appname",
				Labels:    map[string]string{"test": "foo"},
				ClusterId: "cluster1",
			}

			d.AllAppTargetClusters = &deploymentpb.TargetClusters{
				AppName:   "test-appname",
				Labels:    map[string]string{"test1": "foo1"},
				ClusterId: "cluster2",
			}

			err := mergeAllAppTargetClusters(context.Background(), d)
			Expect(err).ShouldNot(HaveOccurred())
			s, ok := status.FromError(err)
			Expect(s.Code()).To(Equal(codes.OK))
			Expect(ok).To(BeTrue())
		})
		It("successfully merges all targetclusters when target cluster is nil", func() {
			d := setDeployment()
			d.TargetClusters = nil

			d.AllAppTargetClusters = &deploymentpb.TargetClusters{
				Labels: map[string]string{"test": "foobar"},
			}

			d.DeploymentType = string(deploymentv1beta1.AutoScaling)

			err := mergeAllAppTargetClusters(context.Background(), d)
			Expect(err).ShouldNot(HaveOccurred())
			s, ok := status.FromError(err)
			Expect(s.Code()).To(Equal(codes.OK))
			Expect(ok).To(BeTrue())
		})
		It("successfully merges all targetclusters when there is common key in Labels", func() {
			d := setDeployment()
			d.DeploymentType = string(deploymentv1beta1.AutoScaling)

			d.TargetClusters = make([]*deploymentpb.TargetClusters, 1)
			d.TargetClusters[0] = &deploymentpb.TargetClusters{
				AppName: "test-appname",
				Labels:  map[string]string{"test": "foo"},
			}

			d.AllAppTargetClusters = &deploymentpb.TargetClusters{
				AppName: "test-appname",
				Labels:  map[string]string{"test": "foo1"},
			}
			err := mergeAllAppTargetClusters(context.Background(), d)
			Expect(err).ShouldNot(HaveOccurred())
			s, ok := status.FromError(err)
			Expect(s.Code()).To(Equal(codes.OK))
			Expect(ok).To(BeTrue())
		})

	})
})
