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

connect to cassandra by exec into it and cqlsh -u cassandra -p cassandra
connect to redis with redis-cli

minikube addons 
ingress
metrics-server

const response = await fetch("/api", {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify({
        "method": "DELETE",
        "url": apiHOST + "/delete/" + index,
        "headers": {
          "Content-Type": "application/json"
        },
        "body": ""
      }),

python api update requirements.txt

api's need to load env to get dynamic paths
