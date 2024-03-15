// const WebSocket = require("ws");
const { WebSocketPort } = require("./constants/env");
const { validateJWT } = require("./grpcClients/auth");
const {
  addConnectionId,
  deleteConnection,
  getAllConnectionsIds,
  clearConnectionCache,
} = require("./redis/socketConnections");

const express = require("express");
const http = require("http");
const socketIo = require("socket.io");

// Authenticate user with token and profile id
const authenticateUser = async (headers = {}) => {
  try {
    const authToken = headers.authorization;
    const profileID = headers.profileid;
    const user = await validateJWT(authToken, profileID, false);
    return user;
  } catch (error) {
    return null;
  }
};

const app = express();
const server = http.createServer(app);
const io = socketIo(server);

server.on("close", async () => {
  // Call clearConnectionCache function before closing the server
  // it will remove the cache from redis
  const keysDeleted = await clearConnectionCache();
  console.log(`Deleted ${keysDeleted} keys starting`);
  console.log("Server closed");
});

// Store connected clients' socket IDs
const connectedClients = new Map();

function createWebSocketServer() {
  // Listen for new socket connections
  io.on("connection", async (socket) => {
    const headers = socket.handshake.headers || {};
    const profileID = headers.profileid;
    const connectionId = socket.id;
    // check if token is authorized
    const user = await authenticateUser(headers);
    if (!user) {
      // close socket if user not authorized
      socket.disconnect();
      return;
    }
    console.log("User connected with Id:", connectionId);
    connectedClients.set(socket.id, socket);
    await addConnectionId(profileID, connectionId);

    socket.on("notification", (data) => {
      console.log(`Received notification from ${socket.id}:`, data);
    });

    socket.on("error", (error) => {
      console.error("Socket error:", error);
    });

    // Handle disconnection
    socket.on("disconnect", async () => {
      await deleteConnection(profileID, connectionId);
      console.log(`User disconnected: ${socket.id}`);
    });
  });

  // Start the server
  const PORT = WebSocketPort;
  server.listen(PORT, () => {
    console.log(`Socket.IO server listening on port ${PORT}`);
  });
  return io;
}

// Function to emit event to specific profileIds
// This functions gets all the connection ids from redis and
// transmits the message to all the sockets of profile ids.
async function emitToSpecificClients(event, data, profileIds) {
  try {
    let connectionIds = [];
    for (let i = 0; i < profileIds.length; i++) {
      const ids = await getAllConnectionsIds(profileIds[i]);
      connectionIds = [...connectionIds, ...ids];
    }
    connectionIds.forEach((clientId) => {
      const socket = connectedClients.get(clientId);
      if (socket) {
        socket.emit(event, data);
      } else {
        console.log(`Socket with ID ${clientId} not found`);
      }
    });
  } catch (error) {
    console.log(
      "Unable to deliver event:",
      event,
      JSON.stringify(data),
      " to",
      profileIds
    );
  }
}

module.exports = {
  createWebSocketServer,
  emitToSpecificClients,
  connectedClients,
};
