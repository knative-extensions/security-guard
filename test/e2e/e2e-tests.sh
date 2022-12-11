
URL=$1
echo "connecting to $URL"
curl $URL
kubectl logs deployment/httptest-00001-deployment queue-proxy
response=`kubectl logs deployment/httptest-00001-deployment queue-proxy|grep -i "alert"`
responseEnd="${response#*Alert}"
alert=${responseEnd%%\"*}

echo "Alert Value: $alert"
if [ "$alert" != "!" ]; then
   exit 1
fi

curl "$URL?a=2"
kubectl logs deployment/httptest-00001-deployment queue-proxy
response=`kubectl logs deployment/httptest-00001-deployment queue-proxy|grep -i "alert"`
responseEnd="${response#*Alert}"
alert=${responseEnd%%\"*}

echo "Alert Value: $alert"
if [ "$alert" != "!" ]; then
   exit 1
fi
