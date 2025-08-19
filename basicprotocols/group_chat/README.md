# Group Chat Blockchain Chat Module / Decentralized Chat Module

## Overview

Group Chat is a blockchain real-time chat module based on the MetaID-v2 protocol, enabling decentralized chat functionality. This module is built on Socket.IO v4 technology stack, supporting group chat and private messaging features, currently supporting blockchain networks such as BTC, MVC, and others. Chat data is stored and synchronized through blockchain transactions, ensuring data decentralization and immutability. The module uses Pebble database for local caching and indexing, and provides RESTful API and Socket.IO interfaces for client access.

## Quick Start

### 1. Integration in Main Project

In the main project's `app.go`, simply add the following code to start the chat module:

```go
// Start with default configuration
go group_chat.Run(man.ChainAdapter)

// Or start with custom configuration
go group_chat.RunWithConfig(man.ChainAdapter)
```

### 2. Configuration File Setup

Add the following configuration items to your project's configuration file:

```toml
# Add "group_chat" to the syncProtocols array
[sync]
syncProtocols = ["group_chat"]

[groupchat]
port = "7568"           # Chat module HTTP service port
host = "0.0.0.0"        # Chat module service listening address
manHost = "https://man.metaid.io"  # MetaID main service address, mainly used to fetch user information

[socket]
isEnble = true          # Whether to enable Socket.IO functionality
port = 7555             # Socket.IO service port
maxConnections = 10000  # Maximum number of connections
maxMemoryMB = 512       # Maximum memory usage (MB)
cleanupInterval = 2     # Cleanup interval (seconds)
connectionTTL = 20      # Connection time to live (seconds)
```

## Configuration Parameters

### GroupChat Configuration

- `port`: HTTP API service port, default is "7568"
- `host`: Service listening address, default is "0.0.0.0"
- `manHost`: MetaID main service address, used for fetching user information

### Socket Configuration

- `isEnble`: Whether to enable Socket.IO functionality, default is true
- `port`: Socket.IO service port, default is 7555
- `maxConnections`: Maximum simultaneous connections, default is 10000
- `maxMemoryMB`: Maximum memory usage, default is 512MB
- `cleanupInterval`: Connection cleanup interval, default is 2 seconds
- `connectionTTL`: Connection time to live, default is 20 seconds

## Database

The chat module uses **Pebble** database to store data, with database files located at:

```
./pebble_data/group_chat_data/
```

The database contains the following main tables:
- Chat records
- Group information
- User information
- Community information

## Interface Information

### HTTP API Interface

- **Port**: 7568
- **Swagger Documentation**: http://localhost:7568/group-chat/docs/index.html#

### Socket.IO Interface

- **Port**: 7555
- **Connection Address**: http://localhost:7555
- **Supported Features**: Real-time message push, online status synchronization

#### Socket.IO Client Connection Example

```javascript
// Connect using Socket.IO v4 client
const socket = io('http://127.0.0.1:7555', {
  path: '/socket.io',
  query: {
    'metaid': 'your_metaid_here'
  }
});

// Connection event listener
socket.on('connect', () => {
  console.log('Connected to server');
});

// Receive messages
socket.on('message', (data) => {
  console.log('Received message:', data);
});

// Disconnect
socket.on('disconnect', () => {
  console.log('Disconnected from server');
});
```

For detailed usage examples, please refer to: `common/socket_util/client-demo.html`

## Main Features

- Group chat
- Private messaging
- Real-time message push
- Online status management
- Message history

## Development Notes

The chat module is an independent microservice that can run standalone or be integrated into the main project. The module provides complete API documentation and Socket.IO interfaces, supporting multiple client access methods.
