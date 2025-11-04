<h2>kubefs</h2>
<h4>The cli tool that automates & simplifies the creation, testing, and delpoyment of full stack applications onto kubernetes clusters.</h4>
<h4>made by @rahulmedicharla</h4>

<h2>Installation</h2>

<code>brew tap rahulmedicharla/kubefs; brew install kubefs</code>

<p>Installing the cli tool is very easy. It is integrated within homebrew so make sure you have that setup.</p>
<p>Requirements</p>
<ul>
  <li>homebrew</li>
  <li>docker desktop</li>
</ul>

<h2>Features</h2>

<p>this cli tool streamlines your testing & development by taking care of the nitpicky things in a kubernetes native fashion so you don't have to</p>

<ul>
  <li>Ingress Routing: automatically attaches ingresses to frontend resources to route based on a defined host name </li>
  <li>Dynamic Intra-cluster communication: maps the hosts for intra-cluster communication to the proper host name & port regardless of where you are in your development cycle so that you don't have to.</li>
  <li>Env vars as secrets: stores things you want to be kept secret, as a secret in a kubernetes native fashion</li>
  <li>Automatic multiarch Docker image & container creation and cleanup</li>
  <li>Common Addons: Currently supports a self-contained headless <a href="https://github.com/rahulmedicharla/kubefs/tree/main/addons/auth">auth</a> server and a credentials manager api <a href="https://github.com/rahulmedicharla/kubefs/tree/main/addons/gateway">gateway</a> for use.
  <li>Resilience through Replication</li>
</ul>

<h2>Tutorial</h2>
<ol>
  <li>Start of my running <code>kubefs config docker</code> to set up the context for your docker credentials, and <code>kubefs init</code> to initialize a new project.</li>
  <li>Use <code>kubefs create</code> to create different frontend, api, and database resources with the specified framwork & port number</li>
  <li>Note. when creating a frontend there is a flag for the hostname, you can leave this empty for the ingress to accept all hosts or you can specify some host name, ie. example.kubefs.com. Note. when creating the database there is a flag for the password for the database, the default is just 'password' but you can modify this.</li>
  <li>Use <code>kubefs run</code> to run the development server for an individual resource </li>
  <li>Note. load the url path for intra-cluster communication an environment variable in the format<code>[name]HOST</code>. ie. if you are in a frontend (named tefi) & want to make a request to the api (named axi), in the frontend server-side, load the environment variable process.env.axiHOST. kubefs will take care of mapping this environment variable to the api's proper host name and path throughout the development cycle.</li>
  <li>Once your local development is ready, use <code>kubefs compile</code> to automatically generate a dockerfile & docker image for your resource, and push it up. You can specify custom configuration through the flags.</li>
  <li>Now, you can test the docker containers and how they worker together. use <code>kubefs test</code> to test all or someo of your resources together. This will bring up all the docker images, and if built using the dynamic intra-cluster paths as environment variables, it will automatically be routed to the correct place. Note. if you have any .env files, they will be included in the .dockerignore file. This is so that the environment variables can be consumed through a kubernetes secret later on.</li>
  <li>Once you have tested your container deployment, its time to deploy it to a kubernetes cluster. The first thing we are going to do is start a local kubernetes cluster using minikube. First run <code>minikube start; minikube addons enable ingress</code>. This just starts up your cluster and sets up the any addons we may need. Next, run <code>kubefs deploy</code> to deploy either all or some of your resources onto the cluster. Note, this will create a helm chart and deploy each resource in its own namespace. Note. Again if you were loading the paths as environment variables, it will automatically be mapped to the proper resource. </li>
  <li>You should have your application deployed to minikube, which you can <code>minikube tunnel</code> to access your exposes resources. Note if you specified a host when creating your frontend, make sure to add that to your /etc/hosts path so the DNS reoslution happens.</li>
  <li>When you are ready you can undeploy and close the cluster by running <code>kubefs undeploy</code> with conditional flags. Then, if you want to delete the reosources you can use <code>kubefs remove</code> to remove it locally and/or on dockerhub.</li>
  <li>That's it!</li>
</ol>

<h2>Additional Information</h2>

<p>connect to postgresql by exec into it and <code>PG_PASSWORD=[password] psql -U postgres -d default -h [host] -p [port]</code></p>
<p>connect to redis by exec into it and <code>redis-cli -h [host] -p [port] -a [password]</code></p>
