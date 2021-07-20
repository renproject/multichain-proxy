# Multichain Proxy

## Design Goals

- handle any proxy requests to rpc server
- independent of path/route should be able to proxy to any route of the node, this enables thr proxy to handle nodes that have different paths/routes for different types of rpc calls.
- should be able to handle both jwt and username/password auth mechanisms
- should be able to whitelist rpc calls and block others