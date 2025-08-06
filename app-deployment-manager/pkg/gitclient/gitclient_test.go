// SPDX-FileCopyrightText: (C) 2024 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0
package gitclient

import (
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

const (
	basedir = "/tmp/gitclient"
	uid     = "12345"
)

var _ = Describe("Gitclient", func() {

	Describe("Gitea Gitclient", func() {
		BeforeEach(func() {
			os.Setenv("SECRET_SERVICE_ENABLED", "false")
			os.Setenv("GIT_USER", "foo")
			os.Setenv("GIT_PASSWORD", "bar")
			os.Setenv("GIT_SERVER", "https://localhost:12345")
			os.Setenv("GIT_PROVIDER", "gitea")
		})
		When("creating a new GitClient", func() {
			Context("and all env vars are set", func() {
				It("should succeed", func() {
					_, err := NewGitClient(uid)
					Expect(err).Should(BeNil())
				})
			})
			Context("and GIT_USER is not set", func() {
				It("should fail", func() {
					os.Unsetenv("GIT_USER")
					_, err := NewGitClient(uid)
					Expect(err).ShouldNot(BeNil())
				})
			})
			Context("and GIT_PASSWORD is not set", func() {
				It("should fail", func() {
					os.Unsetenv("GIT_PASSWORD")
					_, err := NewGitClient(uid)
					Expect(err).ShouldNot(BeNil())
				})
			})
			Context("and GIT_SERVER is not set", func() {
				It("should fail", func() {
					os.Unsetenv("GIT_SERVER")
					_, err := NewGitClient(uid)
					Expect(err).ShouldNot(BeNil())
				})
			})
			Context("and GIT_PROVIDER is not set", func() {
				It("should fail", func() {
					os.Unsetenv("GIT_PROVIDER")
					_, err := NewGitClient(uid)
					Expect(err).ShouldNot(BeNil())
				})
			})
		})

		When("calling GetRemoteURL", func() {
			Context("and all env vars are set", func() {
				It("should succeed", func() {
					_, err := GetRemoteURL(uid)
					Expect(err).Should(BeNil())
				})
			})
			Context("and GIT_USER is not set", func() {
				It("should fail", func() {
					os.Unsetenv("GIT_USER")
					_, err := GetRemoteURL(uid)
					Expect(err).ShouldNot(BeNil())
				})
			})
			Context("GIT_PASSWORD is not set", func() {
				It("should succeed", func() {
					os.Unsetenv("GIT_PASSWORD")
					_, err := GetRemoteURL(uid)
					Expect(err).Should(BeNil())
				})
			})
			Context("and GIT_SERVER is not set", func() {
				It("should fail", func() {
					os.Unsetenv("GIT_SERVER")
					_, err := GetRemoteURL(uid)
					Expect(err).ShouldNot(BeNil())
				})
			})
			Context("and GIT_PROVIDER is not set", func() {
				It("should fail", func() {
					os.Unsetenv("GIT_PROVIDER")
					_, err := GetRemoteURL(uid)
					Expect(err).ShouldNot(BeNil())
				})
			})
			Context("and SECRET_SERVICE_ENABLED is not set", func() {
				It("should fail", func() {
					os.Unsetenv("SECRET_SERVICE_ENABLED")
					_, err := GetRemoteURL(uid)
					Expect(err).ShouldNot(BeNil())
				})
			})
		})
		When("checking if repo already exists", func() {
			Context("and remote server is unreachable", func() {
				It("should fail", func() {
					client, err := NewGitClient(uid)
					Expect(err).Should(BeNil())
					_, err = client.ExistsOnRemote()
					Expect(err).ShouldNot(BeNil())

					os.Setenv("GIT_PROXY", "http://localhost:12345")
					client, err = NewGitClient(uid)
					Expect(err).Should(BeNil())
					_, err = client.ExistsOnRemote()
					Expect(err).ShouldNot(BeNil())
					os.Unsetenv("GIT_PROXY")
				})
			})
		})
		When("deleting remote repo", func() {
			Context("and remote server is unreachable", func() {
				It("should fail", func() {
					client, err := NewGitClient(uid)
					Expect(err).Should(BeNil())
					Expect(client.Delete()).ShouldNot(BeNil())
				})
			})

		})
		When("repo is cloned", func() {
			It("should fail when Git server unreachable", func() {
				client, err := NewGitClient(uid)
				Expect(err).Should(BeNil())
				Expect(os.RemoveAll(basedir)).Should(Succeed())
				Expect(client.Clone(basedir)).ShouldNot(Succeed())
			})
		})
		When("repo is initialized", func() {
			Context("and commit called with no changes", func() {
				It("should not throw an error'", func() {
					client, err := NewGitClient(uid)
					Expect(err).Should(BeNil())
					Expect(os.RemoveAll(basedir)).Should(Succeed())
					Expect(client.Initialize(basedir)).Should(Succeed())
					Expect(basedir).Should(BeADirectory())
					err = client.CommitFiles()
					Expect(err).Should(BeNil())
				})
			})
			Context("and commit called with a change", func() {
				It("should succeed", func() {
					client, err := NewGitClient(uid)
					Expect(err).Should(BeNil())
					Expect(os.RemoveAll(basedir)).Should(Succeed())
					Expect(client.Initialize(basedir)).Should(Succeed())
					Expect(os.WriteFile(filepath.Join(basedir, "foo.txt"), []byte("foo"), 0600)).Should(Succeed())
					Expect(client.CommitFiles()).Should(Succeed())
					Expect(client.PushToRemote()).ShouldNot(Succeed())
				})
			})
		})
		When("repo is already initialized", func() {
			It("should fail to initialize twice", func() {
				client, err := NewGitClient(uid)
				Expect(err).Should(BeNil())
				Expect(os.RemoveAll(basedir)).Should(Succeed())
				Expect(client.Initialize(basedir)).Should(Succeed())
				Expect(client.Initialize(basedir)).ShouldNot(Succeed())
			})
		})
		When("repo is not initialized or cloned", func() {
			It("should fail to commit and push", func() {
				client, err := NewGitClient(uid)
				Expect(err).Should(BeNil())
				Expect(client.CommitFiles()).ShouldNot(Succeed())
				Expect(client.PushToRemote()).ShouldNot(Succeed())
			})
		})
	})

	Describe("Gitea Gitclient with self-signed cert", func() {
		BeforeEach(func() {
			os.Setenv("SECRET_SERVICE_ENABLED", "false")
			os.Setenv("GIT_USER", "foo")
			os.Setenv("GIT_PASSWORD", "bar")
			os.Setenv("GIT_SERVER", "https://localhost:12345")
			os.Setenv("GIT_PROVIDER", "gitea")
			os.Setenv("GIT_CA_CERT", "LS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0tLS0tCk1JSUZWekNDQXorZ0F3SUJBZ0lVSXpKVkNBVnIxcWU1akRoZ0drM25QMWJEcitnd0RRWUpLb1pJaHZjTkFRRUwKQlFBd0dqRVlNQllHQTFVRUF3d1BLaTVyYVc1a0xtbHVkR1Z5Ym1Gc01CNFhEVEkwTURneU1qRTNNamN4TWxvWApEVEkxTURneU1qRTNNamN4TWxvd0dqRVlNQllHQTFVRUF3d1BLaTVyYVc1a0xtbHVkR1Z5Ym1Gc01JSUNJakFOCkJna3Foa2lHOXcwQkFRRUZBQU9DQWc4QU1JSUNDZ0tDQWdFQWpMNlVpekNhT2V1M0Q0TTJvQkUwZnltNjR0U3MKOXRwUE53dkFnVnpEeE1UOWJoTWhycEJZUk5vd2R3cXBtekJMUXZSUTIrbmJ3UDYxcXlQK2g4QzZzWTdNRCtRQgpCaHlEOVdISGJ1RUErZHVoc3J1SGxwL21EdGNvcS93NGZBR090aHI0K0tDQ0hMa1ZBYnVsQlNmUnN0eWRGbUhCCmtxT0hYamhkMGNBUnZWekUzL2dQQkVkUFVUbkN1R2xrcGxGZTdNL0pEVE9qMmF0Uis0R2FORTZpSFZCOWh6OXQKcFpHczJZaUNDL3g0TlFBZ0NseVBjSUViaFdINDAwTnV0ek5xdVdEQ2RES1ptRWJNQ3NTYmdZN1d1YVRKeWlCSApnYk10VHJUSitZNW1vajdaZ1RyUXJ5OE1IVUQzc2RUVU5HV09oUU96bXQrSzlzbWw5aVFNbE85THJUa0dyMlJHCnZtU0plQ2xmM3JXZ1VNSFVqdmZrdFNmU0gxeTc4VFdNbzVWejFGd3c1YU90VTIrT3FnbHJtZ1ZhQU9URXkvcDQKVUNMdjk5NDBybVRqRGRvU3hObWd0MG5IVFVnK1ZEdm02cHhXOTZQU1hQcW9CczYydlljYUxDd1AzT0RqVjNiZwp3ZEpOQlJYVzVoMUUyNEY3OFhacDdaNEcxNkhWcWJFTFdlOHhZYUpUMC9ZR1I5NU9Qbmw1c053bm1lSXRkcWFYCnJXbXdPeHdmbHVjUVBkUXNTb1cyaUlUbTlPSDhsQUIxK3VLZnM5TldlaXY5SnA4bU9CbWJiOU80ZEo3M0tjdU4KL3VqMDBZNEtLaG5PMlhqcmJnY1dKazMrblpQQlRaSFcrcTU3ajRMMlplYW5ubmZ1dWowYjNpUFFSbHJlVldWNgp6STlHeEF5NExBYnVPSE1DQXdFQUFhT0JsRENCa1RBZEJnTlZIUTRFRmdRVVdCanRGRkxzSTdnZE1XcGx5V3p4ClBDYVgvMk13SHdZRFZSMGpCQmd3Rm9BVVdCanRGRkxzSTdnZE1XcGx5V3p4UENhWC8yTXdEd1lEVlIwVEFRSC8KQkFVd0F3RUIvekErQmdOVkhSRUVOekExZ2c4cUxtdHBibVF1YVc1MFpYSnVZV3lDSW1kcGRHVmhMV2gwZEhBdQpaMmwwWldFdWMzWmpMbU5zZFhOMFpYSXViRzlqWVd3d0RRWUpLb1pJaHZjTkFRRUxCUUFEZ2dJQkFDMk9pYVVjCkNQc1pyVnkyYzNoUTFFcmFlQ1pxaUloeFEybVFjSFpPeUdoS2JrS2o2N1N3Q2VYRzBVaE5yOGtXR2plWEJUN1cKeWkvSzgxYnc2UmgxQm1Na2RlS01hUFRYK2pFeVJaQ0lzOXJNVEF0R1NTS092UWNZRUtUbmd2SEpQY0ZtazJzUQpockE0UVdETTB0V0wzejNBNDJ6a2xIV0Ixbk9zbmhwcTUyc016QkFydVVhSXdxUEtTN3U5YUJGRThDSTQrd2x1CnB3eHd3emg1ODF6UXo3TXMvSzA1U3FQdkdOcGd2M0pQRTJtTCtoTE1HaUJQSW5PRnlCWFd3Zlh1TGllM1llN0YKNW5RRlB5MkQ0eUZ3Y0pia3I1RFlTcTZZdldSVzhOVVBtYXZlNkdKdjZvM0NWT0RhOHAvaTBkdmhwYTNTK3BlNgoyRXpGdFk0Si9CSHN4UVQwZjFFTmZtSVN6eHJJTU5yZ3p0d2Q3a3NBeDBkaTd2VXFiSjg3VG15aGpuaDB0VnBjClFmUlovWmNRWVZ4RmQxOFRkN1FpZG5ZbDR5ekttWHdMYkxSVGJUWGVIZTQvckRVN1dobmE2SnFzNEV4TTk5UEIKbnRneWNxbmxOckI1NXZHTVlCT2VNL1Uya1ArZU9aeGJJMXNzVUJ3ZFZ4ZEdxT3llS0M3ZUdkNHluaGtHSzNuago5TFBlUmtvWk41UkZjdTd3ekN3VFF2NWtsM3VHUncvbzBKcjdUNlZmYTVzNFVWRkY4MmdzeXdscFk3Vk9HMUpOCmljd0FQd2NkMVIyc0Rpc3BjQjdiR29GU1hpRlkybFU3eVdrOCtwVUExdWRCK2RwWWs4U2tWMFZ4cnJMeVVyMkkKTmduZjg5TzU5ZFJ6dUQ2MDg5WnhsQ3hMazZzVENLSWN6QVVvCi0tLS0tRU5EIENFUlRJRklDQVRFLS0tLS0K")
		})
		When("creating a new GitClient", func() {
			Context("and all env vars are set", func() {
				It("should succeed", func() {
					_, err := NewGitClient(uid)
					Expect(err).Should(BeNil())
				})
			})
			Context("and GIT_USER is not set", func() {
				It("should fail", func() {
					os.Unsetenv("GIT_USER")
					_, err := NewGitClient(uid)
					Expect(err).ShouldNot(BeNil())
				})
			})
			Context("and GIT_PASSWORD is not set", func() {
				It("should fail", func() {
					os.Unsetenv("GIT_PASSWORD")
					_, err := NewGitClient(uid)
					Expect(err).ShouldNot(BeNil())
				})
			})
			Context("and GIT_SERVER is not set", func() {
				It("should fail", func() {
					os.Unsetenv("GIT_SERVER")
					_, err := NewGitClient(uid)
					Expect(err).ShouldNot(BeNil())
				})
			})
			Context("and GIT_PROVIDER is not set", func() {
				It("should fail", func() {
					os.Unsetenv("GIT_PROVIDER")
					_, err := NewGitClient(uid)
					Expect(err).ShouldNot(BeNil())
				})
			})
		})

		When("calling GetRemoteURL", func() {
			Context("and all env vars are set", func() {
				It("should succeed", func() {
					_, err := GetRemoteURL(uid)
					Expect(err).Should(BeNil())
				})
			})
			Context("and GIT_USER is not set", func() {
				It("should fail", func() {
					os.Unsetenv("GIT_USER")
					_, err := GetRemoteURL(uid)
					Expect(err).ShouldNot(BeNil())
				})
			})
			Context("GIT_PASSWORD is not set", func() {
				It("should succeed", func() {
					os.Unsetenv("GIT_PASSWORD")
					_, err := GetRemoteURL(uid)
					Expect(err).Should(BeNil())
				})
			})
			Context("and GIT_SERVER is not set", func() {
				It("should fail", func() {
					os.Unsetenv("GIT_SERVER")
					_, err := GetRemoteURL(uid)
					Expect(err).ShouldNot(BeNil())
				})
			})
			Context("and GIT_PROVIDER is not set", func() {
				It("should fail", func() {
					os.Unsetenv("GIT_PROVIDER")
					_, err := GetRemoteURL(uid)
					Expect(err).ShouldNot(BeNil())
				})
			})
			Context("and SECRET_SERVICE_ENABLED is not set", func() {
				It("should fail", func() {
					os.Unsetenv("SECRET_SERVICE_ENABLED")
					_, err := GetRemoteURL(uid)
					Expect(err).ShouldNot(BeNil())
				})
			})
		})
		When("checking if repo already exists", func() {
			Context("and remote server is unreachable", func() {
				It("should fail", func() {
					client, err := NewGitClient(uid)
					Expect(err).Should(BeNil())
					_, err = client.ExistsOnRemote()
					Expect(err).ShouldNot(BeNil())

					os.Setenv("GIT_PROXY", "http://localhost:12345")
					client, err = NewGitClient(uid)
					Expect(err).Should(BeNil())
					_, err = client.ExistsOnRemote()
					Expect(err).ShouldNot(BeNil())
					os.Unsetenv("GIT_PROXY")
				})
			})
		})
		When("deleting remote repo", func() {
			Context("and remote server is unreachable", func() {
				It("should fail", func() {
					client, err := NewGitClient(uid)
					Expect(err).Should(BeNil())
					Expect(client.Delete()).ShouldNot(BeNil())
				})
			})

		})
		When("repo is cloned", func() {
			It("should fail when Git server unreachable", func() {
				client, err := NewGitClient(uid)
				Expect(err).Should(BeNil())
				Expect(os.RemoveAll(basedir)).Should(Succeed())
				Expect(client.Clone(basedir)).ShouldNot(Succeed())
			})
		})
		When("repo is initialized", func() {
			Context("and commit called with no changes", func() {
				It("should not throw an error'", func() {
					client, err := NewGitClient(uid)
					Expect(err).Should(BeNil())
					Expect(os.RemoveAll(basedir)).Should(Succeed())
					Expect(client.Initialize(basedir)).Should(Succeed())
					Expect(basedir).Should(BeADirectory())
					err = client.CommitFiles()
					Expect(err).Should(BeNil())
				})
			})
			Context("and commit called with a change", func() {
				It("should succeed", func() {
					client, err := NewGitClient(uid)
					Expect(err).Should(BeNil())
					Expect(os.RemoveAll(basedir)).Should(Succeed())
					Expect(client.Initialize(basedir)).Should(Succeed())
					Expect(os.WriteFile(filepath.Join(basedir, "foo.txt"), []byte("foo"), 0600)).Should(Succeed())
					Expect(client.CommitFiles()).Should(Succeed())
					Expect(client.PushToRemote()).ShouldNot(Succeed())
				})
			})
		})
		When("repo is already initialized", func() {
			It("should fail to initialize twice", func() {
				client, err := NewGitClient(uid)
				Expect(err).Should(BeNil())
				Expect(os.RemoveAll(basedir)).Should(Succeed())
				Expect(client.Initialize(basedir)).Should(Succeed())
				Expect(client.Initialize(basedir)).ShouldNot(Succeed())
			})
		})
		When("repo is not initialized or cloned", func() {
			It("should fail to commit and push", func() {
				client, err := NewGitClient(uid)
				Expect(err).Should(BeNil())
				Expect(client.CommitFiles()).ShouldNot(Succeed())
				Expect(client.PushToRemote()).ShouldNot(Succeed())
			})
		})
	})
})
