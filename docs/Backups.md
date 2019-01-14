# Backup Jobs   

Maestro has the ability to capture the backup and restoration process for database applications.  

## MySQL

Create an instance of MySQL using the provded Framework

```bash
$ kubectl apply -f samples/config/mysql.yaml
framework.maestro.k8s.io/mysql created
frameworkversion.maestro.k8s.io/mysql-57 created
instance.maestro.k8s.io/mysql created
```

Query the database to show
```bash
MYSQL_POD=`kubectl get pods -l app=mysql,step=deploy -o jsonpath="{.items[*].metadata.name}"`
kubectl exec -it $MYSQL_POD -- mysql -ppassword  -e "show tables;" maestro
```

Add some data:
```bash
kubectl exec -it $MYSQL_POD -- mysql -ppassword  -e "INSERT INTO example ( id, name ) VALUES ( null, 'New Data' );" maestro
kubectl exec -it $MYSQL_POD -- mysql -ppassword  -e "INSERT INTO example ( id, name ) VALUES ( null, 'New Data' );" maestro
kubectl exec -it $MYSQL_POD -- mysql -ppassword  -e "INSERT INTO example ( id, name ) VALUES ( null, 'New Data' );" maestro
kubectl exec -it $MYSQL_POD -- mysql -ppassword  -e "INSERT INTO example ( id, name ) VALUES ( null, 'New Data' );" maestro
kubectl exec -it $MYSQL_POD -- mysql -ppassword  -e "select * from example;" maestro
```


## Take a backup

```bash
cat <<EOF | kubectl apply -f -
apiVersion: maestro.k8s.io/v1alpha1
kind: PlanExecution
metadata:
  labels:
    framework-version: mysql-57
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

## Delete data from the datbase

```bash
kubectl exec -it $MYSQL_POD -- mysql -ppassword  -e "delete from example;" maestro
kubectl exec -it $MYSQL_POD -- mysql -ppassword  -e "select * from example;" maestro
```

## Perform a restore

```bash
cat <<EOF | kubectl apply -f -
apiVersion: maestro.k8s.io/v1alpha1
kind: PlanExecution
metadata:
  labels:
    framework-version: mysql-57
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
 And then query to see the data from before

 ```bash
kubectl exec -it $MYSQL_POD -- mysql -ppassword  -e "select * from example;" maestro
 ```
