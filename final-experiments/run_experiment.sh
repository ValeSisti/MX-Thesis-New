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



# Function to generate JSON file using expect
generate_json() {
    local name=$1
    local json_file=$2
    local password=$3

    expect << EOF
        spawn mxpy wallet new --format keystore-secret-key --outfile "$json_file"
        expect "Enter a new password (for keystore):"
        send "$password\r"
        expect eof
EOF

    # Check if the command succeeded
    if [ $? -ne 0 ]; then
        echo "Failed to generate JSON for $name"
        return 1
    fi
}

# Function to convert JSON to PEM using expect
convert_to_pem() {
    local json_file=$1
    local pem_file=$2
    local password=$3

    expect << EOF
        spawn mxpy wallet convert --infile "$json_file" --outfile "$pem_file" --in-format keystore-secret-key --out-format pem
        expect "Enter password for keystore file:"
        send "$password\r"
        expect eof
EOF

    # Check if the command succeeded
    if [ $? -ne 0 ]; then
        echo "Failed to convert JSON to PEM for $json_file"
        return 1
    fi
}

# Function to generate JSON and PEM files
generate_wallets() {
    local num_files=$1
    local output_folder=$2

    # Ensure the number of files to create is greater than or equal to 12
    if [ "$num_files" -lt 12 ]; then
        echo "The number of files to create must be at least 12."
        return 1
    fi

    # Calculate the number of commands to execute
    local commands_to_execute=$((num_files - 12))

    # If no commands need to be executed, print a message and return
    if [ "$commands_to_execute" -eq 0 ]; then
        echo "No additional wallets need to be created since the number of files is 12."
        return 0
    fi

    # Ensure the output folder exists
    mkdir -p "$output_folder"

    # Remove the contents of the output folder
    #rm -rf "$output_folder"/*

    # List of names
    local names=(
        "zzzzz" "jack" "jill" "tom" "harry" "larry" "moe" "curly" "shemp" "vincent"
        "victor" "valerie" "vanessa" "oliver" "oscar" "owen" "ophelia" "otis" "olga"
        "quinn" "quincy" "quentin" "quiana" "quade" "queenie" "quinton" "eileen"
        "raymond" "rachel" "randy" "roger" "rita" "rebecca" "regina" "richard"
        "steven" "samantha" "sarah" "simon" "susan" "scott" "sean" "sophia" "sylvia"
        "terry" "tina" "thomas" "timothy" "theresa" "tiffany" "ursula" "ulysses"
        "ursuline" "umberto" "ulf" "ulyssa" "vicky" "veronica" "vinny" "van" "vivian"
        "vance" "wendy" "william" "wayne" "wesley" "wanda" "willa" "xander" "xena"
        "xenon" "ximena" "xavier" "xochitl" "yasmin" "yvette" "yvonne" "yuri" "yara"
        "yasir" "yasmine" "zachary" "zoe" "zane" "zelda" "zora" "zeke" "zuri"
        "aaron" "abigail" "adam" "adrian" "aiden" "alan" "albert" "alejandro" "alex"
        "alexander" "alexis" "alfred" "ali" "alison" "allison" "alma" "amanda" "amber"
        "amelia" "amy" "andrea" "andrew" "angel" "angela" "anna" "anne" "annie"
        "anthony" "antonio" "april" "arthur" "ashley" "austin" "ava" "barbara" "barry"
        "benjamin" "bernard" "beth" "betty" "beverly" "billy" "blake" "brandon"
        "brenda" "brendan" "brian" "bridget" "bruce" "calvin" "cameron" "carmen"
        "caroline" "casey" "cassandra" "catherine" "cathy" "charles" "charlie" "chase"
        "chelsea" "cheryl" "chris" "christina" "christine" "christopher" "cindy"
        "claire" "clara" "clarence" "claudia" "cody" "colin" "connie" "connor"
        "constance" "corey" "corinne" "courtney" "crystal" "dale" "damian" "damien"
        "dana" "daniel" "daphne" "darren" "daryl" "david" "debbie" "deborah" "dennis"
        "diana" "diane" "donald" "donna" "douglas" "dylan" "eddie" "edgar" "edith"
        "edward" "elaine" "elena" "elijah" "elizabeth" "ella" "ellen" "elliot" "emily"
        "emma" "eric" "erica" "ernest" "ethan" "eugene" "evan" "felicia" "felix"
        "fernando" "finn" "florence" "floyd" "gabriel" "gail" "gary" "gene" "george"
        "gerald" "gina" "gloria" "gordon" "greg" "gregory" "haley" "hannah" "harold"
        "harrison" "hazel" "hector" "henry" "herbert" "holly" "howard" "hudson" "hugh"
        "hunter" "ian" "irene" "iris" "isabel" "isabella" "ivy" "jacob" "jasmine"
        "jason"
    )

    # Set a password for the keystore
    local password="password"

    # Ensure the number of commands to execute does not exceed the number of available names
    if [ "$commands_to_execute" -gt "${#names[@]}" ]; then
        echo "There are not enough names to create $num_files files."
        return 1
    fi

    # Iterate over the required number of names
    for ((i=0; i<commands_to_execute; i++)); do
        local name=${names[$i]}

        # Define the filenames
        local json_file="${output_folder}/${name}.json"
        local pem_file="${output_folder}/${name}.pem"

        # Generate the JSON file
        echo "Generating JSON for $name..."
        generate_json "$name" "$json_file" "$password"

        if [ $? -ne 0 ]; then
            continue
        fi

        # Generate the PEM file
        echo "Converting JSON to PEM for $name..."
        convert_to_pem "$json_file" "$pem_file" "$password"

        if [ $? -ne 0 ]; then
            continue
        fi

        echo "$name wallet created and converted to PEM in $output_folder."
    done

    echo "All wallets created and converted to PEM files in $output_folder."
}




