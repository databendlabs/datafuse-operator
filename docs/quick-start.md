# Quick Start Guide
For more detailed introduction about how datafuse operator works, please take a look at [User Guide](user-guide.md).
If you want to contribute and change API, please take a look at [Development Guide](development-guide.md) at first.

## Table of Contents
* [Installation](#installation)
* [Bootstrapping a cluster](#bootstrap-a-cluster)
* [Testing and Debugging](#testing-the-cluster)
* [Configuration](#configuration)

## Installation
### Environment setup
1. Install [kubectl](https://kubernetes.io/docs/tasks/tools/)
2. Install [Minikube](https://minikube.sigs.k8s.io/docs/start/), Here used kubernetes 1.20.2
```bash
minikube start --cpus 8 --memory 16384 --driver kvm2
```

### Install operator through kubectl
The following command will install crds and install datafuse-operator on datafuse-operator-system namespace
```bash
kubectl apply -f ./examples/operator.yaml
```

Wait until operator get ready
```bash
kubectl get deployments.apps -n datafuse-operator-system

NAME                          READY   UP-TO-DATE   AVAILABLE   AGE
datafuse-controller-manager   1/1     1            1           17s
```


## Bootstrap a cluster
The following command will install a operator hold one compute group
this group has 4 pods.
```bash
kubectl apply -f ./examples/demo.yaml
```

## Testing the cluster
Validate the readiness status for each running instances
```bash
kubectl get pods

NAME                                   READY   STATUS    RESTARTS   AGE
multi-worker-leader-6897f84d44-wjdz7   1/1     Running   0          77s
multi-worker-worker0-6f48985b-xbgrv    1/1     Running   0          77s
multi-worker-worker1-9bb968997-6jlht   1/1     Running   0          77s
multi-worker-worker1-9bb968997-h7trt   1/1     Running   0          77s
```

### Query through serivce based on local machine

Check on the compute group service
```bash
kubectl get svc
NAME           TYPE        CLUSTER-IP      EXTERNAL-IP   PORT(S)                                        AGE
kubernetes     ClusterIP   10.96.0.1       <none>        443/TCP                                        18h
multi-worker   ClusterIP   10.105.246.63   <none>        8080/TCP,3307/TCP,9000/TCP,9090/TCP,7070/TCP   4m4s
```

Port forward the mysql port for service
```bash
kubectl port-forward service/multi-worker 3307:3307
```

Connect the compute group based on local machine
```bash
mysql -h localhost -P 3307
```

Check on all nodes in the cluster
```bash
mysql> SELECT * from system.clusters;

+----------------------------------------------+-----------------+----------+
| name                                         | address         | priority |
+----------------------------------------------+-----------------+----------+
| default/multi-worker-worker1-9bb968997-h7trt | 172.17.0.6:9090 |        1 |
| default/multi-worker-worker1-9bb968997-6jlht | 172.17.0.7:9090 |        1 |
| default/multi-worker-worker0-6f48985b-xbgrv  | 172.17.0.5:9090 |        2 |
| default/multi-worker-leader-6897f84d44-wjdz7 | 172.17.0.4:9090 |        1 |
+----------------------------------------------+-----------------+----------+
```

Do sql queries
```bash
mysql> SELECT sum(number) FROM numbers_mt(100000000000);
+--------------------+
| sum(number)        |
+--------------------+
| 932355974711512064 |
+--------------------+
1 row in set (11.71 sec)
```
### Check on logs in each compute instance
Use kubectl logs command to check on query execution plan on each node
```bash
kubectl logs multi-worker-leader-6897f84d44-wjdz7 fusequery 
```