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


# Get the directory of the script
script_directory = os.path.dirname(os.path.realpath(__file__))

# Define relative paths
TRANSACTIONS_DIRECTORY = os.path.join(script_directory, "generated_transactions/")
TRANSACTIONS_DIRECTORY_WITH_CORRECT_LOAD = os.path.join(script_directory, "generated_transactions_with_correct_load/")
TRANSACTIONS_DIRECTORY_WITH_CORRECT_LOAD_BY_BATCH = os.path.join(script_directory, "generated_transactions_with_correct_load_by_batch/")
TRANSACTIONS_DIRECTORY_WITH_CORRECT_LOAD_BY_BATCH_AND_ACCOUNTS_ALLOCATION = os.path.join(script_directory, "generated_transactions_with_correct_load_by_batch_and_accounts_allocation/")
OUTPUT_CSV = os.path.join(script_directory, "output.csv")
OUTPUT_CSV_WITH_REAL_EXECUTION_TIMESTAMP = os.path.join(script_directory, "output.csv")
OUTPUT_CSV_WITH_END_TIMESTAMP = os.path.join(script_directory, "output_with_end_timestamp.csv")
OUTPUT_CSV_WITH_STATISTICS = os.path.join(script_directory, "output_with_statistics.csv")
OUTPUT_CSV_WITH_TIMESTAMP_DIFFERENCE = os.path.join(script_directory, "output_with_timestamp_difference.csv")
GENERATED_TXS_STATISTICS_CSV = os.path.join(script_directory, "generated_txs_statistics.csv")
ACCOUNTS_INFO_JSON_PATH = os.path.join(script_directory, "accounts_info.json")


"""
TRANSACTIONS_DIRECTORY = "./generated_transactions/" #? crearla se non esiste
TRANSACTIONS_DIRECTORY_WITH_CORRECT_LOAD = "./generated_transactions_with_correct_load/" #? crearla se non esiste
TRANSACTIONS_DIRECTORY_WITH_CORRECT_LOAD_BY_BATCH = "./generated_transactions_with_correct_load_by_batch/" #? crearla se non esiste
OUTPUT_CSV = "./output.csv"
OUTPUT_CSV_WITH_END_TIMESTAMP = "./output_with_end_timestamp.csv"
OUTPUT_CSV_WITH_STATISTICS = "./output_with_statistics.csv"
OUTPUT_CSV_WITH_TIMESTAMP_DIFFERENCE = "./output_with_timestamp_difference.csv"
"""

hot_accounts = ["erd1qyu5wthldzr8wx5c9ucg8kjagg0jfs53s8nr3zpz3hypefsdd8ssycr6th", "erd1kyaqzaprcdnv4luvanah0gfxzzsnpaygsy6pytrexll2urtd05ts9vegu7", "erd18tudnj2z8vjh0339yu3vrkgzz2jpz8mjq0uhgnmklnap6z33qqeszq2yn4" ] #alice, dan, eve
hot_sender_probability = 0.9 #TODO modificare a 0.5
cross_shard_probability = 1

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


def run_shell_command_without_waiting(command_data, timestamp_of_generation):
    if command_data["command"] == "ACCOUNT_ALLOCATION":
        url = "http://localhost:10206/node/send-account-allocation"
        #generate_account_migration_transactions(account_allocation_payload)
        # Generate Curl command
        command_to_run = [
            'curl',            # The curl command
            '-X', 'POST',       # HTTP method (GET, POST, etc.)
            url,               # URL to send the request to
            '-H', 'Content-Type: application/json',  # Headers if needed
            '-d', command_data["accountAllocationPayload"],  # Data to send in the request body, converting it to JSON string
        ]
        command_to_run = f'curl -X POST {url} -H "Content-Type: application/json" --data-raw \'{command_data["accountAllocationPayload"]}\''

    else:
        command_to_run = command_data["command"]
    # Start the process without waiting for it to complete
    process = subprocess.Popen(command_to_run, shell=True, text=True, stdout=subprocess.PIPE, stderr=subprocess.PIPE)
    stdout, stderr = process.communicate()  # Capture stdout and stderr
    return process, stdout, stderr, command_data["infile"]
    # Code here will continue executing immediately without waiting for the process to finish

    # Optionally, you can wait for the process to complete using process.wait()
    # process.wait()




#! COMMAND IS THE COMMAND STRING, WHILE ABOVE I HAVE COMMAND_DATA WITH THE INFO ABOUT COMMAND AND THE CORRESPONDING INFILE
def run_shell_command_without_waiting_OLD(command):
    # Start the process without waiting for it to complete
    process = subprocess.Popen(command, shell=True, text=True, stdout=subprocess.PIPE, stderr=subprocess.PIPE)
    stdout, stderr = process.communicate()  # Capture stdout and stderr
    return process, stdout, stderr
    # Code here will continue executing immediately without waiting for the process to finish

    # Optionally, you can wait for the process to complete using process.wait()
    # process.wait()




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


def create_send_tx_command(tx_id="000000001"):
    infile = f"./transaction_{tx_id}"
    
    command = f"mxpy tx send \
            --proxy=http://localhost:7950 \
            --infile={infile};"

    return command


def sendTransactionFromFile(tx_id):
    
    infile = f"./transaction_{tx_id}"
    
    
    command_to_run = f"mxpy tx send \
            --proxy=http://localhost:7950 \
            --infile={infile};"
    

    # Open the JSON file
    with open(infile) as f:
        # Load JSON data
        data = json.load(f)

    # Access the value of the "sender" attribute
    sender_addr = data['emittedTransaction']['sender']
        

    run_shell_command(command_to_run, sender_addr)



def sendBatchOfTransactions(file_names, batch_size, i, input_directory):
    timestamp = datetime.now()
    print(f"Sending new batch of transactions (batch_size = {batch_size}, timestamp = {timestamp})")
    
    processes = []
    txHashes = []
    for file_name in file_names:
        timestamp = datetime.now()
        print(timestamp)
        
        infile = input_directory+file_name
        command_to_run = f"mxpy tx send \
                --proxy=http://localhost:7950 \
                --infile={infile};"

        """
        # Open the JSON file
        with open(infile, 'r', encoding='utf-8') as f:
            print("Reading file: " + infile)
            # Load JSON data
            data = json.load(f)

        # Access the value of the "sender" attribute
        #sender_addr = data['emittedTransaction']['sender']
        txHashes.append(data['emittedTransactionHash'])
        """
            
        #run_shell_command(command_to_run, sender_addr=None)
        process, stdout, stderr = run_shell_command_without_waiting(command_to_run)
        processes.append((process, stdout, stderr, infile))
    
    timestamp_start_closing = datetime.now()
    print(f"Closing processes... (timestamp: {timestamp_start_closing})")
    for process, stdout, stderr, infile in processes:
        print("--- PROCESSING infile: " + infile)
        return_code = process.wait()
        if return_code == 0:
            # Process was successful
            output_data = json.loads(stdout) # Parse JSON output
            txHash = output_data.get('emittedTransactionHash')
            txHashes.append(txHash)  # Add txHash to the list
        else:
            # Process failed
            print("Error:", stderr)
    timestamp_end_closing = datetime.now()
    print(f"Finished closing processes. Moving to the next batch of transactions... (timestamp: {timestamp_end_closing})")    
    
    #saveBatchToCSVAsList(txHashes, timestamp_start_closing, i)
    saveBatchToCSVAsSeparateRow(txHashes, timestamp_start_closing, i)
    
    
