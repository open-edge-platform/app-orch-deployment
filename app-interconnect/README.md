<!---
  SPDX-FileCopyrightText: (C) 2022 Intel Corporation
  SPDX-License-Identifier: Apache-2.0
-->

# Application Interconnect Service

[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](https://opensource.org/licenses/Apache-2.0)


## Overview

The Application Interconnect Service is a microservice that provides a set of APIs for managing the interconnectivity 
between applications running on separate edge clusters. Interconnect uses a third-party component called [Skupper] to
build a layer 7 application mesh. This L7 mesh allows an application
deployed on one edge cluster to connect to a service on a different edge cluster as if it were a local service, unaware that
it resides on a different edge. The L7 mesh uses mutual TLS to secure the connection.

The overall architecture of the Application Orchestration environment is explained in the
Edge Orchestrator [Application Orchestration Developer Guide](https://docs.openedgeplatform.intel.com/edge-manage-docs/main/developer_guide/app_orch/arch/index.html).

## Get Started

The Application Interconnect Service is deployed as a microservice in the Edge Orchestrator environment. It is 
deployed as a Docker image. The image is published to the Edge Orchestrator Release Service OCI registry.

The Interconnect Manager defines its own Kubernetes Custom Resource Definitions (CRDs) and controllers to manage the
networking resources it creates. These are defined in the `deploy/charts/app-interconnect-manager/crds` directory of the application.


## Develop

The Application Interconnect service is developed in the **Go** language and is built as a Docker image, through a `Dockerfile`
in its `build` folder. The CI integration for this repository will publish the container image to the Edge Orchestrator
Release Service OCI registry upon merging to the `main` branch.

The Interconnect Manager has a corresponding Helm chart in its [deploy](deploy) folder.
The CI integration for this repository will
publish this Helm chart to the Edge Orchestrator Release Service OCI registry upon merging to the `main` branch.
The Interconnect Manager is deployed to the Edge Orchestrator using this Helm chart, whose lifecycle is in turn managed by
Argo CD (see [Foundational Platform]).

The repository also contains the source code for the [Developer Guide App Orch Tutorial].

## Contribute

We welcome contributions from the community! To contribute, please open a pull request to have your changes reviewed
and merged into the `main` branch. We encourage you to add appropriate unit tests and end-to-end tests if
your contribution introduces a new feature. See [Contributor Guide] for information on how to contribute to the project.

Additionally, ensure the following commands are successful:

```shell
make test
make lint
make license
```

## Community and Support

To learn more about the project, its community, and governance, visit the [Edge Orchestrator Community].

For support, start with [Troubleshooting] or [Contact us].

## License

The Application Orchestration Catalog is licensed under [Apache 2.0 License]

[Application Orchestration Deployment]: https://github.com/open-edge-platform/app-orch-deployment
[Tenant Controller]: https://github.com/open-edge-platform/app-orch-tenant-controller
[Cluster Extensions]: https://github.com/open-edge-platform/cluster-extensions
[Foundational Platform]: https://docs.openedgeplatform.intel.com/edge-manage-docs/main/developer_guide/platform/index.html
[Contributor Guide]: https://docs.openedgeplatform.intel.com/edge-manage-docs/main/developer_guide/contributor_guide/index.html
[Troubleshooting]: https://docs.openedgeplatform.intel.com/edge-manage-docs/main/developer_guide/troubleshooting/index.html
[Contact us]: https://github.com/open-edge-platform
[Edge Orchestrator Community]: https://docs.openedgeplatform.intel.com/edge-manage-docs/main/index.html
[Apache 2.0 License]: LICENSES/Apache-2.0.txt
[Developer Guide App Orch Tutorial]: app-orch-tutorials/developer-guide-tutorial/README.md
[Skupper]: https://skupper.io/
