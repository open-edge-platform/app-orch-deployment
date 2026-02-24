// SPDX-FileCopyrightText: (C) 2024 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package deployment

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	nexusApi "github.com/open-edge-platform/orch-utils/tenancy-datamodel/build/apis/runtimeproject.edge-orchestrator.intel.com/v1"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	nexusFake "github.com/open-edge-platform/orch-utils/tenancy-datamodel/build/client/clientset/versioned/fake"
	fleetv1alpha1 "github.com/rancher/fleet/pkg/apis/fleet.cattle.io/v1alpha1"
	"github.com/rancher/wrangler/v3/pkg/genericcondition"
	batchv1 "k8s.io/api/batch/v1"
	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/record"

	"github.com/open-edge-platform/app-orch-deployment/app-deployment-manager/api/v1beta1"
	"github.com/open-edge-platform/app-orch-deployment/app-deployment-manager/pkg/gitclient"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	cutil "sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// These Repository objects are used to test recovery from various errors
type mockRepository struct {
	ExistsOnRemoteValue bool
	ExistsOnRemoteError error
	InitializeError     error
	CloneError          error
	CommitFilesError    error
	PushToRemoteError   error
	DeleteError         error
	CreateRepoError     error
	DeleteRepoError     error
}

func (m *mockRepository) ExistsOnRemote() (bool, error) {
	_, _ = fmt.Fprintf(GinkgoWriter, "[DEBUG] ExistsOnRemote returning %v, %v\n", m.ExistsOnRemoteValue, m.ExistsOnRemoteError)
	return m.ExistsOnRemoteValue, m.ExistsOnRemoteError
}
func (m *mockRepository) Initialize(basedir string) error {
	_, _ = fmt.Fprintf(GinkgoWriter, "[DEBUG] Initialize returning %v\n", m.InitializeError)
	return m.InitializeError
}
func (m *mockRepository) Clone(basedir string) error {
	_, _ = fmt.Fprintf(GinkgoWriter, "[DEBUG] Clone returning %v\n", m.CloneError)
	return m.CloneError
}
func (m *mockRepository) CommitFiles() error {
	_, _ = fmt.Fprintf(GinkgoWriter, "[DEBUG] CommitFiles returning %v\n", m.CommitFilesError)
	return m.CommitFilesError
}
func (m *mockRepository) PushToRemote() error {
	_, _ = fmt.Fprintf(GinkgoWriter, "[DEBUG] PushToRemote returning %v\n", m.PushToRemoteError)
	return m.PushToRemoteError
}
func (m *mockRepository) Delete() error {
	_, _ = fmt.Fprintf(GinkgoWriter, "[DEBUG] PushToRemote returning %v\n", m.DeleteError)
	return m.DeleteError
}

func checkWordpressGitRepo(d v1beta1.Deployment, gr fleetv1alpha1.GitRepo) {
	Expect(gr.Name).To(Equal("wordpress-216e7223-1932-4df6-a6c7-828c84479726"))
	Expect(gr.Namespace).To(Equal(d.Namespace))

	// Check Owner reference
	Expect(len(gr.OwnerReferences)).To(Equal(1))
	or := gr.OwnerReferences[0]
	Expect(or.APIVersion).To(Equal("app.edge-orchestrator.intel.com/v1beta1"))
	Expect(or.Kind).To(Equal("Deployment"))
	Expect(or.Name).To(Equal(d.Name))
	Expect(or.UID).To(Equal(d.UID))
	Expect(*or.Controller).To(BeTrue())
	Expect(*or.BlockOwnerDeletion).To(BeTrue())

	// Check Labels
	Expect(gr.Labels[string(v1beta1.BundleName)]).To(Equal("b-1bc91be2-4cba-57"))
	Expect(gr.Labels[string(v1beta1.BundleType)]).To(Equal("app"))

	// Check Spec
	gitServer := os.Getenv("GIT_SERVER")
	Expect(gr.Spec.Repo).To(Equal(gitServer + "/git-user/216e7223-1932-4df6-a6c7-828c84479726.git"))
	Expect(gr.Spec.TargetNamespace).To(BeEmpty()) // Namespace is only specified in fleet.yaml
	Expect(gr.Spec.Paths).To(Equal([]string{"wordpress"}))
	Expect(gr.Spec.Targets[0].ClusterSelector.MatchLabels).To(Equal(map[string]string{"color": "blue"}))
	Expect(gr.Spec.HelmSecretName).To(Equal("wordpress-0.1.0-helmrepo"))
}

func checkNginxGitRepo(d v1beta1.Deployment, gr fleetv1alpha1.GitRepo) {
	Expect(gr.Name).To(Equal("nginx-216e7223-1932-4df6-a6c7-828c84479726"))
	Expect(gr.Namespace).To(Equal(d.Namespace))

	// Check Owner reference
	Expect(len(gr.OwnerReferences)).To(Equal(1))
	or := gr.OwnerReferences[0]
	Expect(or.APIVersion).To(Equal("app.edge-orchestrator.intel.com/v1beta1"))
	Expect(or.Kind).To(Equal("Deployment"))
	Expect(or.Name).To(Equal(d.Name))
	Expect(or.UID).To(Equal(d.UID))
	Expect(*or.Controller).To(BeTrue())
	Expect(*or.BlockOwnerDeletion).To(BeTrue())

	// Check Labels
	Expect(gr.Labels[string(v1beta1.BundleName)]).To(Equal("b-3866e0e6-bb7b-52"))
	Expect(gr.Labels[string(v1beta1.BundleType)]).To(Equal("app"))

	// Check Spec
	gitServer := os.Getenv("GIT_SERVER")
	Expect(gr.Spec.Repo).To(Equal(gitServer + "/git-user/216e7223-1932-4df6-a6c7-828c84479726.git"))
	Expect(gr.Spec.TargetNamespace).To(BeEmpty()) // Namespace is only specified in fleet.yaml
	Expect(gr.Spec.Paths).To(Equal([]string{"nginx"}))
	Expect(gr.Spec.Targets[0].ClusterSelector.MatchLabels).To(Equal(map[string]string{"color": "blue"}))
	Expect(gr.Spec.HelmSecretName).To(Equal("nginx-0.1.0-helmrepo"))
}

func checkGitRepos(d v1beta1.Deployment, grlist []fleetv1alpha1.GitRepo) {
	for i := range grlist {
		gr := grlist[i]
		switch gr.Name {
		case "wordpress-216e7223-1932-4df6-a6c7-828c84479726":
			checkWordpressGitRepo(d, gr)
		case "nginx-216e7223-1932-4df6-a6c7-828c84479726":
			checkNginxGitRepo(d, gr)
		default:
			Fail("unknown GitRepo object")
		}
	}
}