def sendBatchOfTransactionsAllAtOnce(file_names, batch_size, i, input_directory):
    #def p: #TODO: (ho messo un errore) -> magari le operazioni lente sono quelle su disco (si devono aprire file multipli
    #TODO: quindi magari si potrebbe provare un approccio in cui mi butto tutti i comandi (tanto sono stringhe) in un csv e provo facendo run_commands tenendo aperto un unico file, magari lo leggo con pandas (?))
    timestamp = datetime.now()
    print(f"Sending new batch of transactions (batch_size = {batch_size}, timestamp = {timestamp})")
    
    processes = []
    txHashes = []
    for file_name in file_names:
        timestamp = datetime.now()
        print(timestamp)
        
        infile = input_directory+file_name
        command_to_run = f"mxpy tx send \
                --proxy=http://localhost:7950 \
                --infile={infile};"

        """
        # Open the JSON file
        with open(infile, 'r', encoding='utf-8') as f:
            print("Reading file: " + infile)
            # Load JSON data
            data = json.load(f)

        # Access the value of the "sender" attribute
        #sender_addr = data['emittedTransaction']['sender']
        txHashes.append(data['emittedTransactionHash'])
        """
            
        #run_shell_command(command_to_run, sender_addr=None)
        process, stdout, stderr = run_shell_command_without_waiting(command_to_run, timestamp)
        processes.append((process, stdout, stderr, infile))

    return processes    
    
    timestamp_start_closing = datetime.now()
    print(f"Closing processes... (timestamp: {timestamp_start_closing})")
    for process, stdout, stderr, infile in processes:
        print("--- PROCESSING infile: " + infile)
        return_code = process.wait()
        if return_code == 0:
            # Process was successful
            output_data = json.loads(stdout) # Parse JSON output
            txHash = output_data.get('emittedTransactionHash')
            txHashes.append(txHash)  # Add txHash to the list
        else:
            # Process failed
            print("Error:", stderr)
    timestamp_end_closing = datetime.now()
    print(f"Finished closing processes. Moving to the next batch of transactions... (timestamp: {timestamp_end_closing})")    
    
    #saveBatchToCSVAsList(txHashes, timestamp_start_closing, i)
    saveBatchToCSVAsSeparateRow(txHashes, timestamp_start_closing, i)
    


def saveBatchToCSVAsList(batchTxHashes, timestamp, i):
    # Write to CSV file
    with open(OUTPUT_CSV, mode='a', newline='') as file:
        writer = csv.writer(file)
        writer.writerow([timestamp.timestamp(), batchTxHashes]) #? faccio timestamp.timestamp() per avere il timestamp in epoch format invece che come datetime


def saveBatchToCSVAsSeparateRow(batchTxHashes, timestamp, i):
    # Write to CSV file
    with open(OUTPUT_CSV, mode='a', newline='') as file:
        writer = csv.writer(file)
        for txHash in batchTxHashes:
            writer.writerow([timestamp.timestamp(), txHash]) #? faccio timestamp.timestamp() per avere il timestamp in epoch format invece che come datetime



def isFirstBatch(i):
    return i == 0


def readCSVFile():
    # Read CSV file
    txhashes_by_timestamp = defaultdict(list)

    with open(OUTPUT_CSV, mode='r') as file:
        reader = csv.reader(file)
        next(reader)  # Skip header
        for row in reader:
            timestamp, tx_hash = row
            txhashes_by_timestamp[timestamp].append(tx_hash)

    # Display the grouped transaction hashes
    for timestamp, tx_hashes in txhashes_by_timestamp.items():
        print(f"Timestamp: {timestamp}, TxHashes: {tx_hashes}")    



def pick_sender_and_receiver(hash_map, seed):
    # Extract keys from the hash map
    keys_for_sender = list(hash_map.keys())
    keys_for_receiver = []
    
    # Set the random seed
    random.seed(seed)
    
    while True:
        # Randomly select the first key
        first_key = random.choice(keys_for_sender)
        
        # Remove the first key from the list
        keys_for_receiver = keys_for_sender[:] # ? without [:] it does NOT create a copy, but keys_for_receiver will reference the original list, so we would remove elements from the original list, NOT from a copy!
        keys_for_receiver.remove(first_key)
        
        

        # Randomly select the second key from the remaining keys
        second_key = random.choice(keys_for_receiver)
        
        # Yield the pair of keys
        yield first_key, second_key

def pick_sender_and_receiver_with_correct_load(all_accounts, seed):
    # Extract keys from the hash map
    all_accounts_keys = list(all_accounts.keys())
    light_accounts = list(all_accounts.keys())
    for account in hot_accounts:
        light_accounts.remove(account)
    keys_for_receiver = []
    
    # Set the random seed
    random.seed(seed)
    
    while True:
        #? PICK SENDER ----------
        if random.random() < hot_sender_probability:
            first_key = random.choice(hot_accounts)
        else:
            # Randomly select the first key
            first_key = random.choice(light_accounts)
        
        #? PICK RECEIVER (based on picked sender and cross-shard probability) --------
        sender_shard = all_accounts[first_key]["shard"]
        

        # Extract same shard accounts (excluding the given key)
        same_shard_account_keys = [key for key, acc in accounts_info.items() if key != first_key and acc["shard"] == sender_shard]
        # Extract different shard account keys
        different_shard_account_keys = [key for key, acc in accounts_info.items() if key != first_key and acc["shard"] != sender_shard]
        
        # Randomly choose between same shard and different shard accounts based on cross_shard_probability
        if random.random() < cross_shard_probability:
            # Choose from different shard accounts
            account_keys = different_shard_account_keys
        else:
            # Choose from same shard accounts
            account_keys = same_shard_account_keys
        
        # If the chosen account set is empty, return None
        if not account_keys:
            return None
        
        # Randomly choose a key from the selected set
        second_key = random.choice(account_keys)

        
        # Yield the pair of keys
        yield first_key, second_key



def pick_sender_and_receiver_with_correct_load_by_batch(all_accounts, seed, num_txs_per_batch):
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
        first_key = random.choice(hot_accounts)
        second_key = pick_receiver_based_on_cross_shard_probability(all_accounts, first_key)
        batch_sender_receiver_pairs.append((first_key, second_key))

    # Generate sender-receiver pairs for light account senders
    for _ in range(num_txs_per_batch - num_hot_account_senders):
        first_key = random.choice(light_accounts)
        second_key = pick_receiver_based_on_cross_shard_probability(all_accounts, first_key)
        batch_sender_receiver_pairs.append((first_key, second_key))

    # Shuffle the pairs to mix hot and light account senders
    #random.shuffle(batch_sender_receiver_pairs, random.Random(seed))
    random.shuffle(batch_sender_receiver_pairs)

    # Return the list of sender-receiver pairs for this batch
    return batch_sender_receiver_pairs



def pick_sender_and_receiver_with_correct_load_by_batch_and_account_allocations(all_accounts, seed, num_txs_per_batch, with_cross_shard_probability):
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

    writeGeneratedTxsStatisticsCSV(batch_sender_receiver_pairs)

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



