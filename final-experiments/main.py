import subprocess
import random
import time
import os
import json
import csv
from datetime import datetime
from collections import defaultdict
import pandas as pd
from elasticsearch import Elasticsearch
import matplotlib.pyplot as plt
import numpy as np
import requests
import tempfile
import math
from concurrent.futures import ThreadPoolExecutor
import threading
import queue
import argparse
import shutil
import re

# Get the directory of the script
script_directory = os.path.dirname(os.path.realpath(__file__))

# Define relative paths
TRANSACTIONS_DIRECTORY = os.path.join(script_directory, "generated_transactions/")
TRANSACTIONS_DIRECTORY_WITH_CORRECT_LOAD = os.path.join(script_directory, "generated_transactions_with_correct_load/")
TRANSACTIONS_DIRECTORY_WITH_CORRECT_LOAD_BY_BATCH = os.path.join(script_directory, "generated_transactions_with_correct_load_by_batch/")
TRANSACTIONS_DIRECTORY_WITH_CORRECT_LOAD_BY_BATCH_AND_ACCOUNTS_ALLOCATION = os.path.join(script_directory, "generated_transactions_with_correct_load_by_batch_and_accounts_allocation/")
#OUTPUT_CSV = os.path.join(script_directory, "output.csv")
OUTPUT_CSV = "output.csv"
OUTPUT_CSV_WITH_REAL_EXECUTION_TIMESTAMP = os.path.join(script_directory, "output.csv")
OUTPUT_CSV_WITH_END_TIMESTAMP = os.path.join(script_directory, "output_with_end_timestamp.csv")
#OUTPUT_CSV_WITH_STATISTICS = os.path.join(script_directory, "output_with_statistics.csv")
OUTPUT_CSV_WITH_STATISTICS = "output_with_statistics.csv"
OUTPUT_CSV_WITH_TIMESTAMP_DIFFERENCE = os.path.join(script_directory, "output_with_timestamp_difference.csv")
#GENERATED_TXS_STATISTICS_CSV = os.path.join(script_directory, "generated_txs_statistics.csv")
GENERATED_TXS_STATISTICS_CSV = "generated_txs_statistics.csv"
GENERATED_TXS_CSV = os.path.join(script_directory, "generated_txs.csv")
ACCOUNTS_INFO_JSON_PATH = os.path.join(script_directory, "accounts_info.json")
NORMALIZED_OUTPUT_CSV_WITH_STATISTICS = "normalized_output_with_statistics.csv"

INPUT_FOLDER_PATH = "/home/valentina/multiversx-sdk/testwallets/v1.0.0/mx-sdk-testwallets-1.0.0/users"



hot_accounts = ["erd1qyu5wthldzr8wx5c9ucg8kjagg0jfs53s8nr3zpz3hypefsdd8ssycr6th", "erd1kyaqzaprcdnv4luvanah0gfxzzsnpaygsy6pytrexll2urtd05ts9vegu7", "erd18tudnj2z8vjh0339yu3vrkgzz2jpz8mjq0uhgnmklnap6z33qqeszq2yn4" ] #alice, dan, eve
hot_sender_probability = 0.9 #TODO modificare a 0.5
cross_shard_probability = 0.5

global_txs_id = 0

current_accounts_allocation_id = 0

accounts_info = {
    "erd1qyu5wthldzr8wx5c9ucg8kjagg0jfs53s8nr3zpz3hypefsdd8ssycr6th" : {
        "username" : "alice",
        "shard" : 1,
        "nonce" : 5,
        "migrationNonce" : 0,
    },
    "erd1spyavw0956vq68xj8y4tenjpq2wd5a9p2c6j8gsz7ztyrnpxrruqzu66jx" : {
        "username" : "bob",
        "shard" : 0,
        "nonce" : 1,
        "migrationNonce" : 0,
    },
    "erd1k2s324ww2g0yj38qn2ch2jwctdy8mnfxep94q9arncc6xecg3xaq6mjse8" : {
        "username" : "carol",
        "shard" : 2,
        "nonce" : 1,
        "migrationNonce" : 0,
    },
    "erd1kyaqzaprcdnv4luvanah0gfxzzsnpaygsy6pytrexll2urtd05ts9vegu7" : {
        "username" : "dan",
        "shard" : 1,
        "nonce" : 1,
        "migrationNonce" : 0,
    },
    "erd18tudnj2z8vjh0339yu3vrkgzz2jpz8mjq0uhgnmklnap6z33qqeszq2yn4" : {
        "username" : "eve",
        "shard" : 1,
        "nonce" : 1,
        "migrationNonce" : 0,
    },
    "erd1kdl46yctawygtwg2k462307dmz2v55c605737dp3zkxh04sct7asqylhyv" : {
        "username" : "frank",
        "shard" : 1,
        "nonce" : 1,
        "migrationNonce" : 0,
    },
    "erd1r69gk66fmedhhcg24g2c5kn2f2a5k4kvpr6jfw67dn2lyydd8cfswy6ede" : {
        "username" : "grace",
        "shard" : 1,
        "nonce" : 1,
        "migrationNonce" : 0,
    },
    "erd1dc3yzxxeq69wvf583gw0h67td226gu2ahpk3k50qdgzzym8npltq7ndgha" : {
        "username" : "heidi",
        "shard" : 2,
        "nonce" : 1,
        "migrationNonce" : 0,
    },
    "erd13x29rvmp4qlgn4emgztd8jgvyzdj0p6vn37tqxas3v9mfhq4dy7shalqrx" : {
        "username" : "ivan",
        "shard" : 1,
        "nonce" : 1,
        "migrationNonce" : 0,
    },
    "erd1fggp5ru0jhcjrp5rjqyqrnvhr3sz3v2e0fm3ktknvlg7mcyan54qzccnan" : {
        "username" : "judy",
        "shard" : 2,
        "nonce" : 1,
        "migrationNonce" : 0,
    },
    "erd1z32fx8l6wk9tx4j555sxk28fm0clhr0cl88dpyam9zr7kw0hu7hsx2j524" : {
        "username" : "mallory",
        "shard" : 1,
        "nonce" : 1,
        "migrationNonce" : 0,
    },
    "erd1uv40ahysflse896x4ktnh6ecx43u7cmy9wnxnvcyp7deg299a4sq6vaywa" : {
        "username" : "mike",
        "shard" : 0,
        "nonce" : 1,
        "migrationNonce" : 0,
    },
# ! ------------------------------- NEW USERS -------------------------------
    # "erd1xrvst0w2sa60f6g59z6rawxzgmpktj6yh9jgmnseceq458ys7kts2xxac4" : {
    #    "username" : "my_wallet",
    #    "shard" : 1, # TODO: CONTROLLA
    #    "nonce" : 1,
    #    "migrationNonce" : 0,
    #},
}





accountsAllocationData = {
    1: [
        {
            "accountAddressString": "erd1qyu5wthldzr8wx5c9ucg8kjagg0jfs53s8nr3zpz3hypefsdd8ssycr6th",
            "migrationNonce": 0,
            "sourceShard": 1,
            "destinationShard": 0
        },
        {
            "accountAddressString": "erd1kyaqzaprcdnv4luvanah0gfxzzsnpaygsy6pytrexll2urtd05ts9vegu7",
            "migrationNonce": 0,
            "sourceShard": 1,
            "destinationShard": 2
        }
    ],
    2: [
        {
            "accountAddressString": "erd1qyu5wthldzr8wx5c9ucg8kjagg0jfs53s8nr3zpz3hypefsdd8ssycr6th",
            "migrationNonce": 1,
            "sourceShard": 0,
            "destinationShard": 1
        },
        {
            "accountAddressString": "erd1kyaqzaprcdnv4luvanah0gfxzzsnpaygsy6pytrexll2urtd05ts9vegu7",
            "migrationNonce": 1,
            "sourceShard": 2,
            "destinationShard": 1
        }
    ],
    3: [
        {
            "accountAddressString": "erd1qyu5wthldzr8wx5c9ucg8kjagg0jfs53s8nr3zpz3hypefsdd8ssycr6th",
            "migrationNonce": 2,
            "sourceShard": 1,
            "destinationShard": 0
        },
        {
            "accountAddressString": "erd1kyaqzaprcdnv4luvanah0gfxzzsnpaygsy6pytrexll2urtd05ts9vegu7",
            "migrationNonce": 2,
            "sourceShard": 1,
            "destinationShard": 2
        }
    ]    
}



def run_shell_command(command, sender_addr):
    timestamp = datetime.now()
    #print(timestamp)


    try:
        result = subprocess.run(command, shell=True, check=True, capture_output=True, text=True)
        #print("Command output: ", result.stdout)
        
    
    except subprocess.CalledProcessError as e:
        print("Error executing command: ", e)
        # If the command fails, print the error output
        print("Command error: ", e.stderr)
        return
    
    if sender_addr != None:
        accounts_info[sender_addr]["nonce"] += 1
        #print("New nonce of sender " + sender_addr + ": " + str(accounts_info[sender_addr]["nonce"]))
    
    timestamp = datetime.now()
    #print(timestamp)
    return





def create_new_tx_command(nonce, gas_limit, receiver_address, sender_name, tx_id, output_directory):
    formatted_tx_id = "{:07d}".format(tx_id)
    outfile = f"{output_directory}transaction_{formatted_tx_id}.json"
    
    command = f"mxpy tx new \
            --nonce={nonce} \
            --data=\"Hello, World\" \
            --gas-limit={gas_limit} \
            --receiver={receiver_address} \
            --pem=~/multiversx-sdk/testwallets/latest/users/{sender_name}.pem \
            --chain=localnet \
            --proxy=http://localhost:7950 \
            --outfile={outfile};"
    return command






def pick_sender_and_receiver_with_correct_load_by_batch_and_account_allocations(all_accounts, seed, num_txs_per_batch, with_cross_shard_probability, hot_sender_probability):
    # Extract keys from the hash map
    all_accounts_keys = list(all_accounts.keys())
    light_accounts = list(all_accounts.keys())
    for account in hot_accounts:
        light_accounts.remove(account)

    # Set the random seed
    random.seed(seed)

    # Prepare a list to store sender-receiver pairs
    batch_sender_receiver_pairs = []

    # Calculate the number of hot account senders
    num_hot_account_senders = int(num_txs_per_batch * hot_sender_probability)
    print(f"--- Generating {num_hot_account_senders} by hot senders and {num_txs_per_batch - num_hot_account_senders} by light senders in current batch ---")

    # Generate sender-receiver pairs for hot account senders
    for _ in range(num_hot_account_senders):
        sender = random.choice(hot_accounts)
        receiver = pick_receiver_based_on_cross_shard_probability(all_accounts, sender, with_cross_shard_probability)
        sender_shard = accounts_info[sender]["shard"]
        receiver_shard = accounts_info[receiver]["shard"]
        is_cross_shard = sender_shard == receiver_shard
        batch_sender_receiver_pairs.append((sender, receiver, sender_shard, receiver_shard, is_cross_shard))

    # Generate sender-receiver pairs for light account senders
    for _ in range(num_txs_per_batch - num_hot_account_senders):
        sender = random.choice(light_accounts)
        receiver = pick_receiver_based_on_cross_shard_probability(all_accounts, sender, with_cross_shard_probability)
        sender_shard = accounts_info[sender]["shard"]
        receiver_shard = accounts_info[receiver]["shard"]
        is_cross_shard = sender_shard == receiver_shard
        batch_sender_receiver_pairs.append((sender, receiver, sender_shard, receiver_shard, is_cross_shard))

    # Shuffle the pairs to mix hot and light account senders
    #random.shuffle(batch_sender_receiver_pairs, random.Random(seed))
    random.shuffle(batch_sender_receiver_pairs)

    #writeGeneratedTxsStatisticsCSV(batch_sender_receiver_pairs)

    # Return the list of sender-receiver pairs for this batch
    return batch_sender_receiver_pairs



def writeGeneratedTxsStatisticsCSV(batch_sender_receiver_pairs):
    # Write to CSV file
    with open(GENERATED_TXS_STATISTICS_CSV, mode='a', newline='') as file:
        writer = csv.writer(file)
        for sender, receiver, sender_shard, receiver_shard, cross_shard in batch_sender_receiver_pairs:
            writer.writerow([sender, receiver, sender_shard, receiver_shard, cross_shard])




def pick_receiver_based_on_cross_shard_probability(all_accounts, sender_addr, with_cross_shard_probability):
    
    if not with_cross_shard_probability:
        all_account_keys = [key for key, _ in all_accounts.items()]
        return random.choice(all_account_keys)
    
    sender_shard = all_accounts[sender_addr]["shard"]
    same_shard_account_keys = [key for key, acc in all_accounts.items() if key != sender_addr and acc["shard"] == sender_shard]
    different_shard_account_keys = [key for key, acc in all_accounts.items() if key != sender_addr and acc["shard"] != sender_shard]
    
    if random.random() < cross_shard_probability:
        return random.choice(different_shard_account_keys)
    else:
        return random.choice(same_shard_account_keys)



