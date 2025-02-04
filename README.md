# Welcome to kubefs
## kubefs is a cli desgined to automate the creation, testing, and deployment of production-ready fullstack applications onto kubernetes cluster.


### Installation

copy the repo down, and run ```make build```

this will add the kubefs cli to your usr/local/bin which should automatically make it available on the path

then run ```kubefs init``` to download any required dependencies that don't exist & set up project

kubefshelper

three endpoints

localhost:6000/health GET
localhost:6000/env/{key} GET
localhost:6000/api POST

connect to cassandra by exec into it and ```cqlsh [host] [port] -u cassandra -p [password]```
connect to redis by exec into it and ```redis-cli -h [host] -p [port] -a [password]```

minikube addons 
ingress
metrics-server

api's need to load env to get dynamic paths