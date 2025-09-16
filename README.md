<!--
SPDX-FileCopyrightText: (C) 2025 Intel Corporation
SPDX-License-Identifier: Apache-2.0
-->

# Application Orchestration Deployment

[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](https://opensource.org/licenses/Apache-2.0)
[![ADM Component Test](https://github.com/open-edge-platform/app-orch-deployment/actions/workflows/adm-component-test.yml/badge.svg)](https://github.com/open-edge-platform/app-orch-deployment/actions/workflows/adm-component-test.yml)
[![ARM Component Test](https://github.com/open-edge-platform/app-orch-deployment/actions/workflows/arm-component-test.yml/badge.svg)](https://github.com/open-edge-platform/app-orch-deployment/actions/workflows/arm-component-test.yml)
[![ASP Component Test](https://github.com/open-edge-platform/app-orch-deployment/actions/workflows/asp-component-test.yml/badge.svg)](https://github.com/open-edge-platform/app-orch-deployment/actions/workflows/asp-component-test.yml)
[![OpenSSF Scorecard](https://api.scorecard.dev/projects/github.com/open-edge-platform/app-orch-deployment/badge)](https://scorecard.dev/viewer/?uri=github.com/open-edge-platform/app-orch-deployment)

## Overview

Application Orchestration Deployment is a collection of cloud-native applications (microservices) that facilitate the
deployment of user applications to clusters on Edge Nodes in the Open Edge Platform. Together with the [Application Catalog],
these applications constitute the **Application Orchestration** architecture layer.

Application Orchestration Deployment components work with the [Cluster Manager] to provide a powerful and flexible
platform for deploying applications to the Edge.

Application Orchestration Deployment components are all multi-tenant aware, with each instance able to handle multiple
multi-tenancy projects concurrently.

Application Orchestration Deployment components depend on the Edge Orchestrator [Platform Services] for many support
functions such as API Gateway, Authorization, Authentication, etc.

The overall architecture of the Application Orchestration environment is explained in the
Edge Orchestrator [Application Orchestration Developer Guide](https://docs.openedgeplatform.intel.com/edge-manage-docs/main/developer_guide/app_orch/arch/index.html).

## Get Started

Each of the applications has its own documentation that describes how to get started with them.

- [App Deployment Manager](app-deployment-manager/README.md)
- [App Resource Manager](app-resource-manager/README.md)
- [App Interconnect](app-interconnect/README.md)
- [App Service Proxy](app-service-proxy/README.md)

## Develop

All the applications are developed in the **Go** language and are built as Docker images. Each application has a `Dockerfile`
in its `build` folder. The CI integration for this repository will publish container images to the Edge Orchestrator
Release Service OCI registry upon merging to the `main` branch.

Each application has a corresponding Helm chart in its `deployment` folder. The CI integration for this repository will
publish these Helm charts to the Edge Orchestrator Release Service OCI registry upon merging to the `main` branch.
The applications are deployed to the Edge Orchestrator using these Helm charts, whose lifecycle is managed by
Argo CD (see [Platform Services]).

Some of the applications define their own Kubernetes Custom Resource Definitions (CRDs) and controllers to manage the
lifecycle of the resources they create. These are defined in the `api` directory of the application.

Some of the applications interact with the [Application Catalog] to retrieve information about the applications that
are available for deployment.

Some of the applications interact with the [Cluster Manager] and its Cluster API (CAPI) interface to follow the lifecycle
of Edge Node clusters.

### Dependencies

This code requires the following tools to be installed on your development machine:

- [Docker](https://docs.docker.com/engine/install/) to build containers
- [Go\* programming language](https://go.dev)
- [golangci-lint](https://github.com/golangci/golangci-lint)
- [Python\* programming language version 3.10 or later](https://www.python.org/downloads)
- [buf](https://github.com/bufbuild/buf)
- [protoc-gen-doc](https://github.com/pseudomuto/protoc-gen-doc)
- [protoc-gen-connect-go](https://connectrpc.com/) for Connect-RPC client/server generation
- [protoc-gen-connect-openapi](https://github.com/sudorandom/protoc-gen-connect-openapi) for OpenAPI spec generation
- [protoc-gen-go](https://pkg.go.dev/google.golang.org/protobuf)
- [KinD](https://kind.sigs.k8s.io/docs/user/quick-start/) based cluster for end-to-end tests
- [Helm](https://helm.sh/docs/intro/install/) for install helm charts for end-to-end tests

## Build

Below are some of important make targets which developer should be aware about.

Build the component binary as follows:

```bash
# Build go binary
make build
```

Unit test checks are run for each PR and developer can run the unit tests locally as follows:

```bash
# Run unit tests
make test
```

Linter checks are run for each PR and developer can run linter check locally as follows:

```bash
make lint
```

Multiple container images are generated from this repository. They are  `app-service-proxy`,
`app-interconnect-manager`, `adm-gateway`, `adm-controller`, `app-resource-rest-proxy`,
`app-resource-vnc-proxy` and `app-resource-manager`. Command to generate container images is
as follows:

```bash
make docker-build
```

If developer has done any helm chart changes then helm charts can be build as follows:

```bash
make helm-build
```

## Contribute

We welcome contributions from the community! To contribute, please open a pull request to have your changes reviewed
and merged into the `main` branch. We encourage you to add appropriate unit tests and end-to-end tests if
your contribution introduces a new feature. See [Contributor Guide] for information on how to contribute to the project.

## Community and Support

To learn more about the project, its community, and governance, visit the [Edge Orchestrator Community].
For support, start with [Troubleshooting] or [Contact us].

## License

Application Orchestration Deployment is licensed under [Apache 2.0 License].

[Application Catalog]: https://github.com/open-edge-platform/app-orch-catalog
[Cluster Manager]: https://github.com/open-edge-platform/cluster-manager
[Platform Services]: https://docs.openedgeplatform.intel.com/edge-manage-docs/main/developer_guide/platform/index.html
[Contributor Guide]: https://docs.openedgeplatform.intel.com/edge-manage-docs/main/developer_guide/contributor_guide/index.html
[Troubleshooting]: https://docs.openedgeplatform.intel.com/edge-manage-docs/main/developer_guide/troubleshooting/index.html
[Contact us]: https://github.com/open-edge-platform
[Edge Orchestrator Community]: https://docs.openedgeplatform.intel.com/edge-manage-docs/main/index.html
[Apache 2.0 License]: LICENSES/Apache-2.0.txt