def generateTransactionsWithCorrectLoadInBatchesWithAccountAllocations(num_txs_per_batch, num_total_txs, num_txs_threshold_for_account_allocation, output_dir, with_cross_shard_probability, hot_sender_probability):
    print("Num of accounts: " + str(len(accounts_info)))
    print("0.02x of account is: " + str(len(accounts_info) * 0.02))
    seed_value = 42

    global global_txs_id

    num_txs_from_last_account_allocation = 0

    num_batches = math.ceil(num_total_txs / num_txs_per_batch)

    createGeneratedTxsStatisticsCSV()

    for batch in range(num_batches):
        generator = pick_sender_and_receiver_with_correct_load_by_batch_and_account_allocations(accounts_info, seed_value+batch, num_txs_per_batch, with_cross_shard_probability)
        batch_sender_receiver_pairs = list(generator)

        # Prepare a list to store all the commands to run
        #commands_to_run = []

        # Generate commands for each sender-receiver pair in the batch
        for sender_addr, receiver_addr, _, _, _ in batch_sender_receiver_pairs:
            #print("----- GENERATING TX FROM " + accounts_info[sender_addr]["username"] + " TO " + accounts_info[receiver_addr]["username"] + " -----")
            command_to_run = create_new_tx_command(
                    nonce=accounts_info[sender_addr]["nonce"],
                    gas_limit=70000,
                    receiver_address=receiver_addr,
                    sender_name=accounts_info[sender_addr]["username"],
                    tx_id = global_txs_id,
                    output_directory = output_dir
            )
            #commands_to_run.append((command_to_run, sender_addr))
            run_shell_command(command_to_run, sender_addr)


            global_txs_id += 1
            #print(f"GLOBAL_TXS_ID: {global_txs_id}")
            num_txs_from_last_account_allocation += 1
        
        if num_txs_from_last_account_allocation >= num_txs_threshold_for_account_allocation:
            print(f"num_txs_from_last_account_allocation ({num_txs_from_last_account_allocation}) >= num_txs_threshold_for_account_allocation ({num_txs_threshold_for_account_allocation}): computing account allocation")
            compute_next_account_allocation(num_txs_threshold_for_account_allocation, output_dir, hot_sender_probability)
            num_txs_from_last_account_allocation = 0




def update_hot_accounts(shard_id, num_accounts):
    global hot_accounts

    num_picked = 0
    old_hot_accounts = hot_accounts.copy()
    hot_accounts = []

    hot_accounts.append(old_hot_accounts[0])
    shard_id = accounts_info[old_hot_accounts[0]]['shard']
    num_picked += 1

    print(f"OLD HOT ACCOUNTS SET: {old_hot_accounts}")

    for account_addr, account_info in accounts_info.items():
        if account_info["shard"] == shard_id and account_addr not in old_hot_accounts:
        #if account_addr not in old_hot_accounts:
            hot_accounts.append(account_addr)
            num_picked += 1
            if num_picked == num_accounts:
                break
    
    print(f"NEW HOT ACCOUNTS SET: {hot_accounts}")
            


    
    



def generateTransactionsWithCorrectLoadInBatchesWithAccountAllocationsOnCSV(num_txs_per_batch, num_total_txs, num_txs_threshold_for_account_allocation, output_dir, with_cross_shard_probability, hot_sender_probability, hot_accounts_change_threshold):
    print("Num of accounts: " + str(len(accounts_info)))
    print("0.02x of account is: " + str(len(accounts_info) * 0.02))
    seed_value = 42

    global global_txs_id

    num_txs_from_last_account_allocation = 0
    num_txs_from_last_hot_accounts_change = 0

    num_batches = math.ceil(num_total_txs / num_txs_per_batch)

    createGeneratedTxsStatisticsCSV(output_dir)

    for batch in range(num_batches):
        generator = pick_sender_and_receiver_with_correct_load_by_batch_and_account_allocations(accounts_info, seed_value+batch, num_txs_per_batch, with_cross_shard_probability, hot_sender_probability)
        batch_sender_receiver_pairs = list(generator)

        # Prepare a list to store all the commands to run
        #commands_to_run = []

        with open(f"{output_dir}/{GENERATED_TXS_STATISTICS_CSV}", mode='a', newline='') as file:
            writer = csv.writer(file)
            # Generate commands for each sender-receiver pair in the batch
            for sender, receiver, sender_shard, receiver_shard, cross_shard in batch_sender_receiver_pairs:
                writer.writerow([global_txs_id, sender, receiver, accounts_info[sender]["nonce"], sender_shard, receiver_shard, cross_shard])

                accounts_info[sender]["nonce"] += 1

                global_txs_id += 1
                num_txs_from_last_account_allocation += 1
                num_txs_from_last_hot_accounts_change += 1

        if num_txs_from_last_hot_accounts_change >= hot_accounts_change_threshold:
            update_hot_accounts(shard_id=2, num_accounts=3)
            num_txs_from_last_hot_accounts_change = 0


        
        if num_txs_from_last_account_allocation >= num_txs_threshold_for_account_allocation:
            print(f"num_txs_from_last_account_allocation ({num_txs_from_last_account_allocation}) >= num_txs_threshold_for_account_allocation ({num_txs_threshold_for_account_allocation}): computing account allocation")
            compute_next_account_allocation(num_txs_threshold_for_account_allocation, output_dir, hot_sender_probability)
            num_txs_from_last_account_allocation = 0





def compute_next_account_allocation(num_txs_threshold, txs_dir, hot_sender_probability):
    global current_accounts_allocation_id
    global global_txs_id
    starting_index = max(0, global_txs_id - num_txs_threshold)
    print(f"------ STARTING INDEX: {starting_index} -------")
    #ending_index = iteration * (num_txs_threshold)

    n_t, m_t, q_t_1, total_txs_by_shard, x_t, hot_accounts_load = getGeneratedTxsStatisticsFromCSV(starting_index, num_txs_threshold, hot_sender_probability, txs_dir)

    #shard_loads_variance = computeShardLoadsVariance(shard_loads)
    account_migrations_list, initial_variance, final_variance = move_accounts_to_improve_variance(hot_accounts_load, n_t, m_t, q_t_1, total_txs_by_shard, x_t)
    # Call the algorithm function
    #allocation_result = account_allocation_algorithm({0,1,2}, hot_accounts, n_t, q_t_1, l_t, x_t, m_t)
    #print(f"allocation_result = {allocation_result}")
    
    if len(account_migrations_list) > 0:
        generate_account_allocation_json_file(account_migrations_list, initial_variance, final_variance, global_txs_id, txs_dir, current_accounts_allocation_id)
        current_accounts_allocation_id += 1


def generate_account_allocation_json_file(account_migrations_list, initial_variance, final_variance, tx_id, txs_dir, id):
    formatted_tx_id = "{:07d}".format(tx_id)
    # File path to save the JSON file
    outfile = f"{txs_dir}/transaction_{formatted_tx_id}_account_allocation.json"

    accountAllocationData = {
        "accountAllocationPayload" : {
            "id" : id,
            "accountAllocation" : account_migrations_list
        },
        "initialVariance" : initial_variance,
        "newVariance" : final_variance,
        "varianceImprovement" : final_variance - initial_variance   
    }

    # Write the list of dictionaries to the JSON file
    with open(outfile, "w") as json_file:
        json.dump(accountAllocationData, json_file, indent=4)
    
    for account_migration in account_migrations_list:
        accounts_info[account_migration["accountAddressString"]]["shard"] = account_migration["destinationShard"]


def compute_V_t(S, A_hot, n_t, q_t_1, m_t, x_t):
    """
    Compute the variance V_t for given parameters.

    Args:
    S (set): Set of shards.
    A_hot (set): Set of hot accounts.
    n_t (dict): Dictionary with predicted upcoming transactions {j: n_j^t}.
    q_t_1 (dict): Dictionary with queuing transactions from last epoch {j: q_j^{t-1}}.
    m_t (dict): Dictionary with predicted transactions for each shard {i: m_i^t}.
    x_t (dict): Dictionary with shard mapping for each account {j: {i: x_{i,j}^t}}.

    Returns:
    float: Computed variance V_t.
    """
    
    # Step 1: Compute l_j^t for each hot account j
    l_t = {j: n_t[j] + q_t_1[j] for j in A_hot}

    # Step 2: Compute the average load per shard l_bar_t
    total_m_t = sum(m_t[i] for i in S)
    total_l_t = sum(l_t[j] for j in A_hot)
    l_bar_t = (total_m_t + total_l_t) / len(S)
    
    # Step 3: Compute the variance V_t
    numerator = 0
    for i in S:
        sum_i = sum(l_t[j] * x_t[j][i] for j in A_hot) + m_t[i]
        numerator += (sum_i - l_bar_t) ** 2
        
    V_t = numerator / len(S)
    
    return V_t


def move_accounts_to_improve_variance(hot_accounts_loads, n_t, m_t, q_t_1, total_txs_by_shard, x_t):
    # Sort hot_accounts_loads by load in descending order
    #sorted_accounts = sorted(hot_accounts_loads.items(), key=lambda x: x[1]['load'], reverse=True)
    #print(f"Sorted hot accounts: {sorted_accounts}" )
    
    # Find the shard with the minimum load
    #min_load_shard = min(shard_loads, key=shard_loads.get)


    # Step 2: Sort shards by load and find most heavy and light-loaded shards
    #S_heavy = sorted([0,1,2], key=lambda i: sum(hot_accounts_loads[j] * x_t[j].get(i, 0) + m_t[i] for j in hot_accounts), reverse=True)
    # Compute the loads for each shard and store in a dictionary
    shard_loads = {
        i: sum(hot_accounts_loads[j] * x_t[j].get(i, 0) for j in hot_accounts) + m_t[i]
        for i in [0, 1, 2]
    }

    # Sort the dictionary by loads in descending order and return as a list of tuples
    S_heavy = dict(sorted(shard_loads.items(), key=lambda item: item[1], reverse=True))

    print(S_heavy)

    # Get the shard with the maximum load (first item in the sorted dictionary)
    max_load_shard = list(S_heavy.keys())[0]

    # Get the shard with the minimum load (last item in the sorted dictionary)
    min_load_shard = list(S_heavy.keys())[-1]

    print(f"Shard with maximum load: {max_load_shard}")
    print(f"Shard with minimum load: {min_load_shard}")
    print(f"S_heavy = shard_loads = {S_heavy}")



    print(f"Shard with minimum load: {min_load_shard}" )
    # Step 3: Sort accounts in heavy-loaded shard by load
    A_heavy = sorted(hot_accounts, key=lambda j: hot_accounts_loads[j], reverse=True)
    
    # Initialize a list to store the tuples (account, source_shard, dest_shard)
    moves = []
    
    # Compute variance before moving any accounts
    #initial_variance = computeShardLoadsVariance(shard_loads)
    initial_variance = compute_V_t({0,1,2}, hot_accounts, n_t, q_t_1, m_t, x_t)
    current_variance = initial_variance
    
    for account in A_heavy:
        source_shard = accounts_info[account]['shard']
        
        # Calculate shard loads after moving the account
        shard_loads_after_move = shard_loads.copy()
        shard_loads_after_move[source_shard] -= hot_accounts_loads[account]
        shard_loads_after_move[min_load_shard] += hot_accounts_loads[account]
        print(f"Shard loads after move: {shard_loads_after_move}")
        
        x_t_new = x_t.copy()
        x_t_new[account][source_shard] = 0
        x_t_new[account][min_load_shard] = 1 
        
        # Compute variance after moving the account
        new_variance = compute_V_t({0,1,2}, hot_accounts, n_t, q_t_1, m_t, x_t_new)
        
        # Check if variance improves
        if new_variance < current_variance:
            print(f"--------IMPROVEMENT FOUND---------. Moving to the next hot account...")
            moves.append({
                            "accountAddressString": account,
                            "migrationNonce": accounts_info[account]["migrationNonce"],
                            "sourceShard": source_shard,
                            "destinationShard": min_load_shard
                        })
            accounts_info[account]["migrationNonce"] += 1
            shard_loads = shard_loads_after_move
            x_t = x_t_new
            current_variance = new_variance
            # Update the variable holding the shard with minimum load, as after the migration it could have been changed
            min_load_shard = min(shard_loads_after_move, key=shard_loads_after_move.get)            
            print("Computed Account Allocation: {moves}")
        else:
            print(f"NO IMPROVEMENT. Moving to the next hot account...")
            
            # Stop if variance doesn't improve
            #break
    
    return moves, initial_variance, current_variance



def computeShardLoadsVariance(shard_loads):
    # Calculate the mean of the values
    mean = sum(shard_loads.values()) / len(shard_loads)
    
    # Calculate the squared differences from the mean for each value
    squared_diffs = [(value - mean) ** 2 for value in shard_loads.values()]
    
    # Calculate the variance as the mean of the squared differences
    variance = sum(squared_diffs) / len(shard_loads)

    print("Shard Loads Variance:", variance)
    
    return variance


def calculate_transactions_for_shard(num_transactions, hot_account_percentage, hot_account_config, shard_index):
    total_hot_accounts = sum(hot_account_config.values())
    hot_tx_percentage_per_shard = 0
    hot_tx_per_shard = 0
    non_hot_tx_per_shard = 0
    total_tx_per_shard = 0
    hot_tx_per_account = 0
    hot_tx_percentage_per_account = 0
    
    if total_hot_accounts > 0:
        hot_tx_percentage_per_shard = (hot_account_config[shard_index] / total_hot_accounts) * hot_account_percentage

    hot_tx_per_shard = (hot_tx_percentage_per_shard / 100) * num_transactions
    non_hot_tx_per_shard = (1 / len(hot_account_config)) * (num_transactions * ((100 - hot_account_percentage) / 100))
    total_tx_per_shard = hot_tx_per_shard + non_hot_tx_per_shard

    hot_accounts_count = hot_account_config[shard_index]
    if hot_accounts_count > 0:
        hot_tx_per_account = hot_tx_per_shard / hot_accounts_count
        hot_tx_percentage_per_account = (hot_tx_per_account / total_tx_per_shard) * 100

    hot_tx_percentage = 0
    non_hot_tx_percentage = 0
    if total_tx_per_shard > 0:
        hot_tx_percentage = (hot_tx_per_shard / total_tx_per_shard) * 100
        non_hot_tx_percentage = (non_hot_tx_per_shard / total_tx_per_shard) * 100

    return {
        "total_tx_for_shard": total_tx_per_shard,
        "hot_tx_percentage_for_shard": hot_tx_percentage,
        "non_hot_tx_percentage_for_shard": non_hot_tx_percentage,
        "num_hot_tx_per_shard": hot_tx_per_shard,
        "num_non_hot_tx_per_shard": non_hot_tx_per_shard,
        "num_tx_per_hot_account": hot_tx_per_account,
        "hot_tx_percentage_per_account": hot_tx_percentage_per_account
    }


