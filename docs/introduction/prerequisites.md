# Prerequisites
To collaborate on the External Secrets Operator (ESO) project, you need to install some tools on your computer. This guide explains what each tool is, why it is needed, the recommended version, and how to install it on the corresponding operating system.
### Supported Operating Systems
To collaborate on the External Secrets Operator (ESO) project, it is recommended to use Unix-based operating systems, such as Linux and macOS. ESO's development environment is primarily designed for these systems, and many of the tools and scripts used during development are built to work on them.
### Can You Develop on Windows?
It is possible to use Windows for development, but there are important considerations to keep in mind. Since ESO's development environment is not optimized for Windows, compatibility issues may arise with tools like Make, Tilt, and shell scripts. The project's automation scripts and commands are written for Unix environments, using bash scripting, which might not be compatible with Windows without adaptations. This tutorial will not cover the installation and configuration of tools on Windows due to its complexity and lack of testing.

---

## Install Go (Golang)

<details>
  <summary>About Golang</summary>
  <h3> What is Go?</h3>
  <p> Go, also known as Golang, is a programming language design at Google by Robert Griesemer, Rob Pike, and Ken Thompson. It is known for being efficient, easy to learn, and excellent for developing fast and scalable applications.</p>
  <h3> Why is Go needed?</h3>
  <p> In the <strong>External Secrets Operator</strong> project, Go is used to develop core parts of the code. It is required to compile, run, and contribute to the project's source code.</p>
</details>

<details>
  <summary>Golang Installation</summary>
  <h3> Required Version</h3>
  <p><strong>Minimum version:</strong> Go 1.20 or higher.</p>
  <p><strong>Recommended version:</strong> Go 1.24.1</p>
  <blockquote> As of this writing, the latest version of Go is <strong>1.24.1</strong>As of this writing, the latest version of Go is 1.24.1 , which worked perfectly with the <strong>External Secrets Operator</strong> project. Previous versions failed to test the application. Before testing the project, check your Go version.</blockquote>

  <h3> How to Install Go</h3>

Please consult the <a href="https://go.dev/doc/install">official documentation</a>.</p>
</details>

## Install Helm

<details>
  <summary>About Helm</summary>
  <h3>What is Helm?</h3>
  <p>Helm is a package manager for Kubernetes, the platform that automates deployment, scaling, and management of containerized applications.</p>

  <h3>Why is Helm necessary?</h3>
  <p>In the <strong>External Secrets Operator</strong> project, Helm is used to simplify the installation and management of applications within Kubernetes, automating complex configuration and deployment processes.</p>
</details>

<details>
  <summary>Installing Helm</summary>
  <h3>Required Version</h3>
  <p><strong>Recommended version:</strong> Helm 3 (latest version of Helm 3).</p>
  <h3>How to Install Helm</h3>


  <p>Please consult the <a href="https://helm.sh/docs/intro/install/">official Helm installation guide</a>.</p>
</details>

---

## Install yq

<details>
  <summary>About yq</summary>
  <h3>What is yq?</h3>
  <p>yq is a command-line tool for reading, manipulating, and writing YAML files, which are widely used for configurations.</p>

  <h3>Why is yq necessary?</h3>
  <p>In the <strong>External Secrets Operator</strong> project, yq is used to automate the editing of YAML configuration files, facilitating adjustments and implementations.</p>
</details>

<details>
  <summary>Installing yq</summary>
  <h3>Required Version</h3>
  <p><strong>Recommended version:</strong> yq v4.44.3 or higher.</p>

  <h3>How to Install yq</h3>

  <p>Please consult the <a href="https://github.com/mikefarah/yq">official yq repository</a>.</p>
</details>


---

## Install jq

<details>
  <summary>About jq</summary>
  <h3>What is jq?</h3>
  <p>jq is a command-line tool for processing and manipulating JSON data.</p>

  <h3>Why is jq needed?</h3>
  <p>In the <strong>External Secrets Operator</strong> project, jq is essential for working with JSON data, enabling efficient filtering and transformation of information.</p>
</details>

<details>
  <summary>Installing jq</summary>
  <h3>Required Version</h3>
  <p><strong>Recommended version:</strong> jq 1.6 or later.</p>

  <h3>How to Install jq</h3>

  <p>Please consult the <a href="https://stedolan.github.io/jq/">official jq website</a>.</p>
</details>

## Kubernetes