def pick_receiver(hash_map, seed):
    # Extract keys from the hash map
    keys = list(hash_map.keys())
    
    # Set the random seed
    random.seed(seed)
    
    while True:
        # Randomly select the first key
        first_key = random.choice(keys)
           
        # Yield the pair of keys
        yield first_key


def generateSingleTransaction():
        sender_addr = "erd1qyu5wthldzr8wx5c9ucg8kjagg0jfs53s8nr3zpz3hypefsdd8ssycr6th"   
        
        command_to_run = create_new_tx_command(
                nonce=6,
                gas_limit=70000,
                receiver_address="erd1spyavw0956vq68xj8y4tenjpq2wd5a9p2c6j8gsz7ztyrnpxrruqzu66jx",
                sender_name="alice",
                tx_id = 2,
                output_directory = TRANSACTIONS_DIRECTORY
        )
        run_shell_command(command_to_run, sender_addr)



def generateRandomTransactions(num_txs):
    print("Num of accounts: " + str(len(accounts_info)))
    print("0.02x of account is: " + str(len(accounts_info) * 0.02))
    seed_value = 42

    global_txs_id = 0

    generator = pick_sender_and_receiver(accounts_info, seed_value)

    # Generate pairs of keys
    for _ in range(num_txs):
        sender_addr, receiver_addr = next(generator)

        print("----- GENERATING TX FROM " + accounts_info[sender_addr]["username"] + " TO " + accounts_info[receiver_addr]["username"] + " -----")
        command_to_run = create_new_tx_command(
                nonce=accounts_info[sender_addr]["nonce"],
                gas_limit=70000,
                receiver_address=receiver_addr,
                sender_name=accounts_info[sender_addr]["username"],
                tx_id = global_txs_id,
                output_directory = TRANSACTIONS_DIRECTORY
        )
        run_shell_command(command_to_run, sender_addr)
        global_txs_id += 1
        #time.sleep(0.1)


def generateTransactionsWithCorrectLoad(num_txs):
    print("Num of accounts: " + str(len(accounts_info)))
    print("0.02x of account is: " + str(len(accounts_info) * 0.02))
    seed_value = 42

    global_txs_id = 0

    generator = pick_sender_and_receiver_with_correct_load(accounts_info, seed_value)

    # Generate pairs of keys
    for _ in range(num_txs):
        sender_addr, receiver_addr = next(generator)

        print("----- GENERATING TX FROM " + accounts_info[sender_addr]["username"] + " TO " + accounts_info[receiver_addr]["username"] + " -----")
        command_to_run = create_new_tx_command(
                nonce=accounts_info[sender_addr]["nonce"],
                gas_limit=70000,
                receiver_address=receiver_addr,
                sender_name=accounts_info[sender_addr]["username"],
                tx_id = global_txs_id,
                output_directory = TRANSACTIONS_DIRECTORY_WITH_CORRECT_LOAD
        )
        run_shell_command(command_to_run, sender_addr)
        global_txs_id += 1
        #time.sleep(0.1)


def generateTransactionsWithCorrectLoadInBatches(num_txs_per_batch, num_batches, output_dir):
    print("Num of accounts: " + str(len(accounts_info)))
    print("0.02x of account is: " + str(len(accounts_info) * 0.02))
    seed_value = 42

    global_txs_id = 0

    for batch in range(num_batches):
        generator = pick_sender_and_receiver_with_correct_load_by_batch(accounts_info, seed_value, num_txs_per_batch)
        batch_sender_receiver_pairs = list(generator)

        # Prepare a list to store all the commands to run
        #commands_to_run = []

        # Generate commands for each sender-receiver pair in the batch
        for sender_addr, receiver_addr in batch_sender_receiver_pairs:
            print("----- GENERATING TX FROM " + accounts_info[sender_addr]["username"] + " TO " + accounts_info[receiver_addr]["username"] + " -----")
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




def generateTransactionsWithCorrectLoadInBatchesWithAccountAllocations(num_txs_per_batch, num_total_txs, num_txs_threshold_for_account_allocation, output_dir, with_cross_shard_probability):
    print("Num of accounts: " + str(len(accounts_info)))
    print("0.02x of account is: " + str(len(accounts_info) * 0.02))
    seed_value = 42

    global global_txs_id

    num_txs_from_last_account_allocation = 0

    num_batches = math.ceil(num_total_txs / num_txs_per_batch)

    createGeneratedTxsStatisticsCSV()

    for batch in range(num_batches):
        generator = pick_sender_and_receiver_with_correct_load_by_batch_and_account_allocations(accounts_info, seed_value, num_txs_per_batch, with_cross_shard_probability)
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
            compute_next_account_allocation(num_txs_threshold_for_account_allocation, output_dir)
            num_txs_from_last_account_allocation = 0





def compute_next_account_allocation(num_txs_threshold, txs_dir):
    global current_accounts_allocation_id
    global global_txs_id
    starting_index = max(0, global_txs_id - num_txs_threshold)
    print(f"------ STARTING INDEX: {starting_index} -------")
    #ending_index = iteration * (num_txs_threshold)

    shard_loads, hot_accounts_load = getGeneratedTxsStatisticsFromCSV(starting_index, num_txs_threshold)

    #shard_loads_variance = computeShardLoadsVariance(shard_loads)
    account_migrations_list, initial_variance, final_variance = move_accounts_to_improve_variance(shard_loads, hot_accounts_load)

    if len(account_migrations_list) > 0:
        generate_account_allocation_json_file(account_migrations_list, initial_variance, final_variance, global_txs_id, txs_dir, current_accounts_allocation_id)
        current_accounts_allocation_id += 1


def generate_account_allocation_json_file(account_migrations_list, initial_variance, final_variance, tx_id, txs_dir, id):
    formatted_tx_id = "{:07d}".format(tx_id)
    # File path to save the JSON file
    outfile = f"{txs_dir}transaction_{formatted_tx_id}_account_allocation.json"

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



def move_accounts_to_improve_variance(shard_loads, hot_accounts_loads):
    # Sort hot_accounts_loads by load in descending order
    sorted_accounts = sorted(hot_accounts_loads.items(), key=lambda x: x[1]['load'], reverse=True)
    print(f"Sorted hot accounts: {sorted_accounts}" )
    
    # Find the shard with the minimum load
    min_load_shard = min(shard_loads, key=shard_loads.get)
    print(f"Shard with minimum load: {min_load_shard}" )

    
    # Initialize a list to store the tuples (account, source_shard, dest_shard)
    moves = []
    
    # Compute variance before moving any accounts
    initial_variance = computeShardLoadsVariance(shard_loads)
    current_variance = initial_variance
    
    for account, info in sorted_accounts:
        source_shard = info['shard']
        
        # Calculate shard loads after moving the account
        shard_loads_after_move = shard_loads.copy()
        shard_loads_after_move[source_shard] -= info['load']
        shard_loads_after_move[min_load_shard] += info['load']
        print(f"Shard loads after move: {shard_loads_after_move}")
        
        # Compute variance after moving the account
        new_variance = computeShardLoadsVariance(shard_loads_after_move)
        
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