def computeQueuingTxsAtIterationWithPercentage(i, b, q_0, txs_generated_per_iteration_for_curr_shard, hot_accounts_percentage, num_hot_accounts, single_hot_account_percentage, aggregated_light_account_percentage): # i = iteration, b = block capacity
    print(f"Iteration {i} corresponds to i x 120 = {i * 120} total txs in the system (all shards)")     
    #txs_generated_per_iteration_for_curr_shard = 120
    
    if i == 1:
        total_txs_1 = txs_generated_per_iteration_for_curr_shard + q_0
        q_1 = total_txs_1 - b if (total_txs_1 - b) > 0 else 0
        print(f"q_1 = {q_1}")
        return q_1 if q_1 >= 0 else 0
    
    
    #return txs_generated_per_iteration + computeQueuingTxsAtIteration(i-1, b) - b
    prev_q = computeQueuingTxsAtIterationWithPercentage(i-1, b, q_0, txs_generated_per_iteration_for_curr_shard, hot_accounts_percentage, num_hot_accounts, single_hot_account_percentage, aggregated_light_account_percentage)
    q_i = txs_generated_per_iteration_for_curr_shard + prev_q - b
    final_q_i = q_i if q_i >= 0 else 0
    #num_queuing_for_single_hot_account = math.ceil(( final_q_i * hot_accounts_percentage ) / num_hot_accounts) if num_hot_accounts != 0 else 0
    #num_queuing_for_aggregated_ligh_accounts = math.ceil(final_q_i * (1.0 - hot_accounts_percentage)) if num_hot_accounts != 0 else final_q_i
    num_queuing_for_single_hot_account = math.ceil(( final_q_i * single_hot_account_percentage / 100))
    num_queuing_for_aggregated_ligh_accounts = math.ceil(final_q_i * aggregated_light_account_percentage / 100 )
    print(f"q_{i} = {final_q_i}, with {num_queuing_for_single_hot_account} for each hot account ({num_hot_accounts} x {single_hot_account_percentage}% = {num_hot_accounts * num_queuing_for_single_hot_account}) and {num_queuing_for_aggregated_ligh_accounts} for aggregated light accounts")
    
    return final_q_i




def getGeneratedTxsStatisticsFromCSV(starting_index, num_txs_to_read, hot_account_percentage, output_dir):
    # Calculate the number of rows to skip
    skiprows = range(1, starting_index)  # Skip the header row
    print(f"--------------- SKIPPING ROWS FROM 1 TO {starting_index} --------------")

    # Read only the specified range of rows from the CSV file into a pandas DataFrame
    df = pd.read_csv(f"{output_dir}/{GENERATED_TXS_STATISTICS_CSV}", skiprows=skiprows, nrows=num_txs_to_read)

    # Define the shard values
    shard_values = [0, 1, 2]

    # Initialize a dictionary to store the loads (tx count) for each shard
    #shard_loads = {shard: 0 for shard in shard_values}

    # Initialize a dictionary to store the predicted upcoming transactions for each account
    n_t = {account: 0 for account in hot_accounts}
    # Initialize a dictionary to store the predicted upcoming transactions for each account
    q_t_1 = {account: 0 for account in hot_accounts}
    # Initialize a dictionary to store the predicted upcoming transactions by aggregated light accounts for each shard
    m_t = {shard: 0 for shard in shard_values}
    
    total_txs_by_shard = {shard: 0 for shard in shard_values}

    x_t = {account: {} for account in hot_accounts}

    hot_accounts_config = {shard: 0 for shard in shard_values}
    
    for account in hot_accounts:
        account_shard = accounts_info[account]['shard']
        hot_accounts_config[account_shard] += 1

    for shard in shard_values:
        df_shard = df[df['sender_shard'] == shard]
        transactions_count = len(df_shard)
        total_txs_by_shard[shard] = transactions_count

        df_light = df_shard[~df_shard['sender'].isin(hot_accounts)]
        # Count the number of transactions generated by the current account
        transactions_count = len(df_light)
        # Update the count in the dictionary
        m_t[shard] = transactions_count


    for account in hot_accounts:
        account_rows = df[df['sender'] == account]
        # Count the number of transactions generated by the current account
        transactions_count = len(account_rows)
        # Update the count in the dictionary
        n_t[account] = transactions_count

        account_shard = accounts_info[account]['shard']
        
        if account_shard == 0:
            x_t[account] = {0: 1, 1: 0, 2: 0}
        elif account_shard == 1:
            x_t[account] = {0: 0, 1: 1, 2: 0}
        elif account_shard == 2:
            x_t[account] = {0: 0, 1: 0, 2: 1}

        res = calculate_transactions_for_shard(num_transactions=total_txs_by_shard[account_shard], hot_account_percentage=int(hot_account_percentage * 100), hot_account_config=hot_accounts_config, shard_index=account_shard)
        #TODO: rendere i, b e q_0 parametrici
        final_q_i = computeQueuingTxsAtIterationWithPercentage(i=10, b=100, q_0 = 0, txs_generated_per_iteration_for_curr_shard=120, hot_accounts_percentage=hot_account_percentage, num_hot_accounts=len(hot_accounts), single_hot_account_percentage=res["hot_tx_percentage_per_account"], aggregated_light_account_percentage=res["non_hot_tx_percentage_for_shard"])
        single_hot_account_percentage = res["hot_tx_percentage_per_account"]
        aggregated_light_account_percentage = res["non_hot_tx_percentage_for_shard"]
        num_queuing_for_single_hot_account = math.ceil(( final_q_i * single_hot_account_percentage / 100))
        num_queuing_for_aggregated_ligh_accounts = math.ceil(final_q_i * aggregated_light_account_percentage / 100 )
        
        q_t_1[account] = num_queuing_for_single_hot_account


    # Compute load for each hot account
    l_t = {j: n_t[j] + q_t_1[j] for j in hot_accounts}

    print(f"n_t: {n_t}")
    print(f"m_t: {m_t}")
    print(f"q_t_1: {q_t_1}")
    print(f"total_txs_by_shard: {total_txs_by_shard}")
    print(f"x_t: {x_t}")
    print(f"l_t: {l_t}")
    return n_t, m_t, q_t_1, total_txs_by_shard, x_t, l_t

    

def createOutputCSV():
    # Open the file in write mode
    with open(OUTPUT_CSV, mode='w', newline='') as file:
        writer = csv.writer(file)
        writer.writerow(["timestamp", "txHashes"])  # Write header 

def createGeneratedTxsStatisticsCSV(experiment_folder):
    # Open the file in write mode
    with open(f"{experiment_folder}/{GENERATED_TXS_STATISTICS_CSV}", mode='w', newline='') as file:
        writer = csv.writer(file)
        writer.writerow(["tx_id", "sender", "receiver", "nonce", "sender_shard", "receiver_shard", "cross_shard"])  # Write header 



# Function to query Elasticsearch for end_timestamp based on txHash
def get_has_corresponding_AAT(tx_hash):
    # Connect to Elasticsearch
    es = Elasticsearch([{'host': 'localhost', 'port': 9200, 'scheme': 'http'}])


    query = {
        "query": {
            "bool": {
                "must": [
                    {
                        "exists": {
                            "field": "originalTxHash"
                        }
                    },
                    {
                        "match": {
                            "originalTxHash": tx_hash
                        }
                    }
                ]
            }
        }
    }

    res = es.search(index="transactions", body=query)
    
    if res['hits']['total']['value'] > 0:
        return 1 #true
    else:
        return 0 #false
    


def get_has_corresponding_AAT_from_JSON(tx_hash, experiment_folder):
    # Load data from JSON file
    json_file = experiment_folder + '/elasticsearch_data.json'  # Replace with the path to your JSON file
    with open(json_file, 'r') as f:
        data = json.load(f)

    # Iterate over each document in the JSON data
    for document in data:
        # Check if the document has the originalTxHash field and its value matches tx_hash
        if 'originalTxHash' in document and document['originalTxHash'] == tx_hash:
            return 1  # True, corresponding AAT found

    # No corresponding AAT found
    return 0  # False




# Function to query Elasticsearch for end_timestamp based on txHash
def get_is_affected_by_AAT(mini_block_hash):
    # Connect to Elasticsearch
    es = Elasticsearch([{'host': 'localhost', 'port': 9200, 'scheme': 'http'}])


    query = {
        "query": {
            "bool": {
                "must": [
                    {
                        "exists": {
                            "field": "originalMiniBlockHash"
                        }
                    },
                    {
                        "match": {
                            "originalMiniBlockHash": mini_block_hash
                        }
                    }
                ]
            }
        }
    }

    try:
        res = es.search(index="transactions", body=query)
        
        if res['hits']['total']['value'] > 0:
            return 1 #true
        else:
            return 0 #false
    except:
        return 0
    
def get_is_affected_by_AAT_from_JSON(mini_block_hash, experiment_folder):
    # Load data from JSON file
    json_file = experiment_folder + '/elasticsearch_data.json'  # Replace with the path to your JSON file

    try:
        with open(json_file, 'r') as f:
            data = json.load(f)

        # Iterate over each document in the JSON data
        for document in data:
            # Check if the document has the originalMiniBlockHash field and its value matches mini_block_hash
            if 'originalMiniBlockHash' in document and document['originalMiniBlockHash'] == mini_block_hash:
                return 1  # True, affected by AAT

        # No matching document found
        return 0  # False
    except Exception as e:
        print(f"Error occurred: {e}")
        return 0



    
# Function to query Elasticsearch for end_timestamp based on txHash
def get_statistics(tx_hash):
    # Connect to Elasticsearch
    es = Elasticsearch([{'host': 'localhost', 'port': 9200, 'scheme': 'http'}])

    query = {
        "query": {
            "match": {
                "_id": tx_hash
            }
        }
    }
    

    res = es.search(index="transactions", body=query)
    
    if res['hits']['total']['value'] > 0:
        end_timestamp = res['hits']['hits'][0]['_source']['timestamp']
        sender_shard = res['hits']['hits'][0]['_source']['senderShard']
        receiver_shard = res['hits']['hits'][0]['_source']['receiverShard']
        sender = res['hits']['hits'][0]['_source']['sender']
        receiver = res['hits']['hits'][0]['_source']['receiver']
        mini_block_hash = res['hits']['hits'][0]['_source']['miniBlockHash']
        return end_timestamp, sender_shard, receiver_shard, sender, receiver, mini_block_hash
    else:
        return (None, None, None, None, None, None)


# Function to query JSON data for statistics based on txHash
def get_statistics_from_JSON(tx_hash, experiment_folder):
    # Load data from JSON file
    json_file = experiment_folder + '/elasticsearch_data.json'  # Replace with the path to your JSON file

    with open(json_file, 'r') as f:
        data = json.load(f)

    # Iterate over each document in the JSON data
    for document in data:
        # Check if the document has the matching _id
        if document['_id'] == tx_hash:
            # Extract statistics from the document
            end_timestamp = document['timestamp']
            sender_shard = document['senderShard']
            receiver_shard = document['receiverShard']
            sender = document['sender']
            receiver = document['receiver']
            mini_block_hash = document['miniBlockHash']
            return end_timestamp, sender_shard, receiver_shard, sender, receiver, mini_block_hash

    # No matching document found
    return (None, None, None, None, None, None)



def addStatisticsToCSV(experiment_folder):
    # Read CSV file
    df = pd.read_csv(f"{experiment_folder}/{OUTPUT_CSV}")

    # Add a new column 'end_timestamp' and populate it by querying Elasticsearch
    df[['end_timestamp', 'sender_shard', 'receiver_shard', 'sender', 'receiver', 'mini_block_hash']] = df['txHashes'].apply(get_statistics).apply(pd.Series)
    # Calculate the timestamp difference and add it as a new column
    df['timestamp_difference'] = df['end_timestamp'] - df['realTimestamp'] # Before: df['end_timestamp'] - df['timestamp']
    df['has_corresponding_AAT'] = df['txHashes'].apply(get_has_corresponding_AAT)
    df['is_affected_by_AAT'] = df['mini_block_hash'].apply(get_is_affected_by_AAT)

    # Write the updated dataframe to the new CSV file
    df.to_csv(f"{experiment_folder}/{OUTPUT_CSV_WITH_STATISTICS}", index=False)

    print(f"Result saved to {experiment_folder}/{OUTPUT_CSV_WITH_STATISTICS}")


