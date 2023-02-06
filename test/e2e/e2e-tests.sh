
URL=$1
HTTPTEST=$2

echo "connecting to $URL"
curl $URL
kubectl logs deployment/${HTTPTEST}-00001-deployment queue-proxy
kubectl logs deployment/guard-service -n knative-serving
response=`kubectl logs deployment/${HTTPTEST}-00001-deployment queue-proxy|grep INFO | grep -i "alert"|tail -1`

if [ -z "${response}" ]; then
  echo ">> No alert - as expected"
else
   echo ">> Expected no alert but received: $response"
   exit 1
fi

curl "$URL?a=2"
kubectl logs deployment/${HTTPTEST}-00001-deployment queue-proxy
response=`kubectl logs deployment/${HTTPTEST}-00001-deployment queue-proxy|grep INFO |grep "ALERT!"|tail -1`
responseEnd="${response#*ALERT}"
alert=${responseEnd%%\"*}


if [[ "$alert" == "! Session ->[HttpRequest:[QueryString:[KeyVal:[Key a is not known,],],],]"* ]];then
   echo ">> Alert as expected for $URL?a=2"
else
   echo ">> Alert value is not as expected: $alert"
   exit 1
fi


curl $URL -H "a:2"
kubectl logs deployment/${HTTPTEST}-00001-deployment queue-proxy
response=`kubectl logs deployment/${HTTPTEST}-00001-deployment queue-proxy|grep INFO |grep "ALERT!"|tail -1`
responseEnd="${response#*ALERT}"
alert=${responseEnd%%\"*}


if [[ "$alert" == "Session ->[HttpRequest:[Headers:[KeyVal:[Key A is not known,],],],]"* ]];then
   echo ">> Alert as expected for $URL -H \"a:2\""
else
   echo ">> Alert value is not as expected: $alert"
   exit 1
fi

echo ">> Done! Test OK!"
exit 0
