
URL=$1
echo "connecting to $URL"
curl $URL
kubectl logs deployment/httptest-00001-deployment queue-proxy
response=`kubectl logs deployment/httptest-00001-deployment queue-proxy|grep -i "alert"`
responseEnd="${response#*Alert}"
alert=${responseEnd%%\"*}
if [ "$alert" != "!" ]; then
   return 1
fi
