#!/bin/bash

# TODO: consider getting rid of Bashisms

set -x

# This is the entrypoint for the image and meant to wrap the
# logic of gathering/reporting results to the Sonobuoy worker.

results_dir="${RESULTS_DIR:-/tmp/results}"

# saveResults prepares the results for handoff to the Sonobuoy worker.
# See: https://github.com/vmware-tanzu/sonobuoy/blob/master/docs/plugins.md
saveResults() {
  cd ${results_dir} || exit 1

  # Sonobuoy worker expects a tar file.
	tar czf results.tar.gz *

	# Signal to the worker that we are done and where to find the results.
	printf ${results_dir}/results.tar.gz > ${results_dir}/done

  sleep 300 # TODO: remove it!
}

# Ensure that we tell the Sonobuoy worker we are done regardless of results.
trap saveResults EXIT

# args_into load the next command-line args into the provided vars using the copy of the $@ array
args=("$@")
arg_pos=0
args_len=$#
args_into() {
  local fargs=($@)
  if [ $# -gt $((args_len-arg_pos)) ]; then
    echo not enough args at ${arg_pos}: need "${fargs[*]}" \("$#"\), got "${args[@]:arg_pos}" \($((args_len-arg_pos))\) >> "${results_dir}"/errors.txt
    exit 1
  fi
  for name in "${fargs[@]}"; do
    eval "${name}=\"${args[arg_pos]}\""
    arg_pos=$((arg_pos+1))
  done;
}
####################################

while [ ${args_len} -gt ${arg_pos} ]; do
  args_into cmd_type
  if [ "${cmd_type}" == "request" ]; then
    args_into req_cmd out
      /bin/sh -c "$req_cmd" > "${results_dir}/${out}"
  else
    args_into cmd namespace label_match out
    pod_names=$(kubectl -n "${namespace}" get pods -l"${label_match}" -o json | jq '.items[] | .metadata.name' -r)
#    echo "$pod_names" > "${results_dir}/pods.txt"
    for pod_name in ${pod_names}; do
      mkdir "${results_dir}/${pod_name}"
      kubectl -n "${namespace}" exec "${pod_name}" ${cmd} > "${results_dir}/${pod_name}/${out}" # 2>&1
    done
  fi;
done;