def plotData(field_to_group_by, experiment_folder):
    # Read the CSV file
    df = pd.read_csv(f"{experiment_folder}/{OUTPUT_CSV_WITH_STATISTICS}")


    migration_starts_at = 1713187141
    x_migration_start = migration_starts_at - df['timestamp'].min()
    print(str(migration_starts_at) + " - " + str(df["timestamp"].min()))
    print(x_migration_start)

    # Add vertical line at specified timestamp
    vertical_timestamp = 1712169682
    x_vertical_ts = vertical_timestamp - df['timestamp'].min()
    print(str(vertical_timestamp) + " - " + str(df["timestamp"].min()))
    print(x_vertical_ts)


    # Convert timestamp to seconds
    df[field_to_group_by] = df[field_to_group_by] - df[field_to_group_by].min()  # Normalize timestamps to start from 0

    # Group by timestamp and calculate the mean of timestamp_difference
    mean_timestamp_diff = df.groupby(field_to_group_by)['timestamp_difference'].mean()

    # Plotting
    plt.figure(figsize=(15, 7))
    plt.plot(mean_timestamp_diff.index, mean_timestamp_diff.values, marker=',', linestyle='-')
    plt.xlabel('Timestamp (Seconds)')
    plt.ylabel('Mean Timestamp Difference (Seconds)')
    plt.title('Mean Timestamp Difference over Time (Seconds)')
    plt.grid(True)
    plt.xticks(rotation=45)
    
    
    #plt.axvline(x=x_migration_start, color='r', linestyle='--', label='Vertical Line at Timestamp')
    #plt.axvline(x=x_vertical_ts, color='r', linestyle='--', label='Vertical Line at Timestamp 2')
    
    
    plt.tight_layout()
    # Save the plot
    plt.savefig(f"{experiment_folder}/plot_overall_{field_to_group_by}.png")

    # Show the plot
    #plt.show()

    # Save the mean timestamp difference to a new CSV file
    mean_timestamp_diff_df = mean_timestamp_diff.reset_index()
    mean_timestamp_diff_df.to_csv('mean_timestamp_difference_seconds.csv', index=False)


def plotAggregatedData(field_to_group_by, experiment_folder):
    # Read the CSV file
    df = pd.read_csv(f"{experiment_folder}/{OUTPUT_CSV_WITH_STATISTICS}")


    migration_starts_at = 1713187141
    x_migration_start = migration_starts_at - df['timestamp'].min()
    print(str(migration_starts_at) + " - " + str(df["timestamp"].min()))
    print(x_migration_start)

    # Add vertical line at specified timestamp
    vertical_timestamp = 1712169682
    x_vertical_ts = vertical_timestamp - df['timestamp'].min()
    print(str(vertical_timestamp) + " - " + str(df["timestamp"].min()))
    print(x_vertical_ts)


    # Convert timestamp to seconds
    df[field_to_group_by] = df[field_to_group_by] - df[field_to_group_by].min()  # Normalize timestamps to start from 0

    # Group by timestamp and calculate the mean of timestamp_difference
    mean_timestamp_diff = df.groupby(field_to_group_by)['timestamp_difference'].mean()

    # Plotting
    plt.figure(figsize=(15, 7))
    plt.plot(mean_timestamp_diff.index, mean_timestamp_diff.values, marker=',', linestyle='-')
    plt.xlabel('Timestamp (Seconds)')
    plt.ylabel('Mean Timestamp Difference (Seconds)')
    plt.title('Mean Timestamp Difference over Time (Seconds)')
    plt.grid(True)
    plt.xticks(rotation=45)
    
    
    #plt.axvline(x=x_migration_start, color='r', linestyle='--', label='Vertical Line at Timestamp')
    #plt.axvline(x=x_vertical_ts, color='r', linestyle='--', label='Vertical Line at Timestamp 2')
    
    
    plt.tight_layout()
    # Save the plot
    plt.savefig(f"{experiment_folder}/aggregated_plot_overall_{field_to_group_by}.png")

    # Show the plot
    #plt.show()

    # Save the mean timestamp difference to a new CSV file
    mean_timestamp_diff_df = mean_timestamp_diff.reset_index()
    mean_timestamp_diff_df.to_csv('mean_timestamp_difference_seconds.csv', index=False)




def plotDataFromStatistics(time_threshold, field_to_group_by, experiment_folder, cross_shard_only, redrawn):
    # Read the CSV file
    df = pd.read_csv(f"{experiment_folder}/{OUTPUT_CSV_WITH_STATISTICS}")

    # Find the row with the maximum timestamp difference
    max_time_difference_row = df.loc[df['timestamp_difference'].idxmax()]
    # Retrieve the corresponding txHash
    max_time_difference_txHash = max_time_difference_row['txHashes']
    max_time_difference_has_corresponding_AAT = max_time_difference_row['has_corresponding_AAT']
    max_time_difference_is_affected_by_AAT = max_time_difference_row['is_affected_by_AAT']
    max_time_difference = max_time_difference_row['timestamp_difference']
    print("Max time difference: " + str(max_time_difference) + " ---- TxHash: " + max_time_difference_txHash + " ---- HasCorrespondingAAT: " + 
    str(max_time_difference_has_corresponding_AAT) + " ---- IsAffectedByAAT: " + str(max_time_difference_is_affected_by_AAT))


    # Filter the DataFrame
    ts_differences_greater_than_70 = df[df['timestamp_difference'] > time_threshold]

    print(f"Transactions with timestamp_difference > {time_threshold}:  {len(ts_differences_greater_than_70)} txs")
    for i, row in ts_differences_greater_than_70.iterrows():
        print(
            f"{str(i)})  "
            + f"TxHash: {row['txHashes']}  "
            + f"HasCorrespondingAAT: {row['has_corresponding_AAT']}  "
            + f"IsAffectedByAAT: {row['is_affected_by_AAT']}  "
            + f"Sender: {row['sender']}  "
            + f"Receiver: {row['receiver']}  "
            + f"SenderShard: {row['sender_shard']}  "
            + f"ReceiverShard: {row['receiver_shard']}  "
        )


    # Apply cross-shard filter if required
    if cross_shard_only:
        df = df[df['sender_shard'] != df['receiver_shard']]

    migration_starts_at = 1713187141 #1712606295 #1712322484 #1712169658
    x_migration_start = migration_starts_at - df['timestamp'].min()
    #print(str(migration_starts_at) + " - " + str(df["timestamp"].min()))
    #print(x_migration_start)

    # Add vertical line at specified timestamp
    vertical_timestamp = 1713000131
    x_vertical_ts = vertical_timestamp - df['timestamp'].min()
    #print(str(vertical_timestamp) + " - " + str(df["timestamp"].min()))
    #print(x_vertical_ts)


    # Convert timestamp to seconds
    df[field_to_group_by] = df[field_to_group_by] - df[field_to_group_by].min()  # Normalize timestamps to start from 0

    # Separate data for each shard
    shard_0_data = df[df['sender_shard'] == 0]
    shard_1_data = df[df['sender_shard'] == 1]
    shard_2_data = df[df['sender_shard'] == 2]
    # Separate data for each shard with only cross-shard transactions
    """shard_0_data = df[(df['sender_shard'] == 0) & (df['sender_shard'] != df['receiver_shard'])]
    shard_1_data = df[(df['sender_shard'] == 1) & (df['sender_shard'] != df['receiver_shard'])]
    shard_2_data = df[(df['sender_shard'] == 2) & (df['sender_shard'] != df['receiver_shard'])]"""

    # Group by timestamp and calculate the mean of timestamp_difference for each shard
    mean_timestamp_diff_shard_0 = shard_0_data.groupby(field_to_group_by)['timestamp_difference'].mean()
    mean_timestamp_diff_shard_1 = shard_1_data.groupby(field_to_group_by)['timestamp_difference'].mean()
    mean_timestamp_diff_shard_2 = shard_2_data.groupby(field_to_group_by)['timestamp_difference'].mean()

    # Plotting
    plt.figure(figsize=(15, 6)) #19,3

    # Plot line for shard 0
    plt.plot(mean_timestamp_diff_shard_0.index, mean_timestamp_diff_shard_0.values, marker=',', linestyle='-', label='Shard 0')

    # Plot line for shard 1
    plt.plot(mean_timestamp_diff_shard_1.index, mean_timestamp_diff_shard_1.values, marker=',', linestyle='-', label='Shard 1')

    # Plot line for shard 2
    plt.plot(mean_timestamp_diff_shard_2.index, mean_timestamp_diff_shard_2.values, marker=',', linestyle='-', label='Shard 2')

    plt.xlabel('Timestamp (Seconds)')
    plt.ylabel('Mean Timestamp Difference (Seconds)')
    plt.title('Mean Timestamp Difference over Time for Each Shard (Seconds)')
    plt.grid(True)
    plt.xticks(rotation=45)
    plt.legend()
    
    
    #plt.axvline(x=x_migration_start, color='r', linestyle='--', label='Vertical Line at Timestamp')
    #plt.axvline(x=x_vertical_ts, color='r', linestyle='--', label='Vertical Line at Timestamp 2')

    plt.tight_layout()

    # Save the plot with the appropriate filename
    filename_suffix = "_redrawn" if redrawn else ""
    plt.savefig(f"{experiment_folder}/plot_by_shard_{field_to_group_by}{filename_suffix}.png")


    # Show the plot
    #plt.show()




def plotAggregatedDataFromStatistics(time_threshold, field_to_group_by, experiment_folder):
    # Read the CSV file
    df = pd.read_csv(f"{experiment_folder}/combined.csv")

    # Find the row with the maximum timestamp difference
    max_time_difference_row = df.loc[df['timestamp_difference'].idxmax()]
    # Retrieve the corresponding txHash
    max_time_difference_txHash = max_time_difference_row['txHashes']
    max_time_difference_has_corresponding_AAT = max_time_difference_row['has_corresponding_AAT']
    max_time_difference_is_affected_by_AAT = max_time_difference_row['is_affected_by_AAT']
    max_time_difference = max_time_difference_row['timestamp_difference']
    print("Max time difference: " + str(max_time_difference) + " ---- TxHash: " + max_time_difference_txHash + " ---- HasCorrespondingAAT: " + 
    str(max_time_difference_has_corresponding_AAT) + " ---- IsAffectedByAAT: " + str(max_time_difference_is_affected_by_AAT))


    # Filter the DataFrame
    ts_differences_greater_than_70 = df[df['timestamp_difference'] > time_threshold]

    print(f"Transactions with timestamp_difference > {time_threshold}:  {len(ts_differences_greater_than_70)} txs")
    for i, row in ts_differences_greater_than_70.iterrows():
        print(
            f"{str(i)})  "
            + f"TxHash: {row['txHashes']}  "
            + f"HasCorrespondingAAT: {row['has_corresponding_AAT']}  "
            + f"IsAffectedByAAT: {row['is_affected_by_AAT']}  "
            + f"Sender: {row['sender']}  "
            + f"Receiver: {row['receiver']}  "
            + f"SenderShard: {row['sender_shard']}  "
            + f"ReceiverShard: {row['receiver_shard']}  "
        )


    migration_starts_at = 1713187141 #1712606295 #1712322484 #1712169658
    x_migration_start = migration_starts_at - df['normalized_timestamp'].min()
    #print(str(migration_starts_at) + " - " + str(df["timestamp"].min()))
    #print(x_migration_start)

    # Add vertical line at specified timestamp
    vertical_timestamp = 1713000131
    x_vertical_ts = vertical_timestamp - df['normalized_timestamp'].min()
    #print(str(vertical_timestamp) + " - " + str(df["timestamp"].min()))
    #print(x_vertical_ts)


    # Convert timestamp to seconds
    df[field_to_group_by] = df[field_to_group_by] - df[field_to_group_by].min()  # Normalize timestamps to start from 0

    # Separate data for each shard
    shard_0_data = df[df['sender_shard'] == 0]
    shard_1_data = df[df['sender_shard'] == 1]
    shard_2_data = df[df['sender_shard'] == 2]
    # Separate data for each shard with only cross-shard transactions
    """shard_0_data = df[(df['sender_shard'] == 0) & (df['sender_shard'] != df['receiver_shard'])]
    shard_1_data = df[(df['sender_shard'] == 1) & (df['sender_shard'] != df['receiver_shard'])]
    shard_2_data = df[(df['sender_shard'] == 2) & (df['sender_shard'] != df['receiver_shard'])]"""

    # Group by timestamp and calculate the mean of timestamp_difference for each shard
    mean_timestamp_diff_shard_0 = shard_0_data.groupby(field_to_group_by)['timestamp_difference'].mean()
    mean_timestamp_diff_shard_1 = shard_1_data.groupby(field_to_group_by)['timestamp_difference'].mean()
    mean_timestamp_diff_shard_2 = shard_2_data.groupby(field_to_group_by)['timestamp_difference'].mean()

    # Plotting
    plt.figure(figsize=(15, 6)) #19,3

    # Plot line for shard 0
    plt.plot(mean_timestamp_diff_shard_0.index, mean_timestamp_diff_shard_0.values, marker=',', linestyle='-', label='Shard 0')

    # Plot line for shard 1
    plt.plot(mean_timestamp_diff_shard_1.index, mean_timestamp_diff_shard_1.values, marker=',', linestyle='-', label='Shard 1')

    # Plot line for shard 2
    plt.plot(mean_timestamp_diff_shard_2.index, mean_timestamp_diff_shard_2.values, marker=',', linestyle='-', label='Shard 2')

    plt.xlabel('Timestamp (Seconds)')
    plt.ylabel('Mean Timestamp Difference (Seconds)')
    plt.title('Mean Timestamp Difference over Time for Each Shard (Seconds)')
    plt.grid(True)
    plt.xticks(rotation=45)
    plt.legend()
    
    
    #plt.axvline(x=x_migration_start, color='r', linestyle='--', label='Vertical Line at Timestamp')
    #plt.axvline(x=x_vertical_ts, color='r', linestyle='--', label='Vertical Line at Timestamp 2')

    plt.tight_layout()

    # Save the plot
    plt.savefig(f"{experiment_folder}/plot_mean_of_means_by_shard_{field_to_group_by}.png")

    # Show the plot
    #plt.show()


