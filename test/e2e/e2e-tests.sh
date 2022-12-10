
URL=$1
echo "connecting to $URL"
curl $URL
kubectl logs deployment/httptest queue-proxy|grep "SECURITY ALERT!"