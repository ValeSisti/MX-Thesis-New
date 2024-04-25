#!/bin/bash


# cd /home/valentina/Thesis/MX-Thesis-New/final-experiments && chmod +x run_experiment.sh && ./run_experiment.sh




# Function to close all terminal windows
close_all_terminals() {
    # Close all terminal windows
    pkill gnome-terminal
    sleep 1
}


# Function to execute commands in a new WSL terminal window with optional delay
execute_command_in_new_terminal() {
    delay="$1"    
    command="$2"


    # Check if a command is provided
    if [ -z "$command" ]; then
        echo "No command provided"
        exit 1
    fi


    # Check if a delay is provided
    if [ -n "$delay" ]; then
        # Add a delay
        echo "Waiting for $delay seconds..."
        sleep "$delay"
    fi


    # Execute the command in a new WSL terminal window
    #wsl -d "$distribution" -- gnome-terminal -- bash -c "$command; exec bash"
    gnome-terminal -- bash -c "$command; exec bash"
    #xterm -e "bash -c '$command; exec bash'"


}

# Example usage
close_all_terminals
#execute_command_in_new_terminal 0 "cd /home/valentina/Thesis/MX-Thesis-New/packages/mx-chain-es-indexer-go-v1.4.18/ && docker compose up"  # Execute "ls -l" in a new terminal window for Ubuntu-20.04 with a 2-second delay
#execute_command_in_new_terminal 10 "cd /home/valentina/Thesis/MX-Thesis-New/packages/mx-chain-es-indexer-go-v1.4.18/cmd/elasticindexer && ./elasticindexer"  # Execute "echo 'Hello, world!'" in a new terminal window for Ubuntu-20.04 with a 3-second delay
#execute_command_in_new_terminal 5 "cd /home/valentina/Thesis/Localnets && mxpy localnet clean && mxpy localnet prerequisites && mxpy localnet build && mxpy localnet config && cp -f /home/valentina/Thesis/Localnets/external.toml /home/valentina/Thesis/Localnets/localnet/observer00/config/ && cp -f /home/valentina/Thesis/Localnets/external.toml /home/valentina/Thesis/Localnets/localnet/observer01/config/ && cp -f /home/valentina/Thesis/Localnets/external.toml /home/valentina/Thesis/Localnets/localnet/observer02/config/ && cp -f /home/valentina/Thesis/Localnets/external.toml /home/valentina/Thesis/Localnets/localnet/observer03/config/ && mxpy localnet start"  # Execute "echo 'Hello, world!'" in a new terminal window for Ubuntu-20.04 with a 3-second delay
#execute_command_in_new_terminal 180 "/bin/python3 /home/valentina/Thesis/MX-Thesis-New/final-experiments/main.py"  # Execute "echo 'Hello, world!'" in a new terminal window for Ubuntu-20.04 with a 3-second delay


execute_command_in_new_terminal 5 "cd /home/valentina/Thesis/Localnets && mxpy localnet clean && mxpy localnet prerequisites && mxpy localnet build && mxpy localnet config && mxpy localnet start"
execute_command_in_new_terminal 60 "/bin/python3 /home/valentina/Thesis/MX-Thesis-New/final-experiments/main.py"  # Execute "echo 'Hello, world!'" in a new terminal window for Ubuntu-20.04 with a 3-second delay
