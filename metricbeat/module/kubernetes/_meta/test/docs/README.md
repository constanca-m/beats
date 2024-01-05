# Testing Metricbeat


## Option 1: Using Taskfile

### Pre requisites

You need to have your own cluster, and you need to be running the stack.
To do both, run:

```shell
task prepare-env
```

This creates the cluster, runs `elastic-package stack up` and connects the cluster to the stack. It also
deploys Kube State Metrics, necessary for the `state_*` metricsets. You can change the stack version by updating
the variable `STACK_VERSION` in the `Taskfile.yaml`.


### Step 1. Build and deploy metricbeat

Go to `01_playground/metricbeat.yaml` and make sure the variables to connect to the stack are configured correctly
according to your environment. If you ran the `task prepare-env` previously, then it should look like:
```yaml
output.elasticsearch:
  hosts: ["https://elasticsearch:9200"]
  username: ${ELASTICSEARCH_USERNAME}
  password: ${ELASTICSEARCH_PASSWORD}
  ssl.verification_mode: "none"
```

After that run:

```shell
task build-and-deploy-metricbeat
```

### Step 2. Copy and execute metricbeat

Metricbeat should be running now, otherwise the command will fail.

To copy and execute the metricbeat binary, run:

```shell
task copy-and-exec-metricbeat
```



### Changing the metricbeat binary

In case a new update is needed in the binary or configurations files, you need to delete metricbeat:

```shell
task delete-metricbeat
```

Now you need to go back to step 1.



## Option 2: Running the commands without Taskfile

### Step 1. Create kubernetes cluster using kind

Follow instructions at https://kind.sigs.k8s.io/docs/user/quick-start/#installation and install kind.

Create a kind kubernetes cluster.
```
kind create cluster  --image 'kindest/node:v1.21.1'
```

### Step 2. Deploy Kube-state-metrics

Prerequisite for collecting kubernetes meaningful metrics is kube-state-metrics.
Deploy it to your cluster manually by
```bash
git clone git@github.com:kubernetes/kube-state-metrics.git
cd kube-state-metrics/

kubectl apply -k .
```

### Step 3. Create ELK stack

You can spin up an ELK stack in two ways
1. [Proposed] Using elastic cloud https://cloud.elastic.co
2. Locally on your kind cluster (EK tuple will suffice).
```bash
# Deploy Elasticsearch and Kibana
cd metricbeat/module/kubernetes/_meta/test/docs
kubectl apply -f 01_playground/ek_stack.yaml

# Expose Kibana with port forwarding. In your browser visit localhost:5601
kubectl port-forward deployment/kibana 5601:5601
```


### Step 4. Playground Metricbeat Pod

A slightly modified (as of beats/deploy/kubernetes/metricbeat-kubernetes.yaml) all-in-one metricbeat manifest resides under 01_playground directory.
The daemonset executes an infinite sleep command instead of starting metricbeat.

ELASTICSEARCH_HOST, ELASTICSEARCH_PORT, ELASTICSEARCH_USERNAME, ELASTICSEARCH_PASSWORD variables are set according to local kind EK stack.

In case of Elastic Cloud deployment configure the variables ELASTIC_CLOUD_ID and ELASTIC_CLOUD_AUTH properly.

Deploy metricbeat
```
kubectl apply -f 01_playground/metricbeat.yaml
```

### Step 5. Build and launch metricbeat process

Next step is to build metricbeat binary and copy it in the running metricbeat pod.

Under beats/metricbeat execute

```bash
# Build metricbeat
GOOS=linux GOARCH=amd64 go build

# Copy binary in pod
kubectl cp ./metricbeat `kubectl get pod -n kube-system -l k8s-app=metricbeat -o jsonpath='{.items[].metadata.name}'`:/usr/share/metricbeat/ -n kube-system
````
The above command only copies metricbeat binary.
In case of configuration files updates it can be modified to copy also those files in the right container paths.

```bash
# Exec in the container and launch metricbeat
kubectl exec `kubectl get pod -n kube-system -l k8s-app=metricbeat -o jsonpath='{.items[].metadata.name}'` -n kube-system -- bash -c "metricbeat -e -c /etc/metricbeat.yml"
```
Metricbeat will launch and the process logs will appear in the terminal.

You can as well exec in metricbeat pod with bash command and then run metricbeat.
This gives the flexibility to easily start and stop the process.


### Test Iterations

In case a new update is needed in the binary or configurations files:
1. Delete the running metricbeat pod.
```bash
# Delete metricbeat
kubectl delete pod `kubectl get pod -n kube-system -l k8s-app=metricbeat -o jsonpath='{.items[].metadata.name}'`
```
2. Execute previous step (Build and launch metricbeat process)

