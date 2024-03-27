import subprocess
import random
import time


accounts_info = {
    "erd1qyu5wthldzr8wx5c9ucg8kjagg0jfs53s8nr3zpz3hypefsdd8ssycr6th" : {
        "username" : "alice",
        "shard" : 1,
        "nonce" : 5,
    },
    "erd1spyavw0956vq68xj8y4tenjpq2wd5a9p2c6j8gsz7ztyrnpxrruqzu66jx" : {
        "username" : "bob",
        "shard" : 1,
        "nonce" : 1,
    },
    "erd1k2s324ww2g0yj38qn2ch2jwctdy8mnfxep94q9arncc6xecg3xaq6mjse8" : {
        "username" : "carol",
        "shard" : 1,
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
        "shard" : 1,
        "nonce" : 1,
    },
    "erd13x29rvmp4qlgn4emgztd8jgvyzdj0p6vn37tqxas3v9mfhq4dy7shalqrx" : {
        "username" : "ivan",
        "shard" : 1,
        "nonce" : 1,
    },
    "erd1fggp5ru0jhcjrp5rjqyqrnvhr3sz3v2e0fm3ktknvlg7mcyan54qzccnan" : {
        "username" : "judy",
        "shard" : 1,
        "nonce" : 1,
    },
    "erd1z32fx8l6wk9tx4j555sxk28fm0clhr0cl88dpyam9zr7kw0hu7hsx2j524" : {
        "username" : "mallory",
        "shard" : 1,
        "nonce" : 1,
    },
    "erd1uv40ahysflse896x4ktnh6ecx43u7cmy9wnxnvcyp7deg299a4sq6vaywa" : {
        "username" : "mike",
        "shard" : 1,
        "nonce" : 1,
    },
# ! ------------------------------- NEW USERS -------------------------------
    # "erd1xrvst0w2sa60f6g59z6rawxzgmpktj6yh9jgmnseceq458ys7kts2xxac4" : {
    #    "username" : "my_wallet",
    #    "shard" : 1, # TODO: CONTROLLA
    #    "nonce" : 1,
    #},
}




def run_shell_command(command, sender_addr):
    try:
        result = subprocess.run(command, shell=True, check=True, capture_output=True, text=True)
        print("Command output: ", result.stdout)
        
    
    except subprocess.CalledProcessError as e:
        print("Error executing command: ", e)
        # If the command fails, print the error output
        print("Command error: ", e.stderr)
        return
    
    accounts_info[sender_addr]["nonce"] += 1
    print("New nonce of sender " + sender_addr + ": " + str(accounts_info[sender_addr]["nonce"]))
    return



def create_command_to_run(nonce, gas_limit, receiver_address, sender_name):    
    command = f"mxpy tx new \
            --nonce={nonce} \
            --data=\"Hello, World\" \
            --gas-limit={gas_limit} \
            --receiver={receiver_address} \
            --pem=~/multiversx-sdk/testwallets/latest/users/{sender_name}.pem \
            --chain=localnet \
            --proxy=http://localhost:7950 \
            --send ;"
    return command


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


def generateRandomTransactions():
    print("Num of accounts: " + str(len(accounts_info)))
    print("0.02x of account is: " + str(len(accounts_info) * 0.02))
    seed_value = 42

    generator = pick_sender_and_receiver(accounts_info, seed_value)

    # Generate pairs of keys
    for _ in range(500):  # Generate 5 pairs
        sender_addr, receiver_addr = next(generator)

        print("----- GENERATING A TX FROM " + accounts_info[sender_addr]["username"] + " TO " + accounts_info[receiver_addr]["username"] + " -----")
        command_to_run = create_command_to_run(
                nonce=accounts_info[sender_addr]["nonce"],
                gas_limit=70000,
                receiver_address=receiver_addr,
                sender_name=accounts_info[sender_addr]["username"]
        )
        run_shell_command(command_to_run, sender_addr)
        time.sleep(0.7)


def generateRandomTransactionsFromSingleSender():
    sender_addr = "erd1qyu5wthldzr8wx5c9ucg8kjagg0jfs53s8nr3zpz3hypefsdd8ssycr6th"
    seed_value = 42

    generator = pick_receiver(accounts_info, seed_value)

    # Generate pairs of keys
    for _ in range(150):
        receiver_addr = next(generator)

        print("----- GENERATING TX FROM " + accounts_info[sender_addr]["username"] + " TO " + accounts_info[receiver_addr]["username"] + " -----")
        command_to_run = create_command_to_run(
                nonce=accounts_info[sender_addr]["nonce"],
                gas_limit=70000,
                receiver_address=receiver_addr,
                sender_name=accounts_info[sender_addr]["username"]
        )
        run_shell_command(command_to_run, sender_addr)
        time.sleep(1)


generateRandomTransactions()
#generateRandomTransactionsFromSingleSender()
