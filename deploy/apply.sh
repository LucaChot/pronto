#!/bin/bash

kubectl apply -f deploy/central-sched.yaml
kubectl apply -f deploy/aggregator.yaml
sleep 1
kubectl apply -f deploy/remote-sched.yaml