"""
def plotAggregatedDataFromStatistics(time_threshold, field_to_group_by, experiment_folder):
    # Read the CSV file
    df = pd.read_csv(f"{experiment_folder}/grouped_means.csv")

    # Find the row with the maximum timestamp difference
    max_time_difference_row = df.loc[df['timestamp_difference'].idxmax()]
    # Retrieve the corresponding txHash
    max_time_difference_txHash = max_time_difference_row['txHashes']
    max_time_difference_has_corresponding_AAT = max_time_difference_row['has_corresponding_AAT']
    max_time_difference_is_affected_by_AAT = max_time_difference_row['is_affected_by_AAT']
    max_time_difference = max_time_difference_row['timestamp_difference']
    print("Max time difference: " + str(max_time_difference) + " ---- TxHash: " + max_time_difference_txHash + " ---- HasCorrespondingAAT: " + 
    str(max_time_difference_has_corresponding_AAT) + " ---- IsAffectedByAAT: " + str(max_time_difference_is_affected_by_AAT))


    # Filter the DataFrame
    ts_differences_greater_than_70 = df[df['timestamp_difference'] > time_threshold]

    print(f"Transactions with timestamp_difference > {time_threshold}:  {len(ts_differences_greater_than_70)} txs")
    for i, row in ts_differences_greater_than_70.iterrows():
        print(
            f"{str(i)})  "
            + f"TxHash: {row['txHashes']}  "
            + f"HasCorrespondingAAT: {row['has_corresponding_AAT']}  "
            + f"IsAffectedByAAT: {row['is_affected_by_AAT']}  "
            + f"Sender: {row['sender']}  "
            + f"Receiver: {row['receiver']}  "
            + f"SenderShard: {row['sender_shard']}  "
            + f"ReceiverShard: {row['receiver_shard']}  "
        )


    migration_starts_at = 1713187141 #1712606295 #1712322484 #1712169658
    x_migration_start = migration_starts_at - df['normalized_timestamp'].min()
    #print(str(migration_starts_at) + " - " + str(df["timestamp"].min()))
    #print(x_migration_start)

    # Add vertical line at specified timestamp
    vertical_timestamp = 1713000131
    x_vertical_ts = vertical_timestamp - df['normalized_timestamp'].min()
    #print(str(vertical_timestamp) + " - " + str(df["timestamp"].min()))
    #print(x_vertical_ts)


    # Convert timestamp to seconds
    df[field_to_group_by] = df[field_to_group_by] - df[field_to_group_by].min()  # Normalize timestamps to start from 0

    # Separate data for each shard
    shard_0_data = df[df['sender_shard'] == 0]
    shard_1_data = df[df['sender_shard'] == 1]
    shard_2_data = df[df['sender_shard'] == 2]

    # Group by timestamp and calculate the mean of timestamp_difference for each shard
    mean_timestamp_diff_shard_0 = shard_0_data.groupby(field_to_group_by)['timestamp_difference'].mean()
    mean_timestamp_diff_shard_1 = shard_1_data.groupby(field_to_group_by)['timestamp_difference'].mean()
    mean_timestamp_diff_shard_2 = shard_2_data.groupby(field_to_group_by)['timestamp_difference'].mean()


    # 2. Trova i valori massimi di end_timestamp per ogni gruppo di normalized_timestamp
    max_end_timestamp_shard_0 = shard_0_data.groupby(field_to_group_by)['end_timestamp'].max()
    max_end_timestamp_shard_1 = shard_1_data.groupby(field_to_group_by)['end_timestamp'].max()
    max_end_timestamp_shard_2 = shard_2_data.groupby(field_to_group_by)['end_timestamp'].max()

    # 3. Unisci i due risultati in un unico DataFrame
    result_df_0 = pd.DataFrame({
        'mean_timestamp_difference': mean_timestamp_diff_shard_0,
        'max_end_timestamp': max_end_timestamp_shard_0
    }).reset_index()
    # 3. Unisci i due risultati in un unico DataFrame
    result_df_1 = pd.DataFrame({
        'mean_timestamp_difference': mean_timestamp_diff_shard_1,
        'max_end_timestamp': max_end_timestamp_shard_1
    }).reset_index()
    # 3. Unisci i due risultati in un unico DataFrame
    result_df_2 = pd.DataFrame({
        'mean_timestamp_difference': mean_timestamp_diff_shard_2,
        'max_end_timestamp': max_end_timestamp_shard_2
    }).reset_index()



    # Plotting
    plt.figure(figsize=(15, 6)) #19,3

    # Plot line for shard 0
    #plt.plot(mean_timestamp_diff_shard_0.index, mean_timestamp_diff_shard_0.values, marker=',', linestyle='-', label='Shard 0')
    plt.plot(result_df_0['max_end_timestamp'], result_df_0['mean_timestamp_difference'], marker=',', linestyle='-', label='Mean Timestamp Difference')

    # Plot line for shard 1
    #plt.plot(mean_timestamp_diff_shard_1.index, mean_timestamp_diff_shard_1.values, marker=',', linestyle='-', label='Shard 1')
    plt.plot(result_df_1['max_end_timestamp'], result_df_1['mean_timestamp_difference'], marker=',', linestyle='-', label='Mean Timestamp Difference')


    # Plot line for shard 2
    #plt.plot(mean_timestamp_diff_shard_2.index, mean_timestamp_diff_shard_2.values, marker=',', linestyle='-', label='Shard 2')
    plt.plot(result_df_2['max_end_timestamp'], result_df_2['mean_timestamp_difference'], marker=',', linestyle='-', label='Mean Timestamp Difference')


    
    plt.xlabel('Timestamp (Seconds)')
    plt.ylabel('Mean Timestamp Difference (Seconds)')
    plt.title('Mean Timestamp Difference over Time for Each Shard (Seconds)')
    plt.grid(True)
    plt.xticks(rotation=45)
    plt.legend()
    
    
    #plt.axvline(x=x_migration_start, color='r', linestyle='--', label='Vertical Line at Timestamp')
    #plt.axvline(x=x_vertical_ts, color='r', linestyle='--', label='Vertical Line at Timestamp 2')

    plt.tight_layout()

    # Save the plot
    plt.savefig(f"{experiment_folder}/plot_mean_of_means_by_shard_{field_to_group_by}.png")

    # Show the plot
    #plt.show()
"""




def plotAllExperimentsFromStatisticsInOnePlot(time_threshold, field_to_group_by, experiment_folder):
    plt.figure(figsize=(15, 6))  # Create a single figure for all plots

    for folder in os.listdir(experiment_folder):
        folder_path = os.path.join(experiment_folder, folder)
        if os.path.isdir(folder_path):
            csv_file = os.path.join(folder_path, OUTPUT_CSV_WITH_STATISTICS)
            if os.path.exists(csv_file):
                df = pd.read_csv(csv_file)

                # Normalize timestamps to start from 0
                df[field_to_group_by] = df[field_to_group_by] - df[field_to_group_by].min()

                # Your existing code for processing the DataFrame and plotting the data
                # ...

                # Separate data for each shard
                shard_0_data = df[df['sender_shard'] == 0]
                shard_1_data = df[df['sender_shard'] == 1]
                shard_2_data = df[df['sender_shard'] == 2]

                # Group by timestamp and calculate the mean of timestamp_difference for each shard
                mean_timestamp_diff_shard_0 = shard_0_data.groupby(field_to_group_by)['timestamp_difference'].mean()
                mean_timestamp_diff_shard_1 = shard_1_data.groupby(field_to_group_by)['timestamp_difference'].mean()
                mean_timestamp_diff_shard_2 = shard_2_data.groupby(field_to_group_by)['timestamp_difference'].mean()

                # Plot line for shard 0
                plt.plot(mean_timestamp_diff_shard_0.index, mean_timestamp_diff_shard_0.values, marker=',', linestyle='-', label=f'Shard 0 - {folder}')

                # Plot line for shard 1
                plt.plot(mean_timestamp_diff_shard_1.index, mean_timestamp_diff_shard_1.values, marker=',', linestyle='-', label=f'Shard 1 - {folder}')

                # Plot line for shard 2
                plt.plot(mean_timestamp_diff_shard_2.index, mean_timestamp_diff_shard_2.values, marker=',', linestyle='-', label=f'Shard 2 - {folder}')

    plt.xlabel('Timestamp (Seconds)')
    plt.ylabel('Mean Timestamp Difference (Seconds)')
    plt.title('Mean Timestamp Difference over Time for Each Shard (Seconds)')
    plt.grid(True)
    plt.xticks(rotation=45)
    plt.legend()
    plt.tight_layout()

    # Save the plot
    plt.savefig(f"{experiment_folder}/combined_plot_by_shard_{field_to_group_by}.png")

    # Show the plot
    plt.show()



def aggregateDataFromStatistics(time_threshold, field_to_group_by, experiment_folder):
    shard_data = {}

    # Iterate over each folder
    for folder in os.listdir(experiment_folder):
        folder_path = os.path.join(experiment_folder, folder)
        if os.path.isdir(folder_path):
            csv_file = os.path.join(folder_path, OUTPUT_CSV_WITH_STATISTICS)
            if os.path.exists(csv_file):
                df = pd.read_csv(csv_file)

                # Normalize timestamps to start from 0
                df[field_to_group_by] = df[field_to_group_by] - df[field_to_group_by].min()

                # Separate data for each shard
                for shard in range(3):
                    shard_df = df[df['sender_shard'] == shard]
                    mean_timestamp_diff = shard_df.groupby(field_to_group_by)['timestamp_difference'].mean()

                    if shard not in shard_data:
                        shard_data[shard] = []
                    shard_data[shard].append(mean_timestamp_diff)

    # Aggregate data across all CSV files for each shard
    aggregated_data = {}
    for shard, shard_dfs in shard_data.items():
        aggregated_data[shard] = pd.concat(shard_dfs).groupby(level=0).mean()

    return aggregated_data

def plotAggregatedData(aggregated_data, field_to_group_by, experiment_folder):
    plt.figure(figsize=(15, 6))  # Create a single figure for all plots

    for shard, data in aggregated_data.items():
        plt.plot(data.index, data.values, marker=',', linestyle='-', label=f'Shard {shard}')

    plt.xlabel('Timestamp (Seconds)')
    plt.ylabel('Mean Timestamp Difference (Seconds)')
    plt.title('Aggregated Mean Timestamp Difference over Time for Each Shard (Seconds)')
    plt.grid(True)
    plt.xticks(rotation=45)
    plt.legend()
    plt.tight_layout()

    # Save the plot
    plt.savefig(f"{experiment_folder}/aggregated_plot_by_shard_{field_to_group_by}.png")

    # Show the plot
    plt.show()



def get_all_commands_to_run(file_names, num_txs_to_send, input_directory):
    file_names_to_process = file_names[:num_txs_to_send]
    commands_to_run = []

    for file_name in file_names_to_process:
           

        timestamp = datetime.now()
        #print(timestamp)
        
        infile = input_directory+file_name

        if file_name.endswith("account_allocation.json"):
            command_to_run = "ACCOUNT_ALLOCATION"
            # Read the JSON file
            with open(infile, 'r') as file:
                data = json.load(file)

            # Extract the value associated with "accountAllocationPayload"
            account_allocation_payload = data["accountAllocationPayload"]

        else:
            command_to_run = f"mxpy tx send \
                    --proxy=http://localhost:7950 \
                    --infile={infile};"
            account_allocation_payload = ""

        commands_to_run.append({"command": command_to_run, "infile": infile, "accountAllocationPayload": json.dumps(account_allocation_payload)})
    
    return commands_to_run



def get_all_commands_to_run_from_CSV(df_generated_txs, num_txs_to_send, tx_ids_for_account_allocation, experiment_folder):
    df_generated_txs_to_process = df_generated_txs[:num_txs_to_send]
    commands_to_run = []
    
    gas_limit = 70000

    for index, tx_row in df_generated_txs_to_process.iterrows():
        tx_id = tx_row["tx_id"]   
        timestamp = datetime.now()
        #print(timestamp)
        
        #infile = input_directory+file_name

        """if file_name.endswith("account_allocation.json"):
            command_to_run = "ACCOUNT_ALLOCATION"
            # Read the JSON file
            with open(infile, 'r') as file:
                data = json.load(file)

            # Extract the value associated with "accountAllocationPayload"
            account_allocation_payload = data["accountAllocationPayload"]

        else:
        command_to_run = f"mxpy tx send \
                    --proxy=http://localhost:7950 \
                    --infile={infile};" """
        
        command_to_run = f"mxpy tx new \
            --nonce={tx_row['nonce']} \
            --data=\"Hello, World\" \
            --gas-limit={gas_limit} \
            --receiver={tx_row['receiver']} \
            --pem=~/multiversx-sdk/testwallets/latest/users/{accounts_info[tx_row['sender']]['username']}.pem \
            --chain=localnet \
            --proxy=http://localhost:7950 \
            --send"

        account_allocation_payload = ""

        commands_to_run.append({"command": command_to_run, "tx_id": tx_row["tx_id"], "accountAllocationPayload": json.dumps(account_allocation_payload)})
    
        if tx_id in tx_ids_for_account_allocation:
            # Construct the filename
            padded_tx_id = str(tx_id).zfill(7)
            file_name = f"transaction_{padded_tx_id}_account_allocation.json"
            infile = os.path.join(experiment_folder, file_name)
            
            # Read the JSON file
            with open(infile, 'r') as file:
                data = json.load(file)
            
            # Extract the value associated with "accountAllocationPayload"
            account_allocation_payload = data["accountAllocationPayload"]
            
            # ACCOUNT_ALLOCATION command
            account_allocation_command = "ACCOUNT_ALLOCATION"
            # Append the ACCOUNT_ALLOCATION command first
            commands_to_run.append({
                "command": account_allocation_command,
                "tx_id": tx_id,
                "accountAllocationPayload": json.dumps(account_allocation_payload)
            })                   

    return commands_to_run



