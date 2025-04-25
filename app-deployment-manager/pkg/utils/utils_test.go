// SPDX-FileCopyrightText: (C) 2024 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package utils

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"testing"
	"time"

	"net/http"
	"net/http/httptest"

	"google.golang.org/grpc/status"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"

	"github.com/golang/mock/gomock"
	fleetv1alpha1 "github.com/rancher/fleet/pkg/apis/fleet.cattle.io/v1alpha1"
	"github.com/rancher/wrangler/pkg/genericcondition"
	"github.com/stretchr/testify/assert"
	"github.com/undefinedlabs/go-mpatch"
	"google.golang.org/grpc/metadata"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/open-edge-platform/app-orch-deployment/app-deployment-manager/api/v1beta1"
)

func TestMain(m *testing.M) {
	os.Setenv("RATE_LIMITER_QPS", "20")
	os.Setenv("RATE_LIMITER_BURST", "1000")

	os.Exit(m.Run())
}

func TestUtils(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Utils Suite")
}

var _ = Describe("Test Utils", func() {
	Describe("Test CreateClient", func() {
		It("failed create client due to unable to load out-cluster config", func() {
			_, err := CreateClient("/var")
			Expect(err).To(HaveOccurred())
			expectedErrMsg := fmt.Sprintf("%v", err)
			Expect(string(expectedErrMsg)).To(ContainSubstring("error loading config file"))
		})

		It("failed create client due to unable to load in-cluster config", func() {
			_, err := CreateClient("")
			Expect(err).To(HaveOccurred())
			expectedErrMsg := fmt.Sprintf("%v", err)
			Expect(string(expectedErrMsg)).To(ContainSubstring("unable to load in-cluster configuration, " +
				"KUBERNETES_SERVICE_HOST and KUBERNETES_SERVICE_PORT must be defined"))
		})
	})

	Describe("Test GetState", func() {
		It("Returns Running when BundleDeployment is Ready", func() {
			expected := v1beta1.Running
			v := GetState(&fleetv1alpha1.BundleDeployment{
				Spec: fleetv1alpha1.BundleDeploymentSpec{
					DeploymentID: "s-12345",
				},
				Status: fleetv1alpha1.BundleDeploymentStatus{
					Ready:               true,
					AppliedDeploymentID: "s-12345",
					Conditions: []genericcondition.GenericCondition{
						{
							Type:   "Ready",
							Status: "True",
						},
					},
				},
			})
			Expect(v).To(Equal(expected))
		})
		It("Returns Running when BundleDeployment is Modified", func() {
			expected := v1beta1.Running
			v := GetState(&fleetv1alpha1.BundleDeployment{
				Spec: fleetv1alpha1.BundleDeploymentSpec{
					DeploymentID: "s-12345",
				},
				Status: fleetv1alpha1.BundleDeploymentStatus{
					Display: fleetv1alpha1.BundleDeploymentDisplay{
						State: "Modified",
					},
					Ready:               true,
					NonModified:         false,
					AppliedDeploymentID: "s-12345",
					Conditions: []genericcondition.GenericCondition{
						{
							Type:   "Ready",
							Status: "False",
						},
					},
				},
			})
			Expect(v).To(Equal(expected))
		})
		It("Returns Down when BundleDeployment is not Ready", func() {
			expected := v1beta1.Down
			v := GetState(&fleetv1alpha1.BundleDeployment{
				Spec: fleetv1alpha1.BundleDeploymentSpec{
					DeploymentID: "s-12345",
				},
				Status: fleetv1alpha1.BundleDeploymentStatus{
					Ready:               false,
					AppliedDeploymentID: "s-12345",
					Conditions: []genericcondition.GenericCondition{
						{
							Type:   "Ready",
							Status: "True",
						},
					},
				},
			})
			Expect(v).To(Equal(expected))
		})
		It("Returns Down when BundleDeployment Ready condition is false", func() {
			expected := v1beta1.Down
			v := GetState(&fleetv1alpha1.BundleDeployment{
				Spec: fleetv1alpha1.BundleDeploymentSpec{
					DeploymentID: "s-12345",
				},
				Status: fleetv1alpha1.BundleDeploymentStatus{
					Ready:               true,
					AppliedDeploymentID: "s-12345",
					Conditions: []genericcondition.GenericCondition{
						{
							Type:   "Ready",
							Status: "False",
						},
					},
				},
			})
			Expect(v).To(Equal(expected))
		})
		It("Returns Down when BundleDeployment is Ready but hasn't been sync'ed yet", func() {
			expected := v1beta1.Down
			v := GetState(&fleetv1alpha1.BundleDeployment{
				Spec: fleetv1alpha1.BundleDeploymentSpec{
					DeploymentID: "s-23456",
				},
				Status: fleetv1alpha1.BundleDeploymentStatus{
					Ready:               true,
					AppliedDeploymentID: "s-12345",
					Conditions: []genericcondition.GenericCondition{
						{
							Type:   "Ready",
							Status: "True",
						},
					},
				},
			})
			Expect(v).To(Equal(expected))
		})
	})

	Describe("Test GetMessage", func() {
		It("Returns no message when BundleDeployment is Ready", func() {
			v := GetMessage(&fleetv1alpha1.BundleDeployment{
				Status: fleetv1alpha1.BundleDeploymentStatus{
					Conditions: []genericcondition.GenericCondition{
						{
							Message: "message 1",
						},
					},
					Ready: true,
				},
			})
			Expect(v).To(Equal(""))
		})
		It("Returns concatenated message when BundleDeployment is not Ready", func() {
			expected := "message 1; message 3"
			v := GetMessage(&fleetv1alpha1.BundleDeployment{
				Status: fleetv1alpha1.BundleDeploymentStatus{
					Conditions: []genericcondition.GenericCondition{
						{
							Status:  v1.ConditionStatus(v1.ConditionFalse),
							Message: "message 1",
						},
						{
							Status:  v1.ConditionStatus(v1.ConditionTrue),
							Message: "message 2",
						},
						{
							Status:  v1.ConditionStatus(v1.ConditionFalse),
							Message: "message 3",
						},
					},
					Ready: false,
				},
			})
			Expect(v).To(Equal(expected))
		})
	})

	Describe("Test GetAppID", func() {
		It("Returns the app ID when label is present", func() {
			expected := "app-id"
			v := GetAppID(&fleetv1alpha1.BundleDeployment{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						string(v1beta1.LabelBundleName): "app-id",
					},
				},
			})
			Expect(v).To(Equal(expected))
		})
		It("Returns nothing when label is not present", func() {
			v := GetAppID(&fleetv1alpha1.BundleDeployment{})
			Expect(v).To(Equal(""))
		})
	})

	Describe("Test GetAppName", func() {
		It("Returns the app name when label is present", func() {
			expected := "app-name"
			v := GetAppName(&fleetv1alpha1.BundleDeployment{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						string(v1beta1.LabelAppName): "app-name",
					},
				},
			})
			Expect(v).To(Equal(expected))
		})
		It("Returns nothing when label is not present", func() {
			v := GetAppName(&fleetv1alpha1.BundleDeployment{})
			Expect(v).To(Equal(""))
		})
	})

	Describe("Test GetDeploymentGeneration", func() {
		var jsonmap map[string]any
		var expected int64

		jsonValues := `{"global":{"fleet":{"deploymentGeneration": 1}}}`
		Expect(json.Unmarshal([]byte(jsonValues), &jsonmap)).To(Succeed())

		It("Returns the deployment generation when value is present", func() {
			expected = 1
			v := GetDeploymentGeneration(&fleetv1alpha1.BundleDeployment{
				Spec: fleetv1alpha1.BundleDeploymentSpec{
					Options: fleetv1alpha1.BundleDeploymentOptions{
						Helm: &fleetv1alpha1.HelmOptions{
							Values: &fleetv1alpha1.GenericMap{
								Data: jsonmap,
							},
						},
					},
				},
			})
			Expect(v).To(Equal(expected))
		})
		It("Returns 0 when value is not present", func() {
			expected = 0
			v := GetDeploymentGeneration(&fleetv1alpha1.BundleDeployment{})
			Expect(v).To(Equal(expected))
		})
	})

	Describe("Test UpdateStatusCondition", func() {
		It("Adds the Status condition when not present", func() {
			v := UpdateStatusCondition([]metav1.Condition{}, "Ready", metav1.ConditionTrue, "Reason", nil)
			Expect(len(v)).To(Equal(1))
			Expect(v[0].Type).To(Equal("Ready"))
			Expect(v[0].Status).To(Equal(metav1.ConditionTrue))
			Expect(v[0].Reason).To(Equal("Reason"))
		})
		It("Updates the Status condition when present", func() {
			conds := []metav1.Condition{
				{
					Type:   "Ready",
					Status: metav1.ConditionFalse,
					Reason: "OldReason",
				},
			}
			v := UpdateStatusCondition(conds, "Ready", metav1.ConditionTrue, "NewReason", nil)
			Expect(len(v)).To(Equal(1))
			Expect(v[0].Type).To(Equal("Ready"))
			Expect(v[0].Status).To(Equal(metav1.ConditionTrue))
			Expect(v[0].Reason).To(Equal("NewReason"))
		})
	})

	Describe("Test GetAppRef", func() {
		It("successfully return app ref", func() {
			d := &v1beta1.Deployment{}
			d.Spec.DeploymentPackageRef.Name = "test-name"
			d.Spec.DeploymentPackageRef.Version = "test-version"

			v := GetAppRef(d)

			Expect(v).To(Equal("test-name-test-version"))
		})
	})

	Describe("Test CreateSecret", func() {
		It("successfully create secret", func() {
			ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				_, err := w.Write([]byte(`{"apiVersion": "v1", "kind": "Secret", "metadata": {"name": "test-name"}}`))
				Expect(err).ToNot(HaveOccurred())
			}))

			defer ts.Close()

			kc := mockK8Client(ts.URL)

			var dataMap = make(map[string]string)
			dataMap["hello"] = "world"

			err := CreateSecret(context.Background(), kc, "test-namespace",
				"test-secretname", dataMap, true)

			Expect(err).ShouldNot(HaveOccurred())
		})

		It("fails due to err when creating secret", func() {
			ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusMethodNotAllowed)
				_, err := w.Write([]byte(`{"apiVersion": "v1", "kind": "Secret", "metadata": {"name": "test-name"}}`))
				Expect(err).ToNot(HaveOccurred())
			}))

			defer ts.Close()

			kc := mockK8Client(ts.URL)

			var dataMap = make(map[string]string)
			dataMap["hello"] = "world"

			err := CreateSecret(context.Background(), kc, "test-namespace",
				"test-secretname", dataMap, true)

			Expect(err).Should(HaveOccurred())
			s, ok := status.FromError(err)
			Expect(ok).To(BeFalse())
			Expect(s.Message()).Should(Equal("the server does not allow this method on the requested resource (post secrets)"))
		})
	})

	Describe("Test DeleteSecret", func() {
		It("successfully delete secret", func() {
			ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				_, err := w.Write([]byte(`{"apiVersion": "v1", "kind": "Secret", "metadata": {"name": "test-name"}}`))
				Expect(err).ToNot(HaveOccurred())
			}))

			defer ts.Close()

			kc := mockK8Client(ts.URL)

			err := DeleteSecret(context.Background(), kc, "test-namespace", "test-secretname")

			Expect(err).ShouldNot(HaveOccurred())
		})

		It("fails due to err when deleting secret", func() {
			ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusMethodNotAllowed)
				_, err := w.Write([]byte(`{"apiVersion": "v1", "kind": "Secret", "metadata": {"name": "test-name"}}`))
				Expect(err).ToNot(HaveOccurred())
			}))

			defer ts.Close()

			kc := mockK8Client(ts.URL)

			err := DeleteSecret(context.Background(), kc, "test-namespace", "test-secretname")

			Expect(err).Should(HaveOccurred())
			s, ok := status.FromError(err)
			Expect(ok).To(BeFalse())
			Expect(s.Message()).Should(Equal("the server does not allow this method on the requested resource (delete secrets test-secretname)"))
		})
	})

	Describe("Test GetSecretValues", func() {
		It("successfully get secret values", func() {
			ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				_, err := w.Write([]byte(`{"apiVersion": "v1", "kind": "Secret", "data": {"values": "dGVzdC12YWx1ZXM="}, "metadata": {"name": "test-secretname"}}`))
				Expect(err).ToNot(HaveOccurred())
			}))

			defer ts.Close()

			kc := mockK8Client(ts.URL)

			v, err := GetSecretValue(context.Background(), kc, "test-namespace", "test-secretname")

			Expect(string(v.Data["values"])).To(Equal("test-values"))
			Expect(err).ShouldNot(HaveOccurred())
		})

		It("fails due to err when deleting secret", func() {
			ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusMethodNotAllowed)
				_, err := w.Write([]byte(`{"apiVersion": "v1", "kind": "Secret", "metadata": {"name": "test-name"}}`))
				Expect(err).ToNot(HaveOccurred())
			}))

			defer ts.Close()

			kc := mockK8Client(ts.URL)

			_, err := GetSecretValue(context.Background(), kc, "test-namespace", "test-secretname")

			Expect(err).Should(HaveOccurred())
			s, ok := status.FromError(err)
			Expect(ok).To(BeFalse())
			Expect(s.Message()).Should(Equal("the server does not allow this method on the requested resource (get secrets test-secretname)"))
		})
	})

	Describe("Test GetGitCACert", func() {
		It("successfully get git cacert from var", func() {
			gitCaCert = []byte("test")
			v := GetGitCaCert()
			Expect(v).To(Equal(string(gitCaCert)))
		})
	})

	Describe("Test GetSecretValue", func() {
		It("successfully get secret value", func() {
			ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				_, err := w.Write([]byte(`{"apiVersion": "v1", "kind": "Secret", "data": {"value": "dGVzdC12YWx1ZQ=="}, "metadata": {"name": "test-secretname"}}`))
				Expect(err).ToNot(HaveOccurred())
			}))

			defer ts.Close()

			kc := mockK8Client(ts.URL)

			v, err := GetSecretValue(context.Background(), kc, "test-namespace", "test-secretname")

			Expect(string(v.Data["value"])).To(Equal("test-value"))
			Expect(err).ShouldNot(HaveOccurred())
		})

		It("fails due to err when deleting secret", func() {
			ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusMethodNotAllowed)
				_, err := w.Write([]byte(`{"apiVersion": "v1", "kind": "Secret", "metadata": {"name": "test-name"}}`))
				Expect(err).ToNot(HaveOccurred())
			}))

			defer ts.Close()

			kc := mockK8Client(ts.URL)

			_, err := GetSecretValue(context.Background(), kc, "test-namespace", "test-secretname")

			Expect(err).Should(HaveOccurred())
			s, ok := status.FromError(err)
			Expect(ok).To(BeFalse())
			Expect(s.Message()).Should(Equal("the server does not allow this method on the requested resource (get secrets test-secretname)"))
		})
	})
})

