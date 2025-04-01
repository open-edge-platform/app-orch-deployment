// SPDX-FileCopyrightText: (C) 2024 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package controller

import (
	"os"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	fleetv1alpha1 "github.com/rancher/fleet/pkg/apis/fleet.cattle.io/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/open-edge-platform/app-orch-deployment/app-deployment-manager/api/v1beta1"
)

var _ = Describe("Cluster Controller", func() {
	const (
		clusterId        = "cluster-12345"
		name             = "mydeployment"
		uid              = "216e7223-1932-4df6-a6c7-828c84479726"
		fleetNs          = "fleet-default"
		aspName          = "orchestrator-service-proxy"
		clientSecretName = "test-clientSecretName"
		repoURL          = "http://test-gitRepoUrl/git/test-gitRemoteRepoName.git"
		updatedRepoURL   = "http://test-updatedRepoURL"
		helmSecretName   = "test-helmSecretName"
	)

	var (
		cluster *v1beta1.Cluster
		r       *ClusterReconciler
		c       client.WithWatch
		gitRepo fleetv1alpha1.GitRepo
	)

	Describe("Reconcile Fleet GitRepo", func() {
		BeforeEach(func() {
			gitRepo = fleetv1alpha1.GitRepo{
				ObjectMeta: metav1.ObjectMeta{Namespace: fleetNs, Name: aspName},
				Spec: fleetv1alpha1.GitRepoSpec{
					HelmSecretName:   helmSecretName,
					Repo:             repoURL,
					ClientSecretName: clientSecretName,
					Targets: []fleetv1alpha1.GitTarget{
						{
							ClusterSelector: &metav1.LabelSelector{},
						},
					},
				},
			}

			cluster = &v1beta1.Cluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:      clusterId,
					Namespace: fleetNs,
				},
				Spec: v1beta1.ClusterSpec{
					DisplayName: "My Cluster",
				},
			}

			c = fake.NewClientBuilder().
				WithScheme(scheme.Scheme).
				WithStatusSubresource(&fleetv1alpha1.GitRepo{}).
				WithStatusSubresource(&v1beta1.Deployment{}).
				Build()
			r = &ClusterReconciler{
				Client:               c,
				Scheme:               scheme.Scheme,
				proxyDomain:          "test-proxyDomain",
				proxyServerURL:       "test-proxyServerURL",
				agentRegistry:        "test-agentRegistry",
				agentChart:           "test-agentChart",
				agentChartVersion:    "test-agentChartVersion",
				agentChartRepoSecret: "test-agentChartRepoSecret",
				agentTargetNamespace: "test-agentTargetNamespace",
				gitRepoName:          "test-gitRepoName",
				gitRemoteRepoName:    "test-gitRemoteRepoName",
				gitRepoURL:           repoURL,
				gitClientSecret:      "test-gitClientSecret",
			}

			os.Setenv("GIT_PROVIDER", "test-provider")
			os.Setenv("GIT_PASSWORD", "test-password")
			os.Setenv("SECRET_SERVICE_ENABLED", "false")
			os.Setenv("FLEET_GIT_REMOTE_TYPE", "test-gitRemoteType")
			os.Setenv("GIT_SERVER", "http://test-gitRepoURL")
		})

		It("successfully update Git URL due to change", func() {
			// First time, no gitrepo so create it
			err := r.reconcileFleetGitRepo(ctx, cluster)
			Expect(err).ToNot(HaveOccurred())
			Expect(r.Create(ctx, &gitRepo)).To(Succeed())

			dep := fleetv1alpha1.GitRepo{}
			key := client.ObjectKey{
				Name:      aspName,
				Namespace: fleetNs,
			}

			Expect(r.Get(ctx, key, &dep)).To(Succeed())

			// Change Git Server URL
			os.Setenv("GIT_SERVER", "http://new-fake-server")

			// Second time, URL change detect so update
			err = r.reconcileFleetGitRepo(ctx, cluster)
			Expect(err).ToNot(HaveOccurred())

			Expect(r.Get(ctx, key, &dep)).To(Succeed())
			Expect(r.Update(ctx, &dep)).To(Succeed())
		})

		It("fails due to gitrepo not found", func() {
			dep := fleetv1alpha1.GitRepo{}
			key := client.ObjectKey{
				Name:      "not-exist",
				Namespace: fleetNs,
			}

			err := r.reconcileFleetGitRepo(ctx, cluster)
			Expect(err).ToNot(HaveOccurred())
			err = r.Get(ctx, key, &dep)

			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("gitrepos.fleet.cattle.io \"not-exist\" not found"))
		})

		It("fails to get GetRemoteURLWithCreds due to GIT_SERVER not set", func() {
			os.Unsetenv("GIT_SERVER")
			dep := fleetv1alpha1.GitRepo{}
			key := client.ObjectKey{
				Name:      aspName,
				Namespace: fleetNs,
			}

			// First time, no gitrepo so create it
			err := r.reconcileFleetGitRepo(ctx, cluster)
			Expect(err).ToNot(HaveOccurred())

			Expect(r.Create(ctx, &gitRepo)).To(BeNil())

			err = r.Get(ctx, key, &dep)
			Expect(err).ToNot(HaveOccurred())

			// Second time, error due to env var not set
			err = r.reconcileFleetGitRepo(ctx, cluster)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("GIT_SERVER env var not set"))
		})

		It("fails to create gitrepo", func() {
			err := r.reconcileFleetGitRepo(ctx, cluster)
			Expect(err).ToNot(HaveOccurred())
			gitRepo.ObjectMeta.Name = ""

			err = r.Create(ctx, &gitRepo)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal(" \"\" is invalid: metadata.name: Required value: name is required"))
		})
	})
})