def getGeneratedTxsStatisticsFromCSV(starting_index, num_txs_to_read):
    # Calculate the number of rows to skip
    skiprows = range(1, starting_index)  # Skip the header row
    print(f"--------------- SKIPPING ROWS FROM 1 TO {starting_index} --------------")

    # Read only the specified range of rows from the CSV file into a pandas DataFrame
    df = pd.read_csv(GENERATED_TXS_STATISTICS_CSV, skiprows=skiprows, nrows=num_txs_to_read)

    # Define the shard values
    shard_values = [0, 1, 2]

    # Initialize a dictionary to store the loads (tx count) for each shard
    shard_loads = {shard: 0 for shard in shard_values}

    # Loop through shard values
    for shard in shard_values:
        # Filter rows based on the sender shard value
        sender_shard_rows = df[df['sender_shard'] == shard]
        # Count the unique sender addresses for the current shard
        shard_load = len(sender_shard_rows)
        # Update the count in the dictionary
        shard_loads[shard] = shard_load

    # Print the load for each shard
    for shard, load in shard_loads.items():
        print(f"Number of txs (load) from shard {shard}: {load}")
    
    # Initialize a dictionary to store the number of transactions generated by each account in the subset
    hot_accounts_load = {account: 0 for account in hot_accounts}

    # Loop through the subset of accounts
    for account in hot_accounts:
        # Filter rows based on the sender address matching the account
        account_rows = df[df['sender'] == account]
        # Count the number of transactions generated by the current account
        transactions_count = len(account_rows)
        # Update the count in the dictionary
        hot_accounts_load[account] = {"load": transactions_count, "shard": accounts_info[account]["shard"]}
    
    # Print the number of transactions generated by each account in the subset
    for account, count in hot_accounts_load.items():
        print(f"Number of transactions generated by {account}: {count}")

    return shard_loads, hot_accounts_load

    

def createOutputCSV():
    # Open the file in write mode
    with open(OUTPUT_CSV, mode='w', newline='') as file:
        writer = csv.writer(file)
        writer.writerow(["timestamp", "txHashes"])  # Write header 

def createGeneratedTxsStatisticsCSV():
    # Open the file in write mode
    with open(GENERATED_TXS_STATISTICS_CSV, mode='w', newline='') as file:
        writer = csv.writer(file)
        writer.writerow(["sender", "receiver", "sender_shard", "receiver_shard", "cross_shard"])  # Write header 

# Function to query Elasticsearch for end_timestamp based on txHash
def get_end_timestamp(tx_hash):
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
        return end_timestamp
    else:
        return None


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


def addEndTimestampToCSV():
    # Read CSV file
    df = pd.read_csv(OUTPUT_CSV)

    # Add a new column 'end_timestamp' and populate it by querying Elasticsearch
    df['end_timestamp'] = df['txHashes'].apply(get_end_timestamp)


    # Write the updated dataframe to the new CSV file
    df.to_csv(OUTPUT_CSV_WITH_END_TIMESTAMP, index=False)

    print(f"Result saved to {OUTPUT_CSV_WITH_END_TIMESTAMP}")


def addStatisticsToCSV():
    # Read CSV file
    df = pd.read_csv(OUTPUT_CSV)

    # Add a new column 'end_timestamp' and populate it by querying Elasticsearch
    df[['end_timestamp', 'sender_shard', 'receiver_shard', 'sender', 'receiver', 'mini_block_hash']] = df['txHashes'].apply(get_statistics).apply(pd.Series)
    # Calculate the timestamp difference and add it as a new column
    df['timestamp_difference'] = df['end_timestamp'] - df['realTimestamp'] # Before: df['end_timestamp'] - df['timestamp']
    df['has_corresponding_AAT'] = df['txHashes'].apply(get_has_corresponding_AAT)
    df['is_affected_by_AAT'] = df['mini_block_hash'].apply(get_is_affected_by_AAT)

    # Write the updated dataframe to the new CSV file
    df.to_csv(OUTPUT_CSV_WITH_STATISTICS, index=False)

    print(f"Result saved to {OUTPUT_CSV_WITH_STATISTICS}")   


def addTimestampDifferenceToCSV():
    # Read CSV file
    df = pd.read_csv(OUTPUT_CSV_WITH_END_TIMESTAMP)

    # Calculate the timestamp difference and add it as a new column
    df['timestamp_difference'] = df['end_timestamp'] - df['timestamp']

    # Write the updated dataframe to the new CSV file
    df.to_csv(OUTPUT_CSV_WITH_TIMESTAMP_DIFFERENCE, index=False)

    print(f"Result saved to {OUTPUT_CSV_WITH_TIMESTAMP_DIFFERENCE}")    


def plotData():
    # Read the CSV file
    df = pd.read_csv(OUTPUT_CSV_WITH_STATISTICS)


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
    df['timestamp'] = df['timestamp'] - df['timestamp'].min()  # Normalize timestamps to start from 0

    # Group by timestamp and calculate the mean of timestamp_difference
    mean_timestamp_diff = df.groupby('timestamp')['timestamp_difference'].mean()

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
    plt.savefig('mean_timestamp_difference_seconds_plot.png')

    # Show the plot
    plt.show()

    # Save the mean timestamp difference to a new CSV file
    mean_timestamp_diff_df = mean_timestamp_diff.reset_index()
    mean_timestamp_diff_df.to_csv('mean_timestamp_difference_seconds.csv', index=False)


def plotDataFromStatistics(time_threshold):
    # Read the CSV file
    df = pd.read_csv(OUTPUT_CSV_WITH_STATISTICS)

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
    x_migration_start = migration_starts_at - df['timestamp'].min()
    #print(str(migration_starts_at) + " - " + str(df["timestamp"].min()))
    #print(x_migration_start)

    # Add vertical line at specified timestamp
    vertical_timestamp = 1713000131
    x_vertical_ts = vertical_timestamp - df['timestamp'].min()
    #print(str(vertical_timestamp) + " - " + str(df["timestamp"].min()))
    #print(x_vertical_ts)


    # Convert timestamp to seconds
    df['timestamp'] = df['timestamp'] - df['timestamp'].min()  # Normalize timestamps to start from 0

    # Separate data for each shard
    shard_0_data = df[df['sender_shard'] == 0]
    shard_1_data = df[df['sender_shard'] == 1]
    shard_2_data = df[df['sender_shard'] == 2]
    # Separate data for each shard with only cross-shard transactions
    """shard_0_data = df[(df['sender_shard'] == 0) & (df['sender_shard'] != df['receiver_shard'])]
    shard_1_data = df[(df['sender_shard'] == 1) & (df['sender_shard'] != df['receiver_shard'])]
    shard_2_data = df[(df['sender_shard'] == 2) & (df['sender_shard'] != df['receiver_shard'])]"""

    # Group by timestamp and calculate the mean of timestamp_difference for each shard
    mean_timestamp_diff_shard_0 = shard_0_data.groupby('timestamp')['timestamp_difference'].mean()
    mean_timestamp_diff_shard_1 = shard_1_data.groupby('timestamp')['timestamp_difference'].mean()
    mean_timestamp_diff_shard_2 = shard_2_data.groupby('timestamp')['timestamp_difference'].mean()

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
    plt.savefig('mean_timestamp_difference_shards_plot.png')

    # Show the plot
    plt.show()


def generate_account_migration_transactions(current_account_allocation_id):
    url = "http://localhost:10206/node/send-account-allocation"

    payload = {
        "id": current_account_allocation_id,
        "accountAllocation": accountsAllocationData[current_account_allocation_id]
    }

    # Send POST request
    response = requests.post(url, json=payload)

    # Check if request was successful (status code 200)
    if response.status_code == 200:
        print("--------REST API call successful--------")
        print(f"GENERATED ACCOUNT ALLOCATION WTIH ID {current_account_allocation_id}:  {accountsAllocationData[current_account_allocation_id]}")
    else:
        print(f"REST API call failed with status code {response.status_code}")



