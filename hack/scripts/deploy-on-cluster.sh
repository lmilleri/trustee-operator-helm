IMAGE=quay.io/rh_ee_lmilleri/trustee-operator:helm
make docker-build docker-push IMG=$IMAGE
make install
make deploy IMG=$IMAGE
kubectl apply -f config/samples/trustee_v1alpha1_trusteeconfig.yaml
