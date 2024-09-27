# okg-sidecar
sidecar for open kruise

## hot-update
1. kubectl apply -f ./example/hot-update.yaml
2. 为pod创建公网访问的service
3. 执行更新 `curl -k -X POST -d "url=https://guox-tos.tos-cn-beijing.volces.com/2048/v2/index.html" http://101.126.47.208:5000/hot-update`
