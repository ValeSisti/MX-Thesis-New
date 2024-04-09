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
import requests


TRANSACTIONS_DIRECTORY = "./generated_transactions/" #? crearla se non esiste
TRANSACTIONS_DIRECTORY_WITH_CORRECT_LOAD = "./generated_transactions_with_correct_load/" #? crearla se non esiste
TRANSACTIONS_DIRECTORY_WITH_CORRECT_LOAD_BY_BATCH = "./generated_transactions_with_correct_load_by_batch/" #? crearla se non esiste
OUTPUT_CSV = "./output.csv"
OUTPUT_CSV_WITH_END_TIMESTAMP = "./output_with_end_timestamp.csv"
OUTPUT_CSV_WITH_STATISTICS = "./output_with_statistics.csv"
OUTPUT_CSV_WITH_TIMESTAMP_DIFFERENCE = "./output_with_timestamp_difference.csv"


hot_accounts = ["erd1qyu5wthldzr8wx5c9ucg8kjagg0jfs53s8nr3zpz3hypefsdd8ssycr6th"] #alice
hot_sender_probability = 0.5 #TODO modificare a 0.5
cross_shard_probability = 1

accounts_info = {
    "erd1qyu5wthldzr8wx5c9ucg8kjagg0jfs53s8nr3zpz3hypefsdd8ssycr6th" : {
        "username" : "alice",
        "shard" : 1,
        "nonce" : 5,
    },
    "erd1spyavw0956vq68xj8y4tenjpq2wd5a9p2c6j8gsz7ztyrnpxrruqzu66jx" : {
        "username" : "bob",
        "shard" : 0,
        "nonce" : 1,
    },
    "erd1k2s324ww2g0yj38qn2ch2jwctdy8mnfxep94q9arncc6xecg3xaq6mjse8" : {
        "username" : "carol",
        "shard" : 2,
        "nonce" : 1,
    },
    "erd1kyaqzaprcdnv4luvanah0gfxzzsnpaygsy6pytrexll2urtd05ts9vegu7" : {
        "username" : "dan",
        "shard" : 1,
        "nonce" : 1,
    },
    "erd18tudnj2z8vjh0339yu3vrkgzz2jpz8mjq0uhgnmklnap6z33qqeszq2yn4" : {
        "username" : "eve",
        "shard" : 1,
        "nonce" : 1,
    },
    "erd1kdl46yctawygtwg2k462307dmz2v55c605737dp3zkxh04sct7asqylhyv" : {
        "username" : "frank",
        "shard" : 1,
        "nonce" : 1,
    },
    "erd1r69gk66fmedhhcg24g2c5kn2f2a5k4kvpr6jfw67dn2lyydd8cfswy6ede" : {
        "username" : "grace",
        "shard" : 1,
        "nonce" : 1,
    },
    "erd1dc3yzxxeq69wvf583gw0h67td226gu2ahpk3k50qdgzzym8npltq7ndgha" : {
        "username" : "heidi",
        "shard" : 2,
        "nonce" : 1,
    },
    "erd13x29rvmp4qlgn4emgztd8jgvyzdj0p6vn37tqxas3v9mfhq4dy7shalqrx" : {
        "username" : "ivan",
        "shard" : 1,
        "nonce" : 1,
    },
    "erd1fggp5ru0jhcjrp5rjqyqrnvhr3sz3v2e0fm3ktknvlg7mcyan54qzccnan" : {
        "username" : "judy",
        "shard" : 2,
        "nonce" : 1,
    },
    "erd1z32fx8l6wk9tx4j555sxk28fm0clhr0cl88dpyam9zr7kw0hu7hsx2j524" : {
        "username" : "mallory",
        "shard" : 1,
        "nonce" : 1,
    },
    "erd1uv40ahysflse896x4ktnh6ecx43u7cmy9wnxnvcyp7deg299a4sq6vaywa" : {
        "username" : "mike",
        "shard" : 0,
        "nonce" : 1,
    },
# ! ------------------------------- NEW USERS -------------------------------
    # "erd1xrvst0w2sa60f6g59z6rawxzgmpktj6yh9jgmnseceq458ys7kts2xxac4" : {
    #    "username" : "my_wallet",
    #    "shard" : 1, # TODO: CONTROLLA
    #    "nonce" : 1,
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
    print(timestamp)


    try:
        result = subprocess.run(command, shell=True, check=True, capture_output=True, text=True)
        print("Command output: ", result.stdout)
        
    
    except subprocess.CalledProcessError as e:
        print("Error executing command: ", e)
        # If the command fails, print the error output
        print("Command error: ", e.stderr)
        return
    
    if sender_addr != None:
        accounts_info[sender_addr]["nonce"] += 1
        print("New nonce of sender " + sender_addr + ": " + str(accounts_info[sender_addr]["nonce"]))
    
    timestamp = datetime.now()
    print(timestamp)
    return


def run_shell_command_without_waiting(command):
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

def pick_receiver_based_on_cross_shard_probability(all_accounts, sender_addr):
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



def createOutputCSV():
    # Open the file in write mode
    with open(OUTPUT_CSV, mode='w', newline='') as file:
        writer = csv.writer(file)
        writer.writerow(["timestamp", "txHashes"])  # Write header 


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
        return None


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
    df['timestamp_difference'] = df['end_timestamp'] - df['timestamp']
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


    migration_starts_at = 1712169658
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


def plotDataFromStatistics():
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
    ts_differences_greater_than_70 = df[df['timestamp_difference'] > 70]

    print("Transactions with timestamp_difference > 70:")
    for i, row in ts_differences_greater_than_70.iterrows():
        print(
            f"{str(i)})  "
            #+ f"TxHash: {row['txHashes']}  "
            + f"HasCorrespondingAAT: {row['has_corresponding_AAT']}  "
            + f"IsAffectedByAAT: {row['is_affected_by_AAT']}  "
            + f"Sender: {row['sender']}  "
            + f"Receiver: {row['receiver']}  "
            + f"SenderShard: {row['sender_shard']}  "
            + f"ReceiverShard: {row['receiver_shard']}  "
        )


    migration_starts_at = 1712606295 #1712322484 #1712169658
    x_migration_start = migration_starts_at - df['timestamp'].min()
    #print(str(migration_starts_at) + " - " + str(df["timestamp"].min()))
    #print(x_migration_start)

    # Add vertical line at specified timestamp
    vertical_timestamp = 1712322532
    x_vertical_ts = vertical_timestamp - df['timestamp'].min()
    #print(str(vertical_timestamp) + " - " + str(df["timestamp"].min()))
    #print(x_vertical_ts)


    # Convert timestamp to seconds
    df['timestamp'] = df['timestamp'] - df['timestamp'].min()  # Normalize timestamps to start from 0

    # Separate data for each shard
    shard_0_data = df[df['sender_shard'] == 0]
    shard_1_data = df[df['sender_shard'] == 1]
    shard_2_data = df[df['sender_shard'] == 2]

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







#! ----- PROVE -----
#generateSingleTransaction()
#sendTransactionFromFile(tx_id="000000001")
#generateRandomTransactions(num_txs=1000)
#addEndTimestampToCSV()
#addTimestampDifferenceToCSV()
#plotData()
#! ------------------


#? ---- COMMANDS TO EXECUTE (Correct load overall) ----
#generateTransactionsWithCorrectLoad(num_txs=1000)
#sendAllGeneratedTransactions(batch_size=5, input_directory=TRANSACTIONS_DIRECTORY_WITH_CORRECT_LOAD)
#addStatisticsToCSV()
#plotDataFromStatistics()



#? ---- COMMANDS TO EXECUTE (Correct load by batch) ----
#generateTransactionsWithCorrectLoadInBatches(num_txs_per_batch=5, num_batches=200, output_dir=TRANSACTIONS_DIRECTORY_WITH_CORRECT_LOAD_BY_BATCH) # 20 tx/batch x 50 batch = 1000 txs
#sendAllGeneratedTransactions(batch_size=5, input_directory=TRANSACTIONS_DIRECTORY_WITH_CORRECT_LOAD_BY_BATCH)
#addStatisticsToCSV()
plotDataFromStatistics()