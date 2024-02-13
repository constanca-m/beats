# Testing filebeat


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

Run:
```shell
task setup:create
```

**Important note**: This task will fail with
```shell
task: `test $(filebeat) = "metricbeat"` failed
task: Failed to run task "setup:create": task: precondition not met
```
This is ok and expected behaviour.

### Step 2: Deploy filebeat

In this step:
1. Build the filebeat binary.
2. Deploy the filebeat manifest file. This is a slightly modified version of the [official deploy filebeat manifest file](https://github.com/elastic/beats/blob/main/deploy/kubernetes/filebeat-kubernetes.yaml).
   The daemonset executes an infinite sleep command instead of starting filebeat.
3. Wait for the filebeat pod to be ready.
4. Copy the filebeat binary to the filebeat container.
5. Execute the filebeat inside the filebeat container.

Pre-condition: There should not be any filebeat pods in the cluster before running this task.

Run:
```shell
task beats:deploy_beats
```


### Changing the filebeat binary

To change the filebeat binary of the container, it is necessary to delete the pod and deploy it again.

In this step:
1. Delete filebeat.
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