var _ = Describe("Deployment controller", func() {

	const (
		name            = "mydeployment"
		namespace       = "fleet-default"
		uid             = "216e7223-1932-4df6-a6c7-828c84479726"
		activeProjectId = "a563356a-b4df-47bb-b620-aae2d74c5129"
	)
	var (
		ctx context.Context
		c   client.WithWatch
		r   *Reconciler
		d   v1beta1.Deployment
		m   mockRepository

		wordpressProfile   v1.Secret
		wordpressOverrides v1.Secret
		nginxProfile       v1.Secret
		nginxOverrides     v1.Secret
	)

	BeforeEach(func() {
		ctx = context.Background()
		d = v1beta1.Deployment{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "app.edge-orchestrator.intel.com/v1beta1",
				Kind:       "Deployment",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: namespace,
				Labels: map[string]string{
					string(v1beta1.AppOrchActiveProjectID): activeProjectId,
				},
				UID:        uid,
				Generation: 1,
				CreationTimestamp: metav1.Time{
					Time: time.Now(),
				},
				Finalizers: []string{v1beta1.FinalizerDependency},
			},
			Spec: v1beta1.DeploymentSpec{
				DisplayName: "My Wordpress Blog",
				Project:     "test-project",
				DeploymentPackageRef: v1beta1.DeploymentPackageRef{
					Name:        "wordpress",
					Version:     "0.1.0",
					ProfileName: "default",
				},
				Applications: []v1beta1.Application{
					{
						Name:      "wordpress",
						Version:   "0.1.0",
						Namespace: "apps0",
						Targets: []map[string]string{{
							"color": "blue",
						},
						},
						ProfileSecretName: "wordpress-0.1.0-profile",
						ValueSecretName:   "wordpress-0.1.0-overrides",
						HelmApp: &v1beta1.HelmApp{
							Chart:          "wordpress",
							Version:        "15.2.42",
							Repo:           "https://charts.bitnami.com/bitnami",
							RepoSecretName: "wordpress-0.1.0-helmrepo",
						},
					},
					{
						Name:      "nginx",
						Version:   "0.1.0",
						Namespace: "apps1",
						Targets: []map[string]string{{
							"color": "blue",
						},
						},
						ProfileSecretName: "nginx-0.1.0-profile",
						ValueSecretName:   "nginx-0.1.0-overrides",
						HelmApp: &v1beta1.HelmApp{
							Chart:          "nginx",
							Version:        "15.4.5",
							Repo:           "https://charts.bitnami.com/bitnami",
							RepoSecretName: "nginx-0.1.0-helmrepo",
						},
					},
				},
				DeploymentType: "auto-scaling",
			},
		}
		m = mockRepository{
			ExistsOnRemoteValue: false,
			ExistsOnRemoteError: nil,
			InitializeError:     nil,
			CloneError:          nil,
			CommitFilesError:    nil,
			PushToRemoteError:   nil,
			DeleteError:         nil,
		}
		c = fake.NewClientBuilder().
			WithScheme(scheme.Scheme).
			WithIndex(&fleetv1alpha1.GitRepo{}, ownerKey, gitRepoIdxFunc).
			WithIndex(&batchv1.Job{}, jobOwnerKey, jobIdxFunc).
			WithStatusSubresource(&v1beta1.Deployment{}).
			Build()
		r = &Reconciler{
			Client:   c,
			Scheme:   scheme.Scheme,
			recorder: record.NewFakeRecorder(1024),
			gitclient: func(uuid string) (gitclient.Repository, error) {
				return &m, nil
			},
			nexusclient: nexusFake.NewSimpleClientset(
				&nexusApi.RuntimeProject{
					TypeMeta: metav1.TypeMeta{
						Kind:       "runtimeproject.edge-orchestrator.intel.com",
						APIVersion: "v1",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name: "mock-runtime-project-hash-12345",
						Labels: map[string]string{
							"multitenancies.tenancy.edge-orchestrator.intel.com": "default",
							"nexus/display_name":   "mock-project-1-in-mock-org-1",
							"nexus/is_name_hashed": "true",
							"runtimefolders.runtimefolder.edge-orchestrator.intel.com": "default",
							"runtimeorgs.runtimeorg.edge-orchestrator.intel.com":       "mock-org-1",
							"runtimes.runtime.edge-orchestrator.intel.com":             "default",
						},
						UID: "a563356a-b4df-47bb-b620-aae2d74c5129",
					},
					Spec:   nexusApi.RuntimeProjectSpec{},
					Status: nexusApi.RuntimeProjectNexusStatus{},
				},
			),
		}
		wordpressProfile = v1.Secret{
			TypeMeta: metav1.TypeMeta{
				Kind:       "Secret",
				APIVersion: "v1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      "wordpress-0.1.0-profile",
				Namespace: namespace,
			},
			Data: map[string][]byte{},
			StringData: map[string]string{
				"foo": "bar",
			},
			Type: v1.SecretTypeOpaque,
		}
		wordpressOverrides = v1.Secret{
			TypeMeta: metav1.TypeMeta{
				Kind:       "Secret",
				APIVersion: "v1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      "wordpress-0.1.0-overrides",
				Namespace: namespace,
			},
			Data: map[string][]byte{},
			StringData: map[string]string{
				"foo": "bar",
			},
			Type: v1.SecretTypeOpaque,
		}
		nginxProfile = v1.Secret{
			TypeMeta: metav1.TypeMeta{
				Kind:       "Secret",
				APIVersion: "v1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      "nginx-0.1.0-profile",
				Namespace: namespace,
			},
			Data: map[string][]byte{},
			StringData: map[string]string{
				"foo": "bar",
			},
			Type: v1.SecretTypeOpaque,
		}
		nginxOverrides = v1.Secret{
			TypeMeta: metav1.TypeMeta{
				Kind:       "Secret",
				APIVersion: "v1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      "nginx-0.1.0-overrides",
				Namespace: namespace,
			},
			Data: map[string][]byte{},
			StringData: map[string]string{
				"foo": "bar",
			},
			Type: v1.SecretTypeOpaque,
		}
		os.Setenv("FLEET_GIT_REMOTE_TYPE", "https")
		os.Setenv("GIT_SERVER", "http://fake-server")
		os.Setenv("GIT_PROVIDER", "gitea")
		os.Setenv("GIT_USER", "git-user")
		os.Setenv("GIT_PASSWORD", "git-password")
		os.Setenv("SECRET_SERVICE_ENABLED", "false")
		os.Setenv("GIT_CA_CERT", "LS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0tLS0tCk1JSUZWekNDQXorZ0F3SUJBZ0lVSXpKVkNBVnIxcWU1akRoZ0drM25QMWJEcitnd0RRWUpLb1pJaHZjTkFRRUwKQlFBd0dqRVlNQllHQTFVRUF3d1BLaTVyYVc1a0xtbHVkR1Z5Ym1Gc01CNFhEVEkwTURneU1qRTNNamN4TWxvWApEVEkxTURneU1qRTNNamN4TWxvd0dqRVlNQllHQTFVRUF3d1BLaTVyYVc1a0xtbHVkR1Z5Ym1Gc01JSUNJakFOCkJna3Foa2lHOXcwQkFRRUZBQU9DQWc4QU1JSUNDZ0tDQWdFQWpMNlVpekNhT2V1M0Q0TTJvQkUwZnltNjR0U3MKOXRwUE53dkFnVnpEeE1UOWJoTWhycEJZUk5vd2R3cXBtekJMUXZSUTIrbmJ3UDYxcXlQK2g4QzZzWTdNRCtRQgpCaHlEOVdISGJ1RUErZHVoc3J1SGxwL21EdGNvcS93NGZBR090aHI0K0tDQ0hMa1ZBYnVsQlNmUnN0eWRGbUhCCmtxT0hYamhkMGNBUnZWekUzL2dQQkVkUFVUbkN1R2xrcGxGZTdNL0pEVE9qMmF0Uis0R2FORTZpSFZCOWh6OXQKcFpHczJZaUNDL3g0TlFBZ0NseVBjSUViaFdINDAwTnV0ek5xdVdEQ2RES1ptRWJNQ3NTYmdZN1d1YVRKeWlCSApnYk10VHJUSitZNW1vajdaZ1RyUXJ5OE1IVUQzc2RUVU5HV09oUU96bXQrSzlzbWw5aVFNbE85THJUa0dyMlJHCnZtU0plQ2xmM3JXZ1VNSFVqdmZrdFNmU0gxeTc4VFdNbzVWejFGd3c1YU90VTIrT3FnbHJtZ1ZhQU9URXkvcDQKVUNMdjk5NDBybVRqRGRvU3hObWd0MG5IVFVnK1ZEdm02cHhXOTZQU1hQcW9CczYydlljYUxDd1AzT0RqVjNiZwp3ZEpOQlJYVzVoMUUyNEY3OFhacDdaNEcxNkhWcWJFTFdlOHhZYUpUMC9ZR1I5NU9Qbmw1c053bm1lSXRkcWFYCnJXbXdPeHdmbHVjUVBkUXNTb1cyaUlUbTlPSDhsQUIxK3VLZnM5TldlaXY5SnA4bU9CbWJiOU80ZEo3M0tjdU4KL3VqMDBZNEtLaG5PMlhqcmJnY1dKazMrblpQQlRaSFcrcTU3ajRMMlplYW5ubmZ1dWowYjNpUFFSbHJlVldWNgp6STlHeEF5NExBYnVPSE1DQXdFQUFhT0JsRENCa1RBZEJnTlZIUTRFRmdRVVdCanRGRkxzSTdnZE1XcGx5V3p4ClBDYVgvMk13SHdZRFZSMGpCQmd3Rm9BVVdCanRGRkxzSTdnZE1XcGx5V3p4UENhWC8yTXdEd1lEVlIwVEFRSC8KQkFVd0F3RUIvekErQmdOVkhSRUVOekExZ2c4cUxtdHBibVF1YVc1MFpYSnVZV3lDSW1kcGRHVmhMV2gwZEhBdQpaMmwwWldFdWMzWmpMbU5zZFhOMFpYSXViRzlqWVd3d0RRWUpLb1pJaHZjTkFRRUxCUUFEZ2dJQkFDMk9pYVVjCkNQc1pyVnkyYzNoUTFFcmFlQ1pxaUloeFEybVFjSFpPeUdoS2JrS2o2N1N3Q2VYRzBVaE5yOGtXR2plWEJUN1cKeWkvSzgxYnc2UmgxQm1Na2RlS01hUFRYK2pFeVJaQ0lzOXJNVEF0R1NTS092UWNZRUtUbmd2SEpQY0ZtazJzUQpockE0UVdETTB0V0wzejNBNDJ6a2xIV0Ixbk9zbmhwcTUyc016QkFydVVhSXdxUEtTN3U5YUJGRThDSTQrd2x1CnB3eHd3emg1ODF6UXo3TXMvSzA1U3FQdkdOcGd2M0pQRTJtTCtoTE1HaUJQSW5PRnlCWFd3Zlh1TGllM1llN0YKNW5RRlB5MkQ0eUZ3Y0pia3I1RFlTcTZZdldSVzhOVVBtYXZlNkdKdjZvM0NWT0RhOHAvaTBkdmhwYTNTK3BlNgoyRXpGdFk0Si9CSHN4UVQwZjFFTmZtSVN6eHJJTU5yZ3p0d2Q3a3NBeDBkaTd2VXFiSjg3VG15aGpuaDB0VnBjClFmUlovWmNRWVZ4RmQxOFRkN1FpZG5ZbDR5ekttWHdMYkxSVGJUWGVIZTQvckRVN1dobmE2SnFzNEV4TTk5UEIKbnRneWNxbmxOckI1NXZHTVlCT2VNL1Uya1ArZU9aeGJJMXNzVUJ3ZFZ4ZEdxT3llS0M3ZUdkNHluaGtHSzNuago5TFBlUmtvWk41UkZjdTd3ekN3VFF2NWtsM3VHUncvbzBKcjdUNlZmYTVzNFVWRkY4MmdzeXdscFk3Vk9HMUpOCmljd0FQd2NkMVIyc0Rpc3BjQjdiR29GU1hpRlkybFU3eVdrOCtwVUExdWRCK2RwWWs4U2tWMFZ4cnJMeVVyMkkKTmduZjg5TzU5ZFJ6dUQ2MDg5WnhsQ3hMazZzVENLSWN6QVVvCi0tLS0tRU5EIENFUlRJRklDQVRFLS0tLS0K")
	})

	When("reconciling a Deployment", func() {

		Context("Deployment is not present", func() {
			It("returns with no error", func() {
				res, err := r.Reconcile(ctx, reconcile.Request{
					NamespacedName: types.NamespacedName{
						Namespace: namespace,
						Name:      name,
					},
				})
				Expect(res).To(Equal(reconcile.Result{}))
				Expect(err).To(BeNil())
			})
		})

		Context("FinalizerGitRemote testing", func() {
			BeforeEach(func() {
				r.deleteGitRepo = true
				d.Status.DeployInProgress = true
				m.ExistsOnRemoteValue = true
				Expect(r.Create(ctx, &d)).To(BeNil())
			})
			It("sets and removes the finalizer", func() {
				res, err := r.Reconcile(ctx, reconcile.Request{
					NamespacedName: types.NamespacedName{
						Namespace: namespace,
						Name:      name,
					},
				})
				Expect(res).To(Equal(reconcile.Result{}))
				Expect(err).To(BeNil())

				dep := v1beta1.Deployment{}
				key := client.ObjectKey{
					Name:      name,
					Namespace: namespace,
				}
				Expect(r.Get(ctx, key, &dep)).To(Succeed())
				Expect(dep.Status.State).To(Equal(v1beta1.Deploying))
				Expect(cutil.ContainsFinalizer(&dep, v1beta1.FinalizerGitRemote)).To(BeTrue())

				Expect(r.Delete(ctx, &dep)).To(Succeed())
				res, err = r.Reconcile(ctx, reconcile.Request{
					NamespacedName: types.NamespacedName{
						Namespace: namespace,
						Name:      name,
					},
				})
				Expect(res).To(Equal(reconcile.Result{}))

				// Since automatic deletion was removed, we expect successful reconciliation
				Expect(err).To(BeNil())

				// Finalizer was removed so Kubernetes completes the deletion - resource no longer exists
				Expect(r.Get(ctx, key, &dep)).ToNot(Succeed())
			})
		})

		Context("Generation matches ReconciledGeneration", func() {
			BeforeEach(func() {
				d.Status.ReconciledGeneration = d.Generation
				d.Status.DeployInProgress = true
				d.Status.Conditions = []metav1.Condition{
					{
						Type:   "Ready",
						Status: "True",
					},
				}
				Expect(r.Create(ctx, &d)).To(BeNil())
			})
			It("updates the Deployment status only", func() {
				res, err := r.Reconcile(ctx, reconcile.Request{
					NamespacedName: types.NamespacedName{
						Namespace: namespace,
						Name:      name,
					},
				})
				Expect(res).To(Equal(reconcile.Result{}))
				Expect(err).To(BeNil())

				dep := v1beta1.Deployment{}
				key := client.ObjectKey{
					Name:      name,
					Namespace: namespace,
				}

				Expect(r.Get(ctx, key, &dep)).To(Succeed())
				Expect(dep.Status.State).To(Equal(v1beta1.Deploying))

				grlist := fleetv1alpha1.GitRepoList{}
				Expect(r.List(ctx, &grlist)).To(BeNil())
				Expect(len(grlist.Items)).To(BeZero())
			})
		})

		Context("Git Server URL has changed", func() {
			BeforeEach(func() {
				d.Status.ReconciledGeneration = d.Generation
				Expect(r.Create(ctx, &d)).To(BeNil())
				Expect(r.Create(ctx, &wordpressProfile)).To(BeNil())
				Expect(r.Create(ctx, &wordpressOverrides)).To(BeNil())
				Expect(r.Create(ctx, &nginxProfile)).To(BeNil())
				Expect(r.Create(ctx, &nginxOverrides)).To(BeNil())
			})
			It("updates the appropriate GitRepo resources", func() {
				res, err := r.Reconcile(ctx, reconcile.Request{
					NamespacedName: types.NamespacedName{
						Namespace: namespace,
						Name:      name,
					},
				})
				Expect(res).To(Equal(reconcile.Result{}))
				Expect(err).To(BeNil())

				dep := v1beta1.Deployment{}
				key := client.ObjectKey{
					Name:      name,
					Namespace: namespace,
				}
				Expect(r.Get(ctx, key, &dep)).To(Succeed())
				Expect(dep.Status.State).To(Equal(v1beta1.Deploying))
				Expect(dep.Status.ReconciledGeneration).To(Equal(dep.Generation))
				grlist := fleetv1alpha1.GitRepoList{}
				Expect(r.List(ctx, &grlist)).To(Succeed())
				Expect(len(grlist.Items)).To(Equal(2))

				checkGitRepos(d, grlist.Items)

				// Change Git Server URL
				os.Setenv("GIT_SERVER", "http://new-fake-server")

				// Run Reconcile again
				res, err = r.Reconcile(ctx, reconcile.Request{
					NamespacedName: types.NamespacedName{
						Namespace: namespace,
						Name:      name,
					},
				})
				Expect(res).To(Equal(reconcile.Result{}))
				Expect(err).To(BeNil())

				Expect(r.Get(ctx, key, &dep)).To(Succeed())
				Expect(dep.Status.State).To(Equal(v1beta1.Deploying))
				Expect(dep.Status.ReconciledGeneration).To(Equal(dep.Generation))
				grlist = fleetv1alpha1.GitRepoList{}
				Expect(r.List(ctx, &grlist)).To(Succeed())
				Expect(len(grlist.Items)).To(Equal(2))

				checkGitRepos(d, grlist.Items)
			})
		})

		Context("Generation does not match ReconciledGeneration", func() {
			BeforeEach(func() {
				d.Status.ReconciledGeneration = d.Generation - 1
				Expect(r.Create(ctx, &d)).To(BeNil())
				Expect(r.Create(ctx, &wordpressProfile)).To(BeNil())
				Expect(r.Create(ctx, &wordpressOverrides)).To(BeNil())
				Expect(r.Create(ctx, &nginxProfile)).To(BeNil())
				Expect(r.Create(ctx, &nginxOverrides)).To(BeNil())
			})
			It("creates / updates the appropriate GitRepo resources", func() {
				res, err := r.Reconcile(ctx, reconcile.Request{
					NamespacedName: types.NamespacedName{
						Namespace: namespace,
						Name:      name,
					},
				})
				Expect(res).To(Equal(reconcile.Result{}))
				Expect(err).To(BeNil())

				dep := v1beta1.Deployment{}
				key := client.ObjectKey{
					Name:      name,
					Namespace: namespace,
				}
				Expect(r.Get(ctx, key, &dep)).To(Succeed())
				Expect(dep.Status.State).To(Equal(v1beta1.Deploying))
				Expect(dep.Status.ReconciledGeneration).To(Equal(dep.Generation))
				grlist := fleetv1alpha1.GitRepoList{}
				Expect(r.List(ctx, &grlist)).To(Succeed())
				Expect(len(grlist.Items)).To(Equal(2))

				checkGitRepos(d, grlist.Items)

				// Make small change to Deployment and update Generation
				dep.Spec.Applications[0].HelmApp.Version = "15.2.43"
				dep.Generation++
				Expect(r.Update(ctx, &dep)).To(BeNil())

				// Run Reconcile again
				res, err = r.Reconcile(ctx, reconcile.Request{
					NamespacedName: types.NamespacedName{
						Namespace: namespace,
						Name:      name,
					},
				})
				Expect(res).To(Equal(reconcile.Result{}))
				Expect(err).To(BeNil())

				Expect(r.Get(ctx, key, &dep)).To(Succeed())
				Expect(dep.Status.State).To(Equal(v1beta1.Deploying))
				Expect(dep.Status.ReconciledGeneration).To(Equal(dep.Generation))
				grlist = fleetv1alpha1.GitRepoList{}
				Expect(r.List(ctx, &grlist)).To(Succeed())
				Expect(len(grlist.Items)).To(Equal(2))

				checkGitRepos(d, grlist.Items)

			})
			It("deletes / adds the appropriate GitRepo resources when app number changes", func() {
				res, err := r.Reconcile(ctx, reconcile.Request{
					NamespacedName: types.NamespacedName{
						Namespace: namespace,
						Name:      name,
					},
				})
				Expect(res).To(Equal(reconcile.Result{}))
				Expect(err).To(BeNil())

				dep := v1beta1.Deployment{}
				key := client.ObjectKey{
					Name:      name,
					Namespace: namespace,
				}
				Expect(r.Get(ctx, key, &dep)).To(Succeed())
				Expect(dep.Status.State).To(Equal(v1beta1.Deploying))
				Expect(dep.Status.ReconciledGeneration).To(Equal(dep.Generation))
				grlist := fleetv1alpha1.GitRepoList{}
				Expect(r.List(ctx, &grlist)).To(Succeed())
				Expect(len(grlist.Items)).To(Equal(2))

				checkGitRepos(d, grlist.Items)

				// Delete NGINX app from Spec
				dep.Spec.Applications = []v1beta1.Application{
					d.Spec.Applications[0],
				}
				dep.Generation++
				Expect(r.Update(ctx, &dep)).To(BeNil())

				// Run Reconcile again
				res, err = r.Reconcile(ctx, reconcile.Request{
					NamespacedName: types.NamespacedName{
						Namespace: namespace,
						Name:      name,
					},
				})
				Expect(res).To(Equal(reconcile.Result{}))
				Expect(err).To(BeNil())

				Expect(r.Get(ctx, key, &dep)).To(Succeed())
				Expect(dep.Status.State).To(Equal(v1beta1.Deploying))
				Expect(dep.Status.ReconciledGeneration).To(Equal(dep.Generation))
				grlist = fleetv1alpha1.GitRepoList{}
				Expect(r.List(ctx, &grlist)).To(Succeed())
				Expect(len(grlist.Items)).To(Equal(1))

				checkGitRepos(d, grlist.Items)

				// Delete NGINX app from Spec
				dep.Spec.Applications = []v1beta1.Application{
					d.Spec.Applications[0],
				}
				dep.Generation++
				Expect(r.Update(ctx, &dep)).To(BeNil())

				// Run Reconcile again
				res, err = r.Reconcile(ctx, reconcile.Request{
					NamespacedName: types.NamespacedName{
						Namespace: namespace,
						Name:      name,
					},
				})
				Expect(res).To(Equal(reconcile.Result{}))
				Expect(err).To(BeNil())

				Expect(r.Get(ctx, key, &dep)).To(Succeed())
				Expect(dep.Status.State).To(Equal(v1beta1.Deploying))
				Expect(dep.Status.ReconciledGeneration).To(Equal(dep.Generation))
				grlist = fleetv1alpha1.GitRepoList{}
				Expect(r.List(ctx, &grlist)).To(Succeed())
				Expect(len(grlist.Items)).To(Equal(1))

				checkGitRepos(d, grlist.Items)

				// Restore NGINX app to Spec
				dep.Spec.Applications = d.Spec.Applications
				dep.Generation++
				Expect(r.Update(ctx, &dep)).To(BeNil())

				// Run Reconcile again
				res, err = r.Reconcile(ctx, reconcile.Request{
					NamespacedName: types.NamespacedName{
						Namespace: namespace,
						Name:      name,
					},
				})
				Expect(res).To(Equal(reconcile.Result{}))
				Expect(err).To(BeNil())

				Expect(r.Get(ctx, key, &dep)).To(Succeed())
				Expect(dep.Status.State).To(Equal(v1beta1.Deploying))
				Expect(dep.Status.ReconciledGeneration).To(Equal(dep.Generation))
				grlist = fleetv1alpha1.GitRepoList{}
				Expect(r.List(ctx, &grlist)).To(Succeed())
				Expect(len(grlist.Items)).To(Equal(2))

				checkGitRepos(d, grlist.Items)
			})
		})

		Context("A Deployment with no clusters", func() {
			BeforeEach(func() {
				Expect(r.Create(ctx, &wordpressProfile)).To(BeNil())
				Expect(r.Create(ctx, &wordpressOverrides)).To(BeNil())
				Expect(r.Create(ctx, &nginxProfile)).To(BeNil())
				Expect(r.Create(ctx, &nginxOverrides)).To(BeNil())
			})
			It("a new one should be in Deploying state", func() {
				d.ObjectMeta.CreationTimestamp.Time = time.Now()
				Expect(r.Create(ctx, &d)).To(BeNil())

				res, err := r.Reconcile(ctx, reconcile.Request{
					NamespacedName: types.NamespacedName{
						Namespace: namespace,
						Name:      name,
					},
				})
				Expect(res).To(Equal(reconcile.Result{}))
				Expect(err).To(BeNil())

				dep := v1beta1.Deployment{}
				key := client.ObjectKey{
					Name:      name,
					Namespace: namespace,
				}
				Expect(r.Get(ctx, key, &dep)).To(Succeed())
				Expect(dep.Status.State).To(Equal(v1beta1.Deploying))
			})
			It("an older one should be in NoTargetClusters state", func() {
				d.ObjectMeta.CreationTimestamp.Time = time.Now().Add(-10 * time.Minute)
				Expect(r.Create(ctx, &d)).To(BeNil())

				res, err := r.Reconcile(ctx, reconcile.Request{
					NamespacedName: types.NamespacedName{
						Namespace: namespace,
						Name:      name,
					},
				})
				Expect(res).To(Equal(reconcile.Result{}))
				Expect(err).To(BeNil())

				dep := v1beta1.Deployment{}
				key := client.ObjectKey{
					Name:      name,
					Namespace: namespace,
				}
				Expect(r.Get(ctx, key, &dep)).To(Succeed())
				Expect(dep.Status.State).To(Equal(v1beta1.NoTargetClusters))
			})
		})
	})

	When("reconciling the Git repository", func() {

		Context("remote does not exist, all operations succeed", func() {
			It("succeeds", func() {
				Expect(r.Create(ctx, &wordpressProfile)).To(BeNil())
				Expect(r.Create(ctx, &wordpressOverrides)).To(BeNil())
				Expect(r.Create(ctx, &nginxProfile)).To(BeNil())
				Expect(r.Create(ctx, &nginxOverrides)).To(BeNil())

				m.ExistsOnRemoteValue = false
				_, err := r.reconcileRepository(ctx, &d)
				Expect(err).To(BeNil())
				Expect(meta.IsStatusConditionTrue(d.Status.Conditions, "GitSynced")).To(BeTrue())
			})
		})

		Context("remote exists, all operations succeed", func() {
			It("succeeds", func() {
				Expect(r.Create(ctx, &wordpressProfile)).To(BeNil())
				Expect(r.Create(ctx, &wordpressOverrides)).To(BeNil())
				Expect(r.Create(ctx, &nginxProfile)).To(BeNil())
				Expect(r.Create(ctx, &nginxOverrides)).To(BeNil())

				m.ExistsOnRemoteValue = true
				_, err := r.reconcileRepository(ctx, &d)
				Expect(err).To(BeNil())
				Expect(meta.IsStatusConditionTrue(d.Status.Conditions, "GitSynced")).To(BeTrue())
			})
		})

		Context("NewGitClient() fails", func() {
			It("returns an error and sets condition appropriately", func() {
				message := "An error!"
				r.gitclient = func(uuid string) (gitclient.Repository, error) {
					return &m, errors.New(message)
				}
				_, err := r.reconcileRepository(ctx, &d)
				Expect(err).ToNot(BeNil())
				cond := meta.FindStatusCondition(d.Status.Conditions, "GitSynced")
				Expect(cond.Status).To(Equal(metav1.ConditionFalse))
				Expect(cond.Reason).To(Equal(reasonNewGitClientFailed))
				Expect(cond.Message).To(Equal(message))
			})
		})

		Context("ExistsOnRemote() fails", func() {
			It("returns an error and sets condition appropriately", func() {
				message := "An error!"
				m.ExistsOnRemoteError = errors.New(message)
				_, err := r.reconcileRepository(ctx, &d)
				Expect(err).ToNot(BeNil())
				cond := meta.FindStatusCondition(d.Status.Conditions, "GitSynced")
				Expect(cond.Status).To(Equal(metav1.ConditionFalse))
				Expect(cond.Reason).To(Equal(reasonGitRemoteCheckFailed))
				Expect(cond.Message).To(Equal(message))
			})
		})

		Context("Clone() fails", func() {
			It("returns an error and sets condition appropriately", func() {
				message := "An error!"
				m.ExistsOnRemoteValue = true
				m.CloneError = errors.New(message)
				_, err := r.reconcileRepository(ctx, &d)
				Expect(err).ToNot(BeNil())
				cond := meta.FindStatusCondition(d.Status.Conditions, "GitSynced")
				Expect(cond.Status).To(Equal(metav1.ConditionFalse))
				Expect(cond.Reason).To(Equal(reasonGitCloneFailed))
				Expect(cond.Message).To(Equal(message))
			})
		})

		Context("Initialize() fails", func() {
			It("returns an error and sets condition appropriately", func() {
				message := "An error!"
				m.ExistsOnRemoteValue = false
				m.InitializeError = errors.New(message)
				_, err := r.reconcileRepository(ctx, &d)
				Expect(err).ToNot(BeNil())
				cond := meta.FindStatusCondition(d.Status.Conditions, "GitSynced")
				Expect(cond.Status).To(Equal(metav1.ConditionFalse))
				Expect(cond.Reason).To(Equal(reasonGitInitializationFailed))
				Expect(cond.Message).To(Equal(message))
			})
		})

		Context("CommitFiles() fails", func() {
			It("returns an error and sets condition appropriately", func() {
				Expect(r.Create(ctx, &wordpressProfile)).To(BeNil())
				Expect(r.Create(ctx, &wordpressOverrides)).To(BeNil())
				Expect(r.Create(ctx, &nginxProfile)).To(BeNil())
				Expect(r.Create(ctx, &nginxOverrides)).To(BeNil())

				message := "An error!"
				m.CommitFilesError = errors.New(message)
				_, err := r.reconcileRepository(ctx, &d)
				Expect(err).ToNot(BeNil())
				cond := meta.FindStatusCondition(d.Status.Conditions, "GitSynced")
				Expect(cond.Status).To(Equal(metav1.ConditionFalse))
				Expect(cond.Reason).To(Equal(reasonGitCommitFailed))
				Expect(cond.Message).To(Equal(message))
			})
		})

		Context("Push() fails", func() {
			It("returns an error and sets condition appropriately", func() {
				Expect(r.Create(ctx, &wordpressProfile)).To(BeNil())
				Expect(r.Create(ctx, &wordpressOverrides)).To(BeNil())
				Expect(r.Create(ctx, &nginxProfile)).To(BeNil())
				Expect(r.Create(ctx, &nginxOverrides)).To(BeNil())

				message := "An error!"
				m.PushToRemoteError = errors.New(message)
				_, err := r.reconcileRepository(ctx, &d)
				Expect(err).ToNot(BeNil())
				cond := meta.FindStatusCondition(d.Status.Conditions, "GitSynced")
				Expect(cond.Status).To(Equal(metav1.ConditionFalse))
				Expect(cond.Reason).To(Equal(reasonGitPushFailed))
				Expect(cond.Message).To(Equal(message))
			})
		})
	})

	When("calculating Deployment status", func() {
		const (
			generation = 1
		)
		var (
			d        v1beta1.Deployment
			gr1, gr2 fleetv1alpha1.GitRepo
			dc1, dc2 v1beta1.DeploymentCluster
			c        client.WithWatch
			r        *Reconciler
		)

		BeforeEach(func() {
			d = v1beta1.Deployment{
				ObjectMeta: metav1.ObjectMeta{
					UID:        "12345",
					Generation: generation,
				},
			}
			gr1 = fleetv1alpha1.GitRepo{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "app1-12345",
					Namespace: "fleet-default",
					Labels:    map[string]string{},
				},
				Spec:   fleetv1alpha1.GitRepoSpec{},
				Status: fleetv1alpha1.GitRepoStatus{},
			}
			gr2 = fleetv1alpha1.GitRepo{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "app2-12345",
					Namespace: "fleet-default",
					Labels:    map[string]string{},
				},
				Spec:   fleetv1alpha1.GitRepoSpec{},
				Status: fleetv1alpha1.GitRepoStatus{},
			}
			dc1 = v1beta1.DeploymentCluster{
				ObjectMeta: metav1.ObjectMeta{
					Name: "deploymentcluster-1",
				},
				Status: v1beta1.DeploymentClusterStatus{
					Conditions: []metav1.Condition{
						{
							Type:   "Ready",
							Status: "True",
						},
					},
					Status: v1beta1.Status{
						State: v1beta1.Running,
					},
					Apps: []v1beta1.App{
						{
							Name:                 "app1",
							DeploymentGeneration: generation,
							Status: v1beta1.Status{
								State: v1beta1.Running,
							},
						},
						{
							Name:                 "app2",
							DeploymentGeneration: generation,
							Status: v1beta1.Status{
								State: v1beta1.Running,
							},
						},
					},
				},
			}
			dc2 = v1beta1.DeploymentCluster{
				ObjectMeta: metav1.ObjectMeta{
					Name: "deploymentcluster-2",
				},
				Status: v1beta1.DeploymentClusterStatus{
					Conditions: []metav1.Condition{
						{
							Type:   "Ready",
							Status: "True",
						},
					},
					Status: v1beta1.Status{
						State: v1beta1.Running,
					},
					Apps: []v1beta1.App{
						{
							Name:                 "app1",
							DeploymentGeneration: generation,
							Status: v1beta1.Status{
								State: v1beta1.Running,
							},
						},
						{
							Name:                 "app2",
							DeploymentGeneration: generation,
							Status: v1beta1.Status{
								State: v1beta1.Running,
							},
						},
					},
				},
			}

			idxfunc := func(rawObj client.Object) []string {
				return []string{"app2"}
			}
			c = fake.NewClientBuilder().
				WithScheme(scheme.Scheme).
				WithObjects(&gr1).
				WithIndex(&fleetv1alpha1.GitRepo{}, ".metadata.controller", idxfunc).
				Build()
			r = &Reconciler{
				Client:   c,
				recorder: record.NewFakeRecorder(10),
			}
		})

		Context("multiple apps but no deploymentclusters", func() {
			It("is in state NoTargetClusters", func() {
				d.Status.DeployInProgress = true
				grslice := []fleetv1alpha1.GitRepo{gr1, gr2}
				dcslice := []v1beta1.DeploymentCluster{}
				r.updateDeploymentStatus(&d, grslice, dcslice)
				Expect(d.Status.State).To(Equal(v1beta1.NoTargetClusters))
				Expect(d.Status.DeployInProgress).To(BeTrue())
				Expect(d.Status.Summary).To(Equal(v1beta1.ClusterSummary{
					Total:   0,
					Running: 0,
					Down:    0,
					Unknown: 0,
				}))
			})
		})

		Context("deploy in progress with multiple apps and one is stalled", func() {
			It("is in state Error with a message", func() {
				d.Status.DeployInProgress = true
				gr2.Status.Conditions = []genericcondition.GenericCondition{
					{
						Type:    "Stalled",
						Status:  v1.ConditionTrue,
						Message: "An error message",
					},
				}
				grslice := []fleetv1alpha1.GitRepo{gr1, gr2}
				dcslice := []v1beta1.DeploymentCluster{}
				r.updateDeploymentStatus(&d, grslice, dcslice)
				Expect(d.Status.State).To(Equal(v1beta1.Error))
				Expect(d.Status.Message).To(Equal("App app2: An error message"))
				Expect(d.Status.DeployInProgress).To(BeTrue())
				Expect(d.Status.Summary).To(Equal(v1beta1.ClusterSummary{
					Total:   0,
					Running: 0,
					Down:    0,
					Unknown: 0,
				}))
			})
		})

		Context("deploy in progress with multiple apps and one has Progress deadline exceeded", func() {
			It("is in state Error with a message", func() {
				d.Status.DeployInProgress = true
				dc2.Status.Status.State = v1beta1.Down
				dc2.Status.Status.Message = "Progress deadline exceeded"
				grslice := []fleetv1alpha1.GitRepo{gr1, gr2}
				dcslice := []v1beta1.DeploymentCluster{dc1, dc2}
				r.updateDeploymentStatus(&d, grslice, dcslice)
				Expect(d.Status.State).To(Equal(v1beta1.Error))
				Expect(d.Status.Message).To(Equal("Progress deadline exceeded"))
				Expect(d.Status.DeployInProgress).To(BeTrue())
				Expect(d.Status.Summary).To(Equal(v1beta1.ClusterSummary{
					Total:   2,
					Running: 1,
					Down:    1,
					Unknown: 0,
				}))
			})
		})

		Context("deployment in progress with multiple apps and all are running", func() {
			It("is in state Running", func() {
				d.Status.DeployInProgress = true
				grslice := []fleetv1alpha1.GitRepo{gr1, gr2}
				dcslice := []v1beta1.DeploymentCluster{dc1, dc2}
				r.updateDeploymentStatus(&d, grslice, dcslice)
				Expect(d.Status.State).To(Equal(v1beta1.Running))
				Expect(d.Status.DeployInProgress).To(BeFalse())
				Expect(d.Status.Summary).To(Equal(v1beta1.ClusterSummary{
					Total:   2,
					Running: 2,
					Down:    0,
					Unknown: 0,
				}))
			})
		})

		Context("deploy in progress with multiple apps and one is not running", func() {
			It("is in state Deploying with a message", func() {
				d.Status.DeployInProgress = true
				gr2.Status.Display.Message = "In progress"
				dc2.Status.Status.State = v1beta1.Down
				dc2.Status.Apps[1].Status.State = v1beta1.Down
				grslice := []fleetv1alpha1.GitRepo{gr1, gr2}
				dcslice := []v1beta1.DeploymentCluster{dc1, dc2}
				r.updateDeploymentStatus(&d, grslice, dcslice)
				Expect(d.Status.State).To(Equal(v1beta1.Deploying))
				Expect(d.Status.Message).To(Equal("App app2: In progress"))
				Expect(d.Status.DeployInProgress).To(BeTrue())
				Expect(d.Status.Summary).To(Equal(v1beta1.ClusterSummary{
					Total:   2,
					Running: 1,
					Down:    1,
					Unknown: 0,
				}))
			})
		})

		Context("deploy in progress with multiple apps and all are running but one has old generation", func() {
			It("is in state Updating", func() {
				d.Status.DeployInProgress = true
				d.Generation = generation + 1
				dc1.Status.Apps[0].DeploymentGeneration = generation + 1
				dc1.Status.Apps[1].DeploymentGeneration = generation + 1
				dc2.Status.Apps[0].DeploymentGeneration = generation + 1
				grslice := []fleetv1alpha1.GitRepo{gr1, gr2}
				dcslice := []v1beta1.DeploymentCluster{dc1, dc2}
				r.updateDeploymentStatus(&d, grslice, dcslice)
				Expect(d.Status.State).To(Equal(v1beta1.Updating))
				Expect(d.Status.DeployInProgress).To(BeTrue())
				Expect(d.Status.Summary).To(Equal(v1beta1.ClusterSummary{
					Total:   2,
					Running: 1,
					Down:    1,
					Unknown: 0,
				}))
			})
		})

		Context("multiple apps and all are running", func() {
			It("is in state Running", func() {
				d.Status.DeployInProgress = false
				grslice := []fleetv1alpha1.GitRepo{gr1, gr2}
				dcslice := []v1beta1.DeploymentCluster{dc1, dc2}
				r.updateDeploymentStatus(&d, grslice, dcslice)
				Expect(d.Status.State).To(Equal(v1beta1.Running))
				Expect(d.Status.DeployInProgress).To(BeFalse())
				Expect(d.Status.Summary).To(Equal(v1beta1.ClusterSummary{
					Total:   2,
					Running: 2,
					Down:    0,
					Unknown: 0,
				}))
			})
		})

		Context("multiple apps and one is not running", func() {
			It("is in state Running", func() {
				d.Status.DeployInProgress = false
				gr2.Status.Display.Message = "In progress"
				dc2.Status.Status.State = v1beta1.Down
				dc2.Status.Apps[1].Status.State = v1beta1.Down
				grslice := []fleetv1alpha1.GitRepo{gr1, gr2}
				dcslice := []v1beta1.DeploymentCluster{dc1, dc2}
				r.updateDeploymentStatus(&d, grslice, dcslice)
				Expect(d.Status.State).To(Equal(v1beta1.Down))
				Expect(d.Status.Message).To(Equal("App app2: In progress"))
				Expect(d.Status.DeployInProgress).To(BeFalse())
				Expect(d.Status.Summary).To(Equal(v1beta1.ClusterSummary{
					Total:   2,
					Running: 1,
					Down:    1,
					Unknown: 0,
				}))
			})
		})
	})

	When("a Deployment is in progress and an app is not able to continue", func() {

		const (
			gitRepoName = "app1-12345"
			depName     = "mydeployment"
			depUID      = "12345"
			generation  = int64(3)
		)
		var (
			c client.WithWatch
			r *Reconciler
		)

		// Create a fake client and add a GitRepo object that is "owned" by depName
		BeforeEach(func() {
			gitrepo := &fleetv1alpha1.GitRepo{
				TypeMeta: metav1.TypeMeta{},
				ObjectMeta: metav1.ObjectMeta{
					Name: gitRepoName,
				},
				Spec: fleetv1alpha1.GitRepoSpec{
					ForceSyncGeneration: generation,
				},
				Status: fleetv1alpha1.GitRepoStatus{
					StatusBase: fleetv1alpha1.StatusBase{
						Conditions: []genericcondition.GenericCondition{
							{
								Type:    "Ready",
								Status:  v1.ConditionFalse,
								Message: "Unable to continue",
							},
						},
					},
				},
			}

			idxfunc := func(rawObj client.Object) []string {
				return []string{depName}
			}
			c = fake.NewClientBuilder().
				WithScheme(scheme.Scheme).
				WithObjects(gitrepo).
				WithIndex(&fleetv1alpha1.GitRepo{}, ownerKey, idxfunc).
				Build()
			r = &Reconciler{
				Client:   c,
				recorder: record.NewFakeRecorder(10),
			}
		})

		Context("Deployment's LastForceResync timestamp is blank", func() {
			It("sets LastForceResync timestamp", func() {
				d := &v1beta1.Deployment{
					ObjectMeta: metav1.ObjectMeta{
						Name: depName,
						UID:  depUID,
					},
				}
				Expect(r.forceRedeployStuckApps(ctx, d)).To(Succeed())

				gitrepo := fleetv1alpha1.GitRepo{}
				key := client.ObjectKey{
					Name: gitRepoName,
				}
				Expect(r.Get(ctx, key, &gitrepo)).To(Succeed())

				// GitRepo was not force resync'ed
				Expect(gitrepo.Spec.ForceSyncGeneration).To(Equal(generation))

				// LastForceResync was updated
				Expect(d.Status.LastForceResync).ToNot(Equal(""))
			})
		})

		Context("it is not yet time to resync", func() {
			It("returns without taking action", func() {
				lastResync := time.Now().Add(time.Second * -5).Format(time.RFC3339)
				d := &v1beta1.Deployment{
					ObjectMeta: metav1.ObjectMeta{
						Name: depName,
						UID:  depUID,
					},
					Status: v1beta1.DeploymentStatus{
						LastForceResync: lastResync,
					},
				}

				Expect(r.forceRedeployStuckApps(ctx, d)).To(Succeed())

				gitrepo := fleetv1alpha1.GitRepo{}
				key := client.ObjectKey{
					Name: gitRepoName,
				}
				Expect(r.Get(ctx, key, &gitrepo)).To(Succeed())

				// GitRepo was not force resync'ed
				Expect(gitrepo.Spec.ForceSyncGeneration).To(Equal(generation))

				// LastForceResync was not updated
				Expect(d.Status.LastForceResync).To(Equal(lastResync))
			})
		})

		Context("it is time to resync", func() {
			It("increments the GitRepo's ForceResyncGeneration", func() {
				lastResync := time.Now().Add(time.Second * -75).Format(time.RFC3339)
				d := &v1beta1.Deployment{
					ObjectMeta: metav1.ObjectMeta{
						Name: depName,
						UID:  depUID,
					},
					Status: v1beta1.DeploymentStatus{
						LastForceResync: lastResync,
					},
				}

				Expect(r.forceRedeployStuckApps(ctx, d)).To(Succeed())

				gitrepo := fleetv1alpha1.GitRepo{}
				key := client.ObjectKey{
					Name: gitRepoName,
				}
				Expect(r.Get(ctx, key, &gitrepo)).To(Succeed())

				// GitRepo was force resync'ed
				Expect(gitrepo.Spec.ForceSyncGeneration).To(Equal(generation + 1))

				// LastForceResync was updated
				Expect(d.Status.LastForceResync).ToNot(Equal(lastResync))
			})
		})

		When("a Deployment is in progress and an app is stalled", func() {

			const (
				gitRepoName = "app1-12345"
				depName     = "mydeployment"
				depUID      = "12345"
				generation  = int64(3)
			)
			var (
				c client.WithWatch
				r *Reconciler
			)

			// Create a fake client and add a GitRepo object that is "owned" by depName
			BeforeEach(func() {
				gitrepo := &fleetv1alpha1.GitRepo{
					TypeMeta: metav1.TypeMeta{},
					ObjectMeta: metav1.ObjectMeta{
						Name: gitRepoName,
					},
					Spec: fleetv1alpha1.GitRepoSpec{
						ForceSyncGeneration: generation,
					},
					Status: fleetv1alpha1.GitRepoStatus{
						StatusBase: fleetv1alpha1.StatusBase{
							Conditions: []genericcondition.GenericCondition{
								{
									Type:   "Ready",
									Status: v1.ConditionFalse,
								},
								{
									Type:    "Stalled",
									Status:  v1.ConditionTrue,
									Message: "App is stalled",
								},
							},
						},
					},
				}

				idxfunc := func(rawObj client.Object) []string {
					return []string{depName}
				}
				c = fake.NewClientBuilder().
					WithScheme(scheme.Scheme).
					WithObjects(gitrepo).
					WithIndex(&fleetv1alpha1.GitRepo{}, ".metadata.controller", idxfunc).
					Build()
				r = &Reconciler{
					Client:   c,
					recorder: record.NewFakeRecorder(10),
				}
			})

			Context("Deployment's LastForceResync timestamp is blank", func() {
				It("sets LastForceResync timestamp", func() {
					d := &v1beta1.Deployment{
						ObjectMeta: metav1.ObjectMeta{
							Name: depName,
							UID:  depUID,
						},
					}
					Expect(r.forceRedeployStuckApps(ctx, d)).To(Succeed())

					gitrepo := fleetv1alpha1.GitRepo{}
					key := client.ObjectKey{
						Name: gitRepoName,
					}
					Expect(r.Get(ctx, key, &gitrepo)).To(Succeed())

					// GitRepo was not force resync'ed
					Expect(gitrepo.Spec.ForceSyncGeneration).To(Equal(generation))

					// LastForceResync was updated
					Expect(d.Status.LastForceResync).ToNot(Equal(""))
				})
			})

			Context("it is not yet time to resync", func() {
				It("returns without taking action", func() {
					lastResync := time.Now().Add(time.Second * -5).Format(time.RFC3339)
					d := &v1beta1.Deployment{
						ObjectMeta: metav1.ObjectMeta{
							Name: depName,
							UID:  depUID,
						},
						Status: v1beta1.DeploymentStatus{
							LastForceResync: lastResync,
						},
					}

					Expect(r.forceRedeployStuckApps(ctx, d)).To(Succeed())

					gitrepo := fleetv1alpha1.GitRepo{}
					key := client.ObjectKey{
						Name: gitRepoName,
					}
					Expect(r.Get(ctx, key, &gitrepo)).To(Succeed())

					// GitRepo was not force resync'ed
					Expect(gitrepo.Spec.ForceSyncGeneration).To(Equal(generation))

					// LastForceResync was not updated
					Expect(d.Status.LastForceResync).To(Equal(lastResync))
				})
			})

			Context("it is time to resync", func() {
				It("increments the GitRepo's ForceResyncGeneration", func() {
					lastResync := time.Now().Add(time.Second * -75).Format(time.RFC3339)
					d := &v1beta1.Deployment{
						ObjectMeta: metav1.ObjectMeta{
							Name: depName,
							UID:  depUID,
						},
						Status: v1beta1.DeploymentStatus{
							LastForceResync: lastResync,
						},
					}

					Expect(r.forceRedeployStuckApps(ctx, d)).To(Succeed())

					gitrepo := fleetv1alpha1.GitRepo{}
					key := client.ObjectKey{
						Name: gitRepoName,
					}
					Expect(r.Get(ctx, key, &gitrepo)).To(Succeed())

					// GitRepo was force resync'ed
					Expect(gitrepo.Spec.ForceSyncGeneration).To(Equal(generation + 1))

					// LastForceResync was updated
					Expect(d.Status.LastForceResync).ToNot(Equal(lastResync))
				})
			})
		})
	})

	When("cleaning up removed target clusters", func() {
	var (
	ctx context.Context
	r   *Reconciler
	c   client.WithWatch
	d   v1beta1.Deployment
	dc1 v1beta1.DeploymentCluster
	dc2 v1beta1.DeploymentCluster
	dc3 v1beta1.DeploymentCluster
	)

	BeforeEach(func() {
	ctx = context.Background()

	d = v1beta1.Deployment{
	ObjectMeta: metav1.ObjectMeta{
	Name:      "test-deployment",
	Namespace: "default",
	UID:       "test-uid-12345",
	},
	Spec: v1beta1.DeploymentSpec{
	DeploymentType: v1beta1.Targeted,
	Applications: []v1beta1.Application{
	{
	Name:      "app1",
	Version:   "1.0.0",
	Namespace: "default",
	Targets: []map[string]string{
	{string(v1beta1.ClusterName): "cluster-1"},
	{string(v1beta1.ClusterName): "cluster-2"},
	},
	},
	},
	},
	}

	dc1 = v1beta1.DeploymentCluster{
	ObjectMeta: metav1.ObjectMeta{
	Name:      "test-deployment-cluster-1",
	Namespace: "default",
	Labels: map[string]string{
	string(v1beta1.BundleName):   "test-deployment",
	string(v1beta1.DeploymentID): "test-uid-12345",
	},
	OwnerReferences: []metav1.OwnerReference{
	{APIVersion: "app.edge-orchestrator.intel.com/v1beta1", Kind: "Deployment", Name: "test-deployment", UID: "test-uid-12345"},
	},
	},
	Spec: v1beta1.DeploymentClusterSpec{ClusterID: "cluster-1"},
	}

	dc2 = v1beta1.DeploymentCluster{
	ObjectMeta: metav1.ObjectMeta{
	Name:      "test-deployment-cluster-2",
	Namespace: "default",
	Labels: map[string]string{
	string(v1beta1.BundleName):   "test-deployment",
	string(v1beta1.DeploymentID): "test-uid-12345",
	},
	OwnerReferences: []metav1.OwnerReference{
	{APIVersion: "app.edge-orchestrator.intel.com/v1beta1", Kind: "Deployment", Name: "test-deployment", UID: "test-uid-12345"},
	},
	},
	Spec: v1beta1.DeploymentClusterSpec{ClusterID: "cluster-2"},
	}

	dc3 = v1beta1.DeploymentCluster{
	ObjectMeta: metav1.ObjectMeta{
	Name:      "test-deployment-cluster-3",
	Namespace: "default",
	Labels: map[string]string{
	string(v1beta1.BundleName):   "test-deployment",
	string(v1beta1.DeploymentID): "test-uid-12345",
	},
	OwnerReferences: []metav1.OwnerReference{
	{APIVersion: "app.edge-orchestrator.intel.com/v1beta1", Kind: "Deployment", Name: "test-deployment", UID: "test-uid-12345"},
	},
	},
	Spec: v1beta1.DeploymentClusterSpec{ClusterID: "cluster-3"},
	}

	c = fake.NewClientBuilder().
	WithScheme(scheme.Scheme).
	WithObjects(&d, &dc1, &dc2, &dc3).
	WithStatusSubresource(&v1beta1.Deployment{}).
	Build()

	r = &Reconciler{
	Client:   c,
	Scheme:   scheme.Scheme,
	recorder: record.NewFakeRecorder(1024),
	}
	})

	Context("when cluster is removed from targeted deployment", func() {
	It("should delete the DeploymentCluster for removed cluster", func() {
	err := r.cleanupRemovedTargetClusters(ctx, &d)
	Expect(err).NotTo(HaveOccurred())

	err = r.Get(ctx, types.NamespacedName{Name: dc1.Name, Namespace: dc1.Namespace}, &dc1)
	Expect(err).NotTo(HaveOccurred())

	err = r.Get(ctx, types.NamespacedName{Name: dc2.Name, Namespace: dc2.Namespace}, &dc2)
	Expect(err).NotTo(HaveOccurred())

	err = r.Get(ctx, types.NamespacedName{Name: dc3.Name, Namespace: dc3.Namespace}, &dc3)
	Expect(apierrors.IsNotFound(err)).To(BeTrue())
	})
	})

	Context("when deployment is auto-scaling type", func() {
	It("should not delete any DeploymentClusters", func() {
	d.Spec.DeploymentType = v1beta1.AutoScaling
	err := c.Update(ctx, &d)
	Expect(err).NotTo(HaveOccurred())

	err = r.cleanupRemovedTargetClusters(ctx, &d)
	Expect(err).NotTo(HaveOccurred())

	err = r.Get(ctx, types.NamespacedName{Name: dc1.Name, Namespace: dc1.Namespace}, &dc1)
	Expect(err).NotTo(HaveOccurred())

	err = r.Get(ctx, types.NamespacedName{Name: dc2.Name, Namespace: dc2.Namespace}, &dc2)
	Expect(err).NotTo(HaveOccurred())

	err = r.Get(ctx, types.NamespacedName{Name: dc3.Name, Namespace: dc3.Namespace}, &dc3)
	Expect(err).NotTo(HaveOccurred())
	})
	})

	Context("when all clusters are removed from deployment", func() {
	It("should delete all DeploymentClusters", func() {
	d.Spec.Applications[0].Targets = []map[string]string{}
	err := c.Update(ctx, &d)
	Expect(err).NotTo(HaveOccurred())

	err = r.cleanupRemovedTargetClusters(ctx, &d)
	Expect(err).NotTo(HaveOccurred())

	err = r.Get(ctx, types.NamespacedName{Name: dc1.Name, Namespace: dc1.Namespace}, &dc1)
	Expect(apierrors.IsNotFound(err)).To(BeTrue())

	err = r.Get(ctx, types.NamespacedName{Name: dc2.Name, Namespace: dc2.Namespace}, &dc2)
	Expect(apierrors.IsNotFound(err)).To(BeTrue())

	err = r.Get(ctx, types.NamespacedName{Name: dc3.Name, Namespace: dc3.Namespace}, &dc3)
	Expect(apierrors.IsNotFound(err)).To(BeTrue())
	})
	})
	})

})

