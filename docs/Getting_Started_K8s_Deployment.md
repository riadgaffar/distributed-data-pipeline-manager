# **Getting Started Guide for Kubernetes Deployment**

This guide will walk you through the setup and deployment of your distributed data pipeline project in a Kubernetes cluster.

---

## **Prerequisites**
Before starting, ensure you have the following installed and configured:
1. **Kubernetes Cluster**: Use Minikube, Kind, or a cloud-based Kubernetes cluster.
2. **kubectl**: Kubernetes command-line tool.
3. **Docker**: For building and pushing images if required.
4. **kustomize**: Built into `kubectl` for managing overlays.
5. **Metrics Server** (Optional): For resource monitoring.

---

## **Directory Structure**

Ensure your project follows this structure:

```
deployments/
└── k8s/
    ├── base/
    │   ├── deployments/
    │   │   ├── pipeline-manager-deployment.yaml
    │   │   ├── postgres-deployment.yaml
    │   │   ├── redpanda-deployment.yaml
    │   │   └── ...
    │   ├── services/
    │   ├── configmaps/
    │   ├── pvc/
    │   └── secrets/
    ├── overlays/
    │   ├── dev/
    │   │   ├── kustomization.yaml
    │   │   ├── pipeline-manager-patch.yaml
```

---

## **Step 1: Build Docker Images**

1. Build Docker images for your services (if they’re not already pushed to a registry):

   ```bash
   docker build -t pipeline-manager:latest .
   docker build -t postgres:latest .
   docker build -t redpanda:latest .
   ```

2. If you’re using a local Kubernetes cluster (e.g., Minikube or Kind), load the images:

   **Minikube**:
   ```bash
   eval $(minikube docker-env)
   docker build -t pipeline-manager:latest .
   ```

   **Kind**:
   ```bash
   kind load docker-image pipeline-manager:latest
   ```

---

## **Step 2: Create Secrets**

Ensure all required secrets are created for the Kubernetes cluster:

```bash
kubectl create secret generic postgres-secret \
  --from-literal=POSTGRES_DB=pipelines \
  --from-literal=POSTGRES_USER=admin \
  --from-literal=POSTGRES_PASSWORD=password \
  -n dev

kubectl create secret generic grafana-secret \
  --from-literal=GF_SECURITY_ADMIN_USER=admin \
  --from-literal=GF_SECURITY_ADMIN_PASSWORD=admin \
  -n dev
```

---

## **Step 3: Apply the Base Configuration**

Navigate to the `base` directory and verify the configuration:

1. Validate Kubernetes manifests:
   ```bash
   kubectl apply -k deployments/k8s/base --dry-run=client
   ```

2. Apply the base configuration:
   ```bash
   kubectl apply -k deployments/k8s/base
   ```

---

## **Step 4: Apply Development Overlay**

Switch to the `dev` overlay for environment-specific configurations:

1. Navigate to the `dev` overlay directory:
   ```bash
   cd deployments/k8s/overlays/dev
   ```

2. Validate the overlay:
   ```bash
   kubectl apply -k . --dry-run=client
   ```

3. Deploy to the cluster:
   ```bash
   kubectl apply -k .
   ```

---

## **Step 5: Verify Deployment**

1. Check the status of all resources:
   ```bash
   kubectl get all -n dev
   ```

2. View logs for running pods:
   ```bash
   kubectl logs <pod-name> -n dev
   ```

3. Check PVCs and storage:
   ```bash
   kubectl get pvc -n dev
   ```

4. If using the Metrics Server:
   ```bash
   kubectl top pods -n dev
   ```

---

## **Troubleshooting**

### **Common Issues**

#### 1. `ImagePullBackOff`
- **Cause**: Kubernetes cannot find or pull the image.
- **Solution**:
  - Load local images into Minikube/Kind.
  - Push images to a container registry and update deployment YAMLs.

#### 2. `CrashLoopBackOff`
- **Cause**: Missing environment variables or initialization issues.
- **Solution**:
  - Check pod logs:
    ```bash
    kubectl logs <pod-name> -n dev
    ```
  - Verify secrets and environment variables.

#### 3. `Pending`
- **Cause**: PVC not bound or insufficient cluster resources.
- **Solution**:
  - Check PVC status:
    ```bash
    kubectl get pvc -n dev
    ```
  - Adjust resource requests in deployment YAMLs.

---

## **Step 6: Access Services**

Expose and access the services using the Kubernetes `Service` objects:

- Forward ports for local testing:
  ```bash
  kubectl port-forward svc/<service-name> <local-port>:<service-port> -n dev
  ```
  Example for Grafana:
  ```bash
  kubectl port-forward svc/grafana 3000:3000 -n dev
  ```

---

## **Cleanup**

To delete all resources in the `dev` namespace:
```bash
kubectl delete all --all -n dev
kubectl delete namespace dev
```

---
