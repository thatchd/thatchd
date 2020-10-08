# Thatchd

Thatchd is a testing framework for Kubernetes where test cases are first class Kubernetes resources that are dispatched and executed by a controller based on the cluster state. It allows developers to leverage the Kubernetes controller pattern and Custom Resources by
injecting their custom testing logic. 

## Try it

Thatchd is still under early development, but you can try it's functionallity
with the example test suite that's included in the repo.

### Pre-requisites

* [operator-sdk](https://sdk.operatorframework.io/docs/installation/install-operator-sdk/) v1.0.0
* Admin access to a Kubernetes cluster

### Set up

Clone the repo and install resources in the cluster
```sh
git clone https://github.com/sergioifg94/thatchd.git
cd thatchd
make install
```

Start running the operator
```sh
make run ENABLE_WEBHOOKS=false
```

### Create CRs

The example test suite is included in the repo. The logic is injected to the
`TestSuite` and `TestCase` controllers, and defined in the [`example` package](./example)

This example suite tests that Pods in the namespace have specific annotations,
failing the test case if they don't.

#### TestSuite

Create the TestSuite CR with the `PodsSuiteProvider`

```yaml
apiVersion: thatchd.io/v1alpha1
kind: TestSuite
metadata:
  name: test-pods
spec:
  initialState: '{}'
  stateStrategy:
    provider: PodsSuiteProvider
```

Once created, Thatchd will reconcile the status with a list of Pods in the namespace.
Go ahead and create a simple Pod. Thatchd will reconcile the `status` field accordingly

```yaml
status:
  currentState: |-
    {
      "my-pod": true
    }
```

> ℹ You can use any Go type as test state, leveraging the language type information

#### TestCase

The example test case will be dispatched when a specific pod is ready according
to the TestSuite state. Create a TestCase CR to verify that the `foo: bar` annotation
is set on the `test-success` Pod

```yaml
apiVersion: thatchd.io/v1alpha1
kind: TestCase
metadata:
  name: testcase-success
spec:
  strategy:
    configuration:
      expectedAnnotation: foo
      expectedValue: bar
      podName: test-success
    provider: PodAnnotationProvider
```

> ℹ️ The `configuration` field in the CR allows to reuse logic in multiple test cases

The test case won't be dispatched yet as the Pod hasn't been created

Create a Pod called `test-success` with the `foo: bar` annotation

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: test-success
  labels:
    app: hello-openshift
  annotations:
    foo: bar
spec:
  containers:
    - name: hello-openshift
      image: openshift/hello-openshift
      ports:
        - containerPort: 8080
```

Once the Pod is ready, the TestCase will be dispatched, and quickly executed,
setting the status to `Finished`

Create another TestCase that will fail:

```yaml
apiVersion: thatchd.io/v1alpha1
kind: TestCase
metadata:
  name: testcase-fail
spec:
  strategy:
    configuration:
      expectedAnnotation: foo
      expectedValue: baz
      podName: test-fail
    provider: PodAnnotationProvider
```

Create a Pod with no annotations

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: test-fail
  labels:
    app: hello-openshift
spec:
  containers:
    - name: hello-openshift
      image: openshift/hello-openshift
      ports:
        - containerPort: 8080
```

The TestCase will soon be dispatched, and the error message will be reflected
in the status field, among other useful information:

```yaml
failureMessage: 'Annotation foo: baz not found in Pod'
```