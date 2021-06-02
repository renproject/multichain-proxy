FROM golang:1.15-alpine AS builder
# SETUP WORKDIR AWAY FROM $GOTPATH
RUN mkdir /app
WORKDIR /app

# COPY SOURCE
COPY . .

# BUILD
WORKDIR /app/cmd/proxy
RUN CGO_ENABLED=0 go build -o /output/proxy -v

## DEPLOY STAGE
FROM alpine:latest

COPY --from=builder /output/proxy /

# LOAD ENV VARS
ENV PROXY_URL="https://multichain.renproject.io"
ENV PROXY_USER="user"
ENV PROXY_PASSWORD="password"
ENV PROXY_METHODS="estimatesmartfee,estimatefee,getbestblockhash,getblockchaininfo,getblockcount,getrawtransaction,gettransaction,gettxout,listunspent,sendrawtransaction,eth_blockNumber,eth_call,eth_chainId,eth_estimateGas,eth_gasPrice,eth_getBalance,eth_getBlockByHash,eth_getBlockByNumber,eth_getCode,eth_getLogs,eth_getTransactionByHash,eth_getTransactionCount,eth_getTransactionReceipt,eth_pendingTransactions,eth_sendRawTransaction,eth_sendTransaction,eth_syncing,net_version"

# EXPORT PORT FOR HTTP PROXY
EXPOSE 8080

# DEFINE ENTRY FOR RUNNING CONTAINER
ENTRYPOINT ["./proxy"]
