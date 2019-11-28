#!/usr/bin/env bash

KEP_FILES="../keps/*"

for kep in $KEP_FILES;
do
  echo "Processing $kep"

  # Parse KEP-Header (the part between the '---')
  KEP_HEADER=$(awk '/---/{p++} p==2{print; exit} p>=1' "$kep")

#  echo "$KEP_HEADER"

  KEP_NUMBER=$(echo "$KEP_HEADER" | sed -n -E 's/kep-number: ([0-9]+)/\1/p')
  KEP_TITLE=$(echo "$KEP_HEADER" | sed -n -E 's/title: (.*)/\1/p')
  KEP_STATUS=$(echo "$KEP_HEADER" | sed -n -E 's/status: (.*)/\1/p')

  echo "Kep Number: $KEP_NUMBER"
  echo "Kep Title: $KEP_TITLE"
  echo "Kep Status: $KEP_STATUS"

done
