const WebSocketPort = process.env.WS_PORT;
const AuthGrpcClientUrl = process.env.AUTH_GRPC_CLIENT;
const RedisPort = process.env.REDIS_PORT;
const RedisHost = process.env.REDIS_HOST;
const RedisPassword = process.env.REDIS_PASSWORD;

module.exports = {
  WebSocketPort,
  AuthGrpcClientUrl,
  RedisHost,
  RedisPort,
  RedisPassword,
};