func mockK8Client(tsUrl string) *kubernetes.Clientset {
	config := &rest.Config{
		Host: tsUrl,
	}

	gv := metav1.SchemeGroupVersion
	config.GroupVersion = &gv

	config.NegotiatedSerializer = scheme.Codecs.WithoutConversion()
	config.UserAgent = rest.DefaultKubernetesUserAgent()
	config.ContentType = "application/json"

	_kClient, err := kubernetes.NewForConfig(config)
	Expect(err).ToNot(HaveOccurred())

	return _kClient
}

func unpatchAll(list []*mpatch.Patch) error {
	for _, p := range list {
		err := p.Unpatch()
		if err != nil {
			return err
		}
	}
	return nil
}

func TestLogActivity_Name(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	patch := func(ctrl *gomock.Controller) []*mpatch.Patch {
		fromIncomingContext, err := mpatch.PatchMethod(metadata.FromIncomingContext, func(ctx context.Context) (metadata.MD, bool) {
			return metadata.MD{"name": []string{"name"}}, true
		})
		if err != nil {
			t.Errorf("patch error with gomock %s", err.Error())
		}
		return []*mpatch.Patch{fromIncomingContext}
	}
	pList := patch(ctrl)
	LogActivity(context.TODO(), "verb", "thing")
	err := unpatchAll(pList)
	if err != nil {
		t.Errorf("patch error with gomock %s", err.Error())
	}
}

