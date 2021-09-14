# Multichain Proxy [Dual Lotus]

## ENV Settings

| Env Variable   | Description                                              | Optional                  |   Default Value  (in Docker image)                                                                                                                                                                                                                                                                                                                                                                                                                                                     |
|----------------|----------------------------------------------------------|---------------------------|----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| DEV_MODE       | Enables debug mode for verbose logging                   | Yes                       | "false"                                                                                                                                                                                                                                                                                                                                                                                                                                                                                |
| NODE1_URL      | Node 1 url [scheme://ip:port]                            | No                        | ""                                                                                                                                                                                                                                                                                                                                                                                                                                                                                     |
| NODE2_URL      | Node 1 url [scheme://ip:port]                            | No                        | ""                                                                                                                                                                                                                                                                                                                                                                                                                                                                                     |
| NODE1_TOKEN    | JWT token for Node 1, if node uses jwt tokens            | Yes                       | ""                                                                                                                                                                                                                                                                                                                                                                                                                                                                                     |
| NODE2_TOKEN    | JWT token for Node 2, if node uses jwt tokens            | Yes                       | ""                                                                                                                                                                                                                                                                                                                                                                                                                                                                                     |
| NODE1_USER     | Basic Auth username for Node 1, if node uses basic auth  | Yes                       | ""                                                                                                                                                                                                                                                                                                                                                                                                                                                                                     |
| NODE1_PASSWORD | Basic Auth password for Node 2, if node uses basic auth  | Yes                       | ""                                                                                                                                                                                                                                                                                                                                                                                                                                                                                     |
| NODE2_USER     | Basic Auth username for Node 1, if node uses basic auth  | Yes                       | ""                                                                                                                                                                                                                                                                                                                                                                                                                                                                                     |
| NODE2_PASSWORD | Basic Auth password for Node 2, if node uses basic auth  | Yes                       | ""                                                                                                                                                                                                                                                                                                                                                                                                                                                                                     |
| PROXY_TOKEN    | JWT token for proxy, if proxy needs jwt auth             | Yes                       | ""                                                                                                                                                                                                                                                                                                                                                                                                                                                                                     |
| PROXY_USER     | Basic Auth username for Proxy, if proxy needs basic auth | Yes                       | "user"                                                                                                                                                                                                                                                                                                                                                                                                                                                                                 |
| PROXY_PASSWORD | Basic Auth password for Proxy, if proxy needs basic auth | Yes                       | "password"                                                                                                                                                                                                                                                                                                                                                                                                                                                                             |
| PROXY_METHODS  | Allowed RPC methods (whitelist)                          | Yes, if empty all allowed | "estimatesmartfee,estimatefee,getbestblockhash, getblockchaininfo,getblockcount,getrawtransaction, gettransaction,gettxout,listunspent,sendrawtransaction,eth_blockNumber, eth_call,eth_chainId,eth_estimateGas,eth_gasPrice,eth_getBalance, eth_getBlockByHash,eth_getBlockByNumber,eth_getCode,eth_getLogs, eth_getTransactionByHash,eth_getTransactionCount, eth_getTransactionReceipt,eth_pendingTransactions,eth_sendRawTransaction, eth_sendTransaction,eth_syncing,net_version" |
| PROXY_PATHS    | Allowed routes/paths on the node (whitelist)             | Yes, if empty all allowed | ""                                                                                                                                                                                                                                                                                                                                                                                                                                                                                     |
| CONFIG_PATH_1  | Route used by admin to update proxy 1 config             | No                        | "/proxy/config/1"                                                                                                                                                                                                                                                                                                                                                                                                                                                                      |
| CONFIG_PATH_2  | Route used by admin to update proxy 2 config             | No                        | "/proxy/config/2"                                                                                                                                                                                                                                                                                                                                                                                                                                                                      |
| CONFIG_TOKEN   | JWT token for Config, if route uses jwt token            | Yes                       | ""                                                                                                                                                                                                                                                                                                                                                                                                                                                                                     |
| CONFIG_USER    | Basic Auth username for Config, if route uses basic auth | Yes                       | "user"                                                                                                                                                                                                                                                                                                                                                                                                                                                                                 |
| CONFIG_PASSWORD| Basic Auth password for Config, if route uses basic auth | Yes                       | "password"                                                                                                                                                                                                                                                                                                                                                                                                                                                                             |
| NODE_KEY       | Unique key used to identify the node in proxy db         | No                        | ""                                                                                                                                                                                                                                                                                                                                                                                                                                                                                     |
| DB_SERVER      | MongoDB service URL                                      | Yes                       | "mongodb://mongo-service:27017"                                                                                                                                                                                                                                                                                                                                                                                                                                                        |
| DB_USER        | MongoDB Username                                         | Yes                       | "admin"                                                                                                                                                                                                                                                                                                                                                                                                                                                                                |
| DB_PASSWORD    | MongoDB Password                                         | Yes                       | "password"                                                                                                                                                                                                                                                                                                                                                                                                                                                                             |

## Design Goals

- handle any proxy requests to rpc server
- independent of path/route, should be able to proxy to any route of the node, this enables the proxy to handle nodes that have different paths/routes for different types of rpc calls.
- should have path whitelisting to block unwanted path access
- should be able to whitelist rpc calls and block others
- should be able to handle both jwt and username/password auth mechanisms
- proxy and node can have different auth mechanisms and auth credentials
- can proxy between 2 lotus nodes, if the first call fails the request will be forwarded to the second one