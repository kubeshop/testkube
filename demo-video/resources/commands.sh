brew remove testkube

kubectl delete ns testkube; kubectl delete crds executors.executor.testkube.io  scripts.tests.testkube.io tests.tests.testkube.io testsuites.tests.testkube.io webhooks.executor.testkube.io certificaterequests.cert-manager.io certificates.cert-manager.io challenges.acme.cert-manager.io  clusterissuers.cert-manager.io issuers.cert-manager.io managedcertificates.networking.gke.io orders.acme.cert-manager.io

brew install testkube

kubectl testkube version

kubectl testkube install

kubectl testkube version

kubectl get pods -ntestkube

kubectl testkube dashboard

docker build  --platform linux/x86_64 -t kubeshop/chuck-jokes .
docker push kubeshop/chuck-jokes

kubectl apply -f manifests.yaml

kubectl testkube create test --file chuck-jokes.postman_collection.json --name chuck-jokes-postman

# upload with UI k6 
kubectl testkube create test --file chuck-jokes.k6.json --name chuck-jokes-k6

kubectl testkube run test chuck-jokes-postman -f
kubectl testkube run test chuck-jokes-k6 -f 

kubectl testkube create testsuite --name chuck-jokes --file suite.json