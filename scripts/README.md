

# App-Orch Scale testing
This README documents details on the following topics
- App-Orch scale testing
- Tools used
- Different scale strategies (ENiC vs vCluster)
- Script used and how to run them
- Results collected
- Debugging
- Cleanup

## Pre-requisite

### Gnu Plot
```shell  
sudo apt-get update
sudo apt-get install gnuplot
```  
Verified gnuplot version **5.4 patchlevel 2**.

### K6 API load tester tool
Use instructions at https://k6.io/docs/get-started/installation/ to install k6. Recommended version **v0.47.0**.

### vCluster
Refer https://www.vcluster.com/. Please install vcluster version **0.19.6**.

### kubectl
Refer https://kubernetes.io/docs/tasks/tools/ and install kubectl for your platform. Recommended version **v1.28.9**.

### JQ and YQ
- [JQ](https://jqlang.github.io/jq/download/) - **jq-1.6** or above
- [YQ](https://mikefarah.gitbook.io/yq) - **v4.33.3** or above

### Bash shell
Bash shell version **5.1.16** or above.

### Catalog CLI tool
Refer to https://github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-app.cli.

## Running the cluster scale tester scripts

### Using ENIC based edge node emulator for scale tests
The following steps detail scale tests based on ENIC based edge node emulator.

The scale tester scripts can be used to bring up multiple clusters and install multiple apps on  
them. The clusters / apps are installed in batches of 10 by default. After each batch is ready,  
the test scripts will run API latency tests and collect relevant metrics from the Observability service.

To create a number of clusters using the Configured hosts and wait for all clusters to be Ready:
```shell
cd ./enic-scale-tester
./add-N-clusters.sh -c <total-clusters-to-setup> -o <observability-api-credentials>
```  

#### Other options to the scripts

- `-b` : Batch Size of clusters to install. **Default 10**
- `-f` : Cluster FQDN. **Default integration12.maestro.intel.com**
- `-u` : Keycloak username. **Default all-groups-example-user**
- `-p` : Keycloak password. **Default ChangeMeOn1stLogin!**
- `-a` : Apps per enic. **Default 3**


### Using vCluster based edge node emulator for scale tests
The following steps detail using [vCluster](https://www.vcluster.com/) based scale tests.

#### Case1: A host cluster already exists to host the vClusters
**Pre-requisite:**
- Set KUBECONFIG env variable to kubeconfig of the target cluster
- The script expects that `apps` namespace is pre-created on the target clusters. If vclusters are to be installed in to a different NS, the script needs to be modified.
- The target namespace has sufficient resource limits to host the required number of vclusters


To create a number of clusters using the Configured hosts and wait for all clusters to be Ready:
```shell
cd vcluster-scale-tester/
./add-N-virtual-clusters.sh -c <total-clusters-to-setup> -o <observability-api-credentials>
```  

Use help option like below to learn more options that is exposed by the script

```shell
./add-N-virtual-clusters.sh -h
```  

#### Case2: No host cluster but only bare-metal servers with Ubuntu 22.04

Do the following on all the servers that are used to host vClusters

##### Install Tools
```shell
# Install KinD
[ $(uname -m) = x86_64 ] && curl -Lo ./kind https://kind.sigs.k8s.io/dl/v0.23.0/kind-linux-amd64
# For ARM64
[ $(uname -m) = aarch64 ] && curl -Lo ./kind https://kind.sigs.k8s.io/dl/v0.23.0/kind-linux-arm64
chmod +x ./kind
sudo mv ./kind /usr/local/bin/kind

# Install Helm
curl https://baltocdn.com/helm/signing.asc | gpg --dearmor | sudo tee /usr/share/keyrings/helm.gpg > /dev/null
sudo apt-get install apt-transport-https --yes
echo "deb [arch=$(dpkg --print-architecture) signed-by=/usr/share/keyrings/helm.gpg] https://baltocdn.com/helm/stable/debian/ all main" | sudo tee /etc/apt/sources.list.d/helm-stable-debian.list
sudo apt-get update
sudo apt-get install helm

# Install vCluster
wget https://github.com/loft-sh/vcluster/releases/download/v0.19.6/vcluster-linux-amd64
chmod +x vcluster-linux-amd64
sudo mv vcluster-linux-amd64 /usr/local/bin/vcluster

# Install kubectl
curl -LO "https://dl.k8s.io/release/$(curl -L -s https://dl.k8s.io/release/stable.txt)/bin/linux/amd64/kubectl"
chmod +x kubectl
sudo mv kubectl /usr/local/bin/kubectl
```
##### Copy intel-harbor-ca.crt
Do this in the home folder of the server
```shell
openssl s_client -showcerts -connect amr-registry.caas.intel.com:443 < /dev/null | sed -ne '/-BEGIN CERTIFICATE-/,/-END CERTIFICATE-/p' > ~/intel-harbor-ca.crt
```

##### Create KinD cluster config file
Create file named `kind-config.yaml` in the home folder of the server with below content.
**Note**: Change the `apiServerAddress` to the actual IP address of the server.

```shell
kind: Cluster
apiVersion: kind.x-k8s.io/v1alpha4
networking:
  apiServerAddress: 10.3.162.65
nodes:
- role: control-plane
  extraMounts:
    - hostPath: /tmp/var/log/
      containerPath: /var/log/
    - hostPath: /tmp/var/log/containers/
      containerPath: /var/log/containers/
    - hostPath: /tmp/var/lib/rancher/rke2/agent/logs/
      containerPath: /var/lib/rancher/rke2/agent/logs/
    - hostPath: /tmp/var/lib/rancher/rke2/server/logs/
      containerPath: /var/lib/rancher/rke2/server/logs/
    - hostPath: /tmp/dev/lvmvg/
      containerPath: /dev/lvmvg/
containerdConfigPatches:
  - |-
    [plugins."io.containerd.grpc.v1.cri".registry]
      [plugins."io.containerd.grpc.v1.cri".registry.mirrors]
        [plugins."io.containerd.grpc.v1.cri".registry.mirrors."*"]
          endpoint = ["https://dockerhubcache.caas.intel.com"]
      [plugins."io.containerd.grpc.v1.cri".registry.configs]
        [plugins."io.containerd.grpc.v1.cri".registry.configs."amr-registry.caas.intel.com".tls]
          ca_file = "/usr/local/share/ca-certificates/intel-harbor-ca.crt"
        [plugins."io.containerd.grpc.v1.cri".registry.configs."dockerhubcache.caas.intel.com".tls]
          ca_file = "/usr/local/share/ca-certificates/intel-harbor-ca.crt"
kubeadmConfigPatches:
  - |
    apiVersion: kubelet.config.k8s.io/v1beta1
    kind: KubeletConfiguration
    maxPods: 500
```

##### Enable public key access
Enable public key access to these servers from the server which is used to run the automation scripts.

*We are finally ready to run the automation script*

```shell
cd vcluster-scale-tester/
./add-virtual-clusters-on-kind.sh -s "<username1>@<ip1>/<totalVClusters1>,<username2>@<ip2>/<totalVClusters2>"

## Example
./add-virtual-clusters-on-kind.sh -s "labrat@10.123.232.168/10,labrat@10.123.232.172/10,root@10.237.213.34/10,root@10.237.213.151/10,testbeduser@10.228.254.158/5,labuser@10.3.162.217/10,labuser@10.3.162.88/10,labuser@10.3.162.105/10,labuser@10.3.162.65/10"
```

Other options to the script ->
```shell
❯ ./add-virtual-clusters-on-kind.sh -h
Usage: ./add-virtual-clusters-on-kind.sh [options] [--] [arguments]

Options:
  -s VALUE      List of servers (comma separated) and vclusters to install on those servers. Ex: "labuser@10.3.162.217/30,labuser@10.3.162.105/30"
  -u VALUE      Keycloak username, default all-groups-example-user
  -p VALUE      Keycloak password, default ChangeMeOn1stLogin!
  -f VALUE      Orch FQDN, default integration12.maestro.intel.com
  -b VALUE      Cluster install batch size, default 10
  -o VALUE      Observability API credentials base64 encoded
  -a VALUE      Apps per ENIC, default 1
  -k VALUE      vClusters per kind host, default 30
  -r VALUE      Path of amr-registry.caas.intel.com registry public certificate.
  -h            Print this help menu

  In the below example, we ask to install 30 vclusters on labuser@10.3.162.217 and 10 vclusters on labuser@10.3.162.105 (using -s).
  We also specify the path of Intel AMR CaaS public certificate (using -r). We then specify that only 10 vclusters should be hosted per kind cluster

  ./add-virtual-clusters-on-kind.sh -s "labuser@10.3.162.217/30,labuser@10.3.162.105/10" -r ./intel_harbor_ca.crt -k 10
```

### Running the app scale test script and collecting metrics

#### ADM API Scale tester
To deploy multiple copies of the "dummy app" to place load on the App Orch control plane without exhausting scarce edge resources:
```shell  
cd app-scale-tester/
./add-N-apps.sh -a <total-apps-to-setup> -o <observability-api-credentials>
```
The script generates various plots at the end of the test on API latency and Resource usage. Look for the results in the `./test-results/<timestamp>` folder.
The exact folder where the results are generated is logged at the end of the script execution.

*NOTE:* Refer section [Getting observability api credentials](#getting-observability-api-credentials) further in this README on how to get Observability API Token to get the observability metrics.

#### ARM API and ASP Scale tester

To run the ARM and API scale tester do the following. This script scales the number of concurrent users linearly,
and then measures the ARM API and ASP latency, and also Maestro-App-System Namespace resource usage during the process
```shell
cd k6-scripts
./arm-asp-api-performance-test.sh -a <app-id> -o <observability-api-credentials>
```
For more details on the options to the script use below command
```shell
./arm-asp-api-performance-test.sh -h
```

At the end of the test result, all the metrics are collected and graphs are generated in the `./test-results/<timestamp>` folder.
The exact folder where the results are generated is logged at the end of the script execution.

## Getting observability api credentials

```shell  
export KUBECONFIG=<provide kubeconfig file for orch cluster>  
  
kubectl get secret mp-observability-grafana -n maestro-platform-system -o go-template='{{range $k,$v := .data}}{{printf "%s: " $k}}{{if not $v}}{{$v}}{{else}}{{$v | base64decode}}{{end}}{{"\n"}}{{end}}'  
  
admin-password: <admin-password>
admin-user: <admin-user>
ldap-toml:  
```  

The output credentials are decoded so now base64 encode `<admin-user>:<admin-password>` with the colon char inbetween and provide that as `<observability-api-credentials>`

**Note:** when orchestrator or system restarts, the password will change so you will need to get new credentials and encode again.

## Test Results
All results will be stored in `./test-results/<time-stamp>` folder. It will be a collection of PNG files generated by GNU Plot in PNG file format and the source CSV files that were used to generate the GNU plots.  
The shell script also dumps a lot of logs that could be used to trace the test progress.

## Debugging
### vClusters setup on Scale Load cluster
```shell
# 1: Set the KUBECONFIG to the right kubeconfig file pointing to scale load cluster
export KUBECONFIG=<scale-load cluster kubeconfig file>
# 2: Connect to problematic cluster and execute the kubectl command of interest to debug
vcluster connect <vcluster-name> -n apps -- kubectl get pods -A
```
**NOTE:** All vclusters are hosted in `apps` namespace by default for app-orch scale testing.
### vClusters setup on bare-metal servers
The vCluster name on bare-metal servers have the format - `vcluster-edge-node-<hypen-separated-ip>-<vcluster-index`. An example is `vcluster-edge-node-10-3-162-88-17`, where the IP of the server hosting the vCluster is `10.3.162.88` and the vcluster index on that server is `17`.

Once the IP of the server hosting the vCluster is known, we need to find the KinD cluster index that hosts the vCluster. To do that find the quotient of `vcluster-index / 30`. For instance if the vcluster-index is 17, in that case, `17 / 30 = 0`. Now 0 is the kind cluster index.
Once we know the following details, use instructions below to access the vcluster.

- IP address of the server hosting the vcluster
- KinD cluster index hosting the vCluster on that server
```shell
#### Instructions ####
# 1: Login to the server based on the IP address we found before
# 2. Set the kubeconfig of the kind cluster using the below command formt
# Format: 
# kubectl config use-context kind-kind-node-<hypen-separated-ip-address-of-server>-<kind-cluster-index>
# Example:
kubectl config use-context kind-kind-node-10-3-162-88-0
# 3: list vClusters on that KinD cluster
vcluster list -n apps
# 4: connect to vcluster and execute the kubectl command of interest to debug further
# Example:
vcluster connect -n apps vcluster-edge-node-10-3-162-88-17 -- kubectl get pods -A
```
**NOTE:** All vclusters are hosted in `apps` namespace by default for app-orch scale testing.

## Cleanup
### vClusters cleanup on Scale Load cluster
Use the following instructions
1.  Delete those clusters from Rancher UI. Bulk deletes are possible on Rancher UI by selecting several clusters.
2.  Delete specific vclusters with `vcluster delete -n apps <vcluster-name>`. Make sure you have set the right kubeconfig to be able to access the vcluster. To delete batch of vclusters, use `utils/delete-vclusters.sh -c <total-cluster-to-delete>` script.
3.  If the associated pods are still in Terminating state after vcluster tool has deleted the vclusters, then force delete those pods in Scale Load cluster with below command
```
kubectl delete pod -n apps --force `kubectl get pods -n apps | grep Terminating | awk '{print $1}'`
```
4. Then re-setup the clusters that are necessary to fill the gap on Scale Load cluster with below command

```
 ./add-N-virtual-clusters.sh -c <total-cluster-to-setup> -a <default-apps-to-setup> -i <start-index-to-use-for-vcluster>
```

**NOTE**:

- All vclusters are hosted in `apps` namespace by default for app-orch scale testing.
- Choose `start-index-to-use-for-vcluster` such that it is the first available index with no index in use after that. It defaults to 0 if none specified.

### vClusters cleanup on bare-metal servers
Use below instructions

-  Delete those clusters from Rancher UI. Bulk deletes are possible on Rancher UI by selecting several clusters.
- Login to baremetal hosting the vCluster. Setup the kubeconfig to the KinD cluster hosting the vCluster. Delete the vcluster now

At this time it is not possible to add vClusters on a bare-metal in an additive manner. So please delete all the clusters associated with a server on Rancher UI using bulk delete, then delete all the KinD clusters (which also deletes the vClusters automatically) created for hosting vClusters. Once this cleanup is done, setup the required number of vClusters on this baremetal using the instructions shared in previous sections.
