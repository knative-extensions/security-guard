
URL=$1
URL="http://httptest.default.127.0.0.1.sslip.io"
echo "connecting to $URL"
curl $URL
kubectl logs deployment/httptest queue-proxy|grep "SECURITY ALERT!"