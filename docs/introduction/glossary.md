# Glossary
This glossary includes technologies related to ESO in alphabetic order.


 <details>
  <summary>Cluster</summary>

  <h3> What is it? </h3>
      <p> A group of nodes (computers, VMs) that execute workloads in Kubernetes, i.e., run containerized applications.</p>
      <p>It's a technique that groups multiple computational resources into a single logical unit. These resources are interconnected and work together to execute tasks, process data, and store information in a way that improves application performance, ensures high availability, reduces costs, and increases scalability, as resources can be shared and distributed efficiently to meet real-time application demands. Each computer is a "node," and there's no limit to the number of nodes that can be interconnected. The structure is : Project (Clusters(Nodes(Pods))).</p>
      <p>The cluster is what provides the main advantage of Kubernetes: the ability to program and execute containers on a set of physical, virtual, on-premise, or cloud machines. Kubernetes containers are not tied to individual machines. In fact, they are abstracted across the entire cluster.</p>

  <h3> What is it for? </h3>
      <p>The cluster's function is to group multiple machines into a single, efficient system, allowing distributed applications to be executed with higher performance and scalability. In Kubernetes, it facilitates container management, reducing complexity, ensuring high availability, and reducing costs. A Kubernetes cluster typically has a master node that manages pods and the system's execution environment.</p>

  <h3> Useful links: </h3>

  <ul>
        <li><a href="https://kubernetes.io/docs/concepts/cluster-administration/" target="_blank">Introduction to Clusters</a></li>
        <li><a href="https://aws.amazon.com/pt/what-is/kubernetes-cluster/" target="_blank">What is a Kubernetes cluster?</a></li>
        <li><a href="https://www.atatus.com/blog/kubernetes-clusters-everything-you-need-to-know/" target="_blank">Kubernetes Clusters: Everything You Need to Know</a></li>
      </ul>

   </details>

 <details>
  <summary>Docker</summary>

  <h3>What is it?</h3>
    <p>Docker is an open platform for developing, shipping, and running containerized applications. It allows you to separate your applications from the infrastructure, facilitating the delivery of software quickly and efficiently, enabling the creation, sharing, and execution of containerized applications and microservices.</p>

  <h3>What is it for?</h3>
    <p>It enables infrastructure management. This significantly reduces the time between writing code and executing it in production.
    It simplifies complex processes such as port mapping, file system concerns, and other standard configurations, allowing you to focus on writing code.</p>
    <p>With Docker, you can develop an application and its supporting components using containers. In this context, the container becomes the unit for distributing and testing the application. Once ready, you can deploy the application to the production environment, whether it's local, cloud-based, or hybrid.</p>

  <h3>Useful links:</h3>

  <li><a href="https://docs.docker.com/" target="_blank">Official documentation for Docker</a></li>

   </details>

<details>
  <summary>Golang</summary>

  <h3>What is it?</h3>
    <p>An open-source programming language created by Google, known for its simplicity, performance, clarity, and conciseness.</p>

  <h3>What is it for?</h3>
    <p>
      Used in the development of applications, backend systems, and tools, especially in cloud and Kubernetes environments.
      It's a language that offers concurrency mechanisms that facilitate writing programs capable of taking full advantage of multi-core machines and networks, while its innovative type system enables the construction of flexible and modular programs.
      Go compiles quickly to machine code and, at the same time, offers convenience with garbage collection and the power of runtime reflection. It's a compiled, statically typed language that has the agility of dynamically typed and interpreted languages.
    </p>

  <h3>Useful links:</h3>
  <ul>
    <li><a href="https://go.dev/doc/" target="_blank">Official documentation for Golang</a></li>
  </ul>
</details>

<details>
  <summary>Helm</summary>

  <h3>What is it?</h3>
    <p>A package manager for Kubernetes that facilitates the deployment and management of applications using templates called "charts."</p>

  <h3>What is it for?</h3>
    <p>
      Simplifies the configuration, installation, and update of applications in Kubernetes.
    </p>

  <h3>Useful links:</h3>
  <ul>
    <li><a href="https://helm.sh/docs/" target="_blank">Official documentation for Helm</a></li>
    <li><a href="https://github.com/helm/helm" target="_blank">GitHub project</a></li>
  </ul>
</details>



<details>
  <summary>HPA</summary>

  <h3>What is it?</h3>
    <p>Horizontal Pod Autoscaler (HPA)</p>

  <h3>What is it for?</h3>
    <p>
      It's used to control the number of Pods in a Deployment. For example, if CPU usage is too high, the HPA would increase the number of Pods.
      It's also possible to use the Vertical Pod Autoscaler (VPA), which would increase the amount of resources for each Pod instead of increasing the number of Pods.
    </p>
</details>

<details>
  <summary>Ingress</summary>

  <h3>What is it?</h3>
    <p>
      In a Kubernetes cluster where all requests arrive at the same IP and port, Ingresses are responsible for directing (based on rules you define via the Kubernetes API) these requests to the appropriate Services. It can also be used for other purposes.
    </p>

  <h3>What is it for?</h3>
    <p>
      It provides a single entry point for routing traffic to internal services.
    </p>

  <h3>Useful links:</h3>
  <ul>
    <li><a href="https://kubernetes.io/docs/concepts/services-networking/ingress/" target="_blank">About Ingress</a></li>
  </ul>
</details>

<details>
  <summary>Issuer</summary>

  <h3>What is it?</h3>
    <p>A component in tools like Cert-Manager for issuing certificates.</p>

  <h3>What is it for?</h3>
    <p>
      Manages the issuance of automatic TLS certificates for services in Kubernetes.
      It issues the SSL certificate for Ingresses to encrypt (with HTTPS) incoming and outgoing requests, for example.
    </p>

  <h3>Useful links:</h3>
  <ul>
    <li><a href="https://cert-manager.io/docs/" target="_blank">Cert-Manager documentation</a></li>
  </ul>
