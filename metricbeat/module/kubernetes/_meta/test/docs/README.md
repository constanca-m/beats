# Testing Metricbeat


### Pre requisites

You need to have installed in your computer:
- [Taskfile](https://taskfile.dev/installation/)
- [Elastic package](https://github.com/elastic/elastic-package)
- [kubectl](https://kubernetes.io/docs/tasks/tools/)
- [kind](https://kind.sigs.k8s.io/docs/user/quick-start/#installation)


### Step 1: Prepare local environment

In this step:
1. Create the elastic stack.
2. Create a local cluster.
3. Connect the node of the local cluster to the elastic stack network.
4. Deploy kube state metrics.

Run:
```shell
task setup:create
```


### Step 2: Deploy metricbeat

In this step:
1. Build the metricbeat binary.
2. Deploy the metricbeat manifest file. This is a slightly modified version of the [official deploy metricbeat manifest file](https://github.com/elastic/beats/blob/main/deploy/kubernetes/metricbeat-kubernetes.yaml).
   The daemonset executes an infinite sleep command instead of starting metricbeat.
3. Wait for the metricbeat pod to be ready.
4. Copy the metricbeat binary to the metricbeat container.
5. Execute the metricbeat inside the metricbeat container.

Pre-condition: There should not be any metricbeat pods in the cluster before running this task.

Run:
```shell
task beats:deploy_beats
```


### Changing the metricbeat binary

To change the metricbeat binary of the container, it is necessary to delete the pod and deploy it again.

In this step:
1. Delete metricbeat.
2. Go back to **step 2**.

Run:
```shell
task beats:delete_beats
```

Now go back to **step 2**.


### Cleanup

To delete the elastic stack and the cluster, run:

```shell
task setup:delete
```
