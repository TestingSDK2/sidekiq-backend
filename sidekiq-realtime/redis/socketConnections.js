const redis = require("./index");

const WebSocketProfilePrefix = "wsid";

function getIdWithPrefix(id) {
  return `${WebSocketProfilePrefix}_${id}`;
}

// Function to add a connection ID to a user's set
async function addConnectionId(userId, connectionId) {
  try {
    let parsedId = getIdWithPrefix(userId);
    await redis.sadd(parsedId, connectionId);
    console.log("Connection added to Redis:", parsedId, connectionId);
  } catch (error) {
    console.error("Error adding connection to Redis:", error);
  }
}

// Function to get all connection IDs for a user
async function getAllConnectionsIds(userId) {
  try {
    let parsedId = getIdWithPrefix(userId);
    const connections = await redis.smembers(parsedId);
    console.log("Connections for user:", connections);
    return connections;
  } catch (error) {
    console.error("Error getting connections from Redis:", error);
    return [];
  }
}

// Function to delete the connectionId
async function deleteConnection(userId, connectionId) {
  try {
    let parsedId = getIdWithPrefix(userId);
    const result = await redis.srem(parsedId, connectionId);
    console.log("Connection deleted from Redis:", connectionId);
  } catch (error) {
    console.error("Error deleting connection from Redis:", error);
  }
}

// clears all the keys stored for connections
async function clearConnectionCache() {
  let cursor = "0";
  let keysDeleted = 0;
  let pattern = WebSocketProfilePrefix + "*";
  do {
    // Scan for keys matching the pattern
    const result = await redis.scan(cursor, "MATCH", pattern);
    console.log({ result });

    // Update the cursor for the next iteration
    cursor = result[0];

    // Delete keys returned by the scan
    const keys = result[1];
    if (keys.length > 0) {
      const deletedCount = await redis.del(...keys);
      keysDeleted += deletedCount;
    }
  } while (cursor !== "0");

  return keysDeleted;
}

module.exports = {
  addConnectionId,
  getAllConnectionsIds,
  deleteConnection,
  WebSocketProfilePrefix,
  clearConnectionCache,
};
