#!/bin/bash

# Check if workflow name is passed as an argument
if [ "$#" -ne 1 ]; then
    echo "Usage: $0 <specific_workflow_name>"
    exit 1
fi

# Define the specific workflow name you're looking for
specific_workflow_name="$1"
is_workflow_queued=true
is_workflow_pending=true

# Loop as long as workflows are queued or pending
while [ "$is_workflow_queued" == "true" ] || [ "$is_workflow_pending" == "true" ]; do

    # Fetch queued workflows
    response_queued=$(curl --location 'https://api.github.com/repos/armosec/shared-workflows/actions/runs?status=queued' \
        --header 'Accept: application/vnd.github+json' \
        --header 'Authorization: Bearer $GITHUB_TOKEN' \
        --header 'X-GitHub-Api-Version: 2022-11-28')

    response_pending=$(curl --location 'https://api.github.com/repos/armosec/shared-workflows/actions/runs?status=pending' \
        --header 'Accept: application/vnd.github+json' \
        --header 'Authorization: Bearer $GITHUB_TOKEN' \
        --header 'X-GitHub-Api-Version: 2022-11-28')

    # Clean the response and extract total workflows
    clean_queued_response=$(echo "$response_queued" | tr -d '\000-\031')
    clean_pending_response=$(echo "$response_pending" | tr -d '\000-\031')
    total_workflows_in_queue=$(echo "$clean_queued_response" | jq '.total_count')
    total_workflows_in_pending=$(echo "$clean_pending_response" | jq '.total_count')

    # Inform about total number of workflows
    echo "Checking queued and pending workflows for '$specific_workflow_name'."
    echo "Total queued workflows: $total_workflows_in_queue"
    echo "Total pending workflows: $total_workflows_in_pending"

    # Check if there are no workflows in the queue
    if [ "$total_workflows_in_queue" == "0" ]; then
        is_workflow_queued=false
        echo "No workflows are currently queued."
    fi

    # Check if there are no workflows in the pending state
    if [ "$total_workflows_in_pending" == "0" ]; then
        is_workflow_pending=false
        echo "No workflows are currently pending."
    fi

    # Check if there are no workflows in the queue or pending
    if [ "$is_workflow_pending" == "false" ] && [ "$is_workflow_queued" == "false" ]; then
        echo "No queued or pending workflows found. Triggering CD process."
        break
    fi

    # Loop through each queued workflow
    for ((i=0; i<total_workflows_in_queue; i++)); do
        queued_workflow_num=$(($i+1))
        workflow_name=$(echo "$clean_queued_response" | jq -r ".workflow_runs[$i].name")
        echo "Examining queued workflow $queued_workflow_num of $total_workflows_in_queue: $workflow_name"

        # Check if the current workflow name matches the specific name
        if [ "$workflow_name" == "$specific_workflow_name" ]; then
            is_workflow_queued=true
            echo "Match found: Workflow '$specific_workflow_name' is currently queued."
            break
        else
            is_workflow_queued=false
            echo "Workflow '$specific_workflow_name' is not among queued workflows."
        fi
    done

    # Loop through each pending workflow
    for ((j=0; j<total_workflows_in_pending; j++)); do
        pending_workflow_num=$(($j+1))
        workflow_name_pending=$(echo "$clean_pending_response" | jq -r ".workflow_runs[$j].name")
        echo "Examining pending workflow $pending_workflow_num of $total_workflows_in_pending: $workflow_name_pending"

        # Check if the current workflow name matches the specific name
        if [ "$workflow_name_pending" == "$specific_workflow_name" ]; then
            is_workflow_pending=true
            echo "Match found: Workflow '$specific_workflow_name' is in pending state."
            break
        else
            is_workflow_pending=false
            echo "Workflow '$specific_workflow_name' is not among pending workflows."
        fi
    done

    # Check if any workflow is queued or pending before sleeping
    if [ "$is_workflow_queued" == "false" ] && [ "$is_workflow_pending" == "false" ]; then
        echo "No instances of the workflow '$specific_workflow_name' found in queued or pending states."
        break
    fi

    # Indicate sleep start
    echo "Sleeping for 30 seconds..."
    sleep 30
done