</details>

<details>
  <summary>Kind</summary>

  <h3>What is it?</h3>
    <p>
      Kind means "Kubernetes in Docker", so it is a tool for running local Kubernetes clusters using Docker containers as cluster "nodes."
    </p>

  <h3>What is it for?</h3>
  <p>
    Kind was initially designed for testing Kubernetes itself, but it can also be used for local development or continuous integration (CI).
    It enables the creation of Kubernetes clusters easily in local environments, facilitating testing and development without requiring complex infrastructure.
  </p>

  <h3>Useful links:</h3>
  <ul>
    <li><a href="https://kind.sigs.k8s.io/" target="_blank">Official Website</a></li>
  </ul>
</details>

<details>
  <summary>Kubectl</summary>

  <h3>What is it?</h3>
    <p>
      Kubectl is a command-line tool for communicating with the control plane of a Kubernetes cluster, using the Kubernetes API.
    </p>

  <h3>What is it for?</h3>
    <p>
      It performs operations in Kubernetes, such as creating pods and monitoring the cluster status.
      It allows you to interact with the Kubernetes cluster by performing operations like creating, managing, and viewing resources.
      It searches for a configuration file called <code>config</code> in the <code>$HOME/.kube</code> directory, which contains information about how to connect to the cluster.
    </p>

  <h3>Useful links:</h3>
  <ul>
    <li><a href="https://kubernetes.io/docs/reference/kubectl/" target="_blank">Official Documentation - Kubernetes</a></li>
  </ul>
</details>

<details>
  <summary>Kubernetes</summary>

  <h3>What is it?</h3>
    <p>
      A container orchestration open source platform that automates the deployment, scaling, and management of applications.
    </p>

  <h3>What is it for?</h3>
    <p>
      Ensures high availability, scalability, and monitoring of containerized applications.
    </p>

  <h3>Useful links:</h3>
  <ul>
    <li><a href="https://kubernetes.io/docs/" target="_blank">Official Documentation</a></li>
    <li><a href="https://github.com/kubernetes/kubernetes" target="_blank">Project GitHub</a></li>
  </ul>
</details>


<details>
  <summary>Nginx</summary>

  <h3>What is it?</h3>
    <p>
      It is an open-source HTTP web server that can also function as a reverse proxy, load balancer, content cache, TCP/UDP proxy server, and email proxy server.
      It is widely used due to its high performance and ability to handle large volumes of traffic.
    </p>

  <h3>What is it for?</h3>
    <p>
      Nginx is used to serve web content, manage network traffic, and balance load between servers, as well as act as a reverse proxy and content cache.
      It can be used to improve the scalability and performance of web applications by efficiently distributing requests across multiple servers.
      It has a main process that manages the configuration and several worker processes that handle request processing. The number of worker processes can be adjusted according to the number of processor cores.
    </p>

  <h3>Useful Links:</h3>
  <ul>
    <li><a href="https://nginx.org/en/docs/" target="_blank">Official Documentation</a></li>
  </ul>
</details>

<details>
  <summary>Lint</summary>

  <h3>What is it?</h3>
    <p>
      A static code analysis process for identifying errors, style issues, and non-compliance with best coding practices.
    </p>

  <h3>What is it for?</h3>
    <p>
      Ensures code quality, consistency, and adherence to predefined standards by identifying syntax errors, formatting issues, and poor development practices before code execution.
      It contributes to maintaining clean, readable, and efficient code.
    </p>

  <h3>Useful Links:</h3>
  <ul>
    <li><a href="https://eslint.org/" target="_blank">Introduction to linting</a></li>
  </ul>
</details>

<details>
  <summary>Pod</summary>

  <h3>What is it?</h3>
    <p>
      The smallest unit of computation in Kubernetes, which groups one or more containers.
    </p>

  <h3>What is it for?</h3>
    <p>
      Manages containers that share resources and act as a single entity in a cluster.
      The structure is: Project (Clusters(Nodes(Pods))).
    </p>

  <h3>Useful Links:</h3>
  <ul>
    <li><a href="https://kubernetes.io/docs/concepts/workloads/pods/" target="_blank">About Pods</a></li>
  </ul>
</details>

<details>
  <summary>Secret</summary>

  <h3>What is it?</h3>
    <p>
      Sensitive data we want to store, manage, and use with ESO.
    </p>
</details>

<details>
  <summary>Tilt</summary>

  <h3>What is it?</h3>
    <p>
      A tool that helps with local development for Kubernetes, enabling quick visualization and management of changes to applications.
    </p>

  <h3>What is it for?</h3>
    <p>
      Facilitates the development workflow in Kubernetes by automatically updating the cluster's state based on code changes.
      It has an interface and automates many tasks that would otherwise need to be done manually.
    </p>

  <h3>Useful Links:</h3>
  <ul>
    <li><a href="https://tilt.dev/" target="_blank">Official Website</a></li>
    <li><a href="https://docs.tilt.dev/" target="_blank">Documentation</a></li>
  </ul>
</details>

<details>
  <summary>yq</summary>

  <h3>What is it?</h3>
    <p>
      A tool used to manipulate YAML files in the command line, similar to jq for JSON.
    </p>

  <h3>What is it for?</h3>
    <p>
      Edits, transforms, and queries YAML files. YAML files are used to configure applications, services, or clusters.
    </p>

  <h3>Useful Links:</h3>
  <ul>
    <li><a href="https://github.com/mikefarah/yq" target="_blank">yq GitHub</a></li>
  </ul>
</details>
