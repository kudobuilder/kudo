# Backup Jobs

Kudo has the ability to capture the backup and restoration process for database applications.  

## MySQL

Create an instance of MySQL using the provded Framework

```bash
$ kubectl apply -f samples/config/mysql.yaml
framework.kudo.k8s.io/mysql created
frameworkversion.kudo.k8s.io/mysql-57 created
instance.kudo.k8s.io/mysql created
```

Query the database to show

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

```bash
kubectl kudo start -n mysql -p backup
```

## Delete data from the datbase

```bash
kubectl exec -it $MYSQL_POD -- mysql -ppassword  -e "delete from example;" kudo
kubectl exec -it $MYSQL_POD -- mysql -ppassword  -e "select * from example;" kudo
```

## Perform a restore

```bash
kubectl kudo start -n mysql -p restore
```

 And then query to see the data from before

 ```bash
kubectl exec -it $MYSQL_POD -- mysql -ppassword  -e "select * from example;" kudo
 ```
