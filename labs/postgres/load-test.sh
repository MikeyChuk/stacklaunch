#!/bin/bash

API=http://localhost:8080/users

for i in $(seq 1 5000)
do
    curl -s \
    -X POST $API \
    -H "Content-Type: application/json" \
    -d "{
        \"name\":\"User$i\",
        \"email\":\"user$i@example.com\"
    }" > /dev/null

    echo "Inserted $i"
done

echo "Finished!"