func TestLogActivity_Client(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	patch := func(ctrl *gomock.Controller) []*mpatch.Patch {
		fromIncomingContext, err := mpatch.PatchMethod(metadata.FromIncomingContext, func(ctx context.Context) (metadata.MD, bool) {
			return metadata.MD{"client": []string{"client"}}, false
		})
		if err != nil {
			t.Errorf("patch error with gomock %s", err.Error())
		}
		return []*mpatch.Patch{fromIncomingContext}
	}
	pList := patch(ctrl)
	LogActivity(context.TODO(), "verb", "thing")
	err := unpatchAll(pList)
	if err != nil {
		t.Errorf("patch error with gomock %s", err.Error())
	}
}

func TestCreateRestConfig(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// with empty kubeconfig - success
	patch := func(ctrl *gomock.Controller) []*mpatch.Patch {
		function, err := mpatch.PatchMethod(rest.InClusterConfig, func() (*rest.Config, error) {
			return &rest.Config{}, nil
		})
		if err != nil {
			t.Errorf("patch error with gomock %s", err.Error())
		}
		return []*mpatch.Patch{function}
	}
	pList := patch(ctrl)
	cfg, err := CreateRestConfig("")
	assert.NoError(t, err)
	assert.NotNil(t, cfg)
	err = unpatchAll(pList)
	if err != nil {
		t.Errorf("patch error with gomock %s", err.Error())
	}

	// with empty kubeconfig - fail
	patch = func(ctrl *gomock.Controller) []*mpatch.Patch {
		function, err := mpatch.PatchMethod(rest.InClusterConfig, func() (*rest.Config, error) {
			return nil, errors.New("tmp")
		})
		if err != nil {
			t.Errorf("patch error with gomock %s", err.Error())
		}
		return []*mpatch.Patch{function}
	}
	pList = patch(ctrl)
	cfg, err = CreateRestConfig("")
	assert.Error(t, err)
	assert.Nil(t, cfg)
	err = unpatchAll(pList)
	if err != nil {
		t.Errorf("patch error with gomock %s", err.Error())
	}

	// with non-empty kubeconfig - success
	patch = func(ctrl *gomock.Controller) []*mpatch.Patch {
		function, err := mpatch.PatchMethod(clientcmd.BuildConfigFromFlags, func(masterUrl, kubeconfigPath string) (*rest.Config, error) {
			return &rest.Config{}, nil
		})
		if err != nil {
			t.Errorf("patch error with gomock %s", err.Error())
		}
		return []*mpatch.Patch{function}
	}
	pList = patch(ctrl)
	cfg, err = CreateRestConfig("tmp")
	assert.NoError(t, err)
	assert.NotNil(t, cfg)
	err = unpatchAll(pList)
	if err != nil {
		t.Errorf("patch error with gomock %s", err.Error())
	}

	// with non-empty kubeconfig - fail
	patch = func(ctrl *gomock.Controller) []*mpatch.Patch {
		function, err := mpatch.PatchMethod(clientcmd.BuildConfigFromFlags, func(masterUrl, kubeconfigPath string) (*rest.Config, error) {
			return nil, errors.New("tmp")
		})
		if err != nil {
			t.Errorf("patch error with gomock %s", err.Error())
		}
		return []*mpatch.Patch{function}
	}
	pList = patch(ctrl)
	cfg, err = CreateRestConfig("tmp")
	assert.Error(t, err)
	assert.Nil(t, cfg)
	err = unpatchAll(pList)
	if err != nil {
		t.Errorf("patch error with gomock %s", err.Error())
	}
}