var _ = Describe("Deployment Metadata Cache for Metrics Cleanup", func() {
	var (
		cache *deploymentMetadataCache
	)

	BeforeEach(func() {
		cache = &deploymentMetadataCache{
			cache: make(map[string]*deploymentMetadata),
		}
	})

	Context("when caching deployment metadata", func() {
		It("should store deployment metadata correctly", func() {
			d := &v1beta1.Deployment{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-deployment",
					Namespace: "test-namespace",
					UID:       types.UID("test-uid-12345"),
					Labels: map[string]string{
						string(v1beta1.AppOrchActiveProjectID): "test-project-id",
					},
				},
				Spec: v1beta1.DeploymentSpec{
					DisplayName: "Test Display Name",
				},
			}

			cache.cacheDeploymentMetadata(d.Namespace, d.Name, d)

			metadata := cache.getAndRemoveMetadata(d.Namespace, d.Name)
			Expect(metadata).NotTo(BeNil())
			Expect(metadata.deploymentID).To(Equal("test-uid-12345"))
			Expect(metadata.displayName).To(Equal("Test Display Name"))
			Expect(metadata.projectID).To(Equal("test-project-id"))
		})

		It("should return nil for non-existent entries", func() {
			metadata := cache.getAndRemoveMetadata("non-existent", "deployment")
			Expect(metadata).To(BeNil())
		})

		It("should remove metadata after retrieval", func() {
			d := &v1beta1.Deployment{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-deployment",
					Namespace: "test-namespace",
					UID:       types.UID("test-uid-12345"),
					Labels: map[string]string{
						string(v1beta1.AppOrchActiveProjectID): "test-project-id",
					},
				},
				Spec: v1beta1.DeploymentSpec{
					DisplayName: "Test Display Name",
				},
			}

			cache.cacheDeploymentMetadata(d.Namespace, d.Name, d)

			// First retrieval should succeed
			metadata := cache.getAndRemoveMetadata(d.Namespace, d.Name)
			Expect(metadata).NotTo(BeNil())

			// Second retrieval should return nil (already removed)
			metadata = cache.getAndRemoveMetadata(d.Namespace, d.Name)
			Expect(metadata).To(BeNil())
		})
	})

	Context("when cleaning up old cache entries", func() {
		It("should remove entries older than TTL", func() {
			d := &v1beta1.Deployment{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-deployment",
					Namespace: "test-namespace",
					UID:       types.UID("test-uid-12345"),
					Labels: map[string]string{
						string(v1beta1.AppOrchActiveProjectID): "test-project-id",
					},
				},
				Spec: v1beta1.DeploymentSpec{
					DisplayName: "Test Display Name",
				},
			}

			cache.cacheDeploymentMetadata(d.Namespace, d.Name, d)

			// Manually set the timestamp to be older than TTL
			key := d.Namespace + "/" + d.Name
			cache.cache[key].timestamp = time.Now().Add(-40 * time.Minute)

			cache.cleanupOldEntries()

			metadata := cache.getAndRemoveMetadata(d.Namespace, d.Name)
			Expect(metadata).To(BeNil())
		})

		It("should keep entries newer than TTL", func() {
			d := &v1beta1.Deployment{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-deployment",
					Namespace: "test-namespace",
					UID:       types.UID("test-uid-12345"),
					Labels: map[string]string{
						string(v1beta1.AppOrchActiveProjectID): "test-project-id",
					},
				},
				Spec: v1beta1.DeploymentSpec{
					DisplayName: "Test Display Name",
				},
			}

			cache.cacheDeploymentMetadata(d.Namespace, d.Name, d)

			cache.cleanupOldEntries()

			metadata := cache.getAndRemoveMetadata(d.Namespace, d.Name)
			Expect(metadata).NotTo(BeNil())
		})
	})

	Context("when handling deployment without project ID label", func() {
		It("should handle missing project ID label gracefully", func() {
			d := &v1beta1.Deployment{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-deployment",
					Namespace: "test-namespace",
					UID:       types.UID("test-uid-12345"),
					Labels:    map[string]string{}, // No project ID label
				},
				Spec: v1beta1.DeploymentSpec{
					DisplayName: "Test Display Name",
				},
			}

			cache.cacheDeploymentMetadata(d.Namespace, d.Name, d)

			metadata := cache.getAndRemoveMetadata(d.Namespace, d.Name)
			Expect(metadata).NotTo(BeNil())
			Expect(metadata.projectID).To(Equal(""))
		})
	})
})
