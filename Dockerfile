FROM golang:1.16-alpine AS builder
# SETUP WORKDIR AWAY FROM $GOTPATH
RUN mkdir /app
WORKDIR /app

# COPY SOURCE
COPY . .

# BUILD
WORKDIR /app
RUN CGO_ENABLED=0 go build -o /output/proxy -v

## DEPLOY STAGE
FROM alpine:latest

COPY --from=builder /output/proxy /

##########
# ENV VARS
##########

# rpc node URL
ENV NODE1_URL="https://multichain.renproject.io"
ENV NODE2_URL="https://multichain.renproject.io"

# proxy username and password defaults
ENV PROXY_USER="user"
ENV PROXY_PASSWORD="password"

# rpc node username and password defaults
ENV NODE1_USER=""
ENV NODE1_PASSWORD=""
ENV NODE2_USER=""
ENV NODE2_PASSWORD=""

# proxy config username and password defaults
ENV CONFIG_USER="user"
ENV CONFIG_PASSWORD="password"

# jwt token defaults
ENV PROXY_TOKEN=""
ENV NODE1_TOKEN=""
ENV NODE2_TOKEN=""
ENV CONFIG_TOKEN=""

# whitelisted rpc methods
ENV PROXY_METHODS="estimatesmartfee,estimatefee,getbestblockhash,getblockchaininfo,getblockcount,getrawtransaction,gettransaction,gettxout,listunspent,sendrawtransaction,eth_blockNumber,eth_call,eth_chainId,eth_estimateGas,eth_gasPrice,eth_getBalance,eth_getBlockByHash,eth_getBlockByNumber,eth_getCode,eth_getLogs,eth_getTransactionByHash,eth_getTransactionCount,eth_getTransactionReceipt,eth_pendingTransactions,eth_sendRawTransaction,eth_sendTransaction,eth_syncing,net_version"

# all paths on the node are accessible
ENV PROXY_PATHS=""

# config path used to remotely update proxy
ENV CONFIG_PATH_1="/proxy/config/1"
ENV CONFIG_PATH_2="/proxy/config/2"

# EXPORT PORT FOR HTTP PROXY
EXPOSE 8080

# DEFINE ENTRY FOR RUNNING CONTAINER
ENTRYPOINT ["./proxy"]