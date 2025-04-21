<!--
SPDX-FileCopyrightText: (C) 2025 Intel Corporation
SPDX-License-Identifier: Apache-2.0
-->

# Application Deployment Manager

Application Deployment Manager is focused on providing a friendly, high-level interface on top of the GitOps-based deployment tool [Rancher Fleet] and [CAPI] (k8s Cluster API) for cluster management. Application Deployment Manager provides a simple REST API for creating and managing the lifecycle of Deployments. Internally it uses [k8s Operator pattern] and [k8s CRDs] to automate generating Fleet resources, pushing them to Git repositories, and creating the Fleet Custom Resources necessary to bootstrap GitOps for a Deployment.

See the links below for further details:

- [Developer's Guide] | additional information on Application Deployment Manager and other application orchestration components

- [Tutorials and Examples] | examples for developing, deploying and managing applications

- [Developer's Troubleshooting Guide] | guidance on troubleshooting issues related to Application Deployment Manager

- [API Spec] | Application Deployment Manager openAPI specification

# Getting Started
Instructions on how to test, build and redeploy Application Deployment Manager on your development machine. You can use a [KinD] cluster as the orchestator for local developing. Most targets will install the required tools to execute the target. There's 1 target which installs dependencies to generate protobuf code and openAPI spec (see `make install-protoc-plugins` below)

## Common Test, Build and Redeploy Make Targets

```bash
# If changes to controller CRDs then generate WebhookConfiguration, ClusterRole and CustomResourceDefinition objects
make manifests
```

```bash
# Install dependencies like protoc plugins and Buf. This is used to generate protobuf code and openAPI spec
make install-protoc-plugins
```

```bash
# If changes to protobuf files then generate the protobuf code using Buf tool
make proto-generate
```

```bash
# Build the docker images for the Controllers and Gateway
make docker-build
```

```bash
# Test OPA policies
make opa-test
```

```bash
# Controllers and Gateway unit tests
make test
```

```bash
# Go fuzzing tests
make go-fuzz
```

### Development Targets

```bash
# If helm value changes, redeploy the Controllers and Gateway in the KinD cluster. Note: this will not build images
make coder-redeploy
```

```bash
# If CRD changes, redeploy the Controllers, Gateway and CRDs in the KinD cluster. Note: this will not build images
make coder-redeploy-adm-crd
```

```bash
# If changes to Gateway, build and redeploy the Gateway and use same version as in the Orchestrator
make coder-rebuild-gateway
```

```bash
# If changes to any of the controllers, build and redeploy the controllers and use same version as in the Orchestrator
make coder-rebuild-adm
```


## License
The Application Deployment Manager is licensed under [Apache 2.0 License]

[Buf]: https://buf.build/docs/
[k8s CRDs]: https://kubernetes.io/docs/tasks/extend-kubernetes/custom-resources/custom-resource-definitions
[CAPI]: https://github.com/kubernetes-sigs/cluster-api/tree/main
[k8s Operator pattern]: https://kubernetes.io/docs/concepts/extend-kubernetes/operator
[Rancher Fleet]: https://fleet.rancher.io
[KinD]: https://sigs.k8s.io/kind

[Developer's Troubleshooting Guide]: https://docs.openedgeplatform.intel.com/edge-manage-docs/main/developer_guide/troubleshooting/app_orch.html
[Developer's Guide]: https://docs.openedgeplatform.intel.com/edge-manage-docs/main/developer_guide/app_orch/arch/key_components.html#application-deployment-manager
[API Spec]: api/spec/openapi.yaml
[Apache 2.0 License]: LICENSES/Apache-2.0.txt
[Tutorials and Examples]: https://docs.openedgeplatform.intel.com/edge-manage-docs/main/developer_guide/app_orch/tutorials/index.html