<details>
  <summary>About Kubernetes</summary>
  <h3>What is Kubernetes?</h3>
  <p>Kubernetes is an open-source platform for automating the deployment, scaling, and management of containerized applications. It orchestrates containerized workloads across a cluster of machines, ensuring high availability and efficient resource utilization.</p>

  <h3>Why is Kubernetes needed?</h3>
  <p>In the <strong>External Secrets Operator</strong> project, Kubernetes provides the infrastructure to deploy and manage containerized applications. It allows integration with cloud-native services, enabling scalability, fault tolerance, and streamlined operations in dynamic environments.</p>

  <p>
  To work with Kubernetes, we need to install and configure some tools first. This will be explained in the sections below.

  For more details, check the [official documentation](https://kubernetes.io/docs/home/).
</p>
</details>

---
## Install Docker

<details>
  <summary>About Docker</summary>
  <h3>What is Docker?</h3>
  <p>Docker is a platform for building, deploying, and running applications in containers. Containers package an application with all its dependencies into a standard unit for development and deployment.</p>

  <h3>Why is Docker needed?</h3>
  <p>In the <strong>External Secrets Operator</strong> project, Docker is used to create container images and run services in isolated environments. It is essential for developing, testing, and deploying the application within a Kubernetes environment.</p>
</details>

<details>
  <summary>Installing Docker</summary>
  <h3>How to Install Docker</h3>

  <p>Please consult the <a href="https://docs.docker.com/get-docker/">official Docker documentation</a>.</p>
</details>

<details>
  <summary>Required Docker Configuration</summary>
  <h3>Configure Docker for Non-Root Usage</h3>
  <p>By default, Docker requires superuser (root) privileges to run. To simplify usage, it is recommended to add the current user to the <code>docker</code> group to execute commands without <code>sudo</code>.</p>

  <details>
    <summary>Steps to configure Docker without root on Linux</summary>
    <h3>1. Create the docker group (if it doesn't exist):</h3>
    <pre><code>sudo groupadd docker</code></pre>
    <h3>2. Add the current user to the docker group:</h3>
    <pre><code>sudo usermod -aG docker $USER</code></pre>
    <h3>3. Apply group changes without logging out:</h3>
    <pre><code>newgrp docker</code></pre>
    <h3>4. Verify Docker can run without sudo:</h3>
    <pre><code>docker run hello-world</code></pre>
    <p>If the command works without errors, the configuration is successful.</p>
  </details>
</details>

---
## Install kubectl

<details>
  <summary>About kubectl</summary>
  <h3>What is kubectl?</h3>
  <p><strong>kubectl</strong> is the command-line tool for managing Kubernetes clusters. It enables running commands on the cluster, managing resources, and debugging applications.</p>
  <h3>Why is kubectl needed?</h3>
  <p>In the <strong>External Secrets Operator</strong> project, kubectl is used to interact with local or remote Kubernetes clusters, apply configurations, and check the state of deployed resources.</p>
</details>

<details>
  <summary>Installing kubectl</summary>
  <h3>Required Version</h3>
  <p><strong>A version compatible with the installed Kubernetes version (usually the latest stable version).</strong></p>
  <h3>How to Install kubectl</h3>

  <p>Please consult the <a href="https://kubernetes.io/docs/tasks/tools/">official kubectl documentation</a>.</p>
</details>

---
## Install ctlptl and Create a Kind Cluster with Local Registry

<details>
  <summary>About ctlptl</summary>
  <h3>What is ctlptl?</h3>
  <p><strong>ctlptl</strong> (Control Plane Tool) is a tool for managing local Kubernetes development clusters. It simplifies the creation and management of clusters like <strong>Kind</strong> (Kubernetes in Docker) and the configuration of local container registries.</p>
  <h3>Why is ctlptl necessary?</h3>
  <p>In the <strong>External Secrets Operator</strong> project, ctlptl is used to create and manage a local Kubernetes cluster using Kind, as well as to configure a local container registry to store Docker images during development.</p>
</details>

<details>
  <summary>Installing ctlptl</summary>
  <h3>Required Version</h3>
  <p><strong>The latest available version of ctlptl.</strong></p>

  <h3>How to Install ctlptl</h3>
    <p>Please consult the <a href="https://github.com/tilt-dev/ctlptl/blob/main/INSTALL.md
    ">official ctlptl installation guide</a>.</p>
</details>


<h3>Create a Kind Cluster with Local Registry</h3>

<details>
  <summary>About Kind</summary>
  <p><strong>Kind</strong> (Kubernetes in Docker) is a tool to run local Kubernetes clusters using Docker containers as cluster nodes.</p>
</details>

<details>
  <summary>How to Create a Kind Cluster with Local Registry</summary>
  <h3>1. Create a local container registry:</h3>
  <pre><code>docker run -d --restart=always -p "5000:5000" --name kind-registry registry:2</code></pre>
  <h3>2. Create a Kind cluster using ctlptl and connect it to the local registry:</h3>
  <pre><code>ctlptl create cluster kind --registry=kind-registry</code></pre>
  <p>This will create a Kind cluster configured to use the local registry at <code>localhost:5000</code>.</p>
  <h3>3. Verify the cluster is running:</h3>
  <pre><code>kubectl cluster-info --context kind-kind</code></pre>
  <h3>4. List clusters managed by ctlptl:</h3>
  <pre><code>ctlptl get clusters</code></pre>
</details>

---
## Install Tilt

<details>
  <summary>About Tilt</summary>
  <h3>What is Tilt?</h3>
  <p><strong>Tilt</strong> is a tool that accelerates development in Kubernetes environments. It automates building, deploying, and monitoring code, enabling a faster development cycle.</p>
  <h3>Why is Tilt necessary?</h3>
  <p>In the <strong>External Secrets Operator</strong> project, Tilt is used to develop and test code changes efficiently, reflecting updates almost instantly in the local Kubernetes environment.</p>
</details>

<details>
  <summary>Installing Tilt</summary>
  <h3>Required Version</h3>
  <ul>
    <li><strong>Prerequisites:</strong> Install Docker, kubectl, Kind, and ctlptl.</li>
    <li><strong>Recommended version:</strong> Latest available version.</li>
  </ul>

  <h3>How to Install Tilt</h3>

  <p>Please consult the <a href="https://docs.tilt.dev/install.html">official Tilt installation guide</a>.</p>
</details>