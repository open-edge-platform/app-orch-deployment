# Dummy App Packages for Scale Testing

The Deployment Packages in this directory can be used to deploy a "dummy app"
to edge clusters.  The "dummy app" Helm chart installs only a Service (no Pods)
so it consumes almost no edge resources.  The "dummy app" is useful for App Orch
scale testing.

To load the Deployment Packages, the ["catalog" CLI tool](https://github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-app.cli.git) is recommended.
The following example shows how to load the Deployment Packages into the orchestrator's
Catalog service using the catalog CLI:

```bash
$ export FQDN=integration12.maestro.intel.com
$ catalog login -v --trust-cert=true --keycloak https://keycloak.${FQDN}/realms/master ${USER} ${PASS}
$ catalog --catalog-endpoint https://app-orch.${FQDN} load dummy-app-package/
$ catalog --catalog-endpoint https://app-orch.${FQDN} load ten-dummy-apps/
$ catalog --catalog-endpoint https://app-orch.${FQDN} list deployment-packages
Publisher Name   Name                Display Name     Version   Default Profile       Is Deployed   Is Visible   Application Count
default          dummy-app-package   Dummy app        0.0.1     default-profile       false         false        1
default          dummy-app-package   Dummy app        0.0.2     default-profile       false         false        1
default          ten-dummy-apps      10 dummy apps    0.0.1     default-profile       false         false        10
```