# Stop and remove mx-chain-es-indexer-go-v1418 container
docker stop mx-chain-es-indexer-go-v1418
docker rm mx-chain-es-indexer-go-v1418

# Stop and remove es-container
docker stop es-container
docker rm es-container

# Stop and remove kb-container
docker stop kb-container
docker rm kb-container

close_all_terminals
#: <<'END_COMMENT' # TODO: comment this line to uncomment lines below
#execute_command_in_new_terminal 0 "cd /home/valentina/Thesis/MX-Thesis-New/packages/mx-chain-es-indexer-go-v1.4.18/ && docker compose up" 
#execute_command_in_new_terminal 10 "cd /home/valentina/Thesis/MX-Thesis-New/packages/mx-chain-es-indexer-go-v1.4.18/cmd/elasticindexer && ./elasticindexer"
#execute_command_in_new_terminal 5 "cd /home/valentina/Thesis/Localnets && mxpy localnet clean && mxpy localnet prerequisites && mxpy localnet build && mxpy localnet config && cp -f /home/valentina/Thesis/Localnets/external.toml /home/valentina/Thesis/Localnets/localnet/observer00/config/ && cp -f /home/valentina/Thesis/Localnets/external.toml /home/valentina/Thesis/Localnets/localnet/observer01/config/ && cp -f /home/valentina/Thesis/Localnets/external.toml /home/valentina/Thesis/Localnets/localnet/observer02/config/ && cp -f /home/valentina/Thesis/Localnets/external.toml /home/valentina/Thesis/Localnets/localnet/observer03/config/ && mxpy localnet start"
#execute_command_in_new_terminal 180 "/bin/python3 /home/valentina/Thesis/MX-Thesis-New/final-experiments/main.py"
#END_COMMENT # TODO: comment this line to uncomment lines above
 

#! BLOCK CAPACITY TEST
#: <<'END_COMMENT' # TODO: comment this line to uncomment lines below

