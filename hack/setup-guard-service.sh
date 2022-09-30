kubectl apply -f config/gateAccount.yaml
kubectl apply -f config/serviceAccount.yaml
kubectl apply -f config/guardiansCrd.yaml

./hack/update-guard-service.sh