def printStatistics():
    # Step 1: Read the CSV file into a DataFrame
    df = pd.read_csv(OUTPUT_CSV_WITH_STATISTICS)

    # Step 2: Calculate statistics by shard
    shard_stats = df.groupby('sender_shard')['sender'].nunique()
    total_transactions = df['sender'].nunique()
    shard_stats_percentage = (shard_stats / total_transactions) * 100

    print("Statistics by Shard:")
    print(shard_stats_percentage)

    # Step 3: Identify the sender with the most transactions
    sender_stats = df.groupby('sender').size().sort_values(ascending=False)
    most_transactions_sender = sender_stats.index[0]
    most_transactions_count = sender_stats.iloc[0]

    print("\nSender with the most transactions:")
    print("Sender:", most_transactions_sender)
    print("Number of transactions:", most_transactions_count)

    # Step 4: Calculate statistics for cross-shard transactions
    most_transactions_df = df[df['sender'] == most_transactions_sender]
    cross_shard_transactions = most_transactions_df[most_transactions_df['sender_shard'] != most_transactions_df['receiver_shard']]
    cross_shard_count = cross_shard_transactions.shape[0]
    cross_shard_percentage = (cross_shard_count / most_transactions_count) * 100

    # Step 5: Calculate statistics for intra-shard transactions
    intra_shard_count = most_transactions_count - cross_shard_count
    intra_shard_percentage = 100 - cross_shard_percentage

    print("\nStatistics for Sender with Most Transactions:")
    print("Percentage of Cross-Shard Transactions:", cross_shard_percentage)
    print("Percentage of Intra-Shard Transactions:", intra_shard_percentage)

    # Step 6: Calculate statistics by shard for intra and cross-shard transactions
    shard_transactions = df.groupby(['sender_shard', (df['sender_shard'] != df['receiver_shard'])])['sender'].count()
    shard_transactions_percentage = (shard_transactions / shard_transactions.groupby('sender_shard').sum()) * 100

    print("\nStatistics by Shard for Intra and Cross-Shard Transactions:")
    print(shard_transactions_percentage)




def createOutputCSVWithRealExecutionTimestamp(experiment_folder):
    # Open the file in write mode
    with open(f"{experiment_folder}/{OUTPUT_CSV}", mode='w', newline='') as file:
        writer = csv.writer(file)
        writer.writerow(["file_name","timestamp", "txHashes", "realTimestamp"])  # Write header 


def saveTxsBatchToOutputCSV(batch_outputs, experiment_folder):
    timestamp_of_generation = batch_outputs["timestamp_of_generation"]
    # Sort transactions based on file name
    sorted_transactions_data = sorted(batch_outputs["batchTxs"].items(), key=lambda x: x[0])

    # Write to CSV file
    with open(f"{experiment_folder}/{OUTPUT_CSV}", mode='a', newline='') as file:
        writer = csv.writer(file)
        for tx_id, data in sorted_transactions_data:
            writer.writerow([tx_id, int(timestamp_of_generation), data["txHash"], int(data["realTimestamp"])])

# Function to retrieve shard information from the REST API
def get_shard_from_api_call(account_address):
    # Make the REST API call to retrieve shard information
    # Replace 'API_ENDPOINT' with the actual endpoint of the API   

    response = requests.get(f'http://localhost:7950/address/{account_address}')
    
    # Parse the response and extract shard information
    if response.status_code == 200:
        response_json = response.json()
        account_data = response_json.get('data', {}).get('account', {})
        shard = account_data.get('shardId')
        return shard
    else:
        print(f"Failed to retrieve shard information for account {account_address}")
        return None


def generateAccountsInfoJsonFileOLD():
    # Update shard information for each account
    for account_address in accounts_info:
        shard = get_shard_from_api_call(account_address)
        accounts_info[account_address]['shard'] = shard

    # Write the dictionary to a JSON file
    with open(ACCOUNTS_INFO_JSON_PATH, "w") as json_file:
        json.dump(accounts_info, json_file, indent=4)


def generateAccountsInfoJsonFile():
    global accounts_info

    num_accounts_per_shard = {0:0, 1:0, 2:0}

    # Read account addresses from JSON files in the input folder
    for filename in os.listdir(INPUT_FOLDER_PATH):
        if filename.endswith('.json'):
            file_path = os.path.join(INPUT_FOLDER_PATH, filename)
            with open(file_path, 'r') as json_file:
                account_data = json.load(json_file)
                account_address = account_data.get('bech32')
                if account_address:
                    shard = get_shard_from_api_call(account_address)
                    # Extract username from filename without extension
                    username = os.path.splitext(filename)[0]
                    # Set nonce as 5 for Alice, 1 for others
                    nonce = 5 if username == "alice" else 1
                    # Set migrationNonce as 0 for all
                    migration_nonce = 0
                    accounts_info[account_address] = {
                        'username': username,
                        'shard': shard,
                        'nonce': nonce,
                        'migrationNonce': migration_nonce
                    }
                    num_accounts_per_shard[shard] += 1

    # Write the updated dictionary to a JSON file
    with open(ACCOUNTS_INFO_JSON_PATH, 'w') as output_json_file:
        json.dump(accounts_info, output_json_file, indent=4)
    print("DONE GENERATING ACCOUNTS INFO JSON FILE")
    print(f"num_accounts_per_shard = {num_accounts_per_shard}")
    #print(f"accounts_info = {accounts_info}")



def updateHotAccounts(shard_of_hot_accounts, num_hot_accounts):
    global hot_accounts

    hot_accounts = []
    count = 0

    # Loop through each account in the dictionary
    for account_address, account_data in accounts_info.items():
        # Check if the shard ID matches and haven't reached the desired number
        if account_data["shard"] == shard_of_hot_accounts and count < num_hot_accounts:
            hot_accounts.append(account_address)
            count += 1

    print(f"{num_hot_accounts} HOT ACCOUNTS SELECTED: {hot_accounts}")






sent = False

def myLastSenderWithBarrier(input_directory, delay_in_seconds, num_txs_to_send, num_txs_per_batch, with_AMTs):
    # Get the current time before the function execution
    start_time = time.time()
    # Define a lock for synchronization
    lock = threading.Lock()

    global sent

    # Define a threading barrier
    barrier = threading.Barrier(num_txs_per_batch)  # Additional +1 for the main thread

    # Define a function to execute a batch of commands and capture their output
    def execute_batch(commands, batch_outputs, batch_index):
        global sent
        current_time = time.time()
        elapsed_time = current_time - start_time
        
        if elapsed_time >= 50 and not sent:
            # Trigger your logic here
            print("2 minutes seconds have passed.")
            #sendAccountAllocation()
            sent = True


        threads = []
        
        timestamp_of_batch_generation = datetime.now()
        batch_outputs["timestamp_of_generation"] = timestamp_of_batch_generation.timestamp()
        batch_outputs["batchTxs"] = {}

        for command in commands:
            thread = threading.Thread(target=run_command, args=(command, batch_outputs, lock, batch_index, barrier))
            threads.append(thread)
            thread.start()
        for thread in threads:
            thread.join()

    # Define a function to run a single command and capture its output
    def run_command(command_data, batch_outputs, lock, batch_index, barrier):
        
        if command_data["command"] == "ACCOUNT_ALLOCATION":
            url = "http://localhost:10206/node/send-account-allocation"
            command_to_run = f'curl -X POST {url} -H "Content-Type: application/json" --data-raw \'{command_data["accountAllocationPayload"]}\''
        else:
            command_to_run = command_data["command"]        

        try:
            if not command_to_run == None:
                result = subprocess.run(command_to_run, shell=True, capture_output=True, text=True)
                real_timestamp = datetime.now()
                output_data = json.loads(result.stdout)
                txHash = output_data.get('emittedTransactionHash')
                # Acquire the lock before updating the outputs dictionary
                with lock:
                    batch_outputs["batchTxs"][command_data["infile"]] = {"txHash" : txHash, "realTimestamp": real_timestamp.timestamp()}
        except Exception as e:
            print(f"Error executing command: {e}")
            # Log the error to a file
            with open("error.log", mode='a') as log_file:
                log_file.write(f"Error executing command: {e}\n")
        finally:
            # Wait for all threads to reach the barrier before proceeding
            #print("Waiting for batch threads to complete...")
            barrier.wait()

    files = os.listdir(input_directory)
    # Filter out only the JSON files
    json_files = [file for file in files if file.endswith('.json')]
    if not with_AMTs:
        json_files = [file for file in json_files if not file.endswith('_account_allocation.json')]
    sorted_json_files = sorted(json_files)

    createOutputCSVWithRealExecutionTimestamp()

    num_txs = len(sorted_json_files) if num_txs_to_send == 0 else num_txs_to_send

    commands_to_run = get_all_commands_to_run(sorted_json_files, num_txs, input_directory)
    num_total_commands = len(commands_to_run)
    num_batches = math.ceil(num_total_commands / num_txs_per_batch)
    print(f"Num txs to send: {num_txs}")
    print(f"Num batches: {num_batches}")

    total_txs_by_batch = {}

    # Process commands in batches
    for i in range(0, num_batches):
        # Dictionary to store the output of each command
        batch_outputs = {}
        start_index = i * num_txs_per_batch
        end_index = min(start_index + num_txs_per_batch, num_total_commands)
        timestamp = datetime.now()
        
        print(f"Generating batch {i} of transactions: commands_to_run[{start_index}:{end_index}]        Timestamp: {timestamp}")
        
        batch_commands = commands_to_run[start_index:end_index]
        execute_batch(batch_commands, batch_outputs, i)
        total_txs_by_batch[i] = batch_outputs

    # After all commands have been executed, you can access the output from the 'outputs' dictionary
    for i in range(0, num_batches):
        saveTxsBatchToOutputCSV(total_txs_by_batch[i])

    # Get the current time after the function execution
    end_time = time.time()
    # Calculate the time taken
    execution_time = end_time - start_time
    print("Execution time:", execution_time, "seconds")




def myLastSenderWithBarrierFromCSV(delay_in_seconds, num_txs_to_send, num_txs_per_batch, with_AMTs, experiment_folder):
    # Get the current time before the function execution
    start_time = time.time()
    # Define a lock for synchronization
    lock = threading.Lock()

    global sent

    # Define a threading barrier
    barrier = threading.Barrier(num_txs_per_batch)  # Additional +1 for the main thread

    # Define a function to execute a batch of commands and capture their output
    def execute_batch(commands, batch_outputs, batch_index):
        global sent
        current_time = time.time()
        elapsed_time = current_time - start_time
        
        if elapsed_time >= 50 and not sent:
            # Trigger your logic here
            print("2 minutes seconds have passed.")
            #sendAccountAllocation()
            sent = True


        threads = []
        
        timestamp_of_batch_generation = datetime.now()
        batch_outputs["timestamp_of_generation"] = timestamp_of_batch_generation.timestamp()
        batch_outputs["batchTxs"] = {}

        for command in commands:
            thread = threading.Thread(target=run_command, args=(command, batch_outputs, lock, batch_index, barrier))
            threads.append(thread)
            thread.start()
        for thread in threads:
            thread.join()

    # Define a function to run a single command and capture its output
    def run_command(command_data, batch_outputs, lock, batch_index, barrier):
        
        if command_data["command"] == "ACCOUNT_ALLOCATION":
            url = "http://localhost:10206/node/send-account-allocation"
            command_to_run = f'curl -X POST {url} -H "Content-Type: application/json" --data-raw \'{command_data["accountAllocationPayload"]}\''
            print(f"command_data['command'] = {command_data['command']}")
        else:
            command_to_run = command_data["command"]        

        try:
            if not command_to_run == None:
                result = subprocess.run(command_to_run, shell=True, capture_output=True, text=True)
                real_timestamp = datetime.now()
                output_data = json.loads(result.stdout)
                #print(output_data)
                txHash = output_data.get('emittedTransactionHash')
                # Acquire the lock before updating the outputs dictionary
                with lock:
                    batch_outputs["batchTxs"][command_data["tx_id"]] = {"txHash" : txHash, "realTimestamp": real_timestamp.timestamp()}
                    #print(batch_outputs)
        except Exception as e:
            print(f"Error executing command: {e}")
            # Log the error to a file
            with open("error.log", mode='a') as log_file:
                log_file.write(f"Error executing command: {e}\n")
        finally:
            # Wait for all threads to reach the barrier before proceeding
            #print("Waiting for batch threads to complete...")
            barrier.wait()

    """files = os.listdir(input_directory)
    # Filter out only the JSON files
    json_files = [file for file in files if file.endswith('.json')]
    if not with_AMTs:
        json_files = [file for file in json_files if not file.endswith('_account_allocation.json')]
    sorted_json_files = sorted(json_files)"""

    df_generated_txs = pd.read_csv(f"{experiment_folder}/{GENERATED_TXS_STATISTICS_CSV}")

    createOutputCSVWithRealExecutionTimestamp(experiment_folder)

    #num_txs = len(sorted_json_files) if num_txs_to_send == 0 else num_txs_to_send
    tx_ids_for_account_allocation = get_tx_ids_for_account_allocation(experiment_folder, with_AMTs)
    print(f"tx_ids_for_account_allocation = {tx_ids_for_account_allocation}")
    commands_to_run = get_all_commands_to_run_from_CSV(df_generated_txs, num_txs_to_send, tx_ids_for_account_allocation, experiment_folder)
    num_total_commands = len(commands_to_run)
    num_batches = math.floor(num_total_commands / num_txs_per_batch)
    print(f"Num txs to send: {num_txs_to_send}")
    print(f"Num batches: {num_batches}")

    total_txs_by_batch = {}

    start_generating_txs_timestamp = datetime.now().timestamp()

    # Process commands in batches
    for i in range(0, num_batches):
        # Dictionary to store the output of each command
        batch_outputs = {}
        start_index = i * num_txs_per_batch
        end_index = min(start_index + num_txs_per_batch, num_total_commands)
        timestamp = datetime.now()
        
        print(f"Generating batch {i} of transactions: commands_to_run[{start_index}:{end_index}]        Timestamp: {timestamp}")
        
        batch_commands = commands_to_run[start_index:end_index]
        execute_batch(batch_commands, batch_outputs, i)
        total_txs_by_batch[i] = batch_outputs
    
    end_generating_txs_timestamp = datetime.now().timestamp()

    saveStartAndEndTimestampToJSON(experiment_folder, start_generating_txs_timestamp, end_generating_txs_timestamp)

    # After all commands have been executed, you can access the output from the 'outputs' dictionary
    for i in range(0, num_batches):
        saveTxsBatchToOutputCSV(total_txs_by_batch[i], experiment_folder)

    # Get the current time after the function execution
    end_time = time.time()
    # Calculate the time taken
    execution_time = end_time - start_time
    print("Execution time:", execution_time, "seconds")