func eventSeparator() { time.Sleep(50 * time.Millisecond) }

func shouldWait(path ...string) bool {
	for _, p := range path {
		if p == "" {
			return false
		}
	}
	return true
}

// touch
func touch(t *testing.T, path ...string) {
	t.Helper()
	if len(path) < 1 {
		t.Fatalf("touch: path must have at least one element: %s", path)
	}
	fp, err := os.Create(filepath.Join(path...))
	if err != nil {
		t.Fatalf("touch(%q): %s", filepath.Join(path...), err)
	}
	err = fp.Close()
	if err != nil {
		t.Fatalf("touch(%q): %s", filepath.Join(path...), err)
	}
	if shouldWait(path...) {
		eventSeparator()
	}
}

func TestWatchGitCaCertFile(t *testing.T) {
	tests := []struct {
		name         string
		certFileName string
	}{
		{
			"read initial content and then read updates",
			"ca.crt",
		},
		{
			"cert file does not exist",
			"ca1.crt",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a context with cancel
			ctx, cancelFunc := context.WithCancel(context.Background())
			defer cancelFunc()

			// Temporary directory for the test and cert file
			gitCaCertFolder := t.TempDir()
			touch(t, gitCaCertFolder, "/"+tt.certFileName)

			// Start watching for cert file
			go WatchGitCaCertFile(ctx, gitCaCertFolder, tt.certFileName)

			// Create a cert file with initial content
			fp, err := os.OpenFile(filepath.Join(gitCaCertFolder, tt.certFileName), os.O_RDWR, 0)
			if err != nil {
				t.Fatal(err)
			}

			if _, err = fp.Write([]byte("X")); err != nil {
				t.Fatal(err)
			}
			if err = fp.Sync(); err != nil {
				t.Fatal(err)
			}
			if err = fp.Close(); err != nil {
				t.Fatal(err)
			}

			// Update the cert file
			fp, err = os.OpenFile(filepath.Join(gitCaCertFolder, tt.certFileName), os.O_RDWR, 0)
			if err != nil {
				t.Fatal(err)
			}
			if _, err = fp.Write([]byte("Y")); err != nil {
				t.Fatal(err)
			}
			if err = fp.Sync(); err != nil {
				t.Fatal(err)
			}

			if err = fp.Close(); err != nil {
				t.Fatal(err)
			}
		})
	}
}
func TestWriteFile(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// success
	patch := func(ctrl *gomock.Controller) []*mpatch.Patch {
		function1, err := mpatch.PatchMethod(os.MkdirAll, func(path string, perm os.FileMode) error {
			return nil
		})
		if err != nil {
			t.Errorf("patch error with gomock %s", err.Error())
		}

		function2, err := mpatch.PatchMethod(os.WriteFile, func(name string, data []byte, perm os.FileMode) error {
			return nil
		})
		if err != nil {
			t.Errorf("patch error with gomock %s", err.Error())
		}

		return []*mpatch.Patch{function1, function2}
	}

	pList := patch(ctrl)
	err := WriteFile("tmpdir", "tmpfile", []byte("test"))
	assert.NoError(t, err)
	err = unpatchAll(pList)
	if err != nil {
		t.Errorf("patch error with gomock %s", err.Error())
	}

	// failed to create dir
	patch = func(ctrl *gomock.Controller) []*mpatch.Patch {
		function1, err := mpatch.PatchMethod(os.MkdirAll, func(path string, perm os.FileMode) error {
			return errors.New("tmp")
		})
		if err != nil {
			t.Errorf("patch error with gomock %s", err.Error())
		}

		function2, err := mpatch.PatchMethod(os.WriteFile, func(name string, data []byte, perm os.FileMode) error {
			return nil
		})
		if err != nil {
			t.Errorf("patch error with gomock %s", err.Error())
		}

		return []*mpatch.Patch{function1, function2}
	}

	pList = patch(ctrl)
	err = WriteFile("tmpdir", "tmpfile", []byte("test"))
	assert.Error(t, err)
	err = unpatchAll(pList)
	if err != nil {
		t.Errorf("patch error with gomock %s", err.Error())
	}

	// failed to write file
	patch = func(ctrl *gomock.Controller) []*mpatch.Patch {
		function1, err := mpatch.PatchMethod(os.MkdirAll, func(path string, perm os.FileMode) error {
			return nil
		})
		if err != nil {
			t.Errorf("patch error with gomock %s", err.Error())
		}

		function2, err := mpatch.PatchMethod(os.WriteFile, func(name string, data []byte, perm os.FileMode) error {
			return errors.New("tmp")
		})
		if err != nil {
			t.Errorf("patch error with gomock %s", err.Error())
		}

		return []*mpatch.Patch{function1, function2}
	}

	pList = patch(ctrl)
	err = WriteFile("tmpdir", "tmpfile", []byte("test"))
	assert.Error(t, err)
	err = unpatchAll(pList)
	if err != nil {
		t.Errorf("patch error with gomock %s", err.Error())
	}
}

func TestToInt32Clamped(t *testing.T) {
	Describe("ToInt32Clamped", func() {
		It("returns 0 when input is negative", func() {
			Expect(ToInt32Clamped(-1)).To(Equal(int32(0)))
		})

		It("returns the input value when it is within the int32 range", func() {
			Expect(ToInt32Clamped(123)).To(Equal(int32(123)))
		})

		It("returns MaxInt32 when input exceeds int32 range", func() {
			Expect(ToInt32Clamped(math.MaxInt32 + 1)).To(Equal(int32(math.MaxInt32)))
		})
	})
}
