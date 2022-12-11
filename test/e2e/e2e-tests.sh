
URL=$1
echo "connecting to $URL"
curl $URL
kubectl logs deployment/httptest-00001-deployment queue-proxy
kubectl logs deployment/httptest-00001-deployment queue-proxy|grep -i "alert"