import math
import pandas as pd



# Compute queuing txs at a particular iteration. Iteration = consensus round = 6 seconds -> in each iteration, 120 txs are generated (20txs al secondo x 6 secondi = 120


GENERATED_TXS_STATISTICS_CSV = "generated_txs_statistics.csv"

hot_accounts = ["erd1qyu5wthldzr8wx5c9ucg8kjagg0jfs53s8nr3zpz3hypefsdd8ssycr6th", "erd1kyaqzaprcdnv4luvanah0gfxzzsnpaygsy6pytrexll2urtd05ts9vegu7", "erd18tudnj2z8vjh0339yu3vrkgzz2jpz8mjq0uhgnmklnap6z33qqeszq2yn4" ] #alice, dan, eve


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




def computeQueuingTxsAtIteration(i, b): # i = iteration, b = block capacity
    q_0 = 0
    txs_generated_per_iteration = 20

    # Print the current state of iteration and block capacity
    print(f"Entering computeQueuingTxsAtIteration with i={i}, b={b}")

    if i == 1:
        total_txs_1 = txs_generated_per_iteration + q_0
        q_1 = total_txs_1 - b
        q_1 = q_1 if q_1 >= 0 else 0
        print(f"q_1 = {q_1}")
        return q_1

    # Recursive call
    prev_q = computeQueuingTxsAtIteration(i-1, b)
    q_i = txs_generated_per_iteration + prev_q - b
    final_q_i = q_i if q_i >= 0 else 0
    print(f"At iteration {i}, prev_q = {prev_q}, q_i = {q_i}, final_q_i = {final_q_i}")
    return final_q_i





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




# Example usage
S = {0, 1, 2}  # Set of shards
A_hot = {'a', 'b', 'c'}  # Set of hot accounts
n_t = {'a': 10, 'b': 15, 'c': 20}  # Predicted upcoming transactions
q_t_1 = {'a': 5, 'b': 3, 'c': 2}  # Queuing transactions from last epoch
m_t = {0: 30, 1: 40, 2: 50}  # Predicted transactions for each shard
x_t = {
    'a': {0: 1, 1: 0, 2: 0},
    'b': {0: 0, 1: 1, 2: 0},
    'c': {0: 0, 1: 0, 2: 1}
}  # Shard mapping for each account


#V_t = compute_V_t(S, A_hot, n_t, q_t_1, m_t, x_t)
#print("V_t =", V_t)





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

# Example usage:
num_transactions = 1000
hot_account_percentage = 70  # 90% 
hot_account_config = {0: 0, 1: 3, 2: 0}
shard_index = 1  # Index of the shard for which to calculate transactions

result = calculate_transactions_for_shard(num_transactions, hot_account_percentage, hot_account_config, shard_index)

print("Total transactions for shard:", result["total_tx_for_shard"])
print("Percentage of hot transactions for shard:", result["hot_tx_percentage_for_shard"])
print("Percentage of non-hot transactions for shard:", result["non_hot_tx_percentage_for_shard"])
print("Number of transactions generated by hot accounts for shard:", result["num_hot_tx_per_shard"])
print("Number of transactions generated by non-hot accounts for shard:", result["num_non_hot_tx_per_shard"])
print("Number of transactions generated per hot account for shard:", result["num_tx_per_hot_account"])
print("Percentage of transactions generated by a single hot account for shard:", result["hot_tx_percentage_per_account"])







#computeQueuingTxsAtIterationGPTRevised(i=10, b=100)
computeQueuingTxsAtIterationWithPercentage(i=40, b=100, q_0 = 0, txs_generated_per_iteration_for_curr_shard=120, hot_accounts_percentage=0.5, num_hot_accounts=5, single_hot_account_percentage=result["hot_tx_percentage_per_account"], aggregated_light_account_percentage=result["non_hot_tx_percentage_for_shard"])



