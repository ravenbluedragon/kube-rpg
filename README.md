# Getting Started

- Get Kubernetes working locally eg docker desktop or minikube
- Create namespace `kubectl create ns rpg`
- Build image `docker build -t rpg-races:dev services/races/`
- Create kubernetes resources `kubectl apply -n rpg -f kubernetes`
- 

# NOTES

Secrets DO NOT belong in git. Secrets are even questionable in Kubernetes. They are included here just so things function.

Database should not live in the same pod as the app. This does not allow the app to scale.