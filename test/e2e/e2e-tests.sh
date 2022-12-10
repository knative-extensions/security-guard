echo "connecting to $1"
curl $1
kubectl logs deployment/httptest queue-proxy|grep "SECURITY ALERT!"