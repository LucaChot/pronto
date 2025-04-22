#!/bin/bash

kubectl delete -f deploy/central-sched.yaml
kubectl delete -f deploy/aggregator.yaml
kubectl delete -f deploy/remote-sched.yaml