def sendAccountAllocation():
    url = "http://localhost:10206/node/send-account-allocation"

    payload = {
        "id": 0,
        "accountAllocation": [
            {
                "accountAddressString": "erd18tudnj2z8vjh0339yu3vrkgzz2jpz8mjq0uhgnmklnap6z33qqeszq2yn4",
                "migrationNonce": 0,
                "sourceShard": 1,
                "destinationShard": 0
            },
            {
                "accountAddressString": "erd1qyu5wthldzr8wx5c9ucg8kjagg0jfs53s8nr3zpz3hypefsdd8ssycr6th",
                "migrationNonce": 0,
                "sourceShard": 1,
                "destinationShard": 2
            }
        ]
    }

    # Send POST request
    response = requests.post(url, json=payload)

    # Check if request was successful (status code 200)
    if response.status_code == 200:
        print("--------REST API call successful--------")
        print(f"----------------------- ACCOUNT ALLOCATION SENT -----------------------")
    else:
        print(f"REST API call failed with status code {response.status_code}")

def sendAllGeneratedTransactions(batch_size, input_directory):
    files = os.listdir(input_directory)
    # Filter out only the JSON files
    json_files = [file for file in files if file.endswith('.json')]
    sorted_json_files = sorted(json_files)
    #print(sorted_json_files)
    
    createOutputCSV()

    total_txs_generated = 0
    current_account_allocation_id = 1 # because accountsAllocationData starts by 1

    for i in range(0, len(sorted_json_files), batch_size):
        batch = sorted_json_files[i:i+batch_size]
        sendBatchOfTransactions(batch, batch_size, i, input_directory)

        # Update the total number of transactions generated
        total_txs_generated += len(batch)
        
        # Check if the threshold for triggering the API call is reached
        #"""
        if (total_txs_generated >= 100 and current_account_allocation_id == 1) or (total_txs_generated >= 500 and current_account_allocation_id == 2): #TODO: RENDERE PARAMETRICO
            # Trigger the call to the REST API
            generate_account_migration_transactions(current_account_allocation_id)
            current_account_allocation_id += 1
            # Reset the total_txs_generated counter
            total_txs_generated = 0
        #"""
    readCSVFile()





def sendAllGeneratedTransactionsAllAtOnce(batch_size, input_directory, delay_in_seconds, num_txs_to_send):
    files = os.listdir(input_directory)
    # Filter out only the JSON files
    json_files = [file for file in files if file.endswith('.json')]
    sorted_json_files = sorted(json_files)
    #print(sorted_json_files)
    
    createOutputCSV()

    total_txs_generated = 0
    current_account_allocation_id = 1 # because accountsAllocationData starts by 1

    num_txs = len(sorted_json_files) if num_txs_to_send == 0 else num_txs_to_send

    processes = []
    txHashes = []

    for i in range(0, num_txs, batch_size): #TODO: RIMETTERE
    #for i in range(0, 100, batch_size): #TODO: TOGLIERE
        batch = sorted_json_files[i:i+batch_size]
        batch_processes = sendBatchOfTransactionsAllAtOnce(batch, batch_size, i, input_directory)
        processes += batch_processes # concatenate batch_processes with the list of all processes

        #TODO: VEDI: https://stackoverflow.com/questions/14533458/python-threading-multiple-bash-subprocesses


    for i in range(0, len(processes), batch_size):
        batch_processes = processes[i:i+batch_size]

        for item in batch_processes:
            if len(item) != 4:
                print("Invalid item found:", item)
            break

        timestamp_start_closing = datetime.now()
        print(f"Closing processes... (timestamp: {timestamp_start_closing})")
        for process, stdout, stderr, infile in batch_processes:
            print("--- PROCESSING infile: " + infile)
            return_code = process.wait()
            if return_code == 0:
                # Process was successful
                output_data = json.loads(stdout) # Parse JSON output
                txHash = output_data.get('emittedTransactionHash')
                txHashes.append(txHash)  # Add txHash to the list
            
                # Update the total number of transactions generated
                total_txs_generated += 1
                
                # Check if the threshold for triggering the API call is reached
                """
                if (total_txs_generated >= 100 and current_account_allocation_id == 1) or (total_txs_generated >= 500 and current_account_allocation_id == 2): #TODO: RENDERE PARAMETRICO
                    # Trigger the call to the REST API
                    generate_account_migration_transactions(current_account_allocation_id)
                    current_account_allocation_id += 1
                    # Reset the total_txs_generated counter
                    total_txs_generated = 0
                """
            
            else:
                # Process failed
                print("Error:", stderr)          
        timestamp_end_closing = datetime.now()
        print(f"Finished closing processes. Moving to the next batch of transactions... (timestamp: {timestamp_end_closing})")    
        
        #saveBatchToCSVAsList(txHashes, timestamp_start_closing, i)
        saveBatchToCSVAsSeparateRow(txHashes, timestamp_start_closing, i)
        time.sleep(delay_in_seconds)
    
    readCSVFile()           



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


#? ------------- UNUSED -------------------
def sendAllGeneratedTransactionsAllAtOnceWithMultithreadingOLD(batch_size, input_directory, delay_in_seconds, num_txs_to_send, num_txs_per_thread):
    files = os.listdir(input_directory)
    # Filter out only the JSON files
    json_files = [file for file in files if file.endswith('.json')]
    sorted_json_files = sorted(json_files)
    #print(sorted_json_files)
    
    createOutputCSV()

    total_txs_generated = 0
    current_account_allocation_id = 1 # because accountsAllocationData starts by 1

    processes = []

    num_txs = len(sorted_json_files) if num_txs_to_send == 0 else num_txs_to_send
    num_threads = math.ceil(num_txs / num_txs_per_thread)


    print(f"Num txs to send: {num_txs_to_send}")
    print(f"Num txs per thread: {num_txs_per_thread}")
    print(f"Num threads: math.ceil({num_txs} / {num_txs_per_thread})  = {num_threads}")


    commands_to_run = get_all_commands_to_run(sorted_json_files, num_txs, input_directory)
    total_processes_by_thread = {}

    
    for i in range(0, num_threads):
        # Use ThreadPoolExecutor to execute mxpy commands in parallel
        with ThreadPoolExecutor() as executor:
            start_index = (i * num_txs_per_thread)
            end_index =  start_index + num_txs_per_thread
            
            timestamp_of_generation = datetime.now()
            print(f"Generating thread {i} to run shell commands:  commands_to_run[{start_index}:{end_index}]           Timestamp: {timestamp_of_generation}")
            #batch_processes = list(executor.map(run_shell_command_without_waiting, commands_to_run[start_index:end_index]))
            # Define a lambda function to pass both command_data and timestamp to run_shell_command_without_waiting #TODO: rimettere questa e rimuovere il parametro del timestamp da run_shell_command_withouth_waiting
            batch_processes = list(executor.map(lambda command_data: run_shell_command_without_waiting(command_data, timestamp_of_generation), commands_to_run[start_index:end_index]))
            total_processes_by_thread[i] = {"timestamp_of_generation": timestamp_of_generation, "batch_processes":batch_processes}


    for i in range(0, num_threads):        
        batchTxHashes = []
        timestamp_start_closing = datetime.now()
        print(f"Closing processes in total_processes_by_thread[{i}] ... (timestamp: {timestamp_start_closing})")

        for process, stdout, stderr, infile, _ in total_processes_by_thread[i]["batch_processes"]: #? _ is a placeholder for the timestamp_of_generation, which is not needed within the for loop
            print("--- PROCESSING infile: " + infile)
            return_code = process.wait()
            if return_code == 0:
                # Process was successful
                output_data = json.loads(stdout) # Parse JSON output
                txHash = output_data.get('emittedTransactionHash')
                batchTxHashes.append(txHash)  # Add txHash to the list
                # Update the total number of transactions generated
                total_txs_generated += 1
                
                # Check if the threshold for triggering the API call is reached
                """
                if (total_txs_generated >= 100 and current_account_allocation_id == 1) or (total_txs_generated >= 500 and current_account_allocation_id == 2): #TODO: RENDERE PARAMETRICO
                    # Trigger the call to the REST API
                    generate_account_migration_transactions(current_account_allocation_id)
                    current_account_allocation_id += 1
                    # Reset the total_txs_generated counter
                    total_txs_generated = 0
                """
            
            else:
                # Process failed
                print("Error:", stderr)          
        timestamp_end_closing = datetime.now()
        print(f"Finished closing processes. Moving to the next batch of transactions... (timestamp: {timestamp_end_closing})")
        timestamp_of_generation = total_processes_by_thread[i]["timestamp_of_generation"]
        saveBatchToCSVAsSeparateRow(batchTxHashes, timestamp_of_generation, i)
        time.sleep(delay_in_seconds)
    
    readCSVFile()       




