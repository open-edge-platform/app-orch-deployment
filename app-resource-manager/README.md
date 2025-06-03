<!---
  SPDX-FileCopyrightText: (C) 2022 Intel Corporation
  SPDX-License-Identifier: Apache-2.0
-->

# App Resource Manager

[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](https://opensource.org/licenses/Apache-2.0)

## Overview

The App Resource Manager (ARM) is a microservice within the Application Orchestration architecture.
The ARM microservice enables users to control and access different kinds of resources,
including VMs, containers, etc., using an API.

The overall architecture of the Application Orchestration environment is explained in the
Edge Orchestrator [Application Orchestration Developer Guide](https://docs.openedgeplatform.intel.com/edge-manage-docs/main/developer_guide/app_orch/arch/index.html).

## Get Started

The AppResource Manager is deployed as a microservice in the Edge Orchestrator environment. It is
deployed as a Docker image. The image is published to the Edge Orchestrator Release Service OCI registry.

## Develop

The App Resource Manager service is developed in the **Go** language and is built as a Docker image, through a `Dockerfile`
in its `build` folder. The CI integration for this repository will publish the container image to the Edge Orchestrator
Release Service OCI registry upon merging to the `main` branch.

The ARM has a corresponding Helm chart in its [deployments](deployments) folder.
The CI integration for this repository will
publish this Helm chart to the Edge Orchestrator Release Service OCI registry upon merging to the `main` branch.
ARM is deployed to the Edge Orchestrator using this Helm chart, whose lifecycle is in turn managed by
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

[Foundational Platform]: https://docs.openedgeplatform.intel.com/edge-manage-docs/main/developer_guide/platform/index.html
[Contributor Guide]: https://docs.openedgeplatform.intel.com/edge-manage-docs/main/developer_guide/contributor_guide/index.html
[Troubleshooting]: https://docs.openedgeplatform.intel.com/edge-manage-docs/main/developer_guide/troubleshooting/index.html
[Contact us]: https://github.com/open-edge-platform
[Edge Orchestrator Community]: https://docs.openedgeplatform.intel.com/edge-manage-docs/main/index.html
[Apache 2.0 License]: LICENSES/Apache-2.0.txt
[Developer Guide App Orch Tutorial]: app-orch-tutorials/developer-guide-tutorial/README.md
