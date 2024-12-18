# Introduction
Raft is a consensus algorithm to manage a replicated log across nodes in a distributed system. It was created as a more understandable alternative to Paxos and for building practical systems that are easier to understand. It ensures consistency and fault tolerance by electing a leader node to handle all client interactions and log replication. Raft operates with three key elements: leader election, log replication, and safety. It simplifies consensus by dividing responsibilities, making it easier to understand and implement compared to algorithms like Paxos.

For our project, we implemented a simplified version of the Raft algorithm that implements the core features of leader election, log replications and safety.

To run the simulations and allow for interaction with nodes, we chose to implement a simple web application. The application consists of a frontend server, a backend server and several docker containers. 

The frontend displays all nodes and the variables within each node such as State, Term, etc . Nodes that are not alive, ie docker container is killed, will be displayed as Red. Alive Nodes are displayed as green except for the leader which will be displayed in Blue. 

The backend is used primarily to restart docker containers, by sending a Docker API Request to restart the given node container.

In each container, we spawn a node instance and also run a HTTP server go routine which is able to access the state of the node, which allows us to communicate with the node from the frontend.


# Folders

## backend
In the backend folder, we have the go implementation of RAFT, as well as the http server. In this folder we also include the docker compose file which is used to create the containers.

## frontend
In the frontend folder, it contains the code for the frontend server, and also a script to run a backend server used for making API calls to Docker for restarting node containers.

# Running Application

You will need to open 3 terminals and run the following set of CLI commands. Ensure the starting directory of each terminal is the root folder of this project. 

## Node Containers
```
cd backend
docker compose down && docker compose build && docker compose up
```


## Backend
```
cd frontend
npm i 
npm run backend
```

## Frontend
```
cd frontend
npm i
npm run dev
```