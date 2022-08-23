FROM golang:1.18-alpine AS builder
# SETUP WORKDIR AWAY FROM $GOTPATH
RUN mkdir /app
WORKDIR /app
RUN apk add git

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
ENV NODE_URL="https://multichain.renproject.io"

# proxy username and password defaults
ENV PROXY_USER="user"
ENV PROXY_PASSWORD="password"

# rpc node username and password defaults
ENV NODE_USER="user"
ENV NODE_PASSWORD="password"

# proxy config username and password defaults
ENV CONFIG_USER="user"
ENV CONFIG_PASSWORD="password"

# jwt token defaults
ENV PROXY_TOKEN=""
ENV NODE_TOKEN=""
ENV CONFIG_TOKEN=""

# whitelisted rpc methods
ENV PROXY_METHODS="estimatesmartfee,estimatefee,getbestblockhash,getblockchaininfo,getblockcount,getrawtransaction,gettransaction,gettxout,listunspent,sendrawtransaction,eth_blockNumber,eth_call,eth_chainId,eth_estimateGas,eth_gasPrice,eth_getBalance,eth_getBlockByHash,eth_getBlockByNumber,eth_getCode,eth_getLogs,eth_getTransactionByHash,eth_getTransactionCount,eth_getTransactionReceipt,eth_pendingTransactions,eth_sendRawTransaction,eth_sendTransaction,eth_syncing,net_version,getHealth,getAccountInfo"

# all paths on the node are accessible
ENV PROXY_PATHS=""

# config path used to remotely update proxy
ENV CONFIG_PATH="/proxy/config"
ENV LOCAL_NODE_PATH="/renlocal"

# config for DB
ENV DB_SERVER="mongodb://mongo-service:27017"
ENV DB_USER="admin"
ENV DB_PASSWORD="password"

# EXPORT PORT FOR HTTP PROXY
EXPOSE 8080

# DEFINE ENTRY FOR RUNNING CONTAINER
ENTRYPOINT ["./proxy"]