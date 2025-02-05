# Welcome to kubefs
## kubefs is a cli desgined to automate the creation, testing, and deployment of production-ready fullstack applications onto kubernetes cluster.


### Installation

<code>brew tap rahulmedicharla/kubefs/kubefs</code>

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

api's need to load env to get dynamic paths

.env stored as secrets, not in docker image

hosts are added to frontend resource ingress
need to add 127.0.0.1 domain to /etc/hosts
minikube tunnel