def sendAllGeneratedTransactionsAllAtOnceWithMultithreading(batch_size, input_directory, delay_in_seconds, num_txs_to_send, num_txs_per_thread):
    files = os.listdir(input_directory)
    # Filter out only the JSON files
    json_files = [file for file in files if file.endswith('.json')]
    sorted_json_files = sorted(json_files)
    #print(sorted_json_files)
    
    createOutputCSV()

    num_txs = len(sorted_json_files) if num_txs_to_send == 0 else num_txs_to_send
    num_threads = math.ceil(num_txs / num_txs_per_thread)


    print(f"Num txs to send: {num_txs_to_send}")
    print(f"Num txs per thread: {num_txs_per_thread}")
    print(f"Num threads: math.ceil({num_txs} / {num_txs_per_thread})  = {num_threads}")


    commands_to_run = get_all_commands_to_run(sorted_json_files, num_txs, input_directory)
    total_processes_by_thread = {}

    
    for i in range(0, num_threads):
        # Use ThreadPoolExecutor to execute mxpy commands in parallel
        with ThreadPoolExecutor() as executor:
            start_index = (i * num_txs_per_thread)
            end_index =  start_index + num_txs_per_thread
            
            timestamp_of_generation = datetime.now()
            print(f"Generating thread {i} to run shell commands:  commands_to_run[{start_index}:{end_index}]           Timestamp: {timestamp_of_generation}")
            #batch_processes = list(executor.map(run_shell_command_without_waiting, commands_to_run[start_index:end_index]))
            # Define a lambda function to pass both command_data and timestamp to run_shell_command_without_waiting #TODO: rimettere questa e rimuovere il parametro del timestamp da run_shell_command_withouth_waiting
            batch_processes = list(executor.map(lambda command_data: run_shell_command_without_waiting(command_data, timestamp_of_generation), commands_to_run[start_index:end_index]))
            timestamp_start_closing = datetime.now()
            print(f"Closing processes in total_processes_by_thread[{i}] ... (timestamp: {timestamp_start_closing})")
            time.sleep(10)
            for process, stdout, stderr, infile, _ in batch_processes:
                batchTxHashes = []
                print(f"--- PROCESSING infile in Thread ({i}): " + infile)
                return_code = process.wait()
                if return_code == 0:
                    # Process was successful
                    output_data = json.loads(stdout) # Parse JSON output
                    txHash = output_data.get('emittedTransactionHash')
                    batchTxHashes.append(txHash)  # Add txHash to the list
                else:
                    # Process failed
                    print("Error:", stderr)                 
            total_processes_by_thread[i] = {"timestamp_of_generation": timestamp_of_generation, "batch_tx_hashes": batchTxHashes}


    for i in range(0, num_threads):
        timestamp_start_closing = datetime.now()
        print(f"Closing processes in total_processes_by_thread[{i}] ... (timestamp: {timestamp_start_closing})")

        timestamp_end_closing = datetime.now()
        print(f"Finished closing processes. Moving to the next batch of transactions... (timestamp: {timestamp_end_closing})")
        timestamp_of_generation_for_thread_txs = total_processes_by_thread[i]["timestamp_of_generation"]
        batch_tx_hashes_for_thread_txs = total_processes_by_thread[i]["batch_tx_hashes"]
        saveBatchToCSVAsSeparateRow(batch_tx_hashes_for_thread_txs, timestamp_of_generation_for_thread_txs, i)
        time.sleep(delay_in_seconds)
    
    readCSVFile()       




def sendAllGeneratedTransactionsAllAtOnceWithMultithreadingByChatGPT(batch_size, input_directory, delay_in_seconds, num_txs_to_send, num_txs_per_thread):
    files = os.listdir(input_directory)
    # Filter out only the JSON files
    json_files = [file for file in files if file.endswith('.json')]
    sorted_json_files = sorted(json_files)
    
    createOutputCSV()

    num_txs = len(sorted_json_files) if num_txs_to_send == 0 else num_txs_to_send
    num_threads = math.ceil(num_txs / num_txs_per_thread)

    print(f"Num txs to send: {num_txs_to_send}")
    print(f"Num txs per thread: {num_txs_per_thread}")
    print(f"Num threads: math.ceil({num_txs} / {num_txs_per_thread})  = {num_threads}")

    commands_to_run = get_all_commands_to_run(sorted_json_files, num_txs, input_directory)
    total_processes_by_thread = {}

    result_queue = queue.Queue()
    lock = threading.Lock()

    def process_batch(thread_id, start_index, end_index):
        timestamp_of_generation = datetime.now()
        print(f"Generating thread {thread_id} to run shell commands: commands_to_run[{start_index}:{end_index}] Timestamp: {timestamp_of_generation}")
        batch_tx_hashes = []
        for command_data in commands_to_run[start_index:end_index]:
            process, stdout, stderr, infile, _ = run_shell_command_without_waiting(command_data, timestamp_of_generation)
            return_code = process.wait()
            if return_code == 0:
                output_data = json.loads(stdout)
                txHash = output_data.get('emittedTransactionHash')
                batch_tx_hashes.append(txHash)
            else:
                print("Error:", stderr)
        result_queue.put((thread_id, timestamp_of_generation, batch_tx_hashes))
        time.sleep(delay_in_seconds)

    threads = []
    for i in range(num_threads):
        start_index = i * num_txs_per_thread
        end_index = min(start_index + num_txs_per_thread, num_txs)
        thread = threading.Thread(target=process_batch, args=(i, start_index, end_index))
        threads.append(thread)
        thread.start()
        time.sleep(delay_in_seconds) #? Generate threads with delay

    for thread in threads:
        thread.join()

    while not result_queue.empty():
        thread_id, timestamp_of_generation, batch_tx_hashes = result_queue.get()
        with lock:
            total_processes_by_thread[thread_id] = {"timestamp_of_generation": timestamp_of_generation, "batch_tx_hashes": batch_tx_hashes}

    for i in range(num_threads):
        timestamp_start_closing = datetime.now()
        print(f"Closing processes in total_processes_by_thread[{i}] ... (timestamp: {timestamp_start_closing})")

        timestamp_end_closing = datetime.now()
        print(f"Finished closing processes. Moving to the next batch of transactions... (timestamp: {timestamp_end_closing})")
        timestamp_of_generation_for_thread_txs = total_processes_by_thread[i]["timestamp_of_generation"]
        batch_tx_hashes_for_thread_txs = total_processes_by_thread[i]["batch_tx_hashes"]
        saveBatchToCSVAsSeparateRow(batch_tx_hashes_for_thread_txs, timestamp_of_generation_for_thread_txs, i)

    readCSVFile()




