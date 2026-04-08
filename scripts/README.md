1. https://github.com/intel-innersource/frameworks.edge.one-intel-edge.scale.test-tools
2. Use scale-badhri branch in above repo.
3. Use this command in scripts/app-scale-tester folder to run nginx test: ./add-N-apps.sh -a 1 -o aW50ZWwtaXRlcC11c2VyOkNoYW5nZU1lT24xc3RMb2dpbiE=
4. How to construct the string after -o flag is explained in README of repo. 
5. The observability part doesnt work yet.
6. Use this command in scripts/app-scale-tester folder to run dummy app test: ./add-N-apps_dummyapp.sh -a 1 -o aW50ZWwtaXRlcC11c2VyOkNoYW5nZU1lT24xc3RMb2dpbiE=
7. scale test was run on orchestrator : scale.espd.infra-host.com
8. helpful notes:
    - label all clusters with label scale=adm : kubectl label -n 84d60f25-685c-4992-b60f-ea5a87549456 clusters.cluster.x-k8s.io --all scale=adm
    - remove label from certain cluster : kubectl label -n 84d60f25-685c-4992-b60f-ea5a87549456 clusters.cluster.x-k8s.io/cl-50-concurrent1 scale-
    - list all clusters with label scle=adm: kubectl get clusters.cluster.x-k8s.io -n 84d60f25-685c-4992-b60f-ea5a87549456 -l scale=adm
9. ARM and ASP tests were not run in this release
    - Need to understand what the scripts are doing. 


