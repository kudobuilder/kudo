---
title: MySQL with Backup
type: docs
---

# Backup Jobs

KUDO has the ability to capture the backup and restoration process for database applications.

## Demo

Watch the explained demo video of the steps beneath [here](https://youtu.be/e_xUVS_bB2g?t=1433).  

## MySQL

Create an instance of MySQL using the provided sample Operator:

```bash
$ kubectl apply -f config/samples/mysql.yaml
operator.kudo.dev/mysql created
operatorversion.kudo.dev/mysql-57 created
instance.kudo.dev/mysql created
```

Query the database to show its schema:

```bash
MYSQL_POD=`kubectl get pods -l app=mysql,step=deploy -o jsonpath="{.items[*].metadata.name}"`
kubectl exec -it $MYSQL_POD -- mysql -ppassword  -e "show tables;" kudo
```

Add some data:

```bash
kubectl exec -it $MYSQL_POD -- mysql -ppassword  -e "INSERT INTO example ( id, name ) VALUES ( null, 'New Data' );" kudo
kubectl exec -it $MYSQL_POD -- mysql -ppassword  -e "INSERT INTO example ( id, name ) VALUES ( null, 'New Data' );" kudo
kubectl exec -it $MYSQL_POD -- mysql -ppassword  -e "INSERT INTO example ( id, name ) VALUES ( null, 'New Data' );" kudo
kubectl exec -it $MYSQL_POD -- mysql -ppassword  -e "INSERT INTO example ( id, name ) VALUES ( null, 'New Data' );" kudo
kubectl exec -it $MYSQL_POD -- mysql -ppassword  -e "select * from example;" kudo
```


## Take a backup
Define and execute a custom plan in order to take a backup:

```bash
cat <<EOF | kubectl apply -f -
apiVersion: kudo.dev/v1alpha1
kind: PlanExecution
metadata:
  labels:
    operator-version: mysql-57
    instance: mysql
  name: mysql-backup
  namespace: default
spec:
  instance:
    kind: Instance
    name: mysql
    namespace: default
  planName: backup
EOF
```


## Delete data from the database

```bash
kubectl exec -it $MYSQL_POD -- mysql -ppassword  -e "delete from example;" kudo
kubectl exec -it $MYSQL_POD -- mysql -ppassword  -e "select * from example;" kudo
```

## Perform a restore
Similar to the backup step, define and execute the restore:

```bash
cat <<EOF | kubectl apply -f -
apiVersion: kudo.dev/v1alpha1
kind: PlanExecution
metadata:
  labels:
    operator-version: mysql-57
    instance: mysql
  name: mysql-restore
  namespace: default
spec:
  instance:
    kind: Instance
    name: mysql
    namespace: default
  planName: restore
EOF
```

And then query to see the data from before:

 ```bash
kubectl exec -it $MYSQL_POD -- mysql -ppassword  -e "select * from example;" kudo
 ```