def executeSingleCurlCommandForTest():
    
    url = "http://localhost:10206/node/send-account-allocation"
    
    payload = {
        "id" : 0,
        "accountAllocation" : [
            {
                "accountAddressString": "erd1qyu5wthldzr8wx5c9ucg8kjagg0jfs53s8nr3zpz3hypefsdd8ssycr6th",
                "migrationNonce": 0,
                "sourceShard": 1,
                "destinationShard": 2
            }
        ]
    }


    command_to_run = f'curl -X POST {url} -H "Content-Type: application/json" --data-raw \'{json.dumps(payload)}\''

    process = subprocess.Popen(command_to_run, shell=True, text=True, stdout=subprocess.PIPE, stderr=subprocess.PIPE)
    stdout, stderr = process.communicate()  # Capture stdout and stderr
    return_code = process.wait()
    if return_code == 0:
        # Process was successful
        output_data = json.loads(stdout) # Parse JSON output
        print("Output data: ", output_data)
    else:
        # Process failed
        print("Error:", stderr)


"""def sendAllGeneratedTransactionsAllAtOnceWithMultithreadingRevisedByMe(batch_size, input_directory, delay_in_seconds, num_txs_to_send, num_txs_per_thread):
    files = os.listdir(input_directory)
    # Filter out only the JSON files
    json_files = [file for file in files if file.endswith('.json')]
    sorted_json_files = sorted(json_files)
    
    createOutputCSV()

    num_txs = len(sorted_json_files) if num_txs_to_send == 0 else num_txs_to_send
    num_threads = math.ceil(num_txs / num_txs_per_thread)

    print(f"Num txs to send: {num_txs_to_send}")
    print(f"Num txs per thread: {num_txs_per_thread}")
    print(f"Num threads: math.ceil({num_txs} / {num_txs_per_thread})  = {num_threads}")

    commands_to_run = get_all_commands_to_run(sorted_json_files, num_txs, input_directory)
    total_processes_by_thread = {}

    result_queue = queue.Queue()
    lock = threading.Lock()

    for i in range(0, num_threads):
        # Use ThreadPoolExecutor to execute mxpy commands in parallel
        with ThreadPoolExecutor() as executor:
            start_index = i * num_txs_per_thread
            end_index = min(start_index + num_txs_per_thread, num_txs)   
            timestamp_of_generation = datetime.now()
            print(f"Generating thread {i} to run shell commands:  commands_to_run[{start_index}:{end_index}]           Timestamp: {timestamp_of_generation}")
            #batch_processes = list(executor.map(run_shell_command_without_waiting, commands_to_run[start_index:end_index]))
            # Define a lambda function to pass both command_data and timestamp to run_shell_command_without_waiting #TODO: rimettere questa e rimuovere il parametro del timestamp da run_shell_command_withouth_waiting
            batch_processes = list(executor.map(lambda command_data: run_shell_command_without_waiting(command_data), commands_to_run[start_index:end_index]))
            #total_processes_by_thread[i] = {"timestamp_of_generation": timestamp_of_generation, "batch_processes":batch_processes}
            result_queue.put((i, batch_processes))"""

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



def sendAllGeneratedTransactionsAllAtOnceWithMultithreadingRevisedByMe(batch_size, input_directory, delay_in_seconds, num_txs_to_send, num_txs_per_thread):
    files = os.listdir(input_directory)
    # Filter out only the JSON files
    json_files = [file for file in files if file.endswith('.json')]
    sorted_json_files = sorted(json_files)
    
    createOutputCSV()

    num_txs = len(sorted_json_files) if num_txs_to_send == 0 else num_txs_to_send
    num_threads = math.ceil(num_txs / num_txs_per_thread)

    print(f"Num txs to send: {num_txs_to_send}")
    print(f"Num txs per thread: {num_txs_per_thread}")
    print(f"Num threads: math.ceil({num_txs} / {num_txs_per_thread})  = {num_threads}")

    commands_to_run = get_all_commands_to_run(sorted_json_files, num_txs, input_directory)
    total_processes_by_thread = {}

    result_queue = queue.Queue()
    lock = threading.Lock()

    with ThreadPoolExecutor(max_workers=num_threads) as executor:
        # Dictionary to store the futures for each thread's tasks
        futures = {}
        for i in range(num_threads):
            start_index = i * num_txs_per_thread
            end_index = min(start_index + num_txs_per_thread, num_txs)
            timestamp_of_generation = datetime.now()
            print(f"Generating thread {i} to run shell commands: commands_to_run[{start_index}:{end_index}] Timestamp: {timestamp_of_generation}")
            # Submitting tasks to executor and storing the futures
            futures[i] = list(executor.map(lambda command_data: run_shell_command_without_waiting(command_data, timestamp_of_generation), commands_to_run[start_index:end_index]))
            #batch_processes = list(executor.map(lambda command_data: run_shell_command_without_waiting(command_data, timestamp_of_generation), commands_to_run[start_index:end_index]))
            
        # Wait for all tasks to complete before continuing
        for i, future in futures.items():
            timestamp_of_generation = datetime.now()
            for process, stdout, stderr, infile in future:
                print(f"--- PROCESSING infile in Thread ({i}): " + infile)
                return_code = process.wait()
                if return_code == 0:
                    output_data = json.loads(stdout)
                    txHash = output_data.get('emittedTransactionHash')
                    total_processes_by_thread.setdefault(i, {"timestamp_of_generation": timestamp_of_generation, "batch_tx_hashes": []})["batch_tx_hashes"].append(txHash)
                else:
                    print("Error:", stderr)
        time.sleep(delay_in_seconds)

    for i in range(num_threads):
        timestamp_start_closing = datetime.now()
        print(f"Closing processes in total_processes_by_thread[{i}] ... (timestamp: {timestamp_start_closing})")

        timestamp_end_closing = datetime.now()
        print(f"Finished closing processes. Moving to the next batch of transactions... (timestamp: {timestamp_end_closing})")
        timestamp_of_generation_for_thread_txs = total_processes_by_thread[i]["timestamp_of_generation"]
        batch_tx_hashes_for_thread_txs = total_processes_by_thread[i]["batch_tx_hashes"]
        saveBatchToCSVAsSeparateRow(batch_tx_hashes_for_thread_txs, timestamp_of_generation_for_thread_txs, i)


