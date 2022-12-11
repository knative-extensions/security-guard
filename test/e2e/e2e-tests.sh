
URL=$1
echo "connecting to $URL"
curl $URL
kubectl logs deployment/httptest-00001-deployment queue-proxy
response=`kubectl logs deployment/httptest-00001-deployment queue-proxy|grep -i "alert"|tail -1`
responseEnd="${response#*Alert}"
alert=${responseEnd%%\"*}

echo "Alert Value: $alert"
if [ "$alert" != "!" ]; then
   exit 1
fi

curl "$URL?a=2"
kubectl logs deployment/httptest-00001-deployment queue-proxy
response=`kubectl logs deployment/httptest-00001-deployment queue-proxy|grep "ALERT!"|tail -1`
responseEnd="${response#*ALERT}"
alert=${responseEnd%%\"*}

echo "Alert Value: $alert"
if [ "$alert" != "! HttpRequest -> [QueryString:[KeyVal:[Key a is not known,],],]" ]; then
   exit 1
fi


curl "$URL -H \"a:2\""
kubectl logs deployment/httptest-00001-deployment queue-proxy
response=`kubectl logs deployment/httptest-00001-deployment queue-proxy|grep "ALERT!"|tail -1`
responseEnd="${response#*ALERT}"
alert=${responseEnd%%\"*}

echo "Alert Value: $alert"
if [ "$alert" != "! HttpRequest -> [Header:[KeyVal:[Key a is not known,],],]" ]; then
   exit 1
fi

echo "Done!"
exit 0
