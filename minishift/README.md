## Running Auth, Cluster services on OpenShift

These instructions will help you run your services on OpenShift using MiniShift.

### Prerequisites

[MiniShift v1.27.0](https://docs.openshift.org/latest/minishift/getting-started/installing.html)

[oc 3.11.0](https://docs.okd.io/latest/cli_reference/get_started_cli.html#installing-the-cli)

[KVM Hypervisor](https://www.linux-kvm.org/page/Downloads)

#### Install Minishift

Make sure you have all prerequisites installed. Please check the list [here](https://docs.openshift.org/latest/minishift/getting-started/installing.html#install-prerequisites)

Download and put `minishift` in your $PATH by following steps [here](https://docs.openshift.org/latest/minishift/getting-started/installing.html#manually)

Verify installation by running following command, you should get version number.
```bash
minishift version
```

#### Install oc
Please install and set up oc on your machine by visiting [oc](https://docs.openshift.org/latest/cli_reference/get_started_cli.html#installing-the-cli)

Verify installation by running following command, you should get version number.
```bash
oc version
```

### Deploying services on Minishift
Here, we are going to deploy `auth`, `cluster`, services on minishift.

#### Start Minishift
We have make target defined to start minishift with required cpu's and configuration.
```bash
make minishift-start
```
Please enter sudo password when prompted, it is needed in order to create an entry in the `/etc/hosts`.
`minishift ip` gives the IP address on which MiniShift is running. This automation creates a host entry as `minishift.local` for that IP. This domain is whitelisted on fabric8-auth.

Make sure to verify that your console is configured to reuse the Minishift Docker daemon by running `docker ps` command. You should be able to see running containers for origin.
If not then follow [this](https://docs.openshift.org/latest/minishift/using/docker-daemon.html#docker-daemon-overview) guide to configure it.

#### Create a Project
Let's create a new project by executing following make target.
```bash
make init-project
```

This will create a new project with name `fabric8-services` with `developer:developer` account and switch to it. Make sure to verify that using `oc project`.

#### Auth Service
##### Deploying Auth

To deploy auth service, we have following make target which will deploy required secrets, postgres DB and create routes for you.
```
make deploy-auth
```

Look for running pods using `oc get pods`. You should be able to see two pods(auth-*, db-auth-*). First time it will take some time as it has download required container images.

##### Check auth service status
You can get auth route by using `oc get routes`. It should be in format `auth-fabric8-services.${minishift ip}.nip.io`
You can check status by hitting this in browser `http://auth-fabric8-services.${minishift ip}.nip.io/api/status`(e.g. `http://auth-fabric8-services.192.168.42.177.nip.io/api/status`).

##### Connecting to Postgres DB
If you wish to access the Postgres database, it is available on the same host but on port 31001.  Use the following command to connect with the Postgres client:

```bash
PGPASSWORD=mysecretpassword psql -h minishift.local -U postgres -d postgres -p 31001
```

#### Cluster Service
##### Deploying Cluster

To deploy cluster service, we have following make target which will deploy required secrets, config map, postgres DB and create routes for you.
```
make deploy-cluster
```

Look for running pods using `oc get pods`. You should be able to see two pods(f8cluster-*, db-f8cluster-*). First time it will take some time as it has download required container images.

##### Check Cluster service status
You can get cluster route by using `oc get routes`. It should be in format `f8cluster-fabric8-services.${minishift ip}.nip.io`
You can check status by hitting this in browser `http://f8cluster-fabric8-services.${minishift ip}.nip.io/api/status`(e.g. `http://f8cluster-fabric8-services.192.168.42.177.nip.io/api/status`).

##### Connecting to Postgres DB
If you wish to access the Postgres database, it is available on the same host but on port 31002.  Use the following command to connect with the Postgres client:

```bash
PGPASSWORD=mysecretpassword psql -h minishift.local -U postgres -d postgres -p 31002
```

#### Deploying Auth, Cluster together
To deploy `auth`, `f8cluster` together we have following target:
```bash
make deploy-all
```

#### Cleaning Up

##### Cleaning Auth
This removes both the `auth` and `db-auth` services from minishift.
```bash
make clean-auth
```

##### Cleaning Cluster
This removes both the `f8cluster` and `db-f8cluster` services from minishift.
```bash
make clean-cluster
```
##### Cleaning Auth, Cluster
This removes `auth`, `f8cluster` services from minishift and deletes the `fabric8-services` project.
```bash
make clean-all
```

#### Redeploying Cluster service
However if you are working on cluster service and wants to redeploy latest code change by building container with latest binary. We have
special target for it which will do that for you.

It won't deploy required secrets and postgres db again. It'll re-deploy auth service only.

```bash
make redeploy-cluster
```

#### Checking services logs

List out all running services in MiniShift using
```
oc get pods
```
Wait until all pods are in running state and then copy pod name and use following command to see logs
```
oc logs <<pod name>> -f
```
