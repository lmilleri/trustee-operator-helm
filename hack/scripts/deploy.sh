  # 1. Create a kind cluster (if you don't have one)
  kind create cluster

  # 2. Build the docker image
  make docker-build IMG=trustee-operator:dev

  # 3. Load the image into kind
  kind load docker-image trustee-operator:dev

  # 4. Install CRDs
  make install

  # 5. Deploy the operator
  make deploy IMG=trustee-operator:dev

  # 6. Verify it's running
  kubectl get pods -n trustee-operator-system

  # 7. Create a Permissive TrusteeConfig
  kubectl apply -f config/samples/trustee_v1alpha1_trusteeconfig.yaml

  # 8. Watch the resources
  kubectl get trusteeconfig,trustee

  # To tear down:

  # kubectl delete -f config/samples/trustee_v1alpha1_trusteeconfig.yaml
  # make undeploy
