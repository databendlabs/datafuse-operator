# Datafuse operator
**NOTICE**: this project is not under active maintainence stage, if you are interested on how to deploy databend on kubernetes, please take a look at our helm chart(https://github.com/datafuselabs/helm-charts)

DataFuse operator manages fuse-query and fuse-store clusters atop [Kubernetes](https://kubernetes.io/) using [CRDs](https://kubernetes.io/docs/concepts/extend-kubernetes/api-extension/custom-resources/). If looking to run FuseQuery on bare metal machine or docker, please refer to [How to Run](https://github.com/datafuselabs/datafuse/blob/master/docs/overview/building-and-running.md). 

## Requirements
* Kubernetes v1.15+ (at least with CRD official support)

## Principles
* **Zero Configuration**
  - Support automated cluster provisioning.
* **Whole process monitoring** 
  - Support to use various integration tools to monitor query/storage healthiness and performance
* **High Availability**
  - Aiming on zero downtime, no single point failure
* **Expandability**
  - Support to run on multi-cloud, hybrid environment

## Introduction


## Roadmap

- [ ] 0.1 Support Basic Install (Automated provisioning and configuration management)
- [ ] 0.2 Support Seemless Upgrades (patch and minor version upgrade supported)
- [ ] 0.3 Support Full lifecycle orchestration(High availability)
- [ ] 0.5 Support Deep Insights (Monitoring metrics, workload analysis)
- [ ] 0.6 Support Auto Pilot (self defined Autoscaling based on metrics)

## Status

#### General

- [ ] CRD definition
- [ ] Workload Lifecycle Controllers
- [ ] Continuous Integration tests
- [ ] High Availability operator cluster and Datafuse cluster
- [ ] Monitoring metrics support to use prometheus and grafana for query time, cpu, memory monitoring
- [ ] Autoscaling based on monitoring metrics


## Contributing

You can learn more about contributing to the Datafuse project by reading our [Contribution Guide](https://github.com/datafuselabs/datafuse/blob/master/docs/development/contributing.md) and by viewing our [Code of Conduct](https://github.com/datafuselabs/datafuse/blob/master/docs/policies/code-of-conduct.md).

## License

Datafuse operator is licensed under [Apache 2.0](LICENSE).
