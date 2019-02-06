#!/bin/bash
IFS=', ' read -r -a array <<< "$CONN"
for zk in "${array[@]}"
do
    echo "Trying to reach ${zk}"
    OK=$(echo ruok | nc ${zk})
    if [ "$OK" == "imok" ]; then
    	echo "${zk} passed validation check."
    else
	echo "${zk} failed validation check."
    fi
done
exit 1
