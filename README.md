# Multichain Proxy

## Design Goals

- handle any proxy requests to rpc server
- independent of path/route, should be able to proxy to any route of the node, this enables the proxy to handle nodes that have different paths/routes for different types of rpc calls.
- should have path whitelisting to block unwanted path access
- should be able to whitelist rpc calls and block others
- should be able to handle both jwt and username/password auth mechanisms
- proxy and node can have different auth mechanisms and auth credentials