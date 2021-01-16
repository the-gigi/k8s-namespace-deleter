# Overview

The namespace_deleter.py program deletes Kubernetes namespaces.

This is a surprisingly non-trivial task. You can `kubectl delete ns` and if you're lucky the namespace will deleted. But, the namespace might also hang forever in `Terminating` state. There is no easy way to force delete a namespace.

# Usage

It takes as input the namespace to delete and a few optional arguments

The following arguments are mandatory:
* the namespace to delete

The following flags are optional:
* --kube-context (will use current context if not specified)
* --kube-config  (kubectl will use its default algorithm to search for credentials)
* --port         (kubectl proxy port, default to 8080)

Example command:

```
k8s-namespace-deleter obsolete-namespace                /
                      --kube-context some-context       /
                      --kube-config  ~/.kube/config     /
                      --port         8888
```

# Kubectl plugin

If you wish to install k8s-namespace-deleter as a kubectl plugin run:

```
go build -o /usr/local/bin/kubectl-ns-delete
```

Now you can force delete any namespace with the following kubectl command:

```
kubectl ns delete <namespace>
```

# Quick test

Let's create a new namespace using the following YAML:

```
apiVersion: v1
kind: Namespace
metadata:
  name: try-to-delete-me-if-you-can
spec:
  finalizers:
    - foregroundDeletion
```

Save it to ns.yaml and type:

```
apiVersion: v1
kind: Namespace
metadata:
  name: delete-me-if-you-can
spec:
  finalizers:
    - foregroundDeletion 
```



```
(🐙)/k8s-namespace-deleter/
$ kubectl create namespace ttt
namespace/ttt created

(🐙)/k8s-namespace-deleter/
$ kg ns ttt
NAME   STATUS   AGE
ttt    Active   39s
```

OK. Let's delete it with the kubectl plugin:

```
[19:40:48] (🐙)/k8s-namespace-deleter/
$ kubectl ns delete ttt
2021/01/15 19:41:46 namespace ttt was deleted successfully.
```

Alright. Looks like it worked 👏!

But, did it? let's try to get the namespace again with kubectl

```
[19:41:46] (🐙)/k8s-namespace-deleter/
$ kubectl get ns ttt
Error from server (NotFound): namespaces "ttt" not found
```

Yeah, it really really worked! 🎉
