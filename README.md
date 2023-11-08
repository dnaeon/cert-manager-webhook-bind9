# cert-manager-webhook-bind9

`cert-manager-webhook-bind9` is an [ACME DNS-01 Solver for Cert
Manager](https://cert-manager.io/docs/configuration/acme/dns01/webhook/),
which uses BIND as the DNS provider.

In order to solve ACME challenges and create the respective TXT
records this webhook uses [TSIG
keys](https://en.wikipedia.org/wiki/TSIG) when communicating with
BIND.

# Installation

Install with Helm.

``` bash
helm repo add cert-manager-webhook-bind9 https://dnaeon.github.io/cert-manager-webhook-bind9

helm install \
	--namespace cert-manager \
	cert-manager-webhook-bind9 \
	cert-manager-webhook-bind9/cert-manager-webhook-bind9 \
	--set groupName=acme.mydomain.tld
```

Install without Helm.

``` bash
make rendered-manifest.yaml
kubectl apply -f _out/rendered-manifest.yaml
```

In order to uninstall the webhook execute the following command.

``` bash
helm uninstall --namespace cert-manager cert-manager-webhook-bind9
```

# Usage

Create a TSIG key, which will be shared between the DNS-01 Solver and
your authoritative DNS servers.

``` bash
tsig-keygen -a hmac-sha256 acme-key > acme-tsig.key
```

Create a secret for the TSIG key.

``` bash
kubectl --namespace cert-manager create secret generic acme-tsig.key \
    --from-file=acme-tsig.key \
    --dry-run=client -o yaml | kubectl --namespace cert-manager apply -f -
```

Create an `Issuer` or `ClusterIssuer`, e.g.

``` yaml
apiVersion: cert-manager.io/v1
kind: Issuer
metadata:
  name: step-ca-issuer
  namespace: cert-manager
spec:
  acme:
    email: you@example.com
    server: https://stepca:9000/acme/acme/directory
    caBundle: <BASE64_CA_Bundle>
    privateKeySecretRef:
      name: step-ca-acme-issuer-account-key
    solvers:
      - dns01:
          webhook:
            groupName: acme.your-domain.tld
            solverName: bind9
            config:
              allowedZones:
                - zone1.your-domain.tld.
                - zone2.your-domain.tld.
              ttl: 300
              tsigKeyRef:
                name: acme-tsig.key
                key: acme-tsig.key
```

And now request a certificate using the issuer.

``` yaml
apiVersion: cert-manager.io/v1
kind: Certificate
metadata:
  name: test-cert-01
  namespace: cert-manager
spec:
  secretName: test-cert-tls-01
  issuerRef:
    name: step-ca-issuer
  dnsNames:
    - "foo.zone1.your-domain.tld"
```

# Tests

In order to run the DNS-01 provider conformance test suite, follow
the steps below.

Create a Docker network, which will be used by the test BIND service
and our test suite.

``` bash
docker network create webhook_test --help --subnet 172.16.0.0/24
```

Build and start the BIND service.

``` bash
docker compose --file docker-compose.test.yaml up --build --remove-orphans bind9
```

Run the conformance test suite.

``` bash
docker compose --file docker-compose.test.yaml up --build --remove-orphans tests
```

While the tests are running you can watch the logs of the `bind9`
service, where you should see zone update events.

All tests should and you should see an output similar to the one
below.

``` text
Attaching to cert-manager-webhook-bind9-tests-1
cert-manager-webhook-bind9-tests-1  | curl -fsSL https://go.kubebuilder.io/test-tools/1.28.3/linux/amd64 -o kubebuilder-tools.tar.gz
cert-manager-webhook-bind9-tests-1  | mkdir -p _test/kubebuilder
cert-manager-webhook-bind9-tests-1  | tar -xvf kubebuilder-tools.tar.gz
cert-manager-webhook-bind9-tests-1  | kubebuilder/
cert-manager-webhook-bind9-tests-1  | kubebuilder/bin/
cert-manager-webhook-bind9-tests-1  | kubebuilder/bin/etcd
cert-manager-webhook-bind9-tests-1  | kubebuilder/bin/kubectl
cert-manager-webhook-bind9-tests-1  | kubebuilder/bin/kube-apiserver
cert-manager-webhook-bind9-tests-1  | mv kubebuilder/bin/* _test/kubebuilder/
cert-manager-webhook-bind9-tests-1  | rm kubebuilder-tools.tar.gz
cert-manager-webhook-bind9-tests-1  | rm -R kubebuilder
cert-manager-webhook-bind9-tests-1  | /usr/local/go/bin/go test -v .
cert-manager-webhook-bind9-tests-1  | === RUN   TestRunsSuite
cert-manager-webhook-bind9-tests-1  | === RUN   TestRunsSuite/Conformance
cert-manager-webhook-bind9-tests-1  | === RUN   TestRunsSuite/Conformance/Basic
cert-manager-webhook-bind9-tests-1  | === RUN   TestRunsSuite/Conformance/Basic/PresentRecord
cert-manager-webhook-bind9-tests-1  |     util.go:70: created fixture "basic-present-record"
cert-manager-webhook-bind9-tests-1  |     suite.go:38: Calling Present with ChallengeRequest: &v1alpha1.ChallengeRequest{UID:"", Action:"", Type:"", DNSName:"example.com", Key:"123d==", ResourceNamespace:"basic-present-record", ResolvedFQDN:"cert-manager-dns01-tests.example.com.", ResolvedZone:"example.com.", AllowAmbientCredentials:false, Config:(*v1.JSON)(0xc00058d218)}
cert-manager-webhook-bind9-tests-1  | === RUN   TestRunsSuite/Conformance/Extended
cert-manager-webhook-bind9-tests-1  | === RUN   TestRunsSuite/Conformance/Extended/DeletingOneRecordRetainsOthers
cert-manager-webhook-bind9-tests-1  |     suite.go:70: skipping test as strict mode is disabled, see: https://github.com/cert-manager/cert-manager/pull/1354
cert-manager-webhook-bind9-tests-1  | --- PASS: TestRunsSuite (10.09s)
cert-manager-webhook-bind9-tests-1  |     --- PASS: TestRunsSuite/Conformance (7.18s)
cert-manager-webhook-bind9-tests-1  |         --- PASS: TestRunsSuite/Conformance/Basic (1.44s)
cert-manager-webhook-bind9-tests-1  |             --- PASS: TestRunsSuite/Conformance/Basic/PresentRecord (1.44s)
cert-manager-webhook-bind9-tests-1  |         --- PASS: TestRunsSuite/Conformance/Extended (0.00s)
cert-manager-webhook-bind9-tests-1  |             --- SKIP: TestRunsSuite/Conformance/Extended/DeletingOneRecordRetainsOthers (0.00s)
cert-manager-webhook-bind9-tests-1  | PASS
cert-manager-webhook-bind9-tests-1  | ok        github.com/dnaeon/cert-manager-webhook-bind9    10.107s
cert-manager-webhook-bind9-tests-1 exited with code 0

```

## Regenerate the test TSIG key

In order to regenerate the test TSIG key follow these steps.

First, create a new TSIG key.

``` bash
tsig-keygen -a hmac-sha256 acme-key  > docker/bind9/acme-tsig.key
```

Update the test suite configuration as well.

``` bash
kubectl create secret generic acme-tsig-key \
	--from-file docker/bind9/acme-tsig.key \
	-o yaml \
	--dry-run=client > testdata/cert-manager-webhook-bind9/tsig-key-secret.yaml
```

# License

`cert-manager-webhook-bind9` is Open Source and licensed under the
[BSD License](https://opensource.org/license/bsd-2-clause/).