def createOutputCSVWithRealExecutionTimestamp():
    # Open the file in write mode
    with open(OUTPUT_CSV, mode='w', newline='') as file:
        writer = csv.writer(file)
        writer.writerow(["file_name","timestamp", "txHashes", "realTimestamp"])  # Write header 


def saveTxsBatchToOutputCSV(batch_outputs):
    timestamp_of_generation = batch_outputs["timestamp_of_generation"]
    # Sort transactions based on file name
    sorted_transactions_data = sorted(batch_outputs["batchTxs"].items(), key=lambda x: x[0])

    # Write to CSV file
    with open(OUTPUT_CSV_WITH_REAL_EXECUTION_TIMESTAMP, mode='a', newline='') as file:
        writer = csv.writer(file)
        for infile, data in sorted_transactions_data:

            # Split the file path by '/'
            infile_parts = infile.split('/')
            # Access the last element of the resulting list
            tx_string = infile_parts[-1]

            #print(f"data: {data}")
            #print(f"infile: {tx_string}")
            #print(f"timestamp_of_generation: {timestamp_of_generation}")
            #print(f"TxHash: {data['txHash']}")
            #print(f"realTimestamp: {data['realTimestamp']}")

            writer.writerow([tx_string, int(timestamp_of_generation), data["txHash"], int(data["realTimestamp"])]) #? faccio timestamp.timestamp() per avere il timestamp in epoch format invece che come datetime


def myLastSender(input_directory, delay_in_seconds, num_txs_to_send, num_txs_per_batch):
    # Define a lock for synchronization
    lock = threading.Lock()

    # Define a function to execute a batch of commands and capture their output
    def execute_batch(commands, batch_outputs, batch_index):
        threads = []
        
        timestamp_of_batch_generation = datetime.now()
        batch_outputs["timestamp_of_generation"] = timestamp_of_batch_generation.timestamp()
        batch_outputs["batchTxs"] = {}

        for command in commands:
            thread = threading.Thread(target=run_command, args=(command, batch_outputs, lock, batch_index))
            threads.append(thread)
            thread.start()
        for thread in threads:
            thread.join()

    # Define a function to run a single command and capture its output
    def run_command(command_data, batch_outputs, lock, batch_index):
        if command_data["command"] == "ACCOUNT_ALLOCATION":
            url = "http://localhost:10206/node/send-account-allocation"
            #generate_account_migration_transactions(account_allocation_payload)
            # Generate Curl command
            command_to_run = [
                'curl',            # The curl command
                '-X', 'POST',       # HTTP method (GET, POST, etc.)
                url,               # URL to send the request to
                '-H', 'Content-Type: application/json',  # Headers if needed
                '-d', command_data["accountAllocationPayload"],  # Data to send in the request body, converting it to JSON string
            ]
            command_to_run = f'curl -X POST {url} -H "Content-Type: application/json" --data-raw \'{command_data["accountAllocationPayload"]}\''

        else:
            command_to_run = command_data["command"]        

        try:
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

    files = os.listdir(input_directory)
    # Filter out only the JSON files
    json_files = [file for file in files if file.endswith('.json')]
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
        end_index = (start_index + num_txs_per_batch)
        timestamp = datetime.now()
        
        print(f"Generating batch {i} of transactions: commands_to_run[{start_index}:{end_index}]        Timestamp: {timestamp}")
        
        batch_commands = commands_to_run[start_index:end_index]
        execute_batch(batch_commands, batch_outputs, i)
        total_txs_by_batch[i] = batch_outputs

    # After all commands have been executed, you can access the output from the 'outputs' dictionary
    for i in range(0, num_batches):
        saveTxsBatchToOutputCSV(total_txs_by_batch[i])


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


def generateAccountsInfoJsonFile():
    # Update shard information for each account
    for account_address in accounts_info:
        shard = get_shard_from_api_call(account_address)
        accounts_info[account_address]['shard'] = shard

    # Write the dictionary to a JSON file
    with open(ACCOUNTS_INFO_JSON_PATH, "w") as json_file:
        json.dump(accounts_info, json_file, indent=4)



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





#! ----- PROVE -----
#generateSingleTransaction()
#sendTransactionFromFile(tx_id="000000001")
#generateRandomTransactions(num_txs=1000)
#addEndTimestampToCSV()
#addTimestampDifferenceToCSV()
#plotData()
#--------sendSingleSavedTxsFromJson()
# --------- generateTransactionsWithCorrectLoadInBatches(num_txs_per_batch=5, num_batches=200, output_dir=TRANSACTIONS_DIRECTORY_WITH_CORRECT_LOAD_BY_BATCH) # 20 tx/batch x 50 batch = 1000 txs


#executeSingleCurlCommandForTest()
#! ------------------ 


#? ---- COMMANDS TO EXECUTE (Correct load overall) ----
#generateTransactionsWithCorrectLoad(num_txs=1000)
#sendAllGeneratedTransactions(batch_size=5, input_directory=TRANSACTIONS_DIRECTORY_WITH_CORRECT_LOAD)
#addStatisticsToCSV()
#plotDataFromStatistics()



#? ---- COMMANDS TO EXECUTE (Correct load by batch) ----

"""generateTransactionsWithCorrectLoadInBatchesWithAccountAllocations(num_txs_per_batch=50, 
                                                                   num_total_txs=20000, 
                                                                   num_txs_threshold_for_account_allocation=6000, 
                                                                   output_dir=TRANSACTIONS_DIRECTORY_WITH_CORRECT_LOAD_BY_BATCH_AND_ACCOUNTS_ALLOCATION, 
                                                                   with_cross_shard_probability=False)"""
# --------- sendAllGeneratedTransactions(batch_size=100, input_directory=TRANSACTIONS_DIRECTORY_WITH_CORRECT_LOAD_BY_BATCH)
#sendAllGeneratedTransactionsAllAtOnce(batch_size=20, input_directory=TRANSACTIONS_DIRECTORY_WITH_CORRECT_LOAD_BY_BATCH_AND_ACCOUNTS_ALLOCATION, delay_in_seconds=5, num_txs_to_send=2000)
#sendAllGeneratedTransactionsAllAtOnceWithMultithreadingRevisedByMe(batch_size=50, input_directory=TRANSACTIONS_DIRECTORY_WITH_CORRECT_LOAD_BY_BATCH_AND_ACCOUNTS_ALLOCATION, delay_in_seconds=5, num_txs_to_send=1000, num_txs_per_thread=50) #? 0: all transactions

#---------- myLastSender(input_directory=TRANSACTIONS_DIRECTORY_WITH_CORRECT_LOAD_BY_BATCH_AND_ACCOUNTS_ALLOCATION, delay_in_seconds=5, num_txs_to_send=4000, num_txs_per_batch=50)
"""myLastSenderWithBarrier(input_directory=TRANSACTIONS_DIRECTORY_WITH_CORRECT_LOAD_BY_BATCH_AND_ACCOUNTS_ALLOCATION, 
                        delay_in_seconds=5, 
                        num_txs_to_send=20000, 
                        num_txs_per_batch=50, 
                        with_AMTs=False)"""

#addStatisticsToCSV()

plotDataFromStatistics(time_threshold=250)

#plotData()

#printStatistics()


#generateAccountsInfoJsonFile()