curl "http://httptest.default.example.com"
kubectl logs deployment/httptest queue-proxy|grep "SECURITY ALERT!"