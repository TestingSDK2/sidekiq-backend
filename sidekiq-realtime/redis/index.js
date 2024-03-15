const Redis = require("ioredis");
const { RedisHost, RedisPort, RedisPassword } = require("../constants/env");

// Create a Redis client
const redisClient = new Redis({
  host: RedisHost, // Redis server host
  port: RedisPort, // Redis server port
  password: RedisPassword, // Redis password
});

// Maximum number of retry attempts
const MAX_RETRY_ATTEMPTS = 10;
let retryAttempts = 0;

// Function to check if Redis is connected
function isRedisConnected() {
  return redisClient.status === "ready";
}

// Function to retry connecting to Redis
function retryConnect() {
  if (!isRedisConnected()) {
    if (retryAttempts < MAX_RETRY_ATTEMPTS) {
      retryAttempts++;
      console.log(
        `Attempt ${retryAttempts}/${MAX_RETRY_ATTEMPTS} to reconnect to Redis...`
      );
      redisClient
        .connect()
        .then(() => {
          console.log("Successfully reconnected to Redis");
          // Reset retry counter on successful connection
          retryAttempts = 0;
        })
        .catch((err) => {
          // Retry after a delay
          // Retry after 5 seconds
          setTimeout(retryConnect, 1000);
        });
    } else {
      console.error(
        `Maximum retry attempts (${MAX_RETRY_ATTEMPTS}) reached. Exiting server...`
      );
      // Exit the server process
      process.exit(1);
    }
  }
}

// Event handler for Redis connection established
redisClient.on("connect", () => {
  console.log("Connected to Redis");
});

// Event handler for Redis connection closed
redisClient.on("close", () => {
  console.log("Connection to Redis closed");
});

// Initial connection check and retry
retryConnect();

module.exports = redisClient;