# Define arguments
function_name="load_distribution_experiment" # load_distribution_experiment    block_capacity_experiment
block_capacity=100
test_num=30
num_txs_per_batch=50
num_total_txs=20000 #20000
num_txs_threshold_for_account_allocation=12000 #6000
with_cross_shard_probability=true #true
hot_sender_probability=0.6
hot_accounts_change_threshold=1000000000000000 #9600
initial_shard_for_hot_accounts=1
with_AMTs=false

num_user_accounts=200
num_to_copy=$((num_user_accounts - 12))


# Generate wallets before running mxpy localnet build
#generate_wallets $num_user_accounts "/home/valentina/Thesis/Localnets/generated_user_accounts"

# Set PYTHONPATH if the parameter is greater than or equal to 12
if [ "$#" -ge 12 ]; then
    export PYTHONPATH="/home/valentina/Thesis/MX-Thesis-New/mx-sdk-py-cli-v9.5.1"
fi





#execute_command_in_new_terminal 5 "cd /home/valentina/Thesis/Localnets && mxpy localnet clean && mxpy localnet prerequisites && mxpy localnet build && mxpy localnet config && mxpy localnet start"
#execute_command_in_new_terminal 60 "/bin/python3 \

execute_command_in_new_terminal 0 "cd /home/valentina/Thesis/MX-Thesis-New/packages/mx-chain-es-indexer-go-v1.4.18/ && docker compose up" 
execute_command_in_new_terminal 10 "cd /home/valentina/Thesis/MX-Thesis-New/packages/mx-chain-es-indexer-go-v1.4.18/cmd/elasticindexer && ./elasticindexer"
execute_command_in_new_terminal 5 "
                                if [ $num_user_accounts -ge 12 ]; then
                                    export PYTHONPATH='/home/valentina/Thesis/MX-Thesis-New/mx-sdk-py-cli-v9.5.1'
                                fi
                                cd /home/valentina/Thesis/Localnets \
                                && mxpy localnet clean \
                                && mxpy localnet prerequisites \
                                && /bin/python3 /home/valentina/Thesis/MX-Thesis-New/final-experiments/copy_files.py /home/valentina/Thesis/Localnets/generated_user_accounts/ /home/valentina/multiversx-sdk/testwallets/v1.0.0/mx-sdk-testwallets-1.0.0/users/ $num_to_copy \
                                && mxpy localnet build \
                                && mxpy localnet config \
                                && cp -f /home/valentina/Thesis/Localnets/external.toml /home/valentina/Thesis/Localnets/localnet/observer00/config/ \
                                && cp -f /home/valentina/Thesis/Localnets/external.toml /home/valentina/Thesis/Localnets/localnet/observer01/config/ \
                                && cp -f /home/valentina/Thesis/Localnets/external.toml /home/valentina/Thesis/Localnets/localnet/observer02/config/ \
                                && cp -f /home/valentina/Thesis/Localnets/external.toml /home/valentina/Thesis/Localnets/localnet/observer03/config/ \
                                && mxpy localnet start"
execute_command_in_new_terminal 120 "/bin/python3 /home/valentina/Thesis/MX-Thesis-New/final-experiments/main.py \
                        $function_name \
                        --block_capacity $block_capacity \
                        --test_num $test_num \
                        --num_txs_per_batch $num_txs_per_batch \
                        --num_total_txs $num_total_txs \
                        --num_txs_threshold_for_account_allocation $num_txs_threshold_for_account_allocation \
                        --with_cross_shard_probability $with_cross_shard_probability \
                        --hot_sender_probability $hot_sender_probability \
                        --hot_accounts_change_threshold $hot_accounts_change_threshold \
                        --num_user_accounts $num_user_accounts \
                        --initial_shard_for_hot_accounts $initial_shard_for_hot_accounts \
                        --with_AMTs $with_AMTs"

#END_COMMENT # TODO: comment this line to uncomment lines above
#! END OF BLOCK CAPACITY TEST