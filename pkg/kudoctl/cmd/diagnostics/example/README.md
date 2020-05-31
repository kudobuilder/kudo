## Diagnostics: POC for Copy and Command Example

- install CRDs
- replace standard Zookeeper operator.yaml with [this one](./operator.yaml) (it has an example of diagnostics section)
- install Zookeeper operator
- run diagnostics
```
diag
├── kudo
│   ├── pod_kudo-controller-manager-0
│   │   ├── kudo-controller-manager-0.yaml
│   │   └── manager.log.gz
│   ├── serviceaccountlist.yaml
│   ├── servicelist.yaml
│   └── statefulsetlist.yaml
├── operator_zookeeper
│   ├── instance_zookeeper-instance
│   │   ├── pod_zookeeper-instance-zookeeper-0
│   │   │   ├── kubernetes-zookeeper
│   │   │   │   ├── env-vars.out
│   │   │   │   ├── java.env
│   │   │   │   ├── nslookup.out.err
│   │   │   │   └── zoo.cfg
│   │   │   ├── kubernetes-zookeeper.log.gz
│   │   │   └── zookeeper-instance-zookeeper-0.yaml
│   │   ├── pod_zookeeper-instance-zookeeper-1
│   │   │   ├── kubernetes-zookeeper
│   │   │   │   ├── env-vars.out
│   │   │   │   ├── java.env
│   │   │   │   ├── nslookup.out.err
│   │   │   │   └── zoo.cfg
│   │   │   ├── kubernetes-zookeeper.log.gz
│   │   │   └── zookeeper-instance-zookeeper-1.yaml
│   │   ├── pod_zookeeper-instance-zookeeper-2
│   │   │   ├── kubernetes-zookeeper
│   │   │   │   ├── env-vars.out
│   │   │   │   ├── java.env
│   │   │   │   ├── nslookup.out.err
│   │   │   │   └── zoo.cfg
│   │   │   ├── kubernetes-zookeeper.log.gz
│   │   │   └── zookeeper-instance-zookeeper-2.yaml
│   │   ├── servicelist.yaml
│   │   ├── statefulsetlist.yaml
│   │   └── zookeeper-instance.yaml
│   ├── operatorversion_zookeeper-0.3.0
│   │   └── zookeeper-0.3.0.yaml
│   └── zookeeper.yaml
├── settings.yaml
└── version.yaml
```