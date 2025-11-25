minikube delete --purge

echo "📌 Запускаем minikube..."
minikube start --driver=docker --addons=ingress 


echo "📌 Устанавливаем cert-manager..."

kubectl apply -f https://github.com/cert-manager/cert-manager/releases/download/v1.13.3/cert-manager.yaml

sleep 30 

echo "📌 Устанавливаем jaeger-operator..."
kubectl create namespace observability
kubectl create -f https://github.com/jaegertracing/jaeger-operator/releases/download/v1.51.0/jaeger-operator.yaml -n observability
sleep 20
kubectl apply -f k8s/jaeger-instance.yaml

echo "📌 Запускаем сборку сервисов..."

minikube image build -t service-a:latest services/service-a/
minikube image build -t service-b:latest services/service-b/

echo "📌 Деплоим..."

kubectl apply -f k8s/services.yaml

echo "📌 Для запуска проверки выполните команду:"
echo 'kubectl exec -it $(kubectl get pods -l app=service-a -o jsonpath='{.items[0].metadata.name}') -- wget -qO- "http://service-a:8080/order?order_id=123"'
echo "📌 Для просмотра трейсов перейдите на http://localhost:16686"
echo "📌 Не закрывайте окно для пользования Jaeger UI"
kubectl port-forward svc/simplest-query 16686:16686
