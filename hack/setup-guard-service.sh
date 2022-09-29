kubectl apply -f deploy/gateAccount.yaml
kubectl apply -f deploy/serviceAccount.yaml
kubectl apply -f deploy/guardiansCrd.yaml

./hack/update-guard-service.sh