def get_tx_ids_for_account_allocation(experiment_folder, with_AMTs):
    if not with_AMTs:
        return []
    
    # Compile the regex pattern to match the desired filename and capture the ID part
    pattern = re.compile(r'transaction_(\d{7})_account_allocation\.json')
    
    # Initialize an empty list to store the IDs
    ids = []
    
    # Iterate over all files in the specified folder
    for filename in os.listdir(experiment_folder):
        # Use regex to find matches
        match = pattern.match(filename)
        if match:
            # Extract the ID part, remove leading zeros, and convert to integer
            id_str = match.group(1).lstrip('0')
            id_int = int(id_str)
            ids.append(id_int)
    
    return ids


def saveStartAndEndTimestampToJSON(experiment_folder, start_timestamp, end_timestamp):
  
    # Define the path for the JSON file
    json_file_path = os.path.join(experiment_folder, 'timestamps.json')
    
    # Create a dictionary to store the timestamps
    timestamps_data = {
        'start_timestamp': str(start_timestamp),
        'end_timestamp': str(end_timestamp)
    }
    
    # Write the timestamps to the JSON file
    with open(json_file_path, 'w') as json_file:
        json.dump(timestamps_data, json_file, indent=4)    



#? ---- COMMANDS TO EXECUTE (Correct load by batch) ----

"""generateTransactionsWithCorrectLoadInBatchesWithAccountAllocations(num_txs_per_batch=50, 
                                                                   num_total_txs=20000, 
                                                                   num_txs_threshold_for_account_allocation=6000, 
                                                                   output_dir=TRANSACTIONS_DIRECTORY_WITH_CORRECT_LOAD_BY_BATCH_AND_ACCOUNTS_ALLOCATION, 
                                                                   with_cross_shard_probability=False)"""


"""generateTransactionsWithCorrectLoadInBatchesWithAccountAllocationsOnCSV(num_txs_per_batch=50, 
                                                                        num_total_txs=20000, 
                                                                        num_txs_threshold_for_account_allocation=6000, 
                                                                        output_dir=TRANSACTIONS_DIRECTORY_WITH_CORRECT_LOAD_BY_BATCH_AND_ACCOUNTS_ALLOCATION, 
                                                                        with_cross_shard_probability=False,
                                                                        hot_sender_probability=0.9,
                                                                        hot_accounts_change_threshold=10000)"""

# --------- sendAllGeneratedTransactions(batch_size=100, input_directory=TRANSACTIONS_DIRECTORY_WITH_CORRECT_LOAD_BY_BATCH)
#sendAllGeneratedTransactionsAllAtOnce(batch_size=20, input_directory=TRANSACTIONS_DIRECTORY_WITH_CORRECT_LOAD_BY_BATCH_AND_ACCOUNTS_ALLOCATION, delay_in_seconds=5, num_txs_to_send=2000)
#sendAllGeneratedTransactionsAllAtOnceWithMultithreadingRevisedByMe(batch_size=50, input_directory=TRANSACTIONS_DIRECTORY_WITH_CORRECT_LOAD_BY_BATCH_AND_ACCOUNTS_ALLOCATION, delay_in_seconds=5, num_txs_to_send=1000, num_txs_per_thread=50) #? 0: all transactions

#---------- myLastSender(input_directory=TRANSACTIONS_DIRECTORY_WITH_CORRECT_LOAD_BY_BATCH_AND_ACCOUNTS_ALLOCATION, delay_in_seconds=5, num_txs_to_send=4000, num_txs_per_batch=50)
"""myLastSenderWithBarrier(input_directory=TRANSACTIONS_DIRECTORY_WITH_CORRECT_LOAD_BY_BATCH_AND_ACCOUNTS_ALLOCATION, 
                        delay_in_seconds=5, 
                        num_txs_to_send=20000, 
                        num_txs_per_batch=50, 
                        with_AMTs=False)"""

"""myLastSenderWithBarrierFromCSV(input_directory=TRANSACTIONS_DIRECTORY_WITH_CORRECT_LOAD_BY_BATCH_AND_ACCOUNTS_ALLOCATION,
                        delay_in_seconds=5,
                        num_txs_to_send=200,
                        num_txs_per_batch=50,
                        with_AMTs=False)

time.sleep(120)

addStatisticsToCSV()"""

#plotDataFromStatistics(time_threshold=250, field_to_group_by='end_timestamp') #? PREVIOUS: field_to_group_by='timestamp'


#plotData(field_to_group_by='end_timestamp') #? PREVIOUS: field_to_group_by='timestamp'


#printStatistics()

#generateAccountsInfoJsonFile()



def snapshotElasticsearchData(experiment_folder):
    # Initialize Elasticsearch client
    es = Elasticsearch([{'host': 'localhost', 'port': 9200, 'scheme': 'http'}])

    # Specify the index and query to retrieve data
    index = 'transactions'
    query = {
        "query": {
            "match_all": {}  # Retrieve all documents
        }
    }

    # Initialize scroll
    scroll = '5m'  # Scroll window time (e.g., '5m' for 5 minutes)
    scroll_id = None

    # Open file for writing
    output_file = experiment_folder + '/elasticsearch_data.json'
    with open(output_file, 'w') as f:
        # Scroll through the documents
        while True:
            # Execute search request
            if scroll_id:
                response = es.scroll(scroll_id=scroll_id, scroll=scroll)
            else:
                response = es.search(index=index, body=query, scroll=scroll)

            # Break if no more documents
            if not response['hits']['hits']:
                break

            # Extract and write documents to file
            for hit in response['hits']['hits']:
                f.write(f"{hit['_source']}\n")

            # Update scroll_id for next iteration
            scroll_id = response['_scroll_id']

    # Clear scroll
    es.clear_scroll(scroll_id=scroll_id)



def copyLocalnetLogsToExperimentFolder(experiment_folder):
    # Specify the path to the localnet directory
    localnet_path = '/home/valentina/Thesis/Localnets/localnet'

    destination_folder = experiment_folder + "/localnet_logs"

    # Check if the folder exists
    if not os.path.exists(destination_folder):
        # Create the folder if it doesn't exist
        os.makedirs(destination_folder)

    # Iterate through each validator folder
    for validator_folder in os.listdir(localnet_path):
        validator_path = os.path.join(localnet_path, validator_folder)
        
        # Check if the validator folder exists and is a directory
        if os.path.isdir(validator_path):
            # Construct the path to the log files directory
            logs_path = os.path.join(validator_path, 'logs')
            
            # Check if the logs directory exists
            if os.path.exists(logs_path):
                # Iterate through each file in the logs directory
                for log_file in os.listdir(logs_path):
                    # Check if the file is a .log file
                    if log_file.endswith('.log'):
                        # Construct the path to the log file
                        log_file_path = os.path.join(logs_path, log_file)
                        
                        # Extract the validator name from the parent directory name
                        validator_name = validator_folder
                        
                        # Construct the new filename (validatorXX.log)
                        new_filename = f"{validator_name}.log"
                        
                        # Construct the destination path
                        destination_path = os.path.join(destination_folder, new_filename)
                        
                        # Copy the log file to the destination folder with the new filename
                        shutil.copy(log_file_path, destination_path)



def run_block_capacity_experiment(block_capacity, 
                                  test_num, 
                                  num_txs_per_batch, 
                                  num_total_txs, 
                                  num_txs_threshold_for_account_allocation, 
                                  with_cross_shard_probability,
                                  hot_sender_probability,
                                  hot_accounts_change_threshold
                                ):
    
    # Get the directory of the script
    script_directory = os.path.dirname(os.path.realpath(__file__))
    experiment_folder = os.path.join(script_directory, "Experiments_Setup/Block_Capacity_Test/" + str(block_capacity) + "/" + str(test_num))
    
    
    # Check if the directory exists
    if os.path.exists(experiment_folder):
        # Directory already exists
        user_input = input(f"The directory {experiment_folder} already exists. Do you want to overwrite its content? (y/n): ")
        if user_input.lower() != 'y':
            # If the user doesn't want to overwrite, exit
            print("Directory creation cancelled.")
            exit()
        else:
            # If the user wants to overwrite, remove the existing directory
            shutil.rmtree(experiment_folder)

    # Create the directory
    os.makedirs(experiment_folder)
    print("Directory created successfully.")
    
    
    # Start the experiment
    generateTransactionsWithCorrectLoadInBatchesWithAccountAllocationsOnCSV(num_txs_per_batch=num_txs_per_batch, 
                                                                        num_total_txs=num_total_txs, 
                                                                        num_txs_threshold_for_account_allocation=num_txs_threshold_for_account_allocation, 
                                                                        output_dir=experiment_folder, 
                                                                        with_cross_shard_probability=with_cross_shard_probability,
                                                                        hot_sender_probability=hot_sender_probability,
                                                                        hot_accounts_change_threshold=hot_accounts_change_threshold)

    myLastSenderWithBarrierFromCSV(
                        delay_in_seconds=5,
                        num_txs_to_send=num_total_txs,
                        num_txs_per_batch=num_txs_per_batch,
                        with_AMTs=False,
                        experiment_folder=experiment_folder)

    time.sleep(120)

    addStatisticsToCSV(experiment_folder)

    copyLocalnetLogsToExperimentFolder(experiment_folder)

    #snapshotElasticsearchData(experiment_folder)

    plotDataFromStatistics(time_threshold=250, field_to_group_by='end_timestamp', experiment_folder=experiment_folder, cross_shard_only=False, redrawn=False)
    plotDataFromStatistics(time_threshold=250, field_to_group_by='timestamp', experiment_folder=experiment_folder, cross_shard_only=False, redrawn=False)


    plotData(field_to_group_by='end_timestamp', experiment_folder=experiment_folder) #? PREVIOUS: field_to_group_by='timestamp'
    plotData(field_to_group_by='timestamp', experiment_folder=experiment_folder) #? PREVIOUS: field_to_group_by='timestamp'



def run_load_distribution_experiment(block_capacity, 
                                  test_num, 
                                  num_txs_per_batch, 
                                  num_total_txs, 
                                  num_txs_threshold_for_account_allocation, 
                                  with_cross_shard_probability,
                                  hot_sender_probability,
                                  hot_accounts_change_threshold,
                                  with_AMTs
                                ):
    
    # Get the directory of the script
    script_directory = os.path.dirname(os.path.realpath(__file__))
    load_distribution_hot = str(int(hot_sender_probability * 100))
    load_distribution_light = str(int((1-hot_sender_probability) * 100))
    experiment_folder = os.path.join(script_directory, f"Experiments_Setup/Load_Distribution_Test/{load_distribution_hot}_{load_distribution_light}/{str(test_num)}")
    
    
    # Check if the directory exists
    if os.path.exists(experiment_folder):
        # Directory already exists
        user_input = input(f"The directory {experiment_folder} already exists. Do you want to overwrite its content? (y/n): ")
        if user_input.lower() != 'y':
            # If the user doesn't want to overwrite, exit
            print("Directory creation cancelled.")
            exit()
        else:
            # If the user wants to overwrite, remove the existing directory
            shutil.rmtree(experiment_folder)

    # Create the directory
    os.makedirs(experiment_folder)
    print("Directory created successfully.")


    # Save parameters to a JSON file
    params = {
        'block_capacity': block_capacity,
        'test_num': test_num,
        'num_txs_per_batch': num_txs_per_batch,
        'num_total_txs': num_total_txs,
        'num_txs_threshold_for_account_allocation': num_txs_threshold_for_account_allocation,
        'with_cross_shard_probability': with_cross_shard_probability,
        'hot_sender_probability': hot_sender_probability,
        'hot_accounts_change_threshold': hot_accounts_change_threshold,
        'with_AMTs': with_AMTs
    }

    params_file = os.path.join(experiment_folder, 'experiment_params.json')
    with open(params_file, 'w') as f:
        json.dump(params, f, indent=4)
    print("Parameters saved to experiment_params.json")

    
    
    # Start the experiment
    generateTransactionsWithCorrectLoadInBatchesWithAccountAllocationsOnCSV(num_txs_per_batch=num_txs_per_batch, 
                                                                        num_total_txs=num_total_txs,
                                                                        num_txs_threshold_for_account_allocation=num_txs_threshold_for_account_allocation,
                                                                        output_dir=experiment_folder,
                                                                        with_cross_shard_probability=with_cross_shard_probability,
                                                                        hot_sender_probability=hot_sender_probability,
                                                                        hot_accounts_change_threshold=hot_accounts_change_threshold)

    myLastSenderWithBarrierFromCSV(
                        delay_in_seconds=5,
                        num_txs_to_send=num_total_txs,
                        num_txs_per_batch=num_txs_per_batch,
                        with_AMTs=with_AMTs,
                        experiment_folder=experiment_folder)

    time.sleep(240)

    addStatisticsToCSV(experiment_folder)

    copyLocalnetLogsToExperimentFolder(experiment_folder)

    #snapshotElasticsearchData(experiment_folder)

    plotDataFromStatistics(time_threshold=250, field_to_group_by='end_timestamp', experiment_folder=experiment_folder, cross_shard_only=False, redrawn=False)
    plotDataFromStatistics(time_threshold=250, field_to_group_by='timestamp', experiment_folder=experiment_folder, cross_shard_only=False, redrawn=False)


    plotData(field_to_group_by='end_timestamp', experiment_folder=experiment_folder) 
    plotData(field_to_group_by='timestamp', experiment_folder=experiment_folder) 



