# Overview

The namespace_deleter.py program deletes Kubernetes namespaces.

This is a surprisingly non-trivial task. You can `kubectl delete ns` and if you're lucky the namespace will deleted. But, the namespace might also hang forever in `Terminating` state. There is no easy way to force delete a namespace.

See [Kubernetes issue 77086](https://github.com/kubernetes/kubernetes/issues/77086) for all the gory details.

# Usage

It takes as input the namespace to delete and a few optional arguments.


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

If you wish to install `k8s-namespace-deleter` as a kubectl plugin run:

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
$ kubectl create -f ns.yaml
namespace/delete-me-if-you-can created
```

This namespace has a finalizer and can't be deleted normally.
This command hangs:

```
$ kubectl delete ns delete-me-if-you-can
namespace "delete-me-if-you-can" deleted
```

It says the namespace was deleted, but actually it wasn't. 

After breaking let's check the namespace:

```
$ kubectl get ns delete-me-if-you-can
NAME                   STATUS        AGE
delete-me-if-you-can   Terminating   3m10s
```

As you can see it is stuck in `Terminating` status.

OK. Let's delete it with the kubectl plugin:

```
$ kubectl ns delete delete-me-if-you-can
2021/01/15 22:46:17 namespace delete-me-if-you-can was deleted successfully.
```

Alright. Looks like it worked 👏!

But, did it? let's try to get the namespace again with kubectl

```
$ kubectl get delete-me-if-you-can
error: the server doesn't have a resource type "delete-me-if-you-can"
```

Yeah, it really really worked! 🎉