def str2bool(v):
    if isinstance(v, bool):
        return v
    if v.lower() in ('yes', 'true', 't', 'y', '1'):
        return True
    elif v.lower() in ('no', 'false', 'f', 'n', '0'):
        return False
    else:
        raise argparse.ArgumentTypeError('Boolean value expected.')



def main():
    # Creazione del parser degli argomenti
    parser = argparse.ArgumentParser(description="Esempio di utilizzo delle funzioni con argomenti")
    
    # Aggiunta degli argomenti per la selezione della funzione
    parser.add_argument("function_name", choices=["block_capacity_experiment", "load_distribution_experiment", "function3"], help="Nome della funzione da eseguire")

    # Aggiunta degli argomenti specifici per la funzione 3
    parser.add_argument("--block_capacity", type=int, help="current block capacity")
    parser.add_argument("--test_num", type=int, help="number of the test repetition")
    parser.add_argument("--num_txs_per_batch", type=int, help="number of transactions to send/generate at a time (in batches)")
    parser.add_argument("--num_total_txs", type=int, help="number of total transactions to send/generate")
    parser.add_argument("--num_txs_threshold_for_account_allocation", type=int, help="after how many txs trigger the account allocation")
    parser.add_argument("--with_cross_shard_probability", type=str2bool, help="whether to generate txs based on some cross-shard probability")
    parser.add_argument("--hot_sender_probability", type=float, help="probability that the sender is a hot account")
    parser.add_argument("--hot_accounts_change_threshold", type=int, help="num of txs after which change the set of hot accounts")
    parser.add_argument("--num_user_accounts", type=int, help="num of user accounts in the system")
    parser.add_argument("--initial_shard_for_hot_accounts", type=int, help="id of initial shard for hot accounts")
    parser.add_argument("--with_AMTs", type=str2bool, help="whether to generate account migrations or not")
    
    # Parsing degli argomenti della riga di comando
    args = parser.parse_args()
    print(args)

    # Update accounts_info and hot_accounts
    generateAccountsInfoJsonFile()
    num_hot_accounts = math.ceil(len(accounts_info) * 0.02)
    updateHotAccounts(shard_of_hot_accounts=args.initial_shard_for_hot_accounts, num_hot_accounts=num_hot_accounts)

    
    # Esecuzione della funzione corrispondente
    if args.function_name == "block_capacity_experiment":
        run_block_capacity_experiment(block_capacity=args.block_capacity,
                                        test_num=args.test_num,
                                        num_txs_per_batch=args.num_txs_per_batch,
                                        num_total_txs=args.num_total_txs,
                                        num_txs_threshold_for_account_allocation=args.num_txs_threshold_for_account_allocation,
                                        with_cross_shard_probability=args.with_cross_shard_probability,
                                        hot_sender_probability=args.hot_sender_probability,
                                        hot_accounts_change_threshold=args.hot_accounts_change_threshold,
                                      )
    elif args.function_name == "load_distribution_experiment":
        run_load_distribution_experiment(block_capacity=args.block_capacity,
                                        test_num=args.test_num,
                                        num_txs_per_batch=args.num_txs_per_batch,
                                        num_total_txs=args.num_total_txs,
                                        num_txs_threshold_for_account_allocation=args.num_txs_threshold_for_account_allocation,
                                        with_cross_shard_probability=args.with_cross_shard_probability,
                                        hot_sender_probability=args.hot_sender_probability,
                                        hot_accounts_change_threshold=args.hot_accounts_change_threshold,
                                        with_AMTs=args.with_AMTs
                                      )
    """elif args.function_name == "function3":
        function3(args.arg1, args.arg2)"""



def redraw_plots(experiment_folder, should_add_statistics_from_elastic, cross_shard_only, redrawn):

    if should_add_statistics_from_elastic:
        addStatisticsToCSV(experiment_folder)

    #snapshotElasticsearchData(experiment_folder)

    plotDataFromStatistics(time_threshold=250, field_to_group_by='end_timestamp', experiment_folder=experiment_folder, cross_shard_only=cross_shard_only, redrawn=redrawn)
    plotDataFromStatistics(time_threshold=250, field_to_group_by='timestamp', experiment_folder=experiment_folder, cross_shard_only=cross_shard_only, redrawn=redrawn)


    plotData(field_to_group_by='end_timestamp', experiment_folder=experiment_folder)
    plotData(field_to_group_by='timestamp', experiment_folder=experiment_folder)



def draw_aggregated_plot(experiment_folder):
    #plotAggregatedDataFromStatistics(time_threshold=250, field_to_group_by='end_timestamp', experiment_folder=experiment_folder)
    #plotAggregatedDataFromStatistics(time_threshold=250, field_to_group_by='timestamp', experiment_folder=experiment_folder)

    aggregatedData = aggregateDataFromStatistics(time_threshold=250, field_to_group_by='timestamp', experiment_folder=experiment_folder)
    plotAggregatedData(aggregated_data=aggregatedData, field_to_group_by='timestamp', experiment_folder=experiment_folder)




def aggregateCSVByMean(experiment_folder):
    # Initialize lists to store DataFrames for each CSV file
    dfs = []

    # Iterate through each folder
    for folder in os.listdir(experiment_folder):
        folder_path = os.path.join(experiment_folder, folder)
        if os.path.isdir(folder_path):
            # Check for CSV files named 'OUTPUT_CSV_WITH_STATISTICS.csv'
            csv_file_path = os.path.join(folder_path, NORMALIZED_OUTPUT_CSV_WITH_STATISTICS)
            if os.path.isfile(csv_file_path):
                # Read the CSV file into a DataFrame and append to the list
                df = pd.read_csv(csv_file_path)
                dfs.append(df)

    # Check if any CSV files were found
    if not dfs:
        print("No CSV files found.")
        return

    # Initialize lists to store mean values
    mean_timestamps = []
    mean_end_timestamps = []

    # Iterate over the rows of the CSV files simultaneously
    for i in range(len(dfs[0])):  # Assuming all CSV files have the same number of rows
        # Initialize variables to store sum of timestamps and end_timestamps
        sum_timestamp = 0
        sum_end_timestamp = 0
        count = 0

        # Iterate over each DataFrame and compute sum of timestamps and end_timestamps
        for df in dfs:
            row = df.iloc[i]
            sum_timestamp += row['normalized_timestamp']
            sum_end_timestamp += row['normalized_end_timestamp']
            count += 1

        # Compute mean values for timestamps and end_timestamps
        mean_timestamp = round(sum_timestamp / count)
        mean_end_timestamp = round(sum_end_timestamp / count)

        # Append mean values to lists
        mean_timestamps.append(mean_timestamp)
        mean_end_timestamps.append(mean_end_timestamp)

    # Create a DataFrame with mean values
    new_df = pd.DataFrame({'normalized_timestamp': mean_timestamps, 'normalized_end_timestamp': mean_end_timestamps})

    # Select all columns except 'normalized_timestamp' and 'normalized_end_timestamp' from one of the original DataFrames
    original_columns_df = dfs[0].drop(columns=['normalized_timestamp', 'normalized_end_timestamp'])

    # Concatenate the original columns with the new DataFrame containing mean values
    combined_df = pd.concat([original_columns_df, new_df], axis=1)

    # Calculate 'timestamp_difference' column
    combined_df['timestamp_difference'] = combined_df['normalized_end_timestamp'] - combined_df['normalized_timestamp']


    # Write the new DataFrame to a new CSV file
    combined_df.to_csv(f"{experiment_folder}/combined.csv", index=False)



def normalize_csv_files(experiment_folder):
    # Iterate through each folder in the experiment_folder
    for root, dirs, files in os.walk(experiment_folder):
        for file in files:
            # Check if the file is the desired CSV file
            if file == OUTPUT_CSV_WITH_STATISTICS:
                # Construct the full path to the CSV file
                csv_file_path = os.path.join(root, file)
                
                # Read the CSV file into a pandas DataFrame
                df = pd.read_csv(csv_file_path)
                
                # Find the minimum timestamp and end_timestamp
                min_timestamp = df[['timestamp', 'end_timestamp']].min().min()
                
                # Normalize the timestamps
                df['normalized_timestamp'] = round(df['timestamp'] - min_timestamp)
                df['normalized_end_timestamp'] = round(df['end_timestamp'] - min_timestamp)
                
                # Recompute the values of timestamp_difference
                df['timestamp_difference'] = round(df['normalized_end_timestamp'] - df['normalized_timestamp'])
                
                # Construct the path for the normalized CSV file
                normalized_csv_file_path = os.path.join(root, NORMALIZED_OUTPUT_CSV_WITH_STATISTICS)
                
                # Save the DataFrame to a normalized CSV file
                df.to_csv(normalized_csv_file_path, index=False)
                print(f"Normalized CSV file saved: {normalized_csv_file_path}")



def addGroupToCSVFile(group_size, csv_file):
    # Read the CSV file into a DataFrame
    df = pd.read_csv(csv_file)

    # Create a column to identify groups
    df['group'] = df.index // group_size

    # Group the DataFrame by the 'group' column and calculate the mean
    #grouped_df = df.groupby('group').mean().reset_index(drop=True)

    # Optionally, remove the 'group' column if it's not needed
    # grouped_df = grouped_df.drop(columns=['group'])

    # Get the directory path of the original file
    dir_path = os.path.dirname(csv_file)

    # Construct the path for the new file
    new_file_path = os.path.join(dir_path, 'grouped_means.csv')

    # Save the new DataFrame to a new CSV file
    df.to_csv(new_file_path, index=False)



def get_AMT_timestamp(experiment_folder):
    results = []
    
    # Define the path to the localnet_logs folder
    logs_folder = os.path.join(experiment_folder, "localnet_logs")
    
    # Define the regex pattern to match the desired log line and extract the timestamp
    pattern = re.compile(r"DEBUG\[(\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2}\.\d{3})\].*ACCOUNT ALLOCATION IS NOT EMPTY: PERFORMING ACCOUNT MIGRATIONS")
    
    # Iterate over all files in the localnet_logs folder
    for root, dirs, files in os.walk(logs_folder):
        for file in files:
            if file.endswith(".log") and "observer" not in file and file != "proxy.log":  # Exclude observer files and proxy.log
                file_path = os.path.join(root, file)
                
                # Open and read the log file
                with open(file_path, 'r', encoding='utf-8', errors='ignore') as f:
                    for line in f:
                        match = pattern.match(line)
                        if match:
                            # Extract the timestamp from the matched line
                            timestamp_str = match.group(1)
                            # Convert the timestamp to epoch format
                            timestamp_dt = datetime.strptime(timestamp_str, '%Y-%m-%d %H:%M:%S.%f')
                            epoch_time = int(timestamp_dt.timestamp())
                            # Append the results as a dictionary
                            results.append({
                                "filename": file,
                                "timestamp_str": timestamp_str,
                                "epoch_time": epoch_time
                            })
    
    print(results)
    return results



# Esegui la funzione principale se lo script è eseguito direttamente
if __name__ == "__main__":
    main()
    #generateAccountsInfoJsonFile()
    #get_AMT_timestamp(experiment_folder="/home/valentina/Thesis/MX-Thesis-New/final-experiments/Experiments_Setup/Load_Distribution_Test/90_9/10")
    #redraw_plots(experiment_folder="/home/valentina/Thesis/MX-Thesis-New/final-experiments/Experiments_Setup/Load_Distribution_Test/90_9/23", should_add_statistics_from_elastic=True, cross_shard_only=False, redrawn=False)
    #get_AMT_timestamp("/home/valentina/Thesis/MX-Thesis-New/final-experiments/Experiments_Setup/Load_Distribution_Test/90_9/22")


#redraw_plots(experiment_folder="/home/valentina/Thesis/MX-Thesis-New/final-experiments/Experiments_Setup/Load_Distribution_Test/90_9/11")
#draw_aggregated_plot(experiment_folder="/home/valentina/Thesis/MX-Thesis-New/final-experiments/Experiments_Setup/Block_Capacity_Test/100")
#aggregateCSVByMean(experiment_folder="/home/valentina/Thesis/MX-Thesis-New/final-experiments/Experiments_Setup/Block_Capacity_Test/100")
#normalize_csv_files(experiment_folder="/home/valentina/Thesis/MX-Thesis-New/final-experiments/Experiments_Setup/Block_Capacity_Test/100")
#plotAggregatedDataFromStatistics(time_threshold=250, field_to_group_by='group', experiment_folder="/home/valentina/Thesis/MX-Thesis-New/final-experiments/Experiments_Setup/Block_Capacity_Test/100")
#addGroupToCSVFile(group_size=50, csv_file="/home/valentina/Thesis/MX-Thesis-New/final-experiments/Experiments_Setup/Block_Capacity_Test/100/combined.csv")


#generateAccountsInfoJsonFile()